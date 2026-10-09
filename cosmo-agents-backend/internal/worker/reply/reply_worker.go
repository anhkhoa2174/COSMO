package reply

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/service/autoreply"
	dailyActionSvc "github.com/rockship/cosmo-agents-go/internal/service/daily_action"
	intentService "github.com/rockship/cosmo-agents-go/internal/service/intent"
	outreachService "github.com/rockship/cosmo-agents-go/internal/service/outreach"
	"time"
)

const (
	TypeHandleReply = "email:handle_reply"
)

// HandleReplyPayload is the payload for handle reply task
type HandleReplyPayload struct {
	EmailID uuid.UUID `json:"email_id"`
}

// outreachUpdater updates contact outreach state through the state machine.
type outreachUpdater interface {
	// The parameter order must match outreach.Service exactly: every argument
	// but ctx is a uuid or a string, so a mismatch still compiles. It used to
	// read (contactID, userID, event, channel, content), which swapped the two
	// IDs and wrote the email body into the varchar(50) channel column, so no
	// reply ever moved a contact's outreach state.
	UpdateOutreachForHandler(ctx context.Context, userID, contactID uuid.UUID, event, content, channel, sentiment string) (*outreachService.UpdateOutreachResult, error)
}

// Worker processes email replies with intent classification
type Worker struct {
	emailRepo        emailRepository
	conversationRepo conversationRepository
	campaignRepo     campaignRepository
	contactRepo      contactRepository
	intentClassifier intentClassifier
	handlerFactory   intentHandlerFactory
	outreachSvc      outreachUpdater
	sseManager       *dailyActionSvc.SSEManager
	logger           *zerolog.Logger

	// Optional. Unset, every AI reply stays a draft — which is exactly how
	// this worker behaved before auto-reply existed.
	autoReply autoReplyDispatcher
	agentRepo agentLookup
}

// autoReplyDispatcher decides whether the draft an intent handler just saved
// may be sent without review, and sends it if so.
type autoReplyDispatcher interface {
	Consider(ctx context.Context, intent string, confidence float64,
		target autoreply.Target, draft autoreply.Draft) autoreply.Outcome
}

// agentLookup resolves the mailbox a reply would be sent from.
type agentLookup interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
}

// WithAutoReply enables automatic sending of AI replies for organisations that
// have opted into it. Both dependencies are required: without an agent lookup
// there is no mailbox to send from, so the feature stays off.
func (w *Worker) WithAutoReply(d autoReplyDispatcher, agents agentLookup) *Worker {
	if d != nil && agents != nil {
		w.autoReply = d
		w.agentRepo = agents
	}
	return w
}

type emailRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Email, error)
	Update(ctx context.Context, id uuid.UUID, email *domain.Email) error
}

type conversationRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error)
	Update(ctx context.Context, id uuid.UUID, conversation *domain.Conversation) error
}

type campaignRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error)
}

type contactRepository interface {
	FindByEmail(ctx context.Context, userID uuid.UUID, email string) (*domain.Contact, error)
}

type intentClassifier interface {
	Classify(ctx context.Context, content string) (domain.IntentType, error)
	ClassifyDetailed(ctx context.Context, content string) (intentService.DetailedIntent, error)
}

type intentHandlerFactory interface {
	Build(campaign *domain.Campaign, intent domain.IntentType) (intentService.IntentHandler, error)
}

// New creates a new handle reply worker
func New(
	emailRepo emailRepository,
	conversationRepo conversationRepository,
	campaignRepo campaignRepository,
	contactRepo contactRepository,
	intentClassifier intentClassifier,
	handlerFactory intentHandlerFactory,
	outreachSvc outreachUpdater,
	log *zerolog.Logger,
	sseManager ...*dailyActionSvc.SSEManager,
) *Worker {
	w := &Worker{
		emailRepo:        emailRepo,
		conversationRepo: conversationRepo,
		campaignRepo:     campaignRepo,
		contactRepo:      contactRepo,
		intentClassifier: intentClassifier,
		handlerFactory:   handlerFactory,
		outreachSvc:      outreachSvc,
		logger:           log,
	}
	if len(sseManager) > 0 {
		w.sseManager = sseManager[0]
	}
	return w
}

