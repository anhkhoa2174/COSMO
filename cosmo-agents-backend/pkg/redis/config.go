package redisutil

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// RedisConfig holds parsed Redis connection configuration
type RedisConfig struct {
	Addr     string // host:port format
	Password string
	DB       int
}

// ParseRedisURL parses a Redis URL and returns connection configuration
// Supports formats like: redis://[:password@]host[:port][/db]
// Falls back to localhost:6379 db 0 if URL is empty or invalid
func ParseRedisURL(redisURL, defaultPassword string) RedisConfig {
	config := RedisConfig{
		Addr:     "localhost:6379", // Default Redis address
		Password: defaultPassword,  // Use provided default password
		DB:       0,                // Default database
	}

	if redisURL == "" {
		return config
	}

	u, err := url.Parse(redisURL)
	if err != nil {
		return config // Return defaults on parse error
	}

	// Extract host and port
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "6379" // Default Redis port
	}
	if host != "" {
		config.Addr = fmt.Sprintf("%s:%s", host, port)
	}

	// Extract database from path (e.g., /0, /1)
	if len(u.Path) > 1 {
		if db, err := strconv.Atoi(strings.TrimPrefix(u.Path, "/")); err == nil {
			config.DB = db
		}
	}

	// Extract password from URL if present
	if u.User != nil {
		if pass, ok := u.User.Password(); ok {
			config.Password = pass
		}
	}

	return config
}
