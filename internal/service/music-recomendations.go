package service

import (
	"music-recomendations/internal/server"
	"os"
)

type Config struct {
	APIKey              string
	SimilarArtistsLimit int
	TopArtistsLimit     int
	CachePath           string
}

func Run() error {
	c := Config{
		SimilarArtistsLimit: 100,
		TopArtistsLimit:     100,
		CachePath:           os.Getenv("CACHE_PATH"),
		APIKey:              os.Getenv("API_KEY"),
	}

	srv, err := server.New(server.Config{
		APIKey:              c.APIKey,
		SimilarArtistsLimit: c.SimilarArtistsLimit,
		TopArtistsLimit:     c.TopArtistsLimit,
		CachePath:           c.CachePath,
	})
	if err != nil {
		return err
	}
	defer srv.Close()

	return srv.Start()
}
