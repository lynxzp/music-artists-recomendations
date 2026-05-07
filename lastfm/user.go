package lastfm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type TopArtist struct {
	Name      string `json:"name"`
	MBID      string `json:"mbid"`
	Playcount string `json:"playcount"`
	URL       string `json:"url"`
}

type topArtistsResponse struct {
	TopArtists struct {
		Artist []TopArtist `json:"artist"`
	} `json:"topartists"`
}

func (c *Client) UserGetTopArtists(ctx context.Context, user, period string, limit, page int) ([]TopArtist, error) {
	if user == "" {
		return nil, fmt.Errorf("user must be provided")
	}

	q := url.Values{}
	q.Set("method", "user.gettopartists")
	q.Set("api_key", c.apiKey)
	q.Set("format", "json")
	q.Set("user", user)

	if period != "" {
		q.Set("period", period)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}

	requestURL := c.baseURL + "?" + q.Encode()

	var body []byte

	if c.cache != nil {
		if cached, ok := c.cache.Get(requestURL); ok {
			body = []byte(cached)
		}
	}

	if body == nil {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if c.cache != nil && len(body) > 0 {
			c.cache.SetWithTTL(requestURL, string(body), 5*time.Minute)
		}
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	var result topArtistsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, '%s'", err, string(body))
	}

	return result.TopArtists.Artist, nil
}
