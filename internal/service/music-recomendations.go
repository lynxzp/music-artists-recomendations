package service

import (
	"fmt"
	"log"
	"music-recomendations/lastfm"
	"music-recomendations/lastfm/cache"
	"music-recomendations/lastfm/ratelimit"
	"time"
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

	apiCache, err := cache.New("./cache.db")
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}
	defer apiCache.Close()

	limiter := ratelimit.New(time.Second)
	client := lastfm.NewClientWithCacheAndLimiter(c.APIKey, apiCache, limiter)
	similar, err := client.ArtistGetSimilar(
		"Manowar",
		"",
		0,
		true,
	)
	if err != nil {
		return fmt.Errorf("failed to get similar artists: %w", err)
	}

	for _, a := range similar {
		fmt.Printf("%.2f %s \n", a.Match, a.Name)
	}

	return nil
}
