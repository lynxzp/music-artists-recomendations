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

type Server struct {
	client *lastfm.Client
	cache  *cache.Cache
}

func New(apiKey string) (*Server, error) {
	apiCache, err := cache.New("./cache.db")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	limiter := ratelimit.New(time.Second)
	client := lastfm.NewClientWithCacheAndLimiter(apiKey, apiCache, limiter)

	return &Server{
		client: client,
		cache:  apiCache,
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
