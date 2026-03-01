package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	repositoryHelper "github.com/rockship/cosmo-agents-go/internal/repository/helper"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	resendService "github.com/rockship/cosmo-agents-go/internal/service/resend"
	"github.com/rockship/cosmo-agents-go/pkg/gmail"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

const (
	TypeAgentSendEmail        = "agent:send_email"
	TypeAgentDailyReset       = "agent:daily_reset"
	TypeSendInviteMemberEmail = "email:send_invite_member_email"
	TypeSendEmail             = "email:send"
	TypeProcessIncoming       = "email:process_incoming"
	TypeSyncGmailHistory      = "email:sync_history"
)

const SUBJECT_INVITATION = "Welcome to Cosmo Agents Your Account Awaits"

// extractEmailFromProfile extracts email from contact's profile JSONB
func extractEmailFromProfile(profile domain.JSONB) string {
	if len(profile) == 0 {
		return ""
	}
	var profileData map[string]interface{}
	if err := json.Unmarshal(profile, &profileData); err != nil {
		return ""
	}
	if email, ok := profileData["email"].(string); ok {
		return email
	}
	return ""
}

// AgentSendEmailPayload represents the payload for sending emails via agent.
type AgentSendEmailPayload struct {
	TaskIDs    []uuid.UUID `json:"task_ids"`
	CampaignID uuid.UUID   `json:"campaign_id"`
	AgentID    uuid.UUID   `json:"agent_id"`
}

