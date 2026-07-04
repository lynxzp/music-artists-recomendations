package service

import (
	"log/slog"
	"music-recomendations/internal/server"
	"os"
)

type Config struct {
	ProxyURL            string
	SimilarArtistsLimit int
	TopArtistsLimit     int
}

func Run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	c := Config{
		SimilarArtistsLimit: 500,
		TopArtistsLimit:     500,
		ProxyURL:            os.Getenv("LASTFM_PROXY_URL"),
	}

	srv := server.New(server.Config{
		ProxyURL:            c.ProxyURL,
		SimilarArtistsLimit: c.SimilarArtistsLimit,
		TopArtistsLimit:     c.TopArtistsLimit,
		Logger:              logger,
	})

	return srv.Start()
}
