package server

import (
	"context"
	"log/slog"
	"music-recomendations/lastfm"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const defaultProxyURL = "http://lastfm-proxy:8080"

type Config struct {
	ProxyURL            string
	SimilarArtistsLimit int
	TopArtistsLimit     int
	Logger              *slog.Logger
}

type MusicClient interface {
	ArtistGetSimilar(ctx context.Context, artist, mbid string, limit int, autocorrect bool) ([]lastfm.SimilarArtist, error)
	ArtistGetInfo(ctx context.Context, artist, mbid, username string, autocorrect bool) (*lastfm.ArtistInfo, error)
	UserGetTopArtists(ctx context.Context, user, period string, limit, page int) ([]lastfm.TopArtist, error)
}

type Server struct {
	client     MusicClient
	config     Config
	logger     *slog.Logger
	httpServer *http.Server
}

func New(cfg Config) *Server {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}

	proxyURL := cfg.ProxyURL
	if proxyURL == "" {
		proxyURL = defaultProxyURL
	}

	return &Server{
		client: lastfm.NewClient(proxyURL),
		config: cfg,
		logger: logger,
	}
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
