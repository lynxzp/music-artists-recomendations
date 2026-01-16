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

type SeedArtist struct {
	Name   string
	Weight float64
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

	seeds := []SeedArtist{
		{Name: "Manowar", Weight: 1487},
	}

	var result []lastfm.SimilarArtist
	for _, seed := range seeds {
		similar, err := client.ArtistGetSimilar(seed.Name, "", 0, true)
		if err != nil {
			return fmt.Errorf("failed to get similar artists for %s: %w", seed.Name, err)
		}
		result = lastfm.AppendSimilarArtists(result, similar, seed.Weight)
	}

	for _, a := range result {
		fmt.Printf("%.2f %s \n", a.Match, a.Name)
	}

	return nil
}
