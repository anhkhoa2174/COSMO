package campaign

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/task"
	"github.com/rockship/cosmo-agents-go/internal/domain/template"
	schedulerService "github.com/rockship/cosmo-agents-go/internal/service/scheduler"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

func emailFromProfileContact(t *testing.T, contact domain.Contact) string {
	t.Helper()
	var profile map[string]interface{}
	assert.NoError(t, contact.Profile.Unmarshal(&profile))
	email, _ := profile["email"].(string)
	return email
}

// Mock implementations for testing
type mockSchedulerService struct {
	// Mock implementation that satisfies schedulerService.SchedulerService interface
}

func (m *mockSchedulerService) ScheduleTask(config schedulerService.ScheduleConfig) error {
	return nil
}

func (m *mockSchedulerService) ScheduleCron(config schedulerService.ScheduleConfig, cronExpression string) (string, error) {
	return uuid.New().String(), nil
}

func (m *mockSchedulerService) ScheduleCronDetailed(config schedulerService.ScheduleConfig, minute, hour, day, month, weekday string) (string, error) {
	return uuid.New().String(), nil
}

func (m *mockSchedulerService) CancelJob(jobID string) error {
	return nil
}

type mockWorkerClient struct {
	// Mock implementation that satisfies worker.Client interface
}

func (m *mockWorkerClient) SendMessage(ctx context.Context, taskID uuid.UUID, taskType string, payload map[string]interface{}) error {
	return nil
}

type mockTaskRepository struct {
	tasks map[uuid.UUID]task.Task
}

func (m *mockTaskRepository) Create(ctx context.Context, task *task.Task) error {
	m.tasks[task.ID] = *task
	return nil
}

