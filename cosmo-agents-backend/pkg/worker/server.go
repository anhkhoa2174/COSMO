package worker

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// Server wraps Asynq server for task processing.
type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// NewServer creates a new worker server.
func NewServer(cfg Config) *Server {
	srv := asynq.NewServer(
		cfg.ClientConfig(),
		*cfg.ServerConfig(),
	)

	mux := asynq.NewServeMux()

	return &Server{
		server: srv,
		mux:    mux,
	}
}

// HandleFunc registers a handler function for a task type.
func (s *Server) HandleFunc(pattern string, handler func(context.Context, *asynq.Task) error) {
	s.mux.HandleFunc(pattern, handler)
}

// Handle registers a handler for a task type.
func (s *Server) Handle(pattern string, handler asynq.Handler) {
	s.mux.Handle(pattern, handler)
}

// Start starts the worker server.
func (s *Server) Start() error {
	logger.Info("Starting worker server")
	if err := s.server.Start(s.mux); err != nil {
		return fmt.Errorf("failed to start worker server: %w", err)
	}
	return nil
}

// Stop gracefully stops the worker server.
func (s *Server) Stop() {
	logger.Info("Stopping worker server")
	s.server.Stop()
	logger.Info("Worker server stopped")
}

// Shutdown gracefully shuts down the worker server.
func (s *Server) Shutdown() {
	logger.Info("Shutting down worker server")
	s.server.Shutdown()
	logger.Info("Worker server shut down")
}
