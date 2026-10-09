package daily_action

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// SSEEvent represents an event to be sent via SSE.
type SSEEvent struct {
	ID        string          `json:"id"`
	EventType string          `json:"event"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

// SSEManager manages per-user SSE connections with Redis pub/sub for cross-instance delivery.
type SSEManager struct {
	connections map[uuid.UUID][]chan *SSEEvent
	mu          sync.RWMutex
	redisClient *redis.Client
	pubsubs     map[uuid.UUID]*redis.PubSub
	cancelFuncs map[uuid.UUID]context.CancelFunc
}

// NewSSEManager creates a new SSE manager.
func NewSSEManager(redisClient *redis.Client) *SSEManager {
	return &SSEManager{
		connections: make(map[uuid.UUID][]chan *SSEEvent),
		redisClient: redisClient,
		pubsubs:     make(map[uuid.UUID]*redis.PubSub),
		cancelFuncs: make(map[uuid.UUID]context.CancelFunc),
	}
}

// Register creates a new SSE event channel for a user.
func (m *SSEManager) Register(ctx context.Context, userID uuid.UUID) chan *SSEEvent {
	ch := make(chan *SSEEvent, 64)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.connections[userID] = append(m.connections[userID], ch)

	// Start Redis subscriber for this user if not already running
	if _, exists := m.pubsubs[userID]; !exists {
		topic := fmt.Sprintf("sse:daily-actions:%s", userID.String())
		pubsub := m.redisClient.Subscribe(ctx, topic)
		m.pubsubs[userID] = pubsub

		subCtx, cancel := context.WithCancel(context.Background())
		m.cancelFuncs[userID] = cancel
		go m.redisReader(subCtx, userID, pubsub)
	}

	return ch
}

// Unregister removes an SSE event channel for a user.
func (m *SSEManager) Unregister(userID uuid.UUID, ch chan *SSEEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	channels := m.connections[userID]
	for i, c := range channels {
		if c == ch {
			m.connections[userID] = append(channels[:i], channels[i+1:]...)
			close(ch)
			break
		}
	}

	// Cleanup Redis subscriber if no more connections
	if len(m.connections[userID]) == 0 {
		delete(m.connections, userID)
		if cancel, exists := m.cancelFuncs[userID]; exists {
			cancel()
			delete(m.cancelFuncs, userID)
		}
		if pubsub, exists := m.pubsubs[userID]; exists {
			pubsub.Close()
			delete(m.pubsubs, userID)
		}
	}
}

// PublishEvent sends an event to all connections for a user (via Redis for cross-instance).
func (m *SSEManager) PublishEvent(ctx context.Context, userID uuid.UUID, event *SSEEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal SSE event: %w", err)
	}

	topic := fmt.Sprintf("sse:daily-actions:%s", userID.String())
	if err := m.redisClient.Publish(ctx, topic, string(data)).Err(); err != nil {
		return fmt.Errorf("publish SSE event: %w", err)
	}

	return nil
}

// deliverLocal sends an event to all local connections for a user.
// Giữ RLock suốt vòng gửi: send là non-blocking (select/default) nên không
// giữ lock lâu, và Unregister (cần write lock) không thể close(ch) giữa chừng
// — tránh panic "send on closed channel".
func (m *SSEManager) deliverLocal(userID uuid.UUID, event *SSEEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ch := range m.connections[userID] {
		select {
		case ch <- event:
		default:
			logger.Logger.Warn().
				Str("user_id", userID.String()).
				Msg("SSE channel full, dropping event")
		}
	}
}

// redisReader continuously reads from Redis pub/sub and delivers to local connections.
func (m *SSEManager) redisReader(ctx context.Context, userID uuid.UUID, pubsub *redis.PubSub) {
	// Goroutine nền không đi qua middleware recovery: một panic ở đây
	// (ví dụ send vào channel đã đóng) sẽ giết cả process nếu không recover.
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Error().
				Interface("panic", r).
				Str("user_id", userID.String()).
				Msg("recovered panic in SSE redisReader")
		}
	}()
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			// Sanitize payload: replace raw control chars that break JSON parsing (from email content)
			payload := strings.ReplaceAll(msg.Payload, "\r", "\\r")
			payload = strings.ReplaceAll(payload, "\n", "\\n")
			payload = strings.ReplaceAll(payload, "\t", "\\t")
			var event SSEEvent
			if err := json.Unmarshal([]byte(payload), &event); err != nil {
				logger.Logger.Error().Err(err).Msg("failed to unmarshal SSE event from Redis")
				continue
			}
			m.deliverLocal(userID, &event)
		}
	}
}

// Close shuts down all connections and Redis subscribers.
func (m *SSEManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, channels := range m.connections {
		for _, ch := range channels {
			close(ch)
		}
		delete(m.connections, userID)
	}

	for userID, cancel := range m.cancelFuncs {
		cancel()
		delete(m.cancelFuncs, userID)
	}

	for userID, pubsub := range m.pubsubs {
		pubsub.Close()
		delete(m.pubsubs, userID)
	}
}
