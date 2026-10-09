package summarizer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	summaryService "github.com/rockship/cosmo-agents-go/internal/service/summary"
)

const (
	TypeSummarizeEmail        = "email:summarize"
	TypeSummarizeConversation = "conversation:summarize"
)

// SummarizeEmailPayload is the payload for email summarization task
type SummarizeEmailPayload struct {
	EmailID     uuid.UUID `json:"email_id"`
	DetailLevel float64   `json:"detail_level"` // 0.0 = most compressed, 1.0 = most detailed
}

// SummarizeConversationPayload is the payload for conversation summarization task
type SummarizeConversationPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	DetailLevel    float64   `json:"detail_level"`
}

// Worker processes email and conversation summarization tasks
type Worker struct {
	// Repositories
	emailRepo        *emailRepo.Repository
	conversationRepo *conversationRepo.ConversationRepository

	// Services
	summarizer *summaryService.Summarizer

	logger *zerolog.Logger
}

// New creates a new summarizer worker
func New(
	emailRepo *emailRepo.Repository,
	conversationRepo *conversationRepo.ConversationRepository,
	summarizer *summaryService.Summarizer,
	log *zerolog.Logger,
) *Worker {
	return &Worker{
		emailRepo:        emailRepo,
		conversationRepo: conversationRepo,
		summarizer:       summarizer,
		logger:           log,
	}
}

// ProcessTask processes a summarization task
func (w *Worker) ProcessTask(ctx context.Context, task *asynq.Task) error {
	switch task.Type() {
	case TypeSummarizeEmail:
		return w.processSummarizeEmail(ctx, task)
	case TypeSummarizeConversation:
		return w.processSummarizeConversation(ctx, task)
	default:
		return fmt.Errorf("unknown task type: %s", task.Type())
	}
}

// processSummarizeEmail summarizes a single email
func (w *Worker) processSummarizeEmail(ctx context.Context, task *asynq.Task) error {
	var payload SummarizeEmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("Failed to unmarshal summarize email payload")
		return fmt.Errorf("json.Unmarshal failed: %w", err)
	}

	w.logger.Info().
		Str("email_id", payload.EmailID.String()).
		Float64("detail_level", payload.DetailLevel).
		Msg("Processing email summarization task")

	// Fetch the email
	email, err := w.emailRepo.FindByID(ctx, payload.EmailID)
	if err != nil {
		w.logger.Error().
			Str("email_id", payload.EmailID.String()).
			Err(err).
			Msg("Email not found")
		return fmt.Errorf("email not found: %w", err)
	}

	// Summarize email content
	summaries, err := w.summarizer.Summarize(ctx, email.Content, payload.DetailLevel, true)
	if err != nil {
		w.logger.Error().
			Str("email_id", payload.EmailID.String()).
			Err(err).
			Msg("Failed to summarize email")
		return fmt.Errorf("summarization failed: %w", err)
	}

	// The Email model has no column to hold a summary, so the result can only
	// be logged and thrown away. Summarize the conversation instead — that
	// handler persists to conversation.cmetadata.
	var summary string
	if len(summaries) > 0 {
		summary = summaries[0].Compressed
	}

	w.logger.Warn().
		Str("email_id", payload.EmailID.String()).
		Int("summary_length", len(summary)).
		Msg("email:summarize has nowhere to store its result — use conversation:summarize")

	return fmt.Errorf("email summary storage not implemented, use conversation:summarize: %w", asynq.SkipRetry)
}

// processSummarizeConversation summarizes an entire conversation thread
func (w *Worker) processSummarizeConversation(ctx context.Context, task *asynq.Task) error {
	var payload SummarizeConversationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("Failed to unmarshal summarize conversation payload")
		return fmt.Errorf("json.Unmarshal failed: %w", err)
	}

	w.logger.Info().
		Str("conversation_id", payload.ConversationID.String()).
		Float64("detail_level", payload.DetailLevel).
		Msg("Processing conversation summarization task")

	// Fetch all emails in the conversation
	filter := baseRepo.Filter{
		"conversation_id": payload.ConversationID,
		"is_deleted":      false,
	}
	pagination := &baseRepo.PaginationParams{
		Limit: 100, // Get up to 100 emails
	}
	emailsResult, err := w.emailRepo.FindAll(ctx, filter, pagination)
	if err != nil {
		w.logger.Error().
			Str("conversation_id", payload.ConversationID.String()).
			Err(err).
			Msg("Failed to fetch conversation emails")
		return fmt.Errorf("failed to fetch emails: %w", err)
	}

	if len(emailsResult.List) == 0 {
		w.logger.Warn().
			Str("conversation_id", payload.ConversationID.String()).
			Msg("No emails found in conversation")
		return nil
	}

	// Concatenate all email contents
	var fullContent string
	for i, email := range emailsResult.List {
		fullContent += fmt.Sprintf("\n--- Email %d ---\n%s", i+1, email.Content)
	}

	// Summarize the entire conversation
	summaries, err := w.summarizer.Summarize(ctx, fullContent, payload.DetailLevel, true)
	if err != nil {
		w.logger.Error().
			Str("conversation_id", payload.ConversationID.String()).
			Err(err).
			Msg("Failed to summarize conversation")
		return fmt.Errorf("summarization failed: %w", err)
	}

	// Get the compressed summary
	var summary string
	if len(summaries) > 0 {
		summary = summaries[0].Compressed
	}

	// Persist onto the conversation so the paid summary survives the task.
	// cmetadata is merged rather than replaced — ai_reply and intent_detail
	// live in the same document.
	conv, err := w.conversationRepo.FindByID(ctx, payload.ConversationID)
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}

	meta := map[string]interface{}{}
	if len(conv.CMetadata) > 0 {
		if err := json.Unmarshal(conv.CMetadata, &meta); err != nil {
			meta = map[string]interface{}{}
		}
	}
	meta["summary"] = map[string]interface{}{
		"content":      summary,
		"email_count":  len(emailsResult.List),
		"detail_level": payload.DetailLevel,
	}
	if encoded, err := json.Marshal(meta); err == nil {
		conv.CMetadata = encoded
		if err := w.conversationRepo.Update(ctx, conv.ID, conv); err != nil {
			return fmt.Errorf("failed to store conversation summary: %w", err)
		}
	}

	w.logger.Info().
		Str("conversation_id", payload.ConversationID.String()).
		Int("email_count", len(emailsResult.List)).
		Int("summary_length", len(summary)).
		Msg("Conversation summarized and stored")

	return nil
}

// NewSummarizeEmailTask creates a new email summarization task
func NewSummarizeEmailTask(emailID uuid.UUID, detailLevel float64) (*asynq.Task, error) {
	payload, err := json.Marshal(SummarizeEmailPayload{
		EmailID:     emailID,
		DetailLevel: detailLevel,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeSummarizeEmail, payload), nil
}

// NewSummarizeConversationTask creates a new conversation summarization task
func NewSummarizeConversationTask(conversationID uuid.UUID, detailLevel float64) (*asynq.Task, error) {
	payload, err := json.Marshal(SummarizeConversationPayload{
		ConversationID: conversationID,
		DetailLevel:    detailLevel,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeSummarizeConversation, payload), nil
}