// SendInviteMemberEmailPayload represents the payload for sending invitation emails.
type SendInviteMemberEmailPayload struct {
	UserID         uuid.UUID `json:"user_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	MemberEmail    string    `json:"member_email"`
	MemberName     string    `json:"member_name"`
	InviterName    string    `json:"inviter_name"`
	RedirectURI    string    `json:"redirect_uri"`
	Url            string    `json:"url"`
	Email          string    `json:"email"`
}

// SendEmailPayload represents the payload for sending general emails.
type SendEmailPayload struct {
	TaskID     *uuid.UUID `json:"task_id,omitempty"`
	CampaignID *uuid.UUID `json:"campaign_id,omitempty"`
	TemplateID *uuid.UUID `json:"template_id,omitempty"`
	AgentID    uuid.UUID  `json:"agent_id"`
	ContactID  uuid.UUID  `json:"contact_id,omitempty"`
	To         string     `json:"to"`
	Cc         []string   `json:"cc,omitempty"`
	Bcc        []string   `json:"bcc,omitempty"`
	Subject    string     `json:"subject"`
	Body       string     `json:"body"`
	IsHTML     bool       `json:"is_html"`
	InReplyTo  string     `json:"in_reply_to,omitempty"`
}

// ProcessIncomingEmailPayload represents the payload for processing incoming emails.
type ProcessIncomingEmailPayload struct {
	AgentID        uuid.UUID `json:"agent_id"`
	GmailMessageID string    `json:"gmail_message_id"`
	GmailThreadID  string    `json:"gmail_thread_id,omitempty"`
}

// SyncGmailHistoryPayload represents the payload for syncing Gmail history.
type SyncGmailHistoryPayload struct {
	AgentID        uuid.UUID `json:"agent_id"`
	StartHistoryID string    `json:"start_history_id,omitempty"`
}

// gmailClient interface for Gmail operations.
type gmailClient interface {
	SendMessage(ctx context.Context, req *gmail.SendMessageRequest) (*gmail.Message, error)
	GetMessage(ctx context.Context, messageID string) (*gmail.Message, error)
	ListHistory(ctx context.Context, startHistoryID string, pageToken string) (*gmail.HistoryResult, error)
	Close()
}

type gmailClientFactory func(ctx context.Context, accessToken string) (gmailClient, error)

// GeneralEmailWorker handles general email-related background tasks (from original file).
type GeneralEmailWorker struct {
	db               *gorm.DB
	oauth2Client     oauth2Client
	agentRepo        generalAgentRepo
	emailRepo        generalEmailRepo
	taskRepo         generalTaskRepo
	contactRepo      generalContactRepo
	conversationRepo generalConversationRepo
	organizationRepo *organization.OrganizationRepository
	gmailFactory     gmailClientFactory
}

// EmailWorker handles agent-related background tasks.
type EmailWorker struct {
	db                 *gorm.DB
	agentRepository    agentRepoInterface
	campaignRepository campaignRepoInterface
	taskRepository     taskRepoInterface
	emailRepository    emailRepoInterface
	conversationRepo   conversationRepoInterface
	relationsHelper    *repositoryHelper.RelationsHelper
	googleAuthService  GoogleAuthService
	gmailService       GmailService
	taskMonitor        TaskMonitor
}

// GoogleAuthService interface for Google OAuth2.
type GoogleAuthService interface {
	Refresh(ctx context.Context, credentials map[string]interface{}, scopes []string) (map[string]interface{}, error)
}

// GmailService interface for Gmail operations.
type GmailService interface {
	SendEmail(ctx context.Context, sender string, to []string, subject string, body map[string]string) (*GmailMessage, error)
}

type oauth2Client interface {
	RefreshToken(ctx context.Context, refreshToken string) (*googleoauth.Token, error)
}

type generalAgentRepo interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	Update(ctx context.Context, id uuid.UUID, agent *domain.Agent) error
	UpdateLastHistoryID(ctx context.Context, agentID uuid.UUID, historyID string) error
}

type generalEmailRepo interface {
	Create(ctx context.Context, email *domain.Email) (*domain.Email, error)
	FindByGmailMessageID(ctx context.Context, gmailMessageID string) (*domain.Email, error)
}

type generalTaskRepo interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	Update(ctx context.Context, id uuid.UUID, task *domain.Task) error
}

type generalContactRepo interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error)
}

type generalConversationRepo interface {
	Create(ctx context.Context, conversation *domain.Conversation) (*domain.Conversation, error)
	FindByGmailThreadID(ctx context.Context, threadID string) (*domain.Conversation, error)
	SetReplied(ctx context.Context, conversationID uuid.UUID, replied bool) error
}

type agentRepoInterface interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	Update(ctx context.Context, id uuid.UUID, agent *domain.Agent) error
	IncrementEmailCount(ctx context.Context, agentID string, count int) error
	ResetDailyEmailCounts(ctx context.Context) error
}

type campaignRepoInterface interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error)
}

type taskRepoInterface interface {
	GetPendingTasksByIDs(ctx context.Context, taskIDs []uuid.UUID) ([]*domain.Task, error)
	Update(ctx context.Context, id uuid.UUID, task *domain.Task) error
}

type emailRepoInterface interface {
	Create(ctx context.Context, email *domain.Email) (*domain.Email, error)
}

type conversationRepoInterface interface {
	Create(ctx context.Context, conversation *domain.Conversation) (*domain.Conversation, error)
	FindByGmailThreadID(ctx context.Context, threadID string) (*domain.Conversation, error)
	SetReplied(ctx context.Context, conversationID uuid.UUID, replied bool) error
	Update(ctx context.Context, id uuid.UUID, conversation *domain.Conversation) error
}

// GmailMessage represents a Gmail message.
type GmailMessage struct {
	ID       string
	ThreadID string
}

// TaskMonitor interface for task monitoring.
type TaskMonitor interface {
	RespondToTask(ctx context.Context, taskIDs []string) error
	ResolveTask(ctx context.Context, taskIDs []string) error
}

// NewEmailWorker creates a new agent email worker.
func NewEmailWorker(
	db *gorm.DB,
	googleAuthService GoogleAuthService,
	gmailService GmailService,
	taskMonitor TaskMonitor,
	agentRepo agentRepoInterface,
	campaignRepo campaignRepoInterface,
	taskRepo taskRepoInterface,
	emailRepo emailRepoInterface,
	conversationRepo conversationRepoInterface,
) *EmailWorker {
	return &EmailWorker{
		db:                 db,
		agentRepository:    agentRepo,
		campaignRepository: campaignRepo,
		taskRepository:     taskRepo,
		emailRepository:    emailRepo,
		conversationRepo:   conversationRepo,
		relationsHelper:    repositoryHelper.NewRelationsHelper(db),
		googleAuthService:  googleAuthService,
		gmailService:       gmailService,
		taskMonitor:        taskMonitor,
	}
}

// HandleAgentSendEmail processes sending emails via agent.
func (w *EmailWorker) HandleAgentSendEmail(ctx context.Context, task *asynq.Task) error {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var payload AgentSendEmailPayload
	if err := worker.ParsePayload(task, &payload); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to parse agent send email payload")
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", payload.AgentID.String()).
		Str("campaign_id", payload.CampaignID.String()).
		Int("task_count", len(payload.TaskIDs)).
		Msg("Processing agent send email")

	// Get agent
	agent, err := w.agentRepository.GetByID(ctx, payload.AgentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Failed to get agent")
		return fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		logger.FromContext(ctx).Error().Str("agent_id", payload.AgentID.String()).Msg("Agent not found")
		return fmt.Errorf("agent not found: %s", payload.AgentID)
	}

	// Get campaign
	campaign, err := w.campaignRepository.GetByID(ctx, payload.CampaignID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("campaign_id", payload.CampaignID.String()).Msg("Failed to get campaign")
		return fmt.Errorf("failed to get campaign: %w", err)
	}
	if campaign == nil {
		logger.FromContext(ctx).Error().Str("campaign_id", payload.CampaignID.String()).Msg("Campaign not found")
		return fmt.Errorf("campaign not found: %s", payload.CampaignID)
	}

	// Refresh Google credentials
	var credMap map[string]interface{}
	if err := agent.Credentials.Unmarshal(&credMap); err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Failed to unmarshal credentials")
		return fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	sanitizeCredentialsForLogging(credMap)

	refreshedCredentials, err := w.googleAuthService.Refresh(ctx, credMap, []string{
		"https://www.googleapis.com/auth/gmail.send",
		"https://www.googleapis.com/auth/gmail.readonly",
	})
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			agent.Status = "invalid_grant"
			if updateErr := w.agentRepository.Update(ctx, agent.ID, agent); updateErr != nil {
				logger.FromContext(ctx).Err(updateErr).Msg("Failed to update agent status")
			}
		}
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Failed to refresh credentials")
		return fmt.Errorf("failed to refresh credentials: %w", err)
	}

	// Sanitize refreshed credentials to prevent accidental logging of sensitive data
	sanitizeCredentialsForLogging(refreshedCredentials)

	// Update agent credentials
	if err := agent.SetCredentials(refreshedCredentials); err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Failed to set credentials")
		return fmt.Errorf("failed to set credentials: %w", err)
	}
	if err := w.agentRepository.Update(ctx, agent.ID, agent); err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Failed to update agent credentials")
		return fmt.Errorf("failed to update agent credentials: %w", err)
	}

	// Check daily limits
	dailyLimit := 0
	emailsSentToday := 0
	if agent.DailyLimit != nil {
		dailyLimit = *agent.DailyLimit
	}
	if agent.EmailsSentToday != nil {
		emailsSentToday = *agent.EmailsSentToday
	}
	if dailyLimit > 0 && dailyLimit <= emailsSentToday {
		logger.FromContext(ctx).Warn().
			Str("agent_id", payload.AgentID.String()).
			Int("limit", dailyLimit).
			Int("sent", emailsSentToday).
			Msg("Agent has reached daily limit")
		return nil
	}

	// Get tasks
	tasks, err := w.taskRepository.GetPendingTasksByIDs(ctx, payload.TaskIDs)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to get tasks")
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) == 0 {
		logger.FromContext(ctx).Info().Msg("No tasks to process")
		return nil
	}

	taskIDs := make([]string, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID.String()
	}
	if err := w.taskMonitor.RespondToTask(ctx, taskIDs); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to mark tasks as responded")
	}

	// Process tasks within a database transaction for consistency
	doneTasks := make([]string, 0, len(tasks))
	numEmails := 0

	// Process all tasks in a single transaction to ensure consistency
	// Use the timeout context to ensure transaction respects the 5-minute timeout
	txErr := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		failedTasks := make([]domain.Task, 0, len(tasks))

		for idx, t := range tasks {
			// Check if context has been cancelled
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Load task with full relations using RelationsHelper
			taskWithRelations, err := w.relationsHelper.GetTaskWithFullRelations(ctx, t.ID)
			if err != nil {
				logger.FromContext(ctx).Err(err).
					Str("task_id", t.ID.String()).
					Msg("Failed to load task with relations - skipping")
				t.Status = "failed"
				errMsg := "Failed to load task relations"
				t.Error = &errMsg
				failedTasks = append(failedTasks, *t)
				continue
			}

			// Check if Contact is properly loaded
			if taskWithRelations.Contact == nil {
				logger.FromContext(ctx).Error().
					Str("task_id", t.ID.String()).
					Msg("Task has nil Contact - skipping")
				t.Status = "failed"
				errMsg := "Task contact is nil"
				t.Error = &errMsg
				failedTasks = append(failedTasks, *t)
				continue
			}

			// Check if template exists
			if taskWithRelations.Template == nil {
				logger.FromContext(ctx).Error().
					Str("task_id", t.ID.String()).
					Msg("Task has nil Template - skipping")
				t.Status = "failed"
				errMsg := "Task template is nil"
				t.Error = &errMsg
				failedTasks = append(failedTasks, *t)
				continue
			}

			// Prepare email content - convert JSONB to map
			var contactData map[string]interface{}
			if taskWithRelations.Contact.Profile == nil {
				// Profile is nil, initialize empty map to prevent panic
				contactData = make(map[string]interface{})
				logger.FromContext(ctx).Warn().
					Str("contact_id", taskWithRelations.Contact.ID.String()).
					Msg("Contact profile is nil, using empty data")
			} else if err := taskWithRelations.Contact.Profile.Unmarshal(&contactData); err != nil {
				logger.FromContext(ctx).Err(err).Str("contact_id", taskWithRelations.Contact.ID.String()).Msg("Failed to unmarshal contact profile")
				contactData = make(map[string]interface{})
			}

			// Format subject and content with variables
			subject := w.formatTemplate(taskWithRelations.Template.Subject, contactData)
			plainContent := w.formatTemplate(taskWithRelations.Template.Content, contactData)
			htmlContent := strings.ReplaceAll(plainContent, "\n", "<br>")

			// Extract contact email from profile
			contactEmail := extractEmailFromProfile(taskWithRelations.Contact.Profile)

			// Determine CC recipients based on campaign config
			_ = w.determineCCRecipients(ctx, campaign, idx, len(tasks))
			if contactEmail == "" {
				logger.FromContext(ctx).Error().
					Str("task_id", t.ID.String()).
					Str("contact_id", taskWithRelations.Contact.ID.String()).
					Msg("Task contact has empty email - skipping")
				t.Status = "failed"
				errMsg := "Task contact email is empty"
				t.Error = &errMsg
				failedTasks = append(failedTasks, *t)
				continue
			}

			// Send email via Gmail
			logger.FromContext(ctx).Info().
				Str("to", contactEmail).
				Str("subject", subject).
				Msg("Sending email")

			msg, err := w.gmailService.SendEmail(ctx, agent.Email, []string{contactEmail}, subject, map[string]string{
				"plain_text": plainContent,
				"html":       htmlContent,
			})
			if err != nil {
				logger.FromContext(ctx).Err(err).
					Str("task_id", t.ID.String()).
					Msg("Failed to send email")
				t.Status = "failed"
				errMsg := err.Error()
				t.Error = &errMsg
				failedTasks = append(failedTasks, *t)
				continue
			}

			// Create conversation
			conversation := domain.Conversation{
				UserID:        agent.UserID,
				GmailThreadID: msg.ThreadID,
				CampaignID:    &campaign.ID,
				Replied:       false,
				AgentID:       &agent.ID,
			}
			createdConversation, err := w.conversationRepo.Create(ctx, &conversation)
			if err != nil {
				logger.FromContext(ctx).Err(err).
					Str("task_id", t.ID.String()).
					Msg("Failed to create conversation - marking task as failed")
				// Mark task as failed due to conversation creation error
				t.Status = "failed"
				errMsg := fmt.Sprintf("Failed to create conversation: %v", err)
				t.Error = &errMsg
				failedTasks = append(failedTasks, *t)
				continue
			}

			// Create email record
			email := domain.Email{
				UserID:         agent.UserID,
				ConversationID: &createdConversation.ID,
				GmailMessageID: msg.ID,
				FromEmail:      agent.Email,
				ToEmail:        contactEmail,
				Subject:        subject,
				Content:        plainContent,
				Status:         "sent",
				CampaignID:     &campaign.ID,
			}
			if _, err := w.emailRepository.Create(ctx, &email); err != nil {
				logger.FromContext(ctx).Err(err).
					Str("task_id", t.ID.String()).
					Msg("Failed to create email record - marking task as failed")
				// Mark task as failed due to email creation error
				t.Status = "failed"
				errMsg := fmt.Sprintf("Failed to create email record: %v", err)
				t.Error = &errMsg
				failedTasks = append(failedTasks, *t)
				continue
			}

			doneTasks = append(doneTasks, t.ID.String())
			numEmails++
		}

		// Batch update failed tasks to prevent resource leaks
		if len(failedTasks) > 0 {
			for _, failedTask := range failedTasks {
				if updateErr := w.taskRepository.Update(ctx, failedTask.ID, &failedTask); updateErr != nil {
					logger.FromContext(ctx).Err(updateErr).
						Str("task_id", failedTask.ID.String()).
						Msg("Failed to update failed task status")
				}
			}
			logger.FromContext(ctx).Warn().
				Int("failed_count", len(failedTasks)).
				Msg("Processed failed tasks")
		}

		// Mark tasks as resolved
		if len(doneTasks) > 0 {
			if err := w.taskMonitor.ResolveTask(ctx, doneTasks); err != nil {
				logger.FromContext(ctx).Err(err).Msg("Failed to resolve tasks")
			}
		}

		// Update agent email count inside transaction for atomicity
		if err := w.agentRepository.IncrementEmailCount(ctx, agent.ID.String(), numEmails); err != nil {
			logger.FromContext(ctx).Err(err).Msg("Failed to increment agent email count atomically")
			return fmt.Errorf("failed to increment agent email count: %w", err)
		}

		// Log success inside transaction
		logger.FromContext(ctx).Info().
			Str("agent_id", payload.AgentID.String()).
			Int("count", numEmails).
			Msg("Successfully sent emails")

		return nil
	})

	if txErr != nil {
		logger.FromContext(ctx).Err(txErr).Msg("Transaction failed during email processing")
		return fmt.Errorf("transaction failed: %w", txErr)
	}
	return nil
}

// HandleAgentDailyReset processes daily reset of agent email counts.
func (w *EmailWorker) HandleAgentDailyReset(ctx context.Context, task *asynq.Task) error {
	logger.FromContext(ctx).Info().Msg("Processing agent daily reset")

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := w.agentRepository.ResetDailyEmailCounts(ctx); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to reset daily email counts")
		return fmt.Errorf("failed to reset daily email counts: %w", err)
	}

	logger.FromContext(ctx).Info().Msg("Successfully reset all agent email counts")
	return nil
}

// RegisterEmailWorkerTasks registers all agent-related tasks.
func (w *EmailWorker) RegisterEmailWorkerTasks(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeAgentSendEmail, w.HandleAgentSendEmail)
	mux.HandleFunc(TypeAgentDailyReset, w.HandleAgentDailyReset)
}

// Helper functions

func (w *EmailWorker) formatTemplate(template string, data map[string]interface{}) string {
	result := template
	for key, value := range data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

func (w *EmailWorker) determineCCRecipients(ctx context.Context, campaign *domain.Campaign, taskIndex, totalTasks int) []string {
	return []string{}
}

// sanitizeCredentialsForLogging removes sensitive information from credential maps
func sanitizeCredentialsForLogging(credMap map[string]interface{}) {
	// Remove sensitive fields that might be logged accidentally
	sensitiveFields := []string{"access_token", "refresh_token", "client_secret", "password"}
	for _, field := range sensitiveFields {
		delete(credMap, field)
	}
}

// Enqueue functions

func EnqueueAgentSendEmail(client *asynq.Client, payload AgentSendEmailPayload) error {
	task, err := worker.NewTask(TypeAgentSendEmail, payload)
	if err != nil {
		return err
	}
	_, err = client.Enqueue(task)
	return err
}

func EnqueueAgentDailyReset(client *asynq.Client) error {
	task, err := worker.NewTask(TypeAgentDailyReset, nil)
	if err != nil {
		return err
	}

	_, err = client.Enqueue(task, asynq.ProcessIn(24*time.Hour))
	return err
}

// NewGeneralEmailWorker creates a new general email worker.
func NewGeneralEmailWorker(
	db *gorm.DB,
	oauth2Client oauth2Client,
	agentRepo generalAgentRepo,
	emailRepo generalEmailRepo,
	taskRepo generalTaskRepo,
	contactRepo generalContactRepo,
	conversationRepo generalConversationRepo,
	organizationRepo *organization.OrganizationRepository,
) *GeneralEmailWorker {
	return &GeneralEmailWorker{
		db:               db,
		oauth2Client:     oauth2Client,
		agentRepo:        agentRepo,
		emailRepo:        emailRepo,
		taskRepo:         taskRepo,
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		organizationRepo: organizationRepo,
		gmailFactory: func(ctx context.Context, accessToken string) (gmailClient, error) {
			return gmail.NewClient(ctx, accessToken)
		},
	}
}

// HandleSendEmail processes general email sending tasks.
func (w *GeneralEmailWorker) HandleSendEmail(ctx context.Context, task *asynq.Task) error {
	var payload SendEmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	logger.Logger.Info().
		Str("agent_id", payload.AgentID.String()).
		Str("task_id", safeUUID(payload.TaskID)).
		Str("campaign_id", safeUUID(payload.CampaignID)).
		Str("template_id", safeUUID(payload.TemplateID)).
		Int("body_length", len(payload.Body)).
		Msg("Processing send email task")

	// Get agent with valid credentials
	agent, err := w.agentRepo.FindByID(ctx, payload.AgentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}
	if agent == nil {
		return fmt.Errorf("agent not found: %s", payload.AgentID)
	}

	// Get and refresh token if needed
	tokenStore, err := agent.GetGoogleTokenStore()
	if err != nil {
		return fmt.Errorf("no Gmail credentials: %w", err)
	}

	// Personalize template content/subject using contact + agent data.
	var contact *domain.Contact
	if payload.ContactID != uuid.Nil {
		contact, err = w.contactRepo.FindByID(ctx, payload.ContactID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to load contact: %w", err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contact = nil
		}
	}

	// Note: Agent.Organization relationship removed to avoid circular imports
	// Organization data should be loaded separately using relations package if needed
	templateData := buildTemplateData(agent, contact)
	subject := applyTemplateData(payload.Subject, templateData)
	body := applyTemplateData(payload.Body, templateData)
	body, sendAsHTML := prepareEmailBody(body, payload.IsHTML)

	if strings.TrimSpace(subject) == "" || strings.TrimSpace(body) == "" {
		err := fmt.Errorf("email subject or body empty after templating")
		logger.Logger.Warn().
			Str("agent_id", payload.AgentID.String()).
			Str("task_id", safeUUID(payload.TaskID)).
			Str("campaign_id", safeUUID(payload.CampaignID)).
			Msg("Skipping send: empty subject/body after templating")
		if payload.TaskID != nil {
			w.updateTaskStatus(ctx, *payload.TaskID, domain.TaskStatusFailed, err.Error())
		}
		return err
	}

	if len(subject) > 998 {
		err := fmt.Errorf("email subject exceeds 998 characters after templating")
		logger.Logger.Warn().
			Str("agent_id", payload.AgentID.String()).
			Str("task_id", safeUUID(payload.TaskID)).
			Str("campaign_id", safeUUID(payload.CampaignID)).
			Int("subject_length", len(subject)).
			Msg("Skipping send: subject too long")
		if payload.TaskID != nil {
			w.updateTaskStatus(ctx, *payload.TaskID, domain.TaskStatusFailed, err.Error())
		}
		return err
	}

	// Check if token expired and refresh
	if tokenStore.Expiry.Before(time.Now().Add(5 * time.Minute)) {
		logger.Logger.Info().Str("agent_id", agent.ID.String()).Msg("Refreshing expired token")

		newToken, err := w.oauth2Client.RefreshToken(ctx, tokenStore.RefreshToken)
		if err != nil {
			if strings.Contains(err.Error(), "invalid_grant") {
				w.markAgentInvalidGrant(ctx, agent)
			}
			logger.Logger.Error().
				Err(err).
				Str("agent_id", agent.ID.String()).
				Msg("Failed to refresh Gmail token")
			if payload.TaskID != nil {
				w.updateTaskStatus(ctx, *payload.TaskID, domain.TaskStatusFailed, err.Error())
			}
			return fmt.Errorf("failed to refresh token: %w", err)
		}

		tokenStore.AccessToken = newToken.AccessToken
		if newToken.RefreshToken != "" {
			tokenStore.RefreshToken = newToken.RefreshToken
		}
		tokenStore.Expiry = newToken.Expiry

		if err := agent.UpdateGoogleTokenStore(tokenStore); err != nil {
			logger.Logger.Error().
				Err(err).
				Str("agent_id", agent.ID.String()).
				Msg("Failed to update Gmail token store on agent")
			if payload.TaskID != nil {
				w.updateTaskStatus(ctx, *payload.TaskID, domain.TaskStatusFailed, err.Error())
			}
			return fmt.Errorf("failed to update token: %w", err)
		}

		if err := w.agentRepo.Update(ctx, agent.ID, agent); err != nil {
			logger.Logger.Error().
				Err(err).
				Str("agent_id", agent.ID.String()).
				Msg("Failed to persist refreshed Gmail token to database")
			if payload.TaskID != nil {
				w.updateTaskStatus(ctx, *payload.TaskID, domain.TaskStatusFailed, err.Error())
			}
			return fmt.Errorf("failed to save agent: %w", err)
		}
	}

	// Create Gmail client
	gmailClient, err := w.gmailFactory(ctx, tokenStore.AccessToken)
	if err != nil {
		logger.Logger.Error().
			Err(err).
			Str("agent_id", agent.ID.String()).
			Msg("Failed to create Gmail client")
		return fmt.Errorf("failed to create Gmail client: %w", err)
	}

	// Ensure proper cleanup of Gmail client
	defer func() {
		gmailClient.Close()
		logger.Logger.Debug().
			Str("agent_id", agent.ID.String()).
			Msg("Gmail client closed successfully")
	}()
	sendStart := time.Now()
	var sendCtx context.Context
	var cancelSend context.CancelFunc
	// Use remaining time from parent context or 30 seconds, whichever is smaller
	if deadline, ok := ctx.Deadline(); ok {
		remainingTime := time.Until(deadline)
		if remainingTime < 30*time.Second && remainingTime > 0 {
			sendCtx, cancelSend = context.WithTimeout(ctx, remainingTime)
		} else {
			sendCtx, cancelSend = context.WithTimeout(ctx, 30*time.Second)
		}
	} else {
		sendCtx, cancelSend = context.WithTimeout(ctx, 30*time.Second)
	}
	defer cancelSend()

	message, err := gmailClient.SendMessage(sendCtx, &gmail.SendMessageRequest{
		From:      agent.Email,
		To:        payload.To,
		Cc:        strings.Join(payload.Cc, ","),
		Bcc:       strings.Join(payload.Bcc, ","),
		Subject:   subject,
		Body:      body,
		IsHTML:    sendAsHTML,
		InReplyTo: payload.InReplyTo,
	})

	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			w.markAgentInvalidGrant(ctx, agent)
		}
		logger.Logger.Error().
			Err(err).
			Str("agent_id", agent.ID.String()).
			Str("task_id", safeUUID(payload.TaskID)).
			Str("campaign_id", safeUUID(payload.CampaignID)).
			Str("template_id", safeUUID(payload.TemplateID)).
			Int("body_length", len(body)).
			Dur("send_duration", time.Since(sendStart)).
			Msg("Failed to send email via Gmail API")

		if payload.TaskID != nil {
			w.updateTaskStatus(ctx, *payload.TaskID, domain.TaskStatusFailed, err.Error())
		}
		return fmt.Errorf("failed to send email: %w", err)
	}
	if message == nil {
		err := fmt.Errorf("gmail send returned nil message")
		logger.Logger.Error().
			Str("agent_id", agent.ID.String()).
			Str("task_id", safeUUID(payload.TaskID)).
			Str("campaign_id", safeUUID(payload.CampaignID)).
			Str("template_id", safeUUID(payload.TemplateID)).
			Msg("Failed to send email: nil response")
		if payload.TaskID != nil {
			w.updateTaskStatus(ctx, *payload.TaskID, domain.TaskStatusFailed, err.Error())
		}
		return err
	}

	logger.Logger.Info().
		Str("message_id", message.ID).
		Str("thread_id", message.ThreadID).
		Dur("send_duration", time.Since(sendStart)).
		Msg("Email sent successfully")

	var conversation *domain.Conversation
	if message.ThreadID != "" {
		newConversation := &domain.Conversation{
			UserID:        agent.UserID,
			GmailThreadID: message.ThreadID,
			AgentID:       &agent.ID,
			CampaignID:    payload.CampaignID,
			Status:        domain.ConversationStatusUnread,
			Replied:       false,
		}
		if created, err := w.conversationRepo.Create(ctx, newConversation); err != nil {
			logger.Logger.Warn().
				Err(err).
				Str("thread_id", message.ThreadID).
				Msg("Failed to create conversation for sent email")
		} else {
			conversation = created
		}
	}

	// Create email record in database
	email := &domain.Email{
		UserID:         agent.UserID,
		CampaignID:     payload.CampaignID,
		GmailMessageID: message.ID,
		FromEmail:      agent.Email,
		ToEmail:        payload.To,
		Subject:        subject,
		Content:        body,
		Status:         domain.EmailStatusSent,
	}

	if conversation != nil {
		email.ConversationID = &conversation.ID
		if conversation.CampaignID != nil {
			email.CampaignID = conversation.CampaignID
		}
	}

	if _, err := w.emailRepo.Create(ctx, email); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to save email record")
		// Don't return error - email was sent successfully
	}

	// Update task status to completed if task_id provided
	if payload.TaskID != nil {
		w.updateTaskStatus(ctx, *payload.TaskID, domain.TaskStatusDone, "")
	}

	// Update conversation if in reply to
	if payload.InReplyTo != "" && message.ThreadID != "" {
		w.updateConversationThread(ctx, message.ThreadID, email.ID)
	}

	return nil
}

// safeUUID returns the UUID string if pointer is non-nil, else empty string for logging.
func safeUUID(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// updateTaskStatus updates a task's status and error message.
func (w *GeneralEmailWorker) updateTaskStatus(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, errorMsg string) {
	task, err := w.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		logger.Logger.Error().Err(err).Str("task_id", taskID.String()).Msg("Failed to find task")
		return
	}

	task.Status = status
	if errorMsg != "" {
		task.Error = &errorMsg
	}

	now := time.Now()
	if status == domain.TaskStatusDone {
		task.DoneAt = &now
	}

	if err := w.taskRepo.Update(ctx, taskID, task); err != nil {
		logger.Logger.Error().Err(err).Str("task_id", taskID.String()).Msg("Failed to update task status")
	}
}

// updateConversationThread updates or creates a conversation for an email thread.
func (w *GeneralEmailWorker) updateConversationThread(ctx context.Context, threadID string, emailID uuid.UUID) {
	// This function would need repository filter support - simplified for now
	logger.Logger.Info().
		Str("thread_id", threadID).
		Str("email_id", emailID.String()).
		Msg("Updating conversation thread")
}

// markAgentInvalidGrant marks the agent as having an invalid Google grant so the UI can prompt re-authorization.
func (w *GeneralEmailWorker) markAgentInvalidGrant(ctx context.Context, agent *domain.Agent) {
	invalid := domain.AgentStatusInvalidGrant
	agent.Status = invalid
	valid := false
	agent.ValidCred = &valid

	if err := w.agentRepo.Update(ctx, agent.ID, agent); err != nil {
		logger.Logger.Error().
			Err(err).
			Str("agent_id", agent.ID.String()).
			Msg("Failed to mark agent invalid_grant")
	}
}

// HandleSyncGmailHistory pulls message changes from Gmail History API and persists inbound emails.
func (w *GeneralEmailWorker) HandleSyncGmailHistory(ctx context.Context, task *asynq.Task) error {

	var payload SyncGmailHistoryPayload
	if err := worker.ParsePayload(task, &payload); err != nil {
		return fmt.Errorf("failed to parse sync_history payload: %w", err)
	}

	if payload.AgentID == uuid.Nil {
		logger.FromContext(ctx).Warn().Msg("sync_history skipped: missing agent_id")
		return nil
	}

	agent, err := w.agentRepo.FindByID(ctx, payload.AgentID)
	if err != nil {
		return fmt.Errorf("failed to load agent %s: %w", payload.AgentID, err)
	}
	if agent == nil {
		logger.FromContext(ctx).Warn().
			Str("agent_id", payload.AgentID.String()).
			Msg("sync_history skipped: agent not found")
		return nil
	}

	tokenStore, err := agent.GetGoogleTokenStore()
	if err != nil {
		return fmt.Errorf("no Gmail credentials: %w", err)
	}

	// Refresh token if needed (reuse logic similar to HandleSendEmail).
	if tokenStore.Expiry.Before(time.Now().Add(5 * time.Minute)) {
		logger.FromContext(ctx).Info().Str("agent_id", agent.ID.String()).Msg("Refreshing expired token (sync history)")

		newToken, err := w.oauth2Client.RefreshToken(ctx, tokenStore.RefreshToken)
		if err != nil {
			if strings.Contains(err.Error(), "invalid_grant") {
				w.markAgentInvalidGrant(ctx, agent)
			}
			return fmt.Errorf("failed to refresh token: %w", err)
		}

		tokenStore.AccessToken = newToken.AccessToken
		if newToken.RefreshToken != "" {
			tokenStore.RefreshToken = newToken.RefreshToken
		}
		tokenStore.Expiry = newToken.Expiry

		if err := agent.UpdateGoogleTokenStore(tokenStore); err != nil {
			return fmt.Errorf("failed to update token store: %w", err)
		}
		if err := w.agentRepo.Update(ctx, agent.ID, agent); err != nil {
			return fmt.Errorf("failed to persist refreshed token: %w", err)
		}
	}

	gmailClient, err := w.gmailFactory(ctx, tokenStore.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %w", err)
	}
	defer gmailClient.Close()

	startHistoryID := strings.TrimSpace(payload.StartHistoryID)
	if startHistoryID == "" {
		startHistoryID = strings.TrimSpace(agent.LastHistoryID)
	}
	if startHistoryID == "" {
		logger.FromContext(ctx).Warn().
			Str("agent_id", agent.ID.String()).
			Msg("sync_history skipped: missing start history id")
		return nil
	}

	nextPage := ""
	latestHistory := startHistoryID
	seenMessages := make(map[string]struct{})
	const maxHistoryPages = 100
	pageCount := 0

	for {
		// Abort if context cancelled to avoid infinite loops.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if pageCount >= maxHistoryPages {
			logger.FromContext(ctx).Warn().
				Int("max_pages", maxHistoryPages).
				Str("agent_id", agent.ID.String()).
				Msg("sync_history stopped after reaching page limit")
			break
		}

		historyResult, err := gmailClient.ListHistory(ctx, startHistoryID, nextPage)
		if err != nil {
			return fmt.Errorf("failed to list Gmail history: %w", err)
		}

		// Fetch emails for unique message IDs.
		for _, hm := range historyResult.Messages {
			if _, exists := seenMessages[hm.ID]; exists {
				continue
			}
			seenMessages[hm.ID] = struct{}{}

			msg, err := gmailClient.GetMessage(ctx, hm.ID)
			if err != nil {
				logger.FromContext(ctx).Warn().
					Err(err).
					Str("agent_id", agent.ID.String()).
					Str("message_id", hm.ID).
					Msg("Failed to fetch Gmail message; skipping")
				continue
			}

			if err := w.persistInboundMessage(ctx, agent, msg); err != nil {
				logger.FromContext(ctx).Warn().
					Err(err).
					Str("agent_id", agent.ID.String()).
					Str("message_id", msg.ID).
					Msg("Failed to persist Gmail message")
			}
		}

		if historyResult.NewestHistory != "" {
			latestHistory = historyResult.NewestHistory
		}

		if historyResult.NextPageToken == "" {
			break
		}
		nextPage = historyResult.NextPageToken
		pageCount++
	}

	if latestHistory != "" && latestHistory != agent.LastHistoryID {
		if err := w.agentRepo.UpdateLastHistoryID(ctx, agent.ID, latestHistory); err != nil {
			return fmt.Errorf("failed to update agent history id: %w", err)
		}
	}
	return nil
}

// persistInboundMessage saves a Gmail message to email/conversation stores and marks replied status.
func (w *GeneralEmailWorker) persistInboundMessage(ctx context.Context, agent *domain.Agent, msg *gmail.Message) error {
	if msg == nil {
		return nil
	}

	// Skip outbound messages (already saved when sending).
	if hasLabel(msg.LabelIDs, "SENT") {
		return nil
	}

	// Skip if already stored.
	existing, err := w.emailRepo.FindByGmailMessageID(ctx, msg.ID)
	if err == nil && existing != nil {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	fromHeader := strings.TrimSpace(msg.Headers["From"])
	toHeader := strings.TrimSpace(msg.Headers["To"])

	fromEmail := fromHeader
	if parsed, err := mail.ParseAddress(fromHeader); err == nil && parsed != nil {
		fromEmail = parsed.Address
	}
	toEmail := toHeader
	if parsed, err := mail.ParseAddress(toHeader); err == nil && parsed != nil {
		toEmail = parsed.Address
	}

	conversation, err := w.conversationRepo.FindByGmailThreadID(ctx, msg.ThreadID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if conversation == nil {
		// Do not create new conversations for unknown threads; skip processing.
		logger.FromContext(ctx).Info().
			Str("thread_id", msg.ThreadID).
			Msg("Skipping inbound email: thread not found")
		return nil
	}

	email := &domain.Email{
		UserID:         agent.UserID,
		ConversationID: &conversation.ID,
		GmailMessageID: msg.ID,
		FromEmail:      fromEmail,
		ToEmail:        toEmail,
		Subject:        msg.Headers["Subject"],
		Content:        firstNonEmpty(msg.Body, msg.Snippet),
		Labels:         pq.StringArray(msg.LabelIDs),
		Status:         domain.EmailStatusInbox,
	}

	if _, err := w.emailRepo.Create(ctx, email); err != nil {
		return fmt.Errorf("failed to save email: %w", err)
	}

	// Mark conversation as replied (contact responded).
	if err := w.conversationRepo.SetReplied(ctx, conversation.ID, true); err != nil {
		return fmt.Errorf("failed to mark conversation replied: %w", err)
	}

	return nil
}

func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, target) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

var (
	mdBoldRe             = regexp.MustCompile(`\*\*(.+?)\*\*`)
	mdItalicStarRe       = regexp.MustCompile(`\*([^\s][^*\n]*?[^\s])\*`)
	mdItalicUnderscoreRe = regexp.MustCompile(`_([^\s][^_\n]*?[^\s])_`)
	mdUnderlineRe        = regexp.MustCompile(`__(.+?)__`)
	mdStrikeRe           = regexp.MustCompile(`~~([^~]+?)~~`)
)

// SendInviteMemberEmail send invite email for member in API POST /v2/organizations/{organization_id}/members/invite
func (w *GeneralEmailWorker) SendInviteMemberEmail(ctx context.Context, task *asynq.Task) error {
	var payload SendInviteMemberEmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	resendSvc := resendService.NewResendService()

	// In invitation.html, change {name} to {{.Name}}, {url} to {{.Url}}
	wd, _ := os.Getwd()
	tmplPath := filepath.Join(wd, "internal", "templates", "invitation.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	type InviteData struct {
		Name string
		Url  string
	}
	var buf bytes.Buffer
	data := InviteData{Name: payload.MemberName, Url: payload.Url}

	if err := tmpl.Execute(&buf, &data); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}
	htmlBody := buf.String()

	err = resendSvc.SendEmail(
		os.Getenv("RESEND_SYSTEM_EMAIL"),
		[]string{payload.Email},
		SUBJECT_INVITATION,
		resendService.EmailContent{HTML: htmlBody},
		nil, // cc
		nil, // bcc
	)
	if err != nil {
		log.Printf("Failed to send invite email to %s: %v", payload.Email, err)
		return err
	}

	log.Printf("Invite email sent successfully to %s", payload.Email)

	return nil
}

func buildTemplateData(agent *domain.Agent, contact *domain.Contact) map[string]string {
	data := map[string]string{}

	// Contact fields
	if contact != nil {
		data["contact_name"] = normalizeTemplateValue(contact.Name)
		// For backwards compatibility, extract first name from full name
		nameParts := strings.Fields(contact.Name)
		if len(nameParts) > 0 {
			data["contact_first_name"] = normalizeTemplateValue(nameParts[0])
		}
		if len(nameParts) > 1 {
			data["contact_last_name"] = normalizeTemplateValue(strings.Join(nameParts[1:], " "))
		}
		// Extract email from profile JSONB
		contactEmail := extractEmailFromProfile(contact.Profile)
		data["contact_email"] = normalizeTemplateValue(contactEmail)
		data["contact_company"] = normalizeTemplateValue(contact.Company)
		data["contact_job_title"] = normalizeTemplateValue(contact.JobTitle)
		data["contact_country"] = normalizeTemplateValue(contact.Country)
	}

	// Agent fields
	senderName := normalizeTemplateValue(agent.Name)
	organizationName := ""
	// Note: Agent.Organization relationship removed to avoid circular imports
	// Organization data should be loaded separately using relations package if needed

	signatureTemplate := strings.TrimSpace(agent.Signature)
	if signatureTemplate == "" {
		signatureTemplate = "Best regards,\n" + senderName
	}

	signatureData := map[string]string{
		"sender_name":       senderName,
		"organization_name": organizationName,
	}
	signature := applyTemplateData(signatureTemplate, signatureData)

	// Ensure signature starts on a new line for readability.
	if signature != "" && !strings.HasPrefix(signature, "\n") {
		signature = "\n" + signature
	}

	data["agent_signature"] = signature
	data["sender_name"] = senderName
	data["organization_name"] = organizationName

	return data
}

func normalizeTemplateValue(val string) string {
	val = strings.TrimSpace(val)
	if val == "" || strings.EqualFold(val, domain.NOT_AVAILABLE) {
		return ""
	}
	return val
}

func prepareEmailBody(body string, isHTML bool) (string, bool) {
	normalized := strings.ReplaceAll(body, "\r\n", "\n")

	if isHTML || looksLikeMarkdown(normalized) {
		return markdownToHTML(normalized), true
	}

	return normalized, false
}

func looksLikeMarkdown(body string) bool {
	return strings.Contains(body, "**") ||
		strings.Contains(body, "__") ||
		strings.Contains(body, "_") ||
		strings.Contains(body, "*") ||
		strings.Contains(body, "~~") ||
		strings.Contains(body, "<")
}

func markdownToHTML(body string) string {
	html := mdStrikeRe.ReplaceAllString(body, "<s>$1</s>")
	html = mdBoldRe.ReplaceAllString(html, "<strong>$1</strong>")
	html = mdUnderlineRe.ReplaceAllString(html, "<u>$1</u>")
	html = mdItalicStarRe.ReplaceAllString(html, "<em>$1</em>")
	html = mdItalicUnderscoreRe.ReplaceAllString(html, "<em>$1</em>")
	html = strings.ReplaceAll(html, "\n", "<br/>")
	return html
}

func applyTemplateData(template string, data map[string]string) string {
	if template == "" || len(data) == 0 {
		return template
	}

	result := template
	for key, value := range data {
		single := "{" + key + "}"
		double := "{{" + key + "}}"
		result = strings.ReplaceAll(result, double, value)
		result = strings.ReplaceAll(result, single, value)
	}
	return result
}
