package service

import (
	"log"
	"music-recomendations/internal/server"
)

type Config struct {
	APIKey              string
	SimilarArtistsLimit int
	TopArtistsLimit     int
}

func Run() error {
	c := Config{
		SimilarArtistsLimit: 100,
		TopArtistsLimit:     100,
	}
	err := loadConfig(&c)
	if err != nil {
		log.Fatal(err)
	}

	srv, err := server.New(server.Config{
		APIKey:              c.APIKey,
		SimilarArtistsLimit: c.SimilarArtistsLimit,
		TopArtistsLimit:     c.TopArtistsLimit,
	})
	if err != nil {
		return err
	}
	defer srv.Close()

	return srv.Start()
}