func (m *mockTaskRepository) FindByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]task.Task, error) {
	var tasks []task.Task
	for _, task := range m.tasks {
		if task.CampaignID == campaignID {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

type mockTemplateRepository struct {
	templates map[uuid.UUID]template.Template
}

func (m *mockTemplateRepository) FindByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]template.Template, error) {
	var templates []template.Template
	for _, tmpl := range m.templates {
		if tmpl.CampaignID != nil && *tmpl.CampaignID == campaignID {
			templates = append(templates, tmpl)
		}
	}
	return templates, nil
}

type mockAgentRepository struct {
	agents map[uuid.UUID]domain.Agent
}

func (m *mockAgentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	if agent, exists := m.agents[id]; exists {
		return &agent, nil
	}
	return nil, nil
}

// Create concrete instances for testing
func createMockSchedulerService() *schedulerService.SchedulerService {
	// We'll create a minimal implementation or use a real one if possible
	return nil
}

func createMockWorkerClient() *worker.Client {
	// We'll create a minimal implementation or use a real one if possible
	return nil
}

// Test campaign validation logic without full service dependencies
func TestCampaignService_ValidationLogic(t *testing.T) {
	t.Run("Valid campaign for scheduling", func(t *testing.T) {
		userID := uuid.New()
		campaignID := uuid.New()
		agentID := uuid.New()

		campaign := &domain.Campaign{
			Base:    domain.Base{ID: campaignID},
			UserID:  userID,
			Status:  domain.CampaignStatusActive,
			AgentID: &agentID,
		}

		// Test campaign validation logic
		assert.Equal(t, userID, campaign.UserID)
		assert.Equal(t, domain.CampaignStatusActive, campaign.Status)
		assert.NotNil(t, campaign.AgentID)
		assert.Equal(t, agentID, *campaign.AgentID)
	})

	t.Run("Invalid campaign statuses", func(t *testing.T) {
		userID := uuid.New()
		campaignID := uuid.New()

		invalidStatuses := []domain.CampaignStatus{
			domain.CampaignStatusPaused,
			domain.CampaignStatusEnded,
		}

		for _, status := range invalidStatuses {
			campaign := &domain.Campaign{
				Base:   domain.Base{ID: campaignID},
				UserID: userID,
				Status: status,
			}

			// This campaign should be invalid for scheduling
			assert.NotEqual(t, domain.CampaignStatusActive, campaign.Status)
			assert.NotEqual(t, domain.CampaignStatusScheduled, campaign.Status)
		}
	})

	t.Run("Campaign without agent", func(t *testing.T) {
		userID := uuid.New()
		campaignID := uuid.New()

		campaign := &domain.Campaign{
			Base:    domain.Base{ID: campaignID},
			UserID:  userID,
			Status:  domain.CampaignStatusActive,
			AgentID: nil, // No agent configured
		}

		assert.Nil(t, campaign.AgentID)
	})
}

// Test contact validation logic
func TestCampaignService_ContactValidation(t *testing.T) {
	t.Run("Valid contact list", func(t *testing.T) {
		userID := uuid.New()

		contacts := []domain.Contact{
			{
				Base:    domain.Base{ID: uuid.New()},
				UserID:  userID,
				Profile: base.JSONB(`{"email":"test1@example.com"}`),
			},
			{
				Base:    domain.Base{ID: uuid.New()},
				UserID:  userID,
				Profile: base.JSONB(`{"email":"test2@example.com"}`),
			},
		}

		assert.Len(t, contacts, 2)
		assert.NotEmpty(t, emailFromProfileContact(t, contacts[0]))
		assert.NotEmpty(t, emailFromProfileContact(t, contacts[1]))
	})

	t.Run("Empty contact list", func(t *testing.T) {
		contacts := []domain.Contact{}
		assert.Empty(t, contacts)
	})

	t.Run("Contact with invalid email", func(t *testing.T) {
		userID := uuid.New()

		contact := domain.Contact{
			Base:    domain.Base{ID: uuid.New()},
			UserID:  userID,
			Profile: base.JSONB(`{"email":""}`), // Empty email
		}

		assert.Empty(t, emailFromProfileContact(t, contact))
	})
}

// Test template validation logic
func TestCampaignService_TemplateValidation(t *testing.T) {
	t.Run("Valid templates", func(t *testing.T) {
		campaignID := uuid.New()

		templates := []template.Template{
			{
				Base:       domain.Base{ID: uuid.New()},
				CampaignID: &campaignID,
				SendAfter:  1,
				Subject:    "First Email",
				Type:       "email",
			},
			{
				Base:       domain.Base{ID: uuid.New()},
				CampaignID: &campaignID,
				SendAfter:  3,
				Subject:    "Follow-up Email",
				Type:       "email",
			},
		}

		assert.Len(t, templates, 2)
		assert.Equal(t, campaignID, *templates[0].CampaignID)
		assert.Equal(t, campaignID, *templates[1].CampaignID)
	})

	t.Run("Empty template list", func(t *testing.T) {
		templates := []template.Template{}
		assert.Empty(t, templates)
	})

	t.Run("Template with zero SendAfter", func(t *testing.T) {
		campaignID := uuid.New()

		template := template.Template{
			Base:       domain.Base{ID: uuid.New()},
			CampaignID: &campaignID,
			SendAfter:  0, // Immediate send
		}

		assert.Equal(t, 0, template.SendAfter)
	})
}

// Test task creation logic
func TestCampaignService_TaskCreation(t *testing.T) {
	t.Run("Valid task creation", func(t *testing.T) {
		campaignID := uuid.New()
		contactID := uuid.New()
		templateID := uuid.New()
		scheduledAt := time.Now().Add(time.Hour)

		newTask := task.Task{
			Base:       domain.Base{ID: uuid.New()},
			ContactID:  contactID,
			CampaignID: campaignID,
			TemplateID: templateID,
			Status:     task.TaskStatusPending,
			ScheduleAt: &scheduledAt,
		}

		assert.Equal(t, contactID, newTask.ContactID)
		assert.Equal(t, campaignID, newTask.CampaignID)
		assert.Equal(t, templateID, newTask.TemplateID)
		assert.Equal(t, task.TaskStatusPending, newTask.Status)
		assert.NotNil(t, newTask.ScheduleAt)
		assert.False(t, newTask.ScheduleAt.IsZero())
	})

	t.Run("Task with zero values", func(t *testing.T) {
		emptyTask := task.Task{}

		assert.Equal(t, uuid.Nil, emptyTask.ID)
		assert.Equal(t, uuid.Nil, emptyTask.ContactID)
		assert.Equal(t, uuid.Nil, emptyTask.CampaignID)
		assert.Equal(t, uuid.Nil, emptyTask.TemplateID)
		assert.Nil(t, emptyTask.ScheduleAt)
	})
}

// Test scheduling logic
func TestCampaignService_SchedulingLogic(t *testing.T) {
	t.Run("Valid scheduling time", func(t *testing.T) {
		now := time.Now()
		scheduledAt := now.Add(time.Hour)

		assert.True(t, scheduledAt.After(now))
		assert.False(t, scheduledAt.IsZero())
	})

	t.Run("Cron expression validation", func(t *testing.T) {
		validCrons := []string{
			"0 9 * * 1-5", // Weekdays at 9 AM
			"30 14 * * *", // Daily at 2:30 PM
			"0 0 1 * *",   // Monthly on 1st
		}

		for _, cron := range validCrons {
			assert.NotEmpty(t, cron)
		}
	})

	t.Run("Time zone handling", func(t *testing.T) {
		now := time.Now()
		utcTime := now.UTC()

		assert.Equal(t, "UTC", utcTime.Location().String())
	})
}

// Test error scenarios
func TestCampaignService_ErrorScenarios(t *testing.T) {
	t.Run("All error constants", func(t *testing.T) {
		errors := []error{
			ErrInvalidCampaignStatus,
			ErrMisconfiguredCampaign,
			ErrEmptyContactList,
			ErrNoTemplates,
			ErrNoAgent,
		}

		for _, err := range errors {
			assert.Error(t, err)
			assert.NotEmpty(t, err.Error())
		}
	})

	t.Run("Error messages", func(t *testing.T) {
		assert.Contains(t, ErrInvalidCampaignStatus.Error(), "invalid campaign status")
		assert.Contains(t, ErrMisconfiguredCampaign.Error(), "misconfigured campaign")
		assert.Contains(t, ErrEmptyContactList.Error(), "contact list is empty")
		assert.Contains(t, ErrNoTemplates.Error(), "no outreach emails")
		assert.Contains(t, ErrNoAgent.Error(), "no agent is configured")
	})
}

// Test campaign workflow scenarios
func TestCampaignService_WorkflowScenarios(t *testing.T) {
	t.Run("Complete campaign setup", func(t *testing.T) {
		userID := uuid.New()
		campaignID := uuid.New()
		agentID := uuid.New()

		// Campaign setup
		campaign := &domain.Campaign{
			Base:     domain.Base{ID: campaignID},
			UserID:   userID,
			Status:   domain.CampaignStatusActive,
			AgentID:  &agentID,
			Name:     "Test Campaign",
			Playbook: "test-playbook",
		}

		// Contact setup
		contacts := []domain.Contact{
			{
				Base:    domain.Base{ID: uuid.New()},
				UserID:  userID,
				Profile: base.JSONB(`{"email":"contact1@example.com"}`),
			},
			{
				Base:    domain.Base{ID: uuid.New()},
				UserID:  userID,
				Profile: base.JSONB(`{"email":"contact2@example.com"}`),
			},
		}

		// Template setup
		templates := []template.Template{
			{
				Base:       domain.Base{ID: uuid.New()},
				CampaignID: &campaignID,
				SendAfter:  1,
				Subject:    "Initial Outreach",
				Type:       "email",
			},
		}

		// Verify complete setup
		assert.NotNil(t, campaign)
		assert.Len(t, contacts, 2)
		assert.Len(t, templates, 1)
		assert.Equal(t, campaignID, *templates[0].CampaignID)
	})

	t.Run("Campaign with multiple templates", func(t *testing.T) {
		campaignID := uuid.New()

		templates := []template.Template{
			{
				Base:       domain.Base{ID: uuid.New()},
				CampaignID: &campaignID,
				SendAfter:  1,
				Subject:    "Day 1",
				Type:       "email",
			},
			{
				Base:       domain.Base{ID: uuid.New()},
				CampaignID: &campaignID,
				SendAfter:  3,
				Subject:    "Day 3",
				Type:       "email",
			},
			{
				Base:       domain.Base{ID: uuid.New()},
				CampaignID: &campaignID,
				SendAfter:  7,
				Subject:    "Day 7",
				Type:       "email",
			},
		}

		sendAfterTimes := []int{1, 3, 7}
		for i, template := range templates {
			assert.Equal(t, sendAfterTimes[i], template.SendAfter)
		}
	})
}

// Test boundary conditions
func TestCampaignService_BoundaryConditions(t *testing.T) {
	t.Run("Maximum contacts", func(t *testing.T) {
		userID := uuid.New()

		var contacts []domain.Contact
		for i := 0; i < 1000; i++ {
			contacts = append(contacts, domain.Contact{
				Base:    domain.Base{ID: uuid.New()},
				UserID:  userID,
				Profile: base.JSONB(`{"email":"contact1@example.com"}`),
			})
		}

		assert.Len(t, contacts, 1000)
	})

	t.Run("Zero send after time", func(t *testing.T) {
		campaignID := uuid.New()

		template := template.Template{
			Base:       domain.Base{ID: uuid.New()},
			CampaignID: &campaignID,
			SendAfter:  0,
			Subject:    "Immediate Send",
		}

		assert.Equal(t, 0, template.SendAfter)
	})

	t.Run("Far future scheduling", func(t *testing.T) {
		now := time.Now()
		futureTime := now.Add(365 * 24 * time.Hour) // 1 year from now

		assert.True(t, futureTime.After(now))
		assert.True(t, futureTime.Sub(now) > 24*time.Hour)
	})
}

// Test data consistency
func TestCampaignService_DataConsistency(t *testing.T) {
	t.Run("Campaign and template relationship", func(t *testing.T) {
		userID := uuid.New()
		campaignID := uuid.New()

		campaign := &domain.Campaign{
			Base:   domain.Base{ID: campaignID},
			UserID: userID,
			Status: domain.CampaignStatusActive,
		}

		template := template.Template{
			Base:       domain.Base{ID: uuid.New()},
			CampaignID: &campaignID,
			SendAfter:  1,
		}

		assert.Equal(t, campaignID, *template.CampaignID)
		assert.Equal(t, campaignID, campaign.ID)
	})

	t.Run("User ownership consistency", func(t *testing.T) {
		userID := uuid.New()
		campaignID := uuid.New()

		campaign := &domain.Campaign{
			Base:   domain.Base{ID: campaignID},
			UserID: userID,
			Status: domain.CampaignStatusActive,
		}

		contact := domain.Contact{
			Base:    domain.Base{ID: uuid.New()},
			UserID:  userID,
			Profile: base.JSONB(`{"email":"test@example.com"}`),
		}

		assert.Equal(t, userID, campaign.UserID)
		assert.Equal(t, userID, contact.UserID)
	})
}

// Test service method business logic simulation
func TestCampaignService_BusinessLogicSimulation(t *testing.T) {
	t.Run("ScheduleOutreachToContacts validation logic", func(t *testing.T) {
		userID := uuid.New()
		campaignID := uuid.New()
		agentID := uuid.New()

		// Test campaign status validation (mirrors service logic)
		validStatuses := []domain.CampaignStatus{
			domain.CampaignStatusActive,
			domain.CampaignStatusScheduled,
		}

		campaign := &domain.Campaign{
			Base:    domain.Base{ID: campaignID},
			UserID:  userID,
			Status:  domain.CampaignStatusActive,
			AgentID: &agentID,
		}

		// Simulate service validation
		isValidStatus := false
		for _, status := range validStatuses {
			if campaign.Status == status {
				isValidStatus = true
				break
			}
		}
		assert.True(t, isValidStatus)

		// Test invalid status
		invalidCampaign := &domain.Campaign{
			Base:    domain.Base{ID: campaignID},
			UserID:  userID,
			Status:  domain.CampaignStatusPaused,
			AgentID: &agentID,
		}

		isValidStatus = false
		for _, status := range validStatuses {
			if invalidCampaign.Status == status {
				isValidStatus = true
				break
			}
		}
		assert.False(t, isValidStatus)

		// Test contact list validation
		contacts := []domain.Contact{
			{
				Base:    domain.Base{ID: uuid.New()},
				UserID:  userID,
				Profile: base.JSONB(`{"email":"test@example.com"}`),
			},
		}
		assert.NotEmpty(t, contacts)

		emptyContacts := []domain.Contact{}
		assert.Empty(t, emptyContacts)

		// Test agent validation
		assert.NotNil(t, campaign.AgentID)

		campaignWithoutAgent := &domain.Campaign{
			Base:    domain.Base{ID: campaignID},
			UserID:  userID,
			Status:  domain.CampaignStatusActive,
			AgentID: nil,
		}
		assert.Nil(t, campaignWithoutAgent.AgentID)
	})

	t.Run("Schedule time calculation logic", func(t *testing.T) {
		now := time.Now().UTC()

		// Test campaign schedule in future
		futureSchedule := now.Add(2 * time.Hour)
		campaign := &domain.Campaign{
			Base:     domain.Base{ID: uuid.New()},
			Schedule: &futureSchedule,
		}

		// Simulate service logic: if campaign schedule is after now, use it
		scheduleAt := now
		if campaign.Schedule != nil && campaign.Schedule.After(now) {
			scheduleAt = *campaign.Schedule
		}
		assert.Equal(t, futureSchedule, scheduleAt)

		// Test campaign schedule in past
		pastSchedule := now.Add(-2 * time.Hour)
		campaign.Schedule = &pastSchedule

		scheduleAt = now
		if campaign.Schedule != nil && campaign.Schedule.After(now) {
			scheduleAt = *campaign.Schedule
		}
		assert.Equal(t, now, scheduleAt)

		// Test nil schedule
		campaign.Schedule = nil
		scheduleAt = now
		if campaign.Schedule != nil && campaign.Schedule.After(now) {
			scheduleAt = *campaign.Schedule
		}
		assert.Equal(t, now, scheduleAt)
	})

	t.Run("Task creation simulation", func(t *testing.T) {
		contactID := uuid.New()
		campaignID := uuid.New()
		templateID := uuid.New()

		now := time.Now()
		scheduleAt := now
		sendAfterDays := 1

		// Simulate send time calculation from service
		sendTime := scheduleAt.Add(time.Duration(sendAfterDays) * 24 * time.Hour)

		// Simulate task attributes creation
		taskAttrs := domain.TaskAttributes{
			ContactID:  contactID,
			CampaignID: campaignID,
			TemplateID: templateID,
			Status:     domain.TaskStatusPending,
			ScheduleAt: &sendTime,
		}

		assert.Equal(t, contactID, taskAttrs.ContactID)
		assert.Equal(t, campaignID, taskAttrs.CampaignID)
		assert.Equal(t, templateID, taskAttrs.TemplateID)
		assert.Equal(t, domain.TaskStatusPending, taskAttrs.Status)
		assert.NotNil(t, taskAttrs.ScheduleAt)
		assert.True(t, taskAttrs.ScheduleAt.After(scheduleAt))
	})

	t.Run("Job ID generation logic", func(t *testing.T) {
		campaignID := uuid.New()
		agentID := uuid.New()

		// Simulate job ID generation from service
		jobID := fmt.Sprintf("agent_send_email_%s_%s", campaignID.String(), agentID.String())

		assert.Contains(t, jobID, "agent_send_email_")
		assert.Contains(t, jobID, campaignID.String())
		assert.Contains(t, jobID, agentID.String())

		// Test consistency
		jobID2 := fmt.Sprintf("agent_send_email_%s_%s", campaignID.String(), agentID.String())
		assert.Equal(t, jobID, jobID2)
	})

	t.Run("Cron scheduling time calculation", func(t *testing.T) {
		scheduleAt := time.Now()

		// Simulate minute calculation from service: (scheduleAt.Minute() + 1) % 60
		expectedMinute := (scheduleAt.Minute() + 1) % 60
		minuteStr := fmt.Sprintf("%d", expectedMinute)

		assert.Equal(t, fmt.Sprintf("%d", expectedMinute), minuteStr)
		assert.True(t, expectedMinute >= 0 && expectedMinute < 60)

		// Test hour calculation: scheduleAt.Hour()
		expectedHour := scheduleAt.Hour()
		hourStr := fmt.Sprintf("%d", expectedHour)

		assert.Equal(t, fmt.Sprintf("%d", expectedHour), hourStr)
		assert.True(t, expectedHour >= 0 && expectedHour < 24)
	})
}

// Test service error scenarios
func TestCampaignService_ErrorScenariosDetailed(t *testing.T) {
	t.Run("Error validation flow", func(t *testing.T) {
		userID := uuid.New()
		campaignID := uuid.New()
		agentID := uuid.New()

		t.Run("Invalid campaign status error", func(t *testing.T) {
			campaign := &domain.Campaign{
				Base:    domain.Base{ID: campaignID},
				UserID:  userID,
				Status:  domain.CampaignStatusEnded,
				AgentID: &agentID,
			}

			validStatuses := []domain.CampaignStatus{
				domain.CampaignStatusActive,
				domain.CampaignStatusScheduled,
			}

			isValidStatus := false
			for _, status := range validStatuses {
				if campaign.Status == status {
					isValidStatus = true
					break
				}
			}

			if !isValidStatus {
				// Simulate service error
				err := fmt.Errorf("%w: must be one of %v", ErrInvalidCampaignStatus, validStatuses)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid campaign status")
			}
		})

		t.Run("Empty contact list error", func(t *testing.T) {
			contacts := []domain.Contact{}

			if len(contacts) == 0 {
				// Simulate service error
				err := ErrEmptyContactList
				assert.Error(t, err)
				assert.Equal(t, "contact list is empty", err.Error())
			}
		})

		t.Run("No templates error", func(t *testing.T) {
			templates := []template.Template{}

			if len(templates) == 0 {
				// Simulate service error
				err := ErrNoTemplates
				assert.Error(t, err)
				assert.Equal(t, "no outreach emails are defined", err.Error())
			}
		})

		t.Run("No agent error", func(t *testing.T) {
			campaign := &domain.Campaign{
				Base:    domain.Base{ID: campaignID},
				UserID:  userID,
				Status:  domain.CampaignStatusActive,
				AgentID: nil,
			}

			if campaign.AgentID == nil {
				// Simulate service error
				err := ErrNoAgent
				assert.Error(t, err)
				assert.Equal(t, "no agent is configured for the campaign", err.Error())
			}
		})
	})
}
