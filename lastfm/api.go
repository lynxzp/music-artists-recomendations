package lastfm

import (
	"music-recomendations/lastfm/cache"
	"music-recomendations/lastfm/ratelimit"
)

const BaseURL = "https://ws.audioscrobbler.com/2.0/"

type Client struct {
	apiKey  string
	cache   *cache.Cache
	limiter *ratelimit.Limiter
}

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

func NewClientWithCache(apiKey string, c *cache.Cache) *Client {
	return &Client{apiKey: apiKey, cache: c}
}

func NewClientWithCacheAndLimiter(apiKey string, c *cache.Cache, l *ratelimit.Limiter) *Client {
	return &Client{apiKey: apiKey, cache: c, limiter: l}
}
