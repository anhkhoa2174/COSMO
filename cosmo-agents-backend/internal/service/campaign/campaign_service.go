package campaign

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	templatedomain "github.com/rockship/cosmo-agents-go/internal/domain/template"
	agent "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	taskRepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	templateRepo "github.com/rockship/cosmo-agents-go/internal/repository/template"
	schedulerService "github.com/rockship/cosmo-agents-go/internal/service/scheduler"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// CampaignServiceErrors
var (
	ErrInvalidCampaignStatus = fmt.Errorf("invalid campaign status")
	ErrMisconfiguredCampaign = fmt.Errorf("misconfigured campaign")
	ErrEmptyContactList      = fmt.Errorf("contact list is empty")
	ErrNoTemplates           = fmt.Errorf("no outreach emails are defined")
	ErrNoAgent               = fmt.Errorf("no agent is configured for the campaign")
)

// CampaignService handles campaign-related operations
type CampaignService struct {
	schedulerService *schedulerService.SchedulerService
	workerClient     *worker.Client
	taskRepo         *taskRepo.TaskRepository
	templateRepo     *templateRepo.TemplateRepository
	agentRepo        *agent.AgentRepository
}

// NewCampaignService creates a new campaign service
func NewCampaignService(
	schedulerService *schedulerService.SchedulerService,
	workerClient *worker.Client,
	taskRepo *taskRepo.TaskRepository,
	templateRepo *templateRepo.TemplateRepository,
	agentRepo *agent.AgentRepository,
) *CampaignService {
	return &CampaignService{
		schedulerService: schedulerService,
		workerClient:     workerClient,
		taskRepo:         taskRepo,
		templateRepo:     templateRepo,
		agentRepo:        agentRepo,
	}
}

