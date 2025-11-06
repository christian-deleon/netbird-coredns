package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"netbird-coredns/internal/logger"
)

// Server represents the HTTP API server
type Server struct {
	storage    *Storage
	httpServer *http.Server
	port       int
}

// NewServer creates a new API server
func NewServer(storage *Storage, port int) *Server {
	return &Server{
		storage: storage,
		port:    port,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/health", s.HealthHandler)
	mux.HandleFunc("/api/v1/records", s.RecordHandler)
	mux.HandleFunc("/api/v1/records/", s.RecordHandler)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Starting API server on port %d", s.port)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("API server failed: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	logger.Info("Stopping API server...")
	return s.httpServer.Shutdown(ctx)
}
