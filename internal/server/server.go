package server

import (
	"context"
	"fmt"
	"log/slog"
	"music-recomendations/lastfm"
	"music-recomendations/lastfm/cache"
	"music-recomendations/lastfm/ratelimit"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	APIKey              string
	SimilarArtistsLimit int
	TopArtistsLimit     int
	CachePath           string
	Logger              *slog.Logger
}

type Server struct {
	client     *lastfm.Client
	cache      *cache.Cache
	config     Config
	logger     *slog.Logger
	httpServer *http.Server
}

func New(cfg Config) (*Server, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}

	cachePath := cfg.CachePath
	if cachePath == "" {
		cachePath = "./cache.db"
	}
	apiCache, err := cache.New(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	limiter := ratelimit.New(time.Second)
	client := lastfm.NewClientWithCacheAndLimiter(cfg.APIKey, apiCache, limiter)

	return &Server{
		client: client,
		cache:  apiCache,
		config: cfg,
		logger: logger,
	}, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	addr := "0.0.0.0:8080"
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to signal shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		s.logger.Info("starting server", "addr", addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-shutdownChan:
		s.logger.Info("received shutdown signal", "signal", sig.String())
	case err := <-serverErr:
		if err != nil {
			return err
		}
	}

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.logger.Info("shutting down server")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("server shutdown error", "error", err)
		return err
	}

	s.logger.Info("server stopped")
	return nil
}

func (s *Server) Close() error {
	if s.cache != nil {
		return s.cache.Close()
	}
	return nil
}
