package campaign

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	templatedomain "github.com/rockship/cosmo-agents-go/internal/domain/template"
	agent "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	workerpayloads "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

type campaignRepo interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error)
}

type contactRepository interface {
	FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Contact, error)
	FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Contact], error)
	FindIDsByListContact(ctx context.Context, listID uuid.UUID, userID uuid.UUID, orgID *uuid.UUID, offset int, limit int) ([]uuid.UUID, error)
}

type templateRepository interface {
	FindByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Template, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Template, error)
}

type taskRepository interface {
	CreateTasks(ctx context.Context, attrs []domain.TaskAttributes) ([]uuid.UUID, error)
	FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Task], error)
	Update(ctx context.Context, id uuid.UUID, entity *domain.Task) error
}

// replyChecker reports which contacts have answered a campaign: an incoming
// interaction logged after one of the campaign's emails went out to them.
type replyChecker interface {
	ContactsRepliedInCampaign(ctx context.Context, campaignID uuid.UUID) (map[uuid.UUID]bool, error)
}

type enqueueClient interface {
	EnqueueTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error)
	EnqueueTaskAt(ctx context.Context, taskType string, payload interface{}, processAt time.Time, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// Worker handles campaign-related background tasks.
type Worker struct {
	db           *gorm.DB
	workerClient enqueueClient
	campaignRepo campaignRepo
	contactRepo  contactRepository
	templateRepo templateRepository
	taskRepo     taskRepository
	agentRepo    *agent.AgentRepository

	// replies, when set, stops a sequence once the contact has answered.
	replies replyChecker
}

// New creates a new campaign worker.
func New(
	db *gorm.DB,
	workerClient enqueueClient,
	campaignRepo campaignRepo,
	contactRepo contactRepository,
	templateRepo templateRepository,
	taskRepo taskRepository,
	agentRepo *agent.AgentRepository,
) *Worker {
	return &Worker{
		db:           db,
		workerClient: workerClient,
		campaignRepo: campaignRepo,
		contactRepo:  contactRepo,
		templateRepo: templateRepo,
		taskRepo:     taskRepo,
		agentRepo:    agentRepo,
	}
}

// WithReplyChecker stops follow-up emails to contacts who have replied.
func (w *Worker) WithReplyChecker(r replyChecker) *Worker {
	w.replies = r
	return w
}

// HandleExecuteCampaign processes campaign execution.
func (w *Worker) HandleExecuteCampaign(ctx context.Context, task *asynq.Task) error {
	var payload workerpayloads.ExecuteCampaignPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	logger.Logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Str("agent_id", payload.AgentID.String()).
		Msg("Executing campaign")

	// Get campaign
	campaign, err := w.campaignRepo.FindByID(ctx, payload.CampaignID)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	// Check if campaign is active
	if campaign.Status != domain.CampaignStatusActive {
		return fmt.Errorf("campaign is not active: %s", campaign.Status)
	}

	// Get campaign templates
	templates, err := w.templateRepo.FindByCampaignID(ctx, campaign.ID)
	if err != nil {
		return fmt.Errorf("failed to get templates: %w", err)
	}

	if len(templates) == 0 {
		return fmt.Errorf("no templates found for campaign")
	}

	// Determine base schedule time (campaign.schedule or now)
	baseSchedule := time.Now()
	if campaign.Schedule != nil {
		baseSchedule = *campaign.Schedule
	}

	// Get contacts to process
	var contactIDs []uuid.UUID
	if len(payload.ContactIDs) > 0 {
		contactIDs = payload.ContactIDs
	} else {
		// Prefer explicit list contact if campaign is tied to a list; fall back to all contacts for the user.
		const maxContacts = 20000
		contactIDs = make([]uuid.UUID, 0, maxContacts)
		offset := 0
		pageSize := 1000

		fetchFromList := campaign.ListContactID != nil
		listID := campaign.ListContactID

		for {
			var batch []uuid.UUID
			var err error

			if fetchFromList {
				batch, err = w.contactRepo.FindIDsByListContact(ctx, *listID, payload.UserID, campaign.OrganizationID, offset, pageSize)
			} else {
				// Get all contacts for user (could be filtered by campaign criteria)
				filter := baseRepo.Filter{
					"user_id": payload.UserID,
				}
				result, errFind := w.contactRepo.FindAll(ctx, filter, &baseRepo.PaginationParams{Offset: offset, Limit: pageSize})
				if errFind != nil {
					return fmt.Errorf("failed to get contacts: %w", errFind)
				}
				for _, contact := range result.List {
					batch = append(batch, contact.ID)
				}
				// When not using list contact, total comes from pagination result.
				if len(result.List) == 0 {
					break
				}
			}

			if err != nil {
				return fmt.Errorf("failed to get contacts from list: %w", err)
			}
			if len(batch) == 0 {
				break
			}

			contactIDs = append(contactIDs, batch...)
			if len(contactIDs) >= maxContacts {
				logger.Logger.Warn().
					Str("campaign_id", campaign.ID.String()).
					Int("max_contacts", maxContacts).
					Msg("Contact list capped for task creation")
				contactIDs = contactIDs[:maxContacts]
				break
			}

			offset += len(batch)
			if len(batch) < pageSize {
				break
			}
		}
	}

	logger.Logger.Info().
		Str("campaign_id", campaign.ID.String()).
		Int("contacts", len(contactIDs)).
		Int("templates", len(templates)).
		Msg("Creating campaign tasks")

	// Each step waits its send_after in working days after the step before
	// it. This used to add send_after hours to the start, so "1 working day"
	// in the editor sent the first follow-up an hour after the first email.
	sendTimes := templatedomain.SequenceSendTimes(baseSchedule, templates)

	// Create tasks for each contact x template combination
	tasksCreated := 0
	for _, contactID := range contactIDs {
		for i, template := range templates {
			// Create task
			taskAttrs := domain.TaskAttributes{
				ContactID:  contactID,
				CampaignID: campaign.ID,
				TemplateID: template.ID,
				Status:     domain.TaskStatusPending,
			}

			scheduleAt := sendTimes[i]
			taskAttrs.ScheduleAt = &scheduleAt

			// Use CreateTasks which handles upserts
			taskIDs, err := w.taskRepo.CreateTasks(ctx, []domain.TaskAttributes{taskAttrs})
			if err != nil {
				logger.Logger.Error().Err(err).Msg("Failed to create task")
				continue
			}

			if len(taskIDs) > 0 {
				tasksCreated++
			}
		}
	}

	logger.Logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Int("tasks_created", tasksCreated).
		Msg("Campaign tasks created successfully")

	// Immediately enqueue scheduler so pending tasks actually get sent.
	schedulePayload := workerpayloads.ScheduleTasksPayload{
		CampaignID: payload.CampaignID,
		ContactIDs: contactIDs,
		AgentID:    payload.AgentID,
	}
	// nolint:errcheck // handled below

	if _, err := w.workerClient.EnqueueTask(ctx, queueworker.TypeScheduleTasks, schedulePayload); err != nil {
		return fmt.Errorf("failed to enqueue schedule tasks: %w", err)
	}

	logger.Logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Int("contacts", len(contactIDs)).
		Msg("Schedule tasks enqueued")

	return nil
}

// HandleGenerateEmail generates email content using AI.
func (w *Worker) HandleGenerateEmail(ctx context.Context, task *asynq.Task) error {
	var payload workerpayloads.GenerateEmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	logger.Logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Str("contact_id", payload.ContactID.String()).
		Str("template_id", payload.TemplateID.String()).
		Msg("Generating email content")

	// TODO: Implement AI email generation
	// 1. Get contact information
	// 2. Get template
	// 3. Get knowledge base context
	// 4. Call OpenAI API to generate personalized content
	// 5. Store generated content
	// 6. Enqueue send email task

	return fmt.Errorf("AI email generation not yet implemented")
}

// HandleScheduleTasks schedules tasks for a campaign.
func (w *Worker) HandleScheduleTasks(ctx context.Context, task *asynq.Task) error {
	var payload workerpayloads.ScheduleTasksPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	logger.Logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Int("contacts", len(payload.ContactIDs)).
		Msg("Scheduling campaign tasks")

	// Get pending tasks for this campaign
	filter := baseRepo.Filter{
		"campaign_id": payload.CampaignID,
		"status":      domain.TaskStatusPending,
	}

	result, err := w.taskRepo.FindAll(ctx, filter, &baseRepo.PaginationParams{Offset: 0, Limit: 10000})
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	logger.Logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Int("pending_tasks", int(result.Total)).
		Msg("Found pending tasks")

	// Enqueue send email tasks for ready tasks
	tasksScheduled := 0
	var nextSchedule *time.Time

	// Batch load contacts and templates to avoid N+1 queries
	contactIDs := make([]uuid.UUID, 0, len(result.List))
	templateIDs := make([]uuid.UUID, 0, len(result.List))
	contactSeen := make(map[uuid.UUID]struct{})
	templateSeen := make(map[uuid.UUID]struct{})
	// This loop is intentionally sequential; guard the seen maps if parallelizing.
	for _, t := range result.List {
		if _, ok := contactSeen[t.ContactID]; !ok {
			contactSeen[t.ContactID] = struct{}{}
			contactIDs = append(contactIDs, t.ContactID)
		}
		if _, ok := templateSeen[t.TemplateID]; !ok {
			templateSeen[t.TemplateID] = struct{}{}
			templateIDs = append(templateIDs, t.TemplateID)
		}
	}

	contactMap, err := w.contactRepo.FindByIDs(ctx, contactIDs)
	if err != nil {
		return fmt.Errorf("failed to batch load contacts: %w", err)
	}
	templateMap, err := w.templateRepo.FindByIDs(ctx, templateIDs)
	if err != nil {
		return fmt.Errorf("failed to batch load templates: %w", err)
	}

	// A follow-up says "if the contact does not reply", so once they have
	// replied the rest of their sequence is cancelled rather than sent. Only
	// contacts who were already emailed by this campaign can appear here, so
	// a first email is never cancelled.
	replied := map[uuid.UUID]bool{}
	if w.replies != nil {
		r, err := w.replies.ContactsRepliedInCampaign(ctx, payload.CampaignID)
		if err != nil {
			// Better to hold the run than to mail people who have answered.
			return fmt.Errorf("failed to check replies: %w", err)
		}
		replied = r
	}

	for _, t := range result.List {
		now := time.Now()
		if replied[t.ContactID] {
			t.Status = domain.TaskStatusCancelled
			reason := "contact replied; follow-up not sent"
			t.Error = &reason
			if err := w.taskRepo.Update(ctx, t.ID, &t); err != nil {
				logger.Logger.Error().Err(err).Str("task_id", t.ID.String()).Msg("Failed to cancel follow-up for a contact who replied")
			}
			continue
		}

		// Check if task is ready to be sent
		if t.ScheduleAt != nil && t.ScheduleAt.After(now) {
			// Schedule for later
			logger.Logger.Info().
				Str("task_id", t.ID.String()).
				Str("campaign_id", payload.CampaignID.String()).
				Time("schedule_at", *t.ScheduleAt).
				Msg("Skipping task; schedule time in the future")
			// Track earliest future schedule to re-enqueue scheduler
			if nextSchedule == nil || t.ScheduleAt.Before(*nextSchedule) {
				nextSchedule = t.ScheduleAt
			}
			continue
		}

		// Get contact and template to prepare email
		contact, ok := contactMap[t.ContactID]
		if !ok || contact == nil {
			logger.Logger.Error().Str("contact_id", t.ContactID.String()).Msg("Contact not found in batch fetch")
			t.Status = domain.TaskStatusFailed
			errMsg := "contact not found"
			t.Error = &errMsg
			_ = w.taskRepo.Update(ctx, t.ID, &t)
			continue
		}

		template, ok := templateMap[t.TemplateID]
		if !ok || template == nil {
			logger.Logger.Error().Str("template_id", t.TemplateID.String()).Msg("Template not found in batch fetch")
			t.Status = domain.TaskStatusFailed
			errMsg := "template not found"
			t.Error = &errMsg
			_ = w.taskRepo.Update(ctx, t.ID, &t)
			continue
		}

		if strings.TrimSpace(template.Subject) == "" || strings.TrimSpace(template.Content) == "" {
			logger.Logger.Warn().
				Str("task_id", t.ID.String()).
				Str("campaign_id", payload.CampaignID.String()).
				Str("template_id", template.ID.String()).
				Msg("Skipping enqueue: template subject/body is empty")
			// Mark task failed to avoid retries on empty template
			t.Status = domain.TaskStatusFailed
			errMsg := "template subject or body is empty"
			t.Error = &errMsg
			if err := w.taskRepo.Update(ctx, t.ID, &t); err != nil {
				logger.Logger.Error().Err(err).Str("task_id", t.ID.String()).Msg("Failed to update task status on empty template")
			}
			continue
		}

		// Enqueue AI generate email task (rewrites template per contact, then auto-enqueues send)
		genPayload := workerpayloads.GenerateEmailPayload{
			CampaignID: payload.CampaignID,
			ContactID:  contact.ID,
			TemplateID: template.ID,
			TaskID:     &t.ID,
		}

		if _, err := w.workerClient.EnqueueTask(ctx, queueworker.TypeGenerateEmail, genPayload); err != nil {
			logger.Logger.Error().Err(err).Str("task_id", t.ID.String()).Msg("Failed to enqueue send email task")
			continue
		}

		// Update task status to running
		t.Status = domain.TaskStatusRunning
		t.TriggeredAt = &now

		if err := w.taskRepo.Update(ctx, t.ID, &t); err != nil {
			logger.Logger.Error().Err(err).Str("task_id", t.ID.String()).Msg("Failed to update task status")
		}

		tasksScheduled++
	}

	logger.Logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Int("tasks_scheduled", tasksScheduled).
		Msg("Campaign tasks scheduled successfully")

	// If there are pending tasks in the future, re-enqueue scheduler at the earliest schedule_at.
	if nextSchedule != nil {
		if _, err := w.workerClient.EnqueueTaskAt(ctx, queueworker.TypeScheduleTasks, payload, *nextSchedule); err != nil {
			logger.Logger.Error().
				Err(err).
				Str("campaign_id", payload.CampaignID.String()).
				Time("next_schedule_at", *nextSchedule).
				Msg("Failed to re-enqueue schedule tasks")
		} else {
			logger.Logger.Info().
				Str("campaign_id", payload.CampaignID.String()).
				Time("next_schedule_at", *nextSchedule).
				Msg("Re-enqueued schedule tasks for future run")
		}
	}

	return nil
}
