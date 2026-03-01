package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// ScheduleConfig represents configuration for scheduling a job
type ScheduleConfig struct {
	TaskType         string        // Asynq task type
	Payload          interface{}   // Task payload
	JobID            string        // Unique job identifier
	ReplaceExisting  bool          // Whether to replace existing job with same ID
	MisfireGraceTime time.Duration // Grace time for missed jobs
	MaxRetry         int           // Max retry attempts
	Queue            string        // Queue name (default: "default")
}

// SchedulerService manages scheduled jobs using Asynq
type SchedulerService struct {
	client       *asynq.Client
	scheduler    *asynq.Scheduler
	mu           sync.RWMutex
	scheduledIDs map[string]string // jobID -> asynq entry ID mapping
}

var (
	schedulerInstance *SchedulerService
	schedulerOnce     sync.Once
)

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(redisAddr string, redisPassword string) *SchedulerService {
	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
	}

	return &SchedulerService{
		client: asynq.NewClient(redisOpt),
		scheduler: asynq.NewScheduler(
			redisOpt,
			&asynq.SchedulerOpts{
				LogLevel: asynq.WarnLevel,
			},
		),
		scheduledIDs: make(map[string]string),
	}
}

// GetSchedulerInstance returns singleton instance
func GetSchedulerInstance(redisAddr string, redisPassword string) *SchedulerService {
	schedulerOnce.Do(func() {
		schedulerInstance = NewSchedulerService(redisAddr, redisPassword)
	})
	return schedulerInstance
}

// Start starts the scheduler
func (s *SchedulerService) Start() error {
	if err := s.scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}
	logger.Logger.Info().Msg("Scheduler started")
	return nil
}

// Shutdown gracefully shuts down the scheduler
func (s *SchedulerService) Shutdown(ctx context.Context) error {
	s.scheduler.Shutdown()
	if err := s.client.Close(); err != nil {
		return fmt.Errorf("failed to close client: %w", err)
	}
	logger.Logger.Info().Msg("Scheduler shutdown")
	return nil
}

// RemoveJob removes a scheduled job by job ID
func (s *SchedulerService) RemoveJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.scheduledIDs[jobID]
	if !exists {
		logger.Logger.Warn().Str("job_id", jobID).Msg("Job not found in scheduler")
		return fmt.Errorf("job %s not found", jobID)
	}

	if err := s.scheduler.Unregister(entryID); err != nil {
		logger.Logger.Error().Err(err).Str("job_id", jobID).Msg("Error removing job")
		return err
	}

	delete(s.scheduledIDs, jobID)
	logger.Logger.Info().Str("job_id", jobID).Msg("Removed job from scheduler successfully")
	return nil
}

// ScheduleAt schedules a one-time job at a specific time
func (s *SchedulerService) ScheduleAt(config ScheduleConfig, runDate time.Time) (string, error) {
	payload, err := json.Marshal(config.Payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(config.TaskType, payload, asynq.TaskID(config.JobID))

	queue := config.Queue
	if queue == "" {
		queue = "default"
	}

	opts := []asynq.Option{
		asynq.ProcessAt(runDate),
		asynq.Queue(queue),
	}

	if config.MaxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(config.MaxRetry))
	}

	// For one-time jobs, we use the client directly
	info, err := s.client.Enqueue(task, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to schedule job: %w", err)
	}

	s.mu.Lock()
	s.scheduledIDs[config.JobID] = info.ID
	s.mu.Unlock()

	logger.Logger.Info().
		Str("job_id", config.JobID).
		Time("run_date", runDate).
		Msg("Scheduled one-time job")

	return info.ID, nil
}

// ScheduleInterval schedules a recurring job with interval
func (s *SchedulerService) ScheduleInterval(
	config ScheduleConfig,
	interval time.Duration,
	startDate *time.Time,
) (string, error) {
	payload, err := json.Marshal(config.Payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(config.TaskType, payload)

	cronspec := fmt.Sprintf("@every %s", interval.String())

	entryID, err := s.scheduler.Register(
		cronspec,
		task,
		asynq.Queue(config.Queue),
	)
	if err != nil {
		return "", fmt.Errorf("failed to schedule interval job: %w", err)
	}

	s.mu.Lock()
	s.scheduledIDs[config.JobID] = entryID
	s.mu.Unlock()

	logger.Logger.Info().
		Str("job_id", config.JobID).
		Dur("interval", interval).
		Str("entry_id", entryID).
		Msg("Scheduled interval job")

	return entryID, nil
}

// ScheduleCron schedules a job using cron expression
func (s *SchedulerService) ScheduleCron(config ScheduleConfig, cronExpr string) (string, error) {
	payload, err := json.Marshal(config.Payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(config.TaskType, payload)

	queue := config.Queue
	if queue == "" {
		queue = "default"
	}

	entryID, err := s.scheduler.Register(
		cronExpr,
		task,
		asynq.Queue(queue),
	)
	if err != nil {
		return "", fmt.Errorf("failed to schedule cron job: %w", err)
	}

	s.mu.Lock()
	s.scheduledIDs[config.JobID] = entryID
	s.mu.Unlock()

	logger.Logger.Info().
		Str("job_id", config.JobID).
		Str("cron_expr", cronExpr).
		Str("entry_id", entryID).
		Msg("Scheduled cron job")

	return entryID, nil
}

// ScheduleCronDetailed schedules a cron job with detailed parameters
func (s *SchedulerService) ScheduleCronDetailed(
	config ScheduleConfig,
	minute, hour, day, month, dayOfWeek string,
) (string, error) {
	// Build cron expression from parts
	// Format: minute hour day month day_of_week
	cronExpr := fmt.Sprintf("%s %s %s %s %s", minute, hour, day, month, dayOfWeek)

	return s.ScheduleCron(config, cronExpr)
}

// GetScheduledJobs returns list of scheduled job IDs
func (s *SchedulerService) GetScheduledJobs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]string, 0, len(s.scheduledIDs))
	for jobID := range s.scheduledIDs {
		jobs = append(jobs, jobID)
	}
	return jobs
}
