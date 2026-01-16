package service

import (
	"log"
	"music-recomendations/internal/server"
)

type Config struct {
	APIKey string
}

func Run() error {
	c := Config{}
	err := loadConfig(&c)
	if err != nil {
		log.Fatal(err)
	}

	srv, err := server.New(c.APIKey)
	if err != nil {
		return err
	}
	defer srv.Close()

	return srv.Start()
}
