package pubsub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/usecase"
)

// MockPubSubUseCase is a mock implementation of PubSubUseCase interface
type MockPubSubUseCase struct {
	mock.Mock
}

func (m *MockPubSubUseCase) PublishMessage(ctx context.Context, input *domain.PubSubPublishInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockPubSubUseCase) SubscribeToTopic(ctx context.Context, topic string, timeout time.Duration) (*domain.PubSubMessage, error) {
	args := m.Called(ctx, topic, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PubSubMessage), args.Error(1)
}

func (m *MockPubSubUseCase) GetTopicInfo(ctx context.Context, topic string) (*domain.PubSubTopicInfo, error) {
	args := m.Called(ctx, topic)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PubSubTopicInfo), args.Error(1)
}

func TestPublishMessage_Success(t *testing.T) {
	mockUseCase := new(MockPubSubUseCase)
	handler := NewHandler(mockUseCase, nil) // nil WebSocketManager for legacy HTTP tests

	app := fiber.New()
	app.Post("/pubsub/publish", handler.PublishMessage)

	reqBody := map[string]interface{}{
		"topic":   "test-topic",
		"message": "test message",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	mockUseCase.On("PublishMessage", mock.Anything, mock.MatchedBy(func(input *domain.PubSubPublishInput) bool {
		return input.Topic == "test-topic" && input.Message == "test message"
	})).Return(nil)

	req := httptest.NewRequest("POST", "/pubsub/publish", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	mockUseCase.AssertExpectations(t)
}

func TestPublishMessage_InvalidRequest(t *testing.T) {
	mockUseCase := new(MockPubSubUseCase)
	handler := NewHandler(mockUseCase, nil)

	app := fiber.New()
	app.Post("/pubsub/publish", handler.PublishMessage)

	req := httptest.NewRequest("POST", "/pubsub/publish", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestPublishMessage_TopicRequired(t *testing.T) {
	mockUseCase := new(MockPubSubUseCase)
	handler := NewHandler(mockUseCase, nil)

	app := fiber.New()
	app.Post("/pubsub/publish", handler.PublishMessage)

	reqBody := map[string]interface{}{
		"topic":   "",
		"message": "test message",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	mockUseCase.On("PublishMessage", mock.Anything, mock.Anything).Return(usecase.ErrTopicRequired)

	req := httptest.NewRequest("POST", "/pubsub/publish", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSubscribeToTopic_Success(t *testing.T) {
	mockUseCase := new(MockPubSubUseCase)
	handler := NewHandler(mockUseCase, nil)

	app := fiber.New()
	app.Get("/pubsub/subscribe/:topic", handler.SubscribeToTopic)

	expectedMessage := &domain.PubSubMessage{
		Topic:     "test-topic",
		Payload:   "test payload",
		Timestamp: time.Now(),
	}

	mockUseCase.On("SubscribeToTopic", mock.Anything, "test-topic", 30*time.Second).Return(expectedMessage, nil)

	req := httptest.NewRequest("GET", "/pubsub/subscribe/test-topic", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	mockUseCase.AssertExpectations(t)
}

func TestSubscribeToTopic_Timeout(t *testing.T) {
	mockUseCase := new(MockPubSubUseCase)
	handler := NewHandler(mockUseCase, nil)

	app := fiber.New()
	app.Get("/pubsub/subscribe/:topic", handler.SubscribeToTopic)

	mockUseCase.On("SubscribeToTopic", mock.Anything, "test-topic", 30*time.Second).Return(nil, usecase.ErrSubscriptionTimeout)

	req := httptest.NewRequest("GET", "/pubsub/subscribe/test-topic", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusRequestTimeout, resp.StatusCode)

	mockUseCase.AssertExpectations(t)
}

func TestSubscribeToTopic_EmptyTopic(t *testing.T) {
	mockUseCase := new(MockPubSubUseCase)
	handler := NewHandler(mockUseCase, nil)

	app := fiber.New()
	app.Get("/pubsub/subscribe/:topic", handler.SubscribeToTopic)

	req := httptest.NewRequest("GET", "/pubsub/subscribe/", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode) // Fiber returns 404 for empty param
}

func TestGetTopicInfo_Success(t *testing.T) {
	mockUseCase := new(MockPubSubUseCase)
	handler := NewHandler(mockUseCase, nil)

	app := fiber.New()
	app.Get("/pubsub/topic/:topic", handler.GetTopicInfo)

	expectedInfo := &domain.PubSubTopicInfo{
		Topic:       "test-topic",
		Subscribers: 5,
	}

	mockUseCase.On("GetTopicInfo", mock.Anything, "test-topic").Return(expectedInfo, nil)

	req := httptest.NewRequest("GET", "/pubsub/topic/test-topic", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	mockUseCase.AssertExpectations(t)
}

func TestGetTopicInfo_Error(t *testing.T) {
	mockUseCase := new(MockPubSubUseCase)
	handler := NewHandler(mockUseCase, nil)

	app := fiber.New()
	app.Get("/pubsub/topic/:topic", handler.GetTopicInfo)

	mockUseCase.On("GetTopicInfo", mock.Anything, "test-topic").Return(nil, errors.New("redis error"))

	req := httptest.NewRequest("GET", "/pubsub/topic/test-topic", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	mockUseCase.AssertExpectations(t)
}
