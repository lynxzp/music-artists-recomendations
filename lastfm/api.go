package lastfm

import (
	"music-recomendations/lastfm/cache"
	"music-recomendations/lastfm/ratelimit"
	"music-recomendations/lastfm/retry"
	"net/http"
	"time"
)

const BaseURL = "https://ws.audioscrobbler.com/2.0/"

type Client struct {
	apiKey     string
	cache      *cache.Cache
	limiter    *ratelimit.Limiter
	httpClient *http.Client
}

func newHTTPClient() *http.Client {
	baseTransport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &retry.RetryTransport{
			Base:   baseTransport,
			Delays: retry.DefaultDelays,
		},
	}
}

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey, httpClient: newHTTPClient()}
}

func NewClientWithCache(apiKey string, c *cache.Cache) *Client {
	return &Client{apiKey: apiKey, cache: c, httpClient: newHTTPClient()}
}

func NewClientWithCacheAndLimiter(apiKey string, c *cache.Cache, l *ratelimit.Limiter) *Client {
	return &Client{apiKey: apiKey, cache: c, limiter: l, httpClient: newHTTPClient()}
}
