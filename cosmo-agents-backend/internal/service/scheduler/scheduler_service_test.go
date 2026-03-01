package scheduler

import (
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
)

func TestSchedulerService_RemoveJob_NotFoundAndGetJobs(t *testing.T) {
	svc := &SchedulerService{
		scheduledIDs: map[string]string{},
	}

	err := svc.RemoveJob("missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Add a job id entry without actual scheduler; RemoveJob should still return error when not found.
	svc.scheduledIDs["job"] = "entry"
	jobs := svc.GetScheduledJobs()
	assert.Equal(t, []string{"job"}, jobs)
}

func TestScheduleCronAndRemoveJob(t *testing.T) {
	redisOpt := asynq.RedisClientOpt{Addr: "localhost:0"}
	svc := &SchedulerService{
		scheduler:    asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{LogLevel: asynq.ErrorLevel}),
		scheduledIDs: map[string]string{},
	}

	entryID, err := svc.ScheduleCronDetailed(ScheduleConfig{
		TaskType: "email:send",
		Payload:  map[string]string{"id": "123"},
		JobID:    "job1",
		Queue:    "mail",
	}, "0", "12", "*", "*", "*")
	assert.NoError(t, err)
	assert.NotEmpty(t, entryID)
	assert.Equal(t, entryID, svc.scheduledIDs["job1"])

	err = svc.RemoveJob("job1")
	assert.NoError(t, err)
	assert.Empty(t, svc.scheduledIDs)
}

func TestScheduleIntervalStoresMapping(t *testing.T) {
	redisOpt := asynq.RedisClientOpt{Addr: "localhost:0"}
	svc := &SchedulerService{
		scheduler:    asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{LogLevel: asynq.ErrorLevel}),
		scheduledIDs: map[string]string{},
	}

	entryID, err := svc.ScheduleInterval(ScheduleConfig{
		TaskType: "cleanup",
		Payload:  map[string]string{"cleanup": "true"},
		JobID:    "interval-job",
		Queue:    "maintenance",
	}, time.Minute, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, entryID)
	assert.Equal(t, entryID, svc.scheduledIDs["interval-job"])
}

func TestScheduleCronInvalidSpec(t *testing.T) {
	redisOpt := asynq.RedisClientOpt{Addr: "localhost:0"}
	svc := &SchedulerService{
		scheduler:    asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{LogLevel: asynq.ErrorLevel}),
		scheduledIDs: map[string]string{},
	}

	_, err := svc.ScheduleCron(ScheduleConfig{
		TaskType: "job",
		Payload:  map[string]string{"k": "v"},
		JobID:    "bad",
	}, "invalid cron")
	assert.Error(t, err)
}
