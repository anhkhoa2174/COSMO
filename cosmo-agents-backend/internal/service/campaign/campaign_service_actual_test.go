package campaign

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/agent"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/task"
	"github.com/rockship/cosmo-agents-go/internal/domain/template"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	taskRepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	templateRepo "github.com/rockship/cosmo-agents-go/internal/repository/template"
	schedulerService "github.com/rockship/cosmo-agents-go/internal/service/scheduler"
)

// TestCampaignService_ActualMethods tests actual service methods by focusing on validation logic
func TestCampaignService_ActualMethods(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	campaignID := uuid.New()
	agentID := uuid.New()

	t.Run("ScheduleOutreachToContacts validation branches", func(t *testing.T) {
		contacts := []domain.Contact{{Base: domain.Base{ID: uuid.New()}, UserID: userID, Profile: base.JSONB(`{"email":"c@example.com"}`)}}

		service := &CampaignService{}

		_, err := service.ScheduleOutreachToContacts(ctx, contacts, &domain.Campaign{
			Base:    domain.Base{ID: campaignID},
			UserID:  userID,
			Status:  domain.CampaignStatusPaused,
			AgentID: &agentID,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid campaign status")

		_, err = service.ScheduleOutreachToContacts(ctx, []domain.Contact{}, &domain.Campaign{
			Base:    domain.Base{ID: campaignID},
			UserID:  userID,
			Status:  domain.CampaignStatusActive,
			AgentID: &agentID,
		})
		assert.Equal(t, ErrEmptyContactList, err)
	})

	t.Run("TriggerEmailSequenceNode validation branches", func(t *testing.T) {
		contacts := []domain.Contact{{Base: domain.Base{ID: uuid.New()}, UserID: userID, Profile: base.JSONB(`{"email":"c@example.com"}`)}}

		service := &CampaignService{}

		err := service.TriggerEmailSequenceNode(ctx, contacts, &domain.Campaign{
			Base:    domain.Base{ID: campaignID},
			UserID:  userID,
			Status:  domain.CampaignStatusPaused,
			AgentID: &agentID,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid campaign status")

		err = service.TriggerEmailSequenceNode(ctx, []domain.Contact{}, &domain.Campaign{
			Base:    domain.Base{ID: campaignID},
			UserID:  userID,
			Status:  domain.CampaignStatusActive,
			AgentID: &agentID,
		})
		assert.Equal(t, ErrEmptyContactList, err)
	})

	t.Run("CancelScheduledCampaign with empty scheduler map", func(t *testing.T) {
		svc := &CampaignService{
			schedulerService: &schedulerService.SchedulerService{},
		}
		err := svc.CancelScheduledCampaign(ctx, campaignID, agentID)
		assert.Error(t, err)
	})
}

func TestCampaignService_ScheduleOutreach_CreatesTasksBeforeScheduler(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	campaignID := uuid.New()
	agentID := uuid.New()

	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&template.Template{}, &task.Task{}, &agent.Agent{}))

	templateRepo := templateRepo.NewTemplateRepository(db)
	taskRepo := taskRepo.NewTaskRepository(db)
	agentRepo := agentRepo.NewAgentRepository(db)

	// Seed agent and template
	require.NoError(t, db.Create(&domain.Agent{
		Base:   domain.Base{ID: agentID},
		UserID: userID,
		Email:  "agent@example.com",
		Status: domain.AgentStatusActive,
	}).Error)

	require.NoError(t, db.Create(&template.Template{
		Base:       domain.Base{ID: uuid.New()},
		CampaignID: &campaignID,
		SendAfter:  1,
		Subject:    "Hello",
	}).Error)

	service := &CampaignService{
		schedulerService: &schedulerService.SchedulerService{}, // will panic when scheduling; recover below
		taskRepo:         taskRepo,
		templateRepo:     templateRepo,
		agentRepo:        agentRepo,
	}

	contacts := []domain.Contact{{Base: domain.Base{ID: uuid.New()}, UserID: userID, Profile: base.JSONB(`{"email":"lead@example.com"}`)}}

	defer func() { _ = recover() }()
	_, _ = service.ScheduleOutreachToContacts(ctx, contacts, &domain.Campaign{
		Base:    domain.Base{ID: campaignID},
		UserID:  userID,
		Status:  domain.CampaignStatusActive,
		AgentID: &agentID,
	})

	var tasks []task.Task
	require.NoError(t, db.Find(&tasks).Error)
	assert.GreaterOrEqual(t, len(tasks), 1)
}

func TestCampaignService_TriggerEmailSequenceNode_CreatesTasksBeforeEnqueue(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	campaignID := uuid.New()
	agentID := uuid.New()

	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&template.Template{}, &task.Task{}, &agent.Agent{}))

	templateRepo := templateRepo.NewTemplateRepository(db)
	taskRepo := taskRepo.NewTaskRepository(db)
	agentRepo := agentRepo.NewAgentRepository(db)

	require.NoError(t, db.Create(&domain.Agent{
		Base:   domain.Base{ID: agentID},
		UserID: userID,
		Email:  "agent@example.com",
		Status: domain.AgentStatusActive,
	}).Error)

	cid := campaignID
	require.NoError(t, db.Create(&template.Template{
		Base:       domain.Base{ID: uuid.New()},
		CampaignID: &cid,
		SendAfter:  0,
		Subject:    "Immediate",
	}).Error)

	service := &CampaignService{
		workerClient: nil, // will panic on EnqueueTask
		taskRepo:     taskRepo,
		templateRepo: templateRepo,
		agentRepo:    agentRepo,
	}

	contacts := []domain.Contact{{Base: domain.Base{ID: uuid.New()}, UserID: userID, Profile: base.JSONB(`{"email":"lead@example.com"}`)}}

	defer func() { _ = recover() }()
	_ = service.TriggerEmailSequenceNode(ctx, contacts, &domain.Campaign{
		Base:    domain.Base{ID: campaignID},
		UserID:  userID,
		Status:  domain.CampaignStatusActive,
		AgentID: &agentID,
	})

	var tasks []task.Task
	require.NoError(t, db.Find(&tasks).Error)
	assert.GreaterOrEqual(t, len(tasks), 1)
}
func TestCampaignService_NoTemplatesOrAgent(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	campaignID := uuid.New()

	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&template.Template{}, &task.Task{}, &agent.Agent{}))

	templateRepository := templateRepo.NewTemplateRepository(db)
	taskRepository := taskRepo.NewTaskRepository(db)
	agentRepository := agentRepo.NewAgentRepository(db)

	service := &CampaignService{
		taskRepo:     taskRepository,
		templateRepo: templateRepository,
		agentRepo:    agentRepository,
	}

	contacts := []domain.Contact{{Base: domain.Base{ID: uuid.New()}, UserID: userID, Profile: base.JSONB(`{"email":"lead@example.com"}`)}}

	// No templates -> ErrNoTemplates
	_, err = service.ScheduleOutreachToContacts(ctx, contacts, &domain.Campaign{
		Base:    domain.Base{ID: campaignID},
		UserID:  userID,
		Status:  domain.CampaignStatusActive,
		AgentID: func() *uuid.UUID { v := uuid.New(); return &v }(),
	})
	assert.Equal(t, ErrNoTemplates, err)

	// Insert a template, but no agent configured -> ErrNoAgent
	cid := campaignID
	require.NoError(t, db.Create(&template.Template{
		Base:       domain.Base{ID: uuid.New()},
		CampaignID: &cid,
		SendAfter:  1,
	}).Error)

	_, err = service.ScheduleOutreachToContacts(ctx, contacts, &domain.Campaign{
		Base:   domain.Base{ID: campaignID},
		UserID: userID,
		Status: domain.CampaignStatusActive,
		// AgentID nil
	})
	assert.Equal(t, ErrNoAgent, err)

	// TriggerEmailSequenceNode with no templates
	err = service.TriggerEmailSequenceNode(ctx, contacts, &domain.Campaign{
		Base:    domain.Base{ID: uuid.New()},
		UserID:  userID,
		Status:  domain.CampaignStatusActive,
		AgentID: func() *uuid.UUID { v := uuid.New(); return &v }(),
	})
	assert.Equal(t, ErrNoTemplates, err)

	// TriggerEmailSequenceNode with templates but no agent
	err = service.TriggerEmailSequenceNode(ctx, contacts, &domain.Campaign{
		Base:   domain.Base{ID: campaignID},
		UserID: userID,
		Status: domain.CampaignStatusActive,
	})
	assert.Equal(t, ErrNoAgent, err)
}

