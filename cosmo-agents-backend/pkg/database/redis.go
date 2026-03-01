package database

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
	customLogger "github.com/rockship/cosmo-agents-go/pkg/logger"
)

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Password string
}

// NewRedisClient creates a new Redis client
// Equivalent to Python's Redis setup in worker.py and core/cache
func NewRedisClient(cfg RedisConfig) (*redis.Client, error) {
	// Parse Redis URL
	parsedURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	// Extract host and port
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "6379"
	}

	// Extract username and password
	username := ""
	password := cfg.Password
	if parsedURL.User != nil {
		username = parsedURL.User.Username()
		if pass, ok := parsedURL.User.Password(); ok {
			password = pass
		}
	}

	// Extract DB number
	db := 0
	if len(parsedURL.Path) > 1 {
		fmt.Sscanf(parsedURL.Path[1:], "%d", &db)
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Username:     username,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	customLogger.Info("Redis connected successfully")

	return client, nil
}

// CloseRedis closes the Redis connection
func CloseRedis(client *redis.Client) error {
	return client.Close()
}
