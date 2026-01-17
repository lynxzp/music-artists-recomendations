package server

import (
	"fmt"
	"log"
	"music-recomendations/lastfm"
	"music-recomendations/lastfm/cache"
	"music-recomendations/lastfm/ratelimit"
	"net/http"
	"time"
)

type Config struct {
	APIKey              string
	SimilarArtistsLimit int
	TopArtistsLimit     int
	CachePath           string
}

type Server struct {
	client *lastfm.Client
	cache  *cache.Cache
	config Config
}

func New(cfg Config) (*Server, error) {
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
	}, nil
}

func (s *Server) Start() error {
	s.registerRoutes()

	addr := "0.0.0.0:8080"
	log.Printf("Starting server on %s", addr)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) Close() error {
	if s.cache != nil {
		return s.cache.Close()
	}
	return nil
}