// ScheduleOutreachToContacts schedules outreach emails to be sent to multiple contacts
//
// Scheduling rules:
// - If the scheduled datetime of the campaign has passed, emails are scheduled at now + 1 minute
// - If the scheduled datetime is in the future, emails are scheduled at that time + 1 minute
//
// Deprecated: This method is deprecated, use TriggerEmailSequenceNode instead
func (s *CampaignService) ScheduleOutreachToContacts(
	ctx context.Context,
	contacts []domain.Contact,
	campaign *domain.Campaign,
) (string, error) {
	// Validate campaign status
	requiredStatuses := []domain.CampaignStatus{
		domain.CampaignStatusActive,
		domain.CampaignStatusScheduled,
	}

	isValidStatus := false
	for _, status := range requiredStatuses {
		if campaign.Status == status {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		return "", fmt.Errorf("%w: must be one of %v", ErrInvalidCampaignStatus, requiredStatuses)
	}

	// Validate inputs
	if len(contacts) == 0 {
		return "", ErrEmptyContactList
	}

	// Get campaign templates
	templates, err := s.templateRepo.FindByCampaignID(ctx, campaign.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get templates: %w", err)
	}

	if len(templates) == 0 {
		return "", ErrNoTemplates
	}

	// Get campaign agent
	if campaign.AgentID == nil {
		return "", ErrNoAgent
	}

	agent, err := s.agentRepo.FindByID(ctx, *campaign.AgentID)
	if err != nil {
		return "", fmt.Errorf("failed to get agent: %w", err)
	}

	// Determine schedule time
	now := time.Now().UTC()
	scheduleAt := now

	if campaign.Schedule != nil && campaign.Schedule.After(now) {
		scheduleAt = *campaign.Schedule
	}

	sendTimes := templatedomain.SequenceSendTimes(scheduleAt, templates)

	// Create tasks for each contact-template combination
	var taskIDs []uuid.UUID

	for _, contact := range contacts {
		for i, template := range templates {
			// Same rule as the campaign worker: working days after the
			// previous step.
			sendTime := sendTimes[i]

			taskAttrs := domain.TaskAttributes{
				ContactID:  contact.ID,
				CampaignID: campaign.ID,
				TemplateID: template.ID,
				Status:     domain.TaskStatusPending,
				ScheduleAt: &sendTime,
			}

			createdIDs, err := s.taskRepo.CreateTasks(ctx, []domain.TaskAttributes{taskAttrs})
			if err != nil {
				logger.Logger.Error().Err(err).Msg("Failed to create task")
				continue
			}

			if len(createdIDs) > 0 {
				taskIDs = append(taskIDs, createdIDs...)
			}
		}
	}

	// Generate unique job ID
	jobID := fmt.Sprintf("agent_send_email_%s_%s", campaign.ID.String(), agent.ID.String())

	// Schedule the email sending task with cron
	// Schedule at specific minute and hour
	minute := fmt.Sprintf("%d", (scheduleAt.Minute()+1)%60)
	hour := fmt.Sprintf("%d", scheduleAt.Hour())

	config := schedulerService.ScheduleConfig{
		TaskType: "agent:send_email",
		Payload: map[string]interface{}{
			"task_ids":    taskIDs,
			"agent_id":    agent.ID.String(),
			"campaign_id": campaign.ID.String(),
		},
		JobID:           jobID,
		ReplaceExisting: true,
		Queue:           "default",
	}

	_, err = s.schedulerService.ScheduleCronDetailed(config, minute, hour, "*", "*", "*")
	if err != nil {
		return "", fmt.Errorf("failed to schedule job: %w", err)
	}

	logger.Logger.Info().
		Str("job_id", jobID).
		Str("campaign_id", campaign.ID.String()).
		Int("tasks_created", len(taskIDs)).
		Msg("Scheduled outreach to contacts")

	return jobID, nil
}

// TriggerEmailSequenceNode immediately triggers a background task to send outreach emails
func (s *CampaignService) TriggerEmailSequenceNode(
	ctx context.Context,
	contacts []domain.Contact,
	campaign *domain.Campaign,
) error {
	// Validate campaign status
	requiredStatuses := []domain.CampaignStatus{
		domain.CampaignStatusActive,
		domain.CampaignStatusScheduled,
	}

	isValidStatus := false
	for _, status := range requiredStatuses {
		if campaign.Status == status {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		return fmt.Errorf("%w: must be one of %v", ErrInvalidCampaignStatus, requiredStatuses)
	}

	// Validate inputs
	if len(contacts) == 0 {
		return ErrEmptyContactList
	}

	// Get campaign templates
	templates, err := s.templateRepo.FindByCampaignID(ctx, campaign.ID)
	if err != nil {
		return fmt.Errorf("failed to get templates: %w", err)
	}

	if len(templates) == 0 {
		return ErrNoTemplates
	}

	// Get campaign agent
	if campaign.AgentID == nil {
		return ErrNoAgent
	}

	agent, err := s.agentRepo.FindByID(ctx, *campaign.AgentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Determine schedule time
	now := time.Now().UTC()
	scheduleAt := now

	if campaign.Schedule != nil && campaign.Schedule.After(now) {
		scheduleAt = *campaign.Schedule
	}

	sendTimes := templatedomain.SequenceSendTimes(scheduleAt, templates)

	// Create tasks for each contact-template combination
	var taskIDs []uuid.UUID

	for _, contact := range contacts {
		for i, template := range templates {
			// Same rule as the campaign worker: working days after the
			// previous step.
			sendTime := sendTimes[i]

			taskAttrs := domain.TaskAttributes{
				ContactID:  contact.ID,
				CampaignID: campaign.ID,
				TemplateID: template.ID,
				Status:     domain.TaskStatusPending,
				ScheduleAt: &sendTime,
			}

			createdIDs, err := s.taskRepo.CreateTasks(ctx, []domain.TaskAttributes{taskAttrs})
			if err != nil {
				logger.Logger.Error().Err(err).Msg("Failed to create task")
				continue
			}

			if len(createdIDs) > 0 {
				taskIDs = append(taskIDs, createdIDs...)
			}
		}
	}

	// Enqueue agent send email task immediately
	payload := map[string]interface{}{
		"task_ids":    taskIDs,
		"campaign_id": campaign.ID.String(),
		"agent_id":    agent.ID.String(),
	}

	_, err = s.workerClient.EnqueueTask(ctx, "agent:send_email", payload)
	if err != nil {
		return fmt.Errorf("failed to enqueue send email task: %w", err)
	}

	logger.Logger.Info().
		Str("campaign_id", campaign.ID.String()).
		Int("tasks_created", len(taskIDs)).
		Int("contacts", len(contacts)).
		Msg("Triggered email sequence node")

	return nil
}

// CancelScheduledCampaign cancels a scheduled campaign job
func (s *CampaignService) CancelScheduledCampaign(ctx context.Context, campaignID, agentID uuid.UUID) error {
	jobID := fmt.Sprintf("agent_send_email_%s_%s", campaignID.String(), agentID.String())

	if err := s.schedulerService.RemoveJob(jobID); err != nil {
		return fmt.Errorf("failed to cancel campaign: %w", err)
	}

	logger.Logger.Info().
		Str("campaign_id", campaignID.String()).
		Str("job_id", jobID).
		Msg("Cancelled scheduled campaign")

	return nil
}
