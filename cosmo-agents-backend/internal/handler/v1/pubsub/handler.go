package pubsub

import (
	"strconv"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/internal/usecase"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/valyala/fasthttp"
)

// WebSocket upgrader
var upgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true // Allow all origins (configure as needed)
	},
}

// Handler handles Redis Pub/Sub operations following clean architecture
// Now includes WebSocket support matching Python implementation
type Handler struct {
	pubsubUseCase PubSubUseCase
	wsManager     *WebSocketManager
}

// NewHandler creates a new PubSub handler
func NewHandler(pubsubUseCase PubSubUseCase, wsManager *WebSocketManager) *Handler {
	return &Handler{
		pubsubUseCase: pubsubUseCase,
		wsManager:     wsManager,
	}
}

// PublishMessage publishes a message to a Redis topic/channel
// Legacy HTTP endpoint - hidden from Swagger docs
func (h *Handler) PublishMessage(c fiber.Ctx) error {
	var req v1schema.PubSubPublishRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		))
	}

	// Convert to domain input
	input := &domain.PubSubPublishInput{
		Topic:   req.Topic,
		Message: req.Message,
	}

	// Publish message
	if err := h.pubsubUseCase.PublishMessage(c.Context(), input); err != nil {
		if err == usecase.ErrTopicRequired || err == usecase.ErrMessageRequired {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				err.Error(),
				"",
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to publish message",
			err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"topic":   req.Topic,
		"message": "Message published successfully",
	}))
}

// SubscribeToTopic subscribes to a Redis topic and returns messages
// Legacy HTTP endpoint - hidden from Swagger docs
func (h *Handler) SubscribeToTopic(c fiber.Ctx) error {
	topic := c.Params("topic")
	if topic == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Topic is required",
			"",
		))
	}

	// Parse timeout from query parameter
	timeoutStr := c.Query("timeout", "30")
	timeoutSec, err := strconv.Atoi(timeoutStr)
	if err != nil {
		timeoutSec = 30
	}
	timeout := time.Duration(timeoutSec) * time.Second

	// Subscribe to topic
	message, err := h.pubsubUseCase.SubscribeToTopic(c.Context(), topic, timeout)
	if err != nil {
		if err == usecase.ErrSubscriptionTimeout {
			return c.Status(fiber.StatusRequestTimeout).JSON(schema.ErrorResponse(
				fiber.StatusRequestTimeout,
				"Request timeout",
				"",
			))
		}
		if err == usecase.ErrTopicRequired {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				err.Error(),
				"",
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to subscribe to topic",
			err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"topic":   message.Topic,
		"message": message.Payload,
	}))
}

// GetTopicInfo returns information about a topic
// Legacy HTTP endpoint - hidden from Swagger docs
func (h *Handler) GetTopicInfo(c fiber.Ctx) error {
	topic := c.Params("topic")
	if topic == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Topic is required",
			"",
		))
	}

	// Get topic info
	info, err := h.pubsubUseCase.GetTopicInfo(c.Context(), topic)
	if err != nil {
		if err == usecase.ErrTopicRequired {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				err.Error(),
				"",
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to get topic info",
			err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"topic":       info.Topic,
		"subscribers": info.Subscribers,
	}))
}

// =====================================================
// NEW METHODS - Match Python implementation
// =====================================================

// HandleWebSocketSubscribe handles WebSocket connection for topic subscription
// Matches Python: @router.websocket("/{topic}/subcribe")
// WebSocket endpoint - not shown in Swagger docs (WebSocket not well supported in Swagger)
func (h *Handler) HandleWebSocketSubscribe(c fiber.Ctx) error {
	topic := c.Params("topic")

	// Check if it's a WebSocket upgrade request
	if string(c.Request().Header.Peek("Upgrade")) != "websocket" {
		return fiber.ErrUpgradeRequired
	}

	// Upgrade to WebSocket
	err := upgrader.Upgrade(c.RequestCtx(), func(conn *websocket.Conn) {
		// Topic is captured by closure
		h.websocketHandler(conn, topic)
	})

	if err != nil {
		logger.Logger.Error().Err(err).Msg("WebSocket upgrade failed")
		return err
	}

	return nil
}

// websocketHandler handles the WebSocket connection lifecycle
func (h *Handler) websocketHandler(conn *websocket.Conn, topic string) {
	log := logger.Logger.With().
		Str("topic", topic).
		Str("remote_addr", conn.RemoteAddr().String()).
		Logger()

	log.Info().Msg("WebSocket connection established")

	// Add subscriber to topic
	err := h.wsManager.AddSubscriberToTopic(topic, conn)
	if err != nil {
		log.Error().Err(err).Msg("Failed to add subscriber")
		conn.WriteJSON(fiber.Map{"error": err.Error()})
		conn.Close()
		return
	}

	// Ensure cleanup on disconnect
	defer func() {
		h.wsManager.RemoveSubscriberFromTopic(topic, conn)
		log.Info().Msg("WebSocket connection closed")
	}()

	// Keep connection alive - just wait for messages (like Python)
	for {
		// Read message (or wait for disconnect)
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Debug().Err(err).Msg("WebSocket read error, disconnecting")
			break
		}
		// We don't process incoming messages, just keep connection alive
	}
}

// PublishToRoom publishes a message to a topic via URL params
// Matches Python: @router.post("/{topic}/publish/{message}")
// @Summary Publish message to room
// @Description Publishes a message to a topic (URL params version)
// @Tags pubsub
// @Accept json
// @Produce json
// @Param topic path string true "Topic name"
// @Param message path string true "Message to publish"
// @Success 200 {object} map[string]interface{} "Message published successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v1/pubsub/{topic}/publish/{message} [post]
func (h *Handler) PublishToRoom(c fiber.Ctx) error {
	topic := c.Params("topic")
	message := c.Params("message")

	if topic == "" || message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Topic and message are required",
			"",
		))
	}

	// Broadcast to room using WebSocketManager
	err := h.wsManager.BroadcastToRoom(topic, message)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to publish message",
			err.Error(),
		))
	}

	// Match Python response format
	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Message " + message + " sent to topic " + topic,
	})
}

// DeleteTopic deletes a topic and closes all connections
// Matches Python: @router.delete("/{topic}")
// @Summary Delete topic
// @Description Deletes a topic and closes all WebSocket connections
// @Tags pubsub
// @Accept json
// @Produce json
// @Param topic path string true "Topic name"
// @Success 200 {object} map[string]interface{} "Topic deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Topic not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v1/pubsub/{topic} [delete]
func (h *Handler) DeleteTopic(c fiber.Ctx) error {
	topic := c.Params("topic")

	if topic == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Topic is required",
			"",
		))
	}

	// Delete topic using WebSocketManager
	err := h.wsManager.DeleteTopic(topic)
	if err != nil {
		if err == ErrTopicNotExist {
			return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
				fiber.StatusNotFound,
				"Topic not found",
				err.Error(),
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to delete topic",
			err.Error(),
		))
	}

	// Match Python response format
	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Topic " + topic + " deleted.",
	})
}
