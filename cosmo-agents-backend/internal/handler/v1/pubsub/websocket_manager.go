package pubsub

import (
	"context"
	"errors"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

var (
	// ErrTopicNotExist is returned when trying to subscribe to a non-existent topic
	ErrTopicNotExist = errors.New("topic does not exist")
)

// wsConn wraps a WebSocket connection with a write mutex to prevent concurrent writes
type wsConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// WriteMessage safely writes a message to the WebSocket connection
func (w *wsConn) WriteMessage(messageType int, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(messageType, data)
}

// Close safely closes the WebSocket connection
func (w *wsConn) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.Close()
}

// WebSocketManager manages WebSocket connections and Redis pub/sub
// Equivalent to Python's PubSubController
type WebSocketManager struct {
	// rooms stores wrapped WebSocket connections per topic
	rooms map[string][]*wsConn

	// subscribers stores Redis PubSub per topic (like Python's self.subcribers)
	subscribers map[string]*redis.PubSub

	// cancelFuncs stores context cancel functions per topic (like Python's self.threads)
	cancelFuncs map[string]context.CancelFunc

	// redisClient is the Redis client for pub/sub operations
	redisClient *redis.Client

	// mu protects concurrent access to maps
	mu sync.RWMutex
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(redisClient *redis.Client) *WebSocketManager {
	return &WebSocketManager{
		rooms:       make(map[string][]*wsConn),
		subscribers: make(map[string]*redis.PubSub),
		cancelFuncs: make(map[string]context.CancelFunc),
		redisClient: redisClient,
	}
}

// AddSubscriberToTopic adds a WebSocket connection to a topic
// Creates the topic if it doesn't exist (fixed: allows subscribe-before-publish)
func (m *WebSocketManager) AddSubscriberToTopic(topic string, conn *websocket.Conn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Wrap connection with write mutex for thread-safe writes
	wrappedConn := &wsConn{conn: conn}

	// Create topic if it doesn't exist (allows first subscriber to create topic)
	if _, exists := m.rooms[topic]; !exists {
		m.rooms[topic] = []*wsConn{}

		// Subscribe to Redis for this topic
		pubsub := m.redisClient.Subscribe(context.Background(), topic)
		m.subscribers[topic] = pubsub

		logger.Logger.Info().
			Str("topic", topic).
			Msg("Topic created by first subscriber")
	}

	// If no background reader exists, start one
	if _, exists := m.cancelFuncs[topic]; !exists {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelFuncs[topic] = cancel
		go m.pubsubDataReader(ctx, topic)
	}

	// Add wrapped connection to room
	m.rooms[topic] = append(m.rooms[topic], wrappedConn)

	logger.Logger.Info().
		Str("topic", topic).
		Int("total_subscribers", len(m.rooms[topic])).
		Msg("Subscriber added to topic")

	return nil
}

// BroadcastToRoom publishes a message to a topic
// Equivalent to Python's broadcast_to_room
func (m *WebSocketManager) BroadcastToRoom(topic string, message string) error {
	// Just publish to Redis - don't create subscriptions here
	// Subscriptions are only created when WebSocket clients subscribe
	// This prevents resource leaks when publishing to topics with no subscribers
	err := m.redisClient.Publish(context.Background(), topic, message).Err()
	if err != nil {
		logger.Logger.Error().Err(err).Str("topic", topic).Msg("Failed to publish to Redis")
		return err
	}

	logger.Logger.Info().
		Str("topic", topic).
		Str("message", message).
		Msg("Message published to topic")

	return nil
}

// RemoveSubscriberFromTopic removes a WebSocket connection from a topic
// Equivalent to Python's remove_subcriber_from_topic
func (m *WebSocketManager) RemoveSubscriberFromTopic(topic string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if connections, exists := m.rooms[topic]; exists {
		// Find and remove the connection
		for i, c := range connections {
			if c.conn == conn {
				m.rooms[topic] = append(connections[:i], connections[i+1:]...)
				logger.Logger.Info().
					Str("topic", topic).
					Int("remaining_subscribers", len(m.rooms[topic])).
					Msg("Subscriber removed from topic")
				break
			}
		}

		// If room is empty, cleanup (like Python)
		if len(m.rooms[topic]) == 0 {
			m.cleanupTopic(topic)
		}
	}
}

// DeleteTopic removes a topic and closes all connections
// Equivalent to Python's delete_topic
func (m *WebSocketManager) DeleteTopic(topic string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if connections, exists := m.rooms[topic]; exists {
		// Close all WebSocket connections
		for _, conn := range connections {
			conn.Close()
		}

		logger.Logger.Info().
			Str("topic", topic).
			Int("closed_connections", len(connections)).
			Msg("Closing all connections for topic")

		// Cleanup resources
		m.cleanupTopic(topic)

		return nil
	}

	return ErrTopicNotExist
}

// cleanupTopic cleans up resources for a topic
// Must be called with lock held
func (m *WebSocketManager) cleanupTopic(topic string) {
	// Cancel background reader (like Python's thread.cancel())
	if cancel, exists := m.cancelFuncs[topic]; exists {
		cancel()
		delete(m.cancelFuncs, topic)
		logger.Logger.Debug().Str("topic", topic).Msg("Cancelled background reader")
	}

	// Unsubscribe from Redis
	if pubsub, exists := m.subscribers[topic]; exists {
		pubsub.Unsubscribe(context.Background(), topic)
		pubsub.Close()
		delete(m.subscribers, topic)
		logger.Logger.Debug().Str("topic", topic).Msg("Unsubscribed from Redis")
	}

	// Remove room
	delete(m.rooms, topic)
	logger.Logger.Info().Str("topic", topic).Msg("Topic cleaned up")
}

// pubsubDataReader continuously reads from Redis and broadcasts to WebSockets
// Equivalent to Python's _pubsub_data_reader
func (m *WebSocketManager) pubsubDataReader(ctx context.Context, topic string) {
	logger.Logger.Info().Str("topic", topic).Msg("Starting background reader for topic")

	for {
		select {
		case <-ctx.Done():
			logger.Logger.Info().Str("topic", topic).Msg("Background reader stopped")
			return
		default:
			// Get subscriber
			m.mu.RLock()
			pubsub, exists := m.subscribers[topic]
			m.mu.RUnlock()

			if !exists {
				logger.Logger.Warn().Str("topic", topic).Msg("Subscriber not found, stopping reader")
				return
			}

			// Receive message from Redis
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					// Context cancelled, exit gracefully
					return
				}
				logger.Logger.Error().Err(err).Str("topic", topic).Msg("Error receiving message from Redis")
				continue
			}

			// Get all connections for this topic
			m.mu.RLock()
			connections := make([]*wsConn, len(m.rooms[topic]))
			copy(connections, m.rooms[topic])
			m.mu.RUnlock()

			if len(connections) == 0 {
				continue
			}

			// Decode and broadcast message
			data := msg.Payload
			logger.Logger.Debug().
				Str("topic", topic).
				Str("message", data).
				Int("subscribers", len(connections)).
				Msg("Broadcasting message to subscribers")

			// Send to all connections
			for _, conn := range connections {
				// Send in goroutine to avoid blocking
				// The wsConn wrapper ensures thread-safe writes with per-connection mutex
				go func(c *wsConn, message string) {
					if err := c.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
						logger.Logger.Error().
							Err(err).
							Str("topic", topic).
							Msg("Failed to send message to WebSocket")
					}
				}(conn, data)
			}
		}
	}
}

// GetTopicInfo returns information about a topic
func (m *WebSocketManager) GetTopicInfo(topic string) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if connections, exists := m.rooms[topic]; exists {
		return len(connections), true
	}
	return 0, false
}

// GetAllTopics returns all active topics
func (m *WebSocketManager) GetAllTopics() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	topics := make([]string, 0, len(m.rooms))
	for topic := range m.rooms {
		topics = append(topics, topic)
	}
	return topics
}

// Close closes all connections and cleans up resources
func (m *WebSocketManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for topic := range m.rooms {
		m.cleanupTopic(topic)
	}
}