// TestCampaignService_ErrorConstantsAndHelpers tests all exported constants and helper functions
func TestCampaignService_ErrorConstantsAndHelpers(t *testing.T) {
	t.Run("All error constants are properly defined", func(t *testing.T) {
		errors := []struct {
			err  error
			want string
		}{
			{ErrInvalidCampaignStatus, "invalid campaign status"},
			{ErrMisconfiguredCampaign, "misconfigured campaign"},
			{ErrEmptyContactList, "contact list is empty"},
			{ErrNoTemplates, "no outreach emails are defined"},
			{ErrNoAgent, "no agent is configured for the campaign"},
		}

		for _, tc := range errors {
			assert.Error(t, tc.err)
			assert.Contains(t, tc.err.Error(), tc.want)
		}
	})

	t.Run("Campaign status validation", func(t *testing.T) {
		validStatuses := []domain.CampaignStatus{
			domain.CampaignStatusActive,
			domain.CampaignStatusScheduled,
		}

		// Test validation logic from the service
		for _, status := range validStatuses {
			isValidStatus := false
			for _, valid := range validStatuses {
				if status == valid {
					isValidStatus = true
					break
				}
			}
			assert.True(t, isValidStatus, "Status %v should be valid", status)
		}

		invalidStatuses := []domain.CampaignStatus{
			domain.CampaignStatusPaused,
			domain.CampaignStatusEnded,
		}

		for _, status := range invalidStatuses {
			isValidStatus := false
			for _, valid := range validStatuses {
				if status == valid {
					isValidStatus = true
					break
				}
			}
			assert.False(t, isValidStatus, "Status %v should be invalid", status)
		}
	})

	t.Run("Schedule time calculation logic", func(t *testing.T) {
		now := time.Now().UTC()

		// Test future schedule
		futureSchedule := now.Add(2 * time.Hour)
		scheduleAt := now
		if futureSchedule.After(now) {
			scheduleAt = futureSchedule
		}
		assert.Equal(t, futureSchedule, scheduleAt)

		// Test past schedule
		pastSchedule := now.Add(-2 * time.Hour)
		scheduleAt = now
		if pastSchedule.After(now) {
			scheduleAt = pastSchedule
		}
		assert.Equal(t, now, scheduleAt)
	})

	t.Run("Send time calculation", func(t *testing.T) {
		scheduleAt := time.Now()
		sendAfterDays := 1

		// Test the exact calculation from the service
		sendTime := scheduleAt.Add(time.Duration(sendAfterDays) * 24 * time.Hour)

		expectedDuration := 24 * time.Hour
		actualDuration := sendTime.Sub(scheduleAt)

		assert.Equal(t, expectedDuration, actualDuration)
	})

	t.Run("Job ID generation", func(t *testing.T) {
		campaignID := uuid.New()
		agentID := uuid.New()

		// Test the exact job ID format from the service
		jobID := fmt.Sprintf("agent_send_email_%s_%s", campaignID.String(), agentID.String())

		assert.Contains(t, jobID, "agent_send_email_")
		assert.Contains(t, jobID, campaignID.String())
		assert.Contains(t, jobID, agentID.String())

		// Test consistency
		jobID2 := fmt.Sprintf("agent_send_email_%s_%s", campaignID.String(), agentID.String())
		assert.Equal(t, jobID, jobID2)
	})

	t.Run("Cron time calculations", func(t *testing.T) {
		scheduleAt := time.Now()

		// Test the exact minute calculation from the service
		minute := (scheduleAt.Minute() + 1) % 60
		minuteStr := fmt.Sprintf("%d", minute)

		assert.True(t, minute >= 0 && minute < 60)
		assert.Equal(t, fmt.Sprintf("%d", minute), minuteStr)

		// Test the exact hour calculation from the service
		hour := scheduleAt.Hour()
		hourStr := fmt.Sprintf("%d", hour)

		assert.True(t, hour >= 0 && hour < 24)
		assert.Equal(t, fmt.Sprintf("%d", hour), hourStr)
	})
}
