package worker

import "github.com/hibiken/asynq"

// Config holds worker configuration.
type Config struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	Concurrency   int // Number of concurrent workers
}

// ClientConfig creates Asynq client config from worker config.
func (c Config) ClientConfig() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     c.RedisAddr,
		Password: c.RedisPassword,
		DB:       c.RedisDB,
	}
}

// ServerConfig creates Asynq server config from worker config.
func (c Config) ServerConfig() *asynq.Config {
	return &asynq.Config{
		Concurrency: c.Concurrency,
		Queues: map[string]int{
			"critical": 6, // 60% of workers
			"default":  3, // 30% of workers
			"low":      1, // 10% of workers
		},
		StrictPriority: false, // Allow low priority tasks to run even if high priority queue is full
	}
}