// ProcessTask processes a handle reply task
func (w *Worker) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload HandleReplyPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("Failed to unmarshal handle reply payload")
		return fmt.Errorf("json.Unmarshal failed: %w", err)
	}

	w.logger.Info().
		Str("email_id", payload.EmailID.String()).
		Msg("Processing handle reply task")

	// Fetch the email
	email, err := w.emailRepo.FindByID(ctx, payload.EmailID)
	if err != nil {
		w.logger.Warn().
			Str("email_id", payload.EmailID.String()).
			Msg("Reply email not found")
		return fmt.Errorf("email not found: %w", err)
	}

	// Fetch the conversation
	conversation, err := w.conversationRepo.FindByID(ctx, *email.ConversationID)
	if err != nil {
		w.logger.Error().Err(err).Msg("Conversation not found")
		return fmt.Errorf("conversation not found: %w", err)
	}

	// Check if conversation was already handled by AI (has a usable ai_reply draft).
	// The presence of the ai_reply key is not enough — an entry whose
	// draft_content is empty means the previous attempt produced nothing, and
	// treating that as "already drafted" would strand the conversation with no
	// draft forever.
	if conversation.Replied {
		var meta map[string]interface{}
		if err := conversation.CMetadata.Unmarshal(&meta); err == nil {
			if aiReply, ok := meta["ai_reply"].(map[string]interface{}); ok {
				draft, _ := aiReply["draft_content"].(string)
				if strings.TrimSpace(draft) != "" {
					w.logger.Info().
						Str("email_id", payload.EmailID.String()).
						Str("conversation_id", conversation.ID.String()).
						Msg("AI already drafted reply for this conversation. Skipped.")
					return nil
				}
				w.logger.Warn().
					Str("email_id", payload.EmailID.String()).
					Str("conversation_id", conversation.ID.String()).
					Msg("Existing ai_reply has an empty draft. Regenerating.")
			}
		}
		w.logger.Info().
			Str("email_id", payload.EmailID.String()).
			Str("conversation_id", conversation.ID.String()).
			Msg("Conversation marked replied but no AI draft yet. Processing.")
	}

	// Fetch the campaign. Email cũ (trước khi inbound gán CampaignID) có thể
	// nil — deref thẳng sẽ panic và giết cả worker, nên thoát êm.
	if email.CampaignID == nil {
		w.logger.Warn().
			Str("email_id", email.ID.String()).
			Msg("Inbound email has no campaign; skipping intent pipeline")
		return nil
	}
	campaign, err := w.campaignRepo.FindByID(ctx, *email.CampaignID)
	if err != nil {
		w.logger.Error().Err(err).Msg("Campaign not found")
		return fmt.Errorf("campaign not found: %w", err)
	}

	// Check if campaign is active
	if campaign.Status != domain.CampaignStatusActive {
		w.logger.Info().
			Str("campaign_id", campaign.ID.String()).
			Str("email_id", email.ID.String()).
			Msg("Campaign is not active. Skipped")
		return nil
	}

	// Classify intent using AI (giữ cả confidence + reasoning cho FE hiển thị)
	detail, err := w.intentClassifier.ClassifyDetailed(ctx, email.Content)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to classify intent")
		return fmt.Errorf("intent classification failed: %w", err)
	}
	intent := detail.Intent

	w.logger.Info().
		Str("email_id", email.ID.String()).
		Str("intent", string(intent)).
		Msg("Classified intent")

	// Update outreach state through state machine (EventReplied)
	if w.outreachSvc != nil && w.contactRepo != nil {
		contact, contactErr := w.contactRepo.FindByEmail(ctx, campaign.UserID, email.FromEmail)
		if contactErr == nil && contact != nil {
			if _, err := w.outreachSvc.UpdateOutreachForHandler(
				ctx, campaign.UserID, contact.ID,
				"replied", email.Content, "Email", "",
			); err != nil {
				w.logger.Warn().Err(err).
					Str("contact_id", contact.ID.String()).
					Msg("Failed to update outreach state via state machine")
			} else {
				w.logger.Info().
					Str("contact_id", contact.ID.String()).
					Str("intent", string(intent)).
					Msg("Outreach state updated via state machine (EventReplied)")
			}
		}
	}

	// Update email with intent (convert to StringArray)
	email.Intents = []string{string(intent)}
	if err := w.emailRepo.Update(ctx, email.ID, email); err != nil {
		w.logger.Error().Err(err).Msg("Failed to update email intent")
		return fmt.Errorf("failed to update email: %w", err)
	}

	// Update conversation with intent (convert to StringArray)
	conversation.Intents = []string{string(intent)}
	// Lưu chi tiết phân loại vào CMetadata: FE dùng reasoning (có trích dẫn
	// nguyên văn) để highlight câu thể hiện intent trong nội dung email.
	var convMeta map[string]interface{}
	if err := conversation.CMetadata.Unmarshal(&convMeta); err != nil || convMeta == nil {
		convMeta = map[string]interface{}{}
	}
	convMeta["intent_detail"] = map[string]interface{}{
		"intent":     string(intent),
		"confidence": detail.Confidence,
		"reasoning":  detail.Reasoning,
		"email_id":   email.ID.String(),
	}
	if metaBytes, err := json.Marshal(convMeta); err == nil {
		_ = conversation.CMetadata.Scan(metaBytes)
	}
	if err := w.conversationRepo.Update(ctx, conversation.ID, conversation); err != nil {
		w.logger.Error().Err(err).Msg("Failed to update conversation intent")
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	// Build the appropriate handler
	handler, err := w.handlerFactory.Build(campaign, intent)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to build intent handler")
		return fmt.Errorf("failed to build handler: %w", err)
	}

	if handler == nil {
		w.logger.Error().
			Str("campaign_id", campaign.ID.String()).
			Str("intent", string(intent)).
			Msg("Failed to resolve handler")
		return fmt.Errorf("no handler available for intent: %s", intent)
	}

	// Execute the handler
	success, err := handler.Execute(ctx, campaign, intent, email)
	if err != nil {
		w.logger.Error().
			Err(err).
			Str("intent", string(intent)).
			Str("email_id", email.ID.String()).
			Msg("Failed to execute intent handler")
		return fmt.Errorf("handler execution failed: %w", err)
	}

	if !success {
		w.logger.Warn().
			Str("intent", string(intent)).
			Str("email_id", email.ID.String()).
			Msg("Intent handler execution was not successful")
		return fmt.Errorf("handler execution unsuccessful")
	}

	w.logger.Info().
		Str("intent", string(intent)).
		Str("email_id", email.ID.String()).
		Msg("Contact email handled successfully with intent handler")

	// The handler has saved its draft. Only now is it decided whether that
	// draft goes out unreviewed: persisting first means there is a record of
	// what was sent even if the send itself fails, and keeping the decision
	// here means one place to audit rather than six handlers.
	w.considerAutoReply(ctx, campaign, email, intent, detail.Confidence)

	// Publish SSE event to notify frontend
	if w.sseManager != nil {
		eventData, _ := json.Marshal(map[string]interface{}{
			"event_type":      "prospect_replied",
			"conversation_id": conversation.ID.String(),
			"email_id":        email.ID.String(),
			"intent":          string(intent),
			"from_email":      email.FromEmail,
			"subject":         email.Subject,
			"preview":         truncate(email.Content, 100),
			"timestamp":       time.Now(),
		})
		if err := w.sseManager.PublishEvent(ctx, campaign.UserID, &dailyActionSvc.SSEEvent{
			EventType: "prospect_replied",
			Data:      eventData,
		}); err != nil {
			w.logger.Warn().Err(err).Msg("Failed to publish SSE event for reply")
		}
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// NewHandleReplyTask creates a new handle reply task
func NewHandleReplyTask(emailID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(HandleReplyPayload{EmailID: emailID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeHandleReply, payload), nil
}

// considerAutoReply sends the just-saved draft when the organisation has opted
// this intent into automatic replies.
//
// Every path that cannot establish permission leaves the draft alone. The
// method reports nothing to its caller because holding a draft is not a
// failure of reply processing — the reply was handled either way, and the
// reason is recorded on the conversation for the inbox to show.
func (w *Worker) considerAutoReply(
	ctx context.Context,
	campaign *domain.Campaign,
	email *domain.Email,
	intent domain.IntentType,
	confidence float64,
) {
	if w.autoReply == nil || campaign == nil || email.ConversationID == nil {
		return
	}

	// The contact is looked up here rather than passed in: the caller resolves
	// it only inside the branch that updates outreach state, and a reply whose
	// contact cannot be resolved must not be auto-answered at all — the
	// do-not-contact flag lives on that record.
	contact, err := w.contactRepo.FindByEmail(ctx, campaign.UserID, email.FromEmail)
	if err != nil || contact == nil {
		w.logger.Warn().Err(err).
			Msg("auto-reply: contact not resolved, leaving draft for review")
		return
	}

	// Re-read the conversation: the handler wrote the draft into it, and the
	// copy this function was handed predates that write.
	conversation, err := w.conversationRepo.FindByID(ctx, *email.ConversationID)
	if err != nil || conversation == nil {
		w.logger.Warn().Err(err).Msg("auto-reply: conversation unreadable, leaving draft")
		return
	}

	draft, drafted := readAIDraft(conversation)
	if !drafted {
		// Not every handler produces a draft — out-of-office, nurture and an
		// assignment to a teammate have nothing to send. The organisation may
		// still have written a fixed message for the intent, so the dispatcher
		// is asked anyway, with the subject the reply would thread under.
		draft = aiDraft{Subject: "Re: " + email.Subject}
	}

	agentID := uuid.Nil
	toEmail := ""
	senderName := ""
	if campaign.AgentID != nil {
		if agent, err := w.agentRepo.FindByID(ctx, *campaign.AgentID); err == nil && agent != nil {
			agentID = agent.ID
			senderName = agent.Name
		}
	}
	if email.FromEmail != "" {
		toEmail = email.FromEmail
	}

	outcome := w.autoReply.Consider(ctx, string(intent), confidence,
		autoreply.Target{
			UserID:     campaign.UserID,
			AgentID:    agentID,
			ContactID:  contact.ID,
			CampaignID: &campaign.ID,
			ToEmail:    toEmail,
			OptedOut:   contact.DoNotContact,
			Merge: autoreply.NewMergeValues(
				contact.Name, contact.Company, contact.JobTitle, senderName),
		},
		autoreply.Draft{
			Subject:   draft.Subject,
			Body:      draft.Body,
			InReplyTo: email.GmailMessageID,
		})

	if !drafted && !outcome.FromTemplate {
		// Nothing was drafted and nothing was written for this intent, so
		// there is no draft to stamp; the dispatcher has logged the reason.
		return
	}

	if err := w.recordAutoReplyOutcome(ctx, conversation, intent, outcome); err != nil {
		w.logger.Warn().Err(err).Msg("auto-reply: could not record the outcome")
	}
}

type aiDraft struct {
	Subject string
	Body    string
}

// readAIDraft pulls the draft an intent handler saved into the conversation.
func readAIDraft(conversation *domain.Conversation) (aiDraft, bool) {
	var meta map[string]interface{}
	if err := conversation.CMetadata.Unmarshal(&meta); err != nil || meta == nil {
		return aiDraft{}, false
	}
	block, ok := meta["ai_reply"].(map[string]interface{})
	if !ok {
		return aiDraft{}, false
	}
	body, _ := block["draft_content"].(string)
	if strings.TrimSpace(body) == "" {
		return aiDraft{}, false
	}
	subject, _ := block["draft_subject"].(string)
	return aiDraft{Subject: subject, Body: body}, true
}

// recordAutoReplyOutcome stamps the decision onto the draft so the inbox knows
// whether a reply still needs a person, and so the reason survives for audit.
func (w *Worker) recordAutoReplyOutcome(
	ctx context.Context,
	conversation *domain.Conversation,
	intent domain.IntentType,
	outcome autoreply.Outcome,
) error {
	var meta map[string]interface{}
	if err := conversation.CMetadata.Unmarshal(&meta); err != nil {
		return fmt.Errorf("conversation metadata unreadable")
	}
	if meta == nil {
		meta = map[string]interface{}{}
	}

	block, ok := meta["ai_reply"].(map[string]interface{})
	if outcome.FromTemplate {
		// The organisation's fixed message replaces whatever the handler
		// drafted: held, it is the message a person now reviews; sent, it is
		// what the prospect received.
		//
		// conversation.Replied is deliberately left alone. The worker skips
		// later messages in a thread that is marked replied and holds a draft,
		// and an out-of-office acknowledgement must not stop the prospect's
		// real answer, when they are back, from being classified.
		block = map[string]interface{}{
			"draft_content": outcome.Draft.Body,
			"draft_subject": outcome.Draft.Subject,
			"intent":        string(intent),
			"generated_by":  "org_template",
			"generated_at":  time.Now().Format(time.RFC3339),
		}
	} else if !ok {
		return fmt.Errorf("no ai_reply block to stamp")
	}

	block["status"] = outcome.Status()
	block["auto_reply_reason"] = outcome.Decision.Reason
	if outcome.SentAt != nil {
		block["auto_sent_at"] = outcome.SentAt.Format(time.RFC3339)
	}
	meta["ai_reply"] = block

	encoded, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	if err := conversation.CMetadata.Scan(encoded); err != nil {
		return err
	}
	return w.conversationRepo.Update(ctx, conversation.ID, conversation)
}
