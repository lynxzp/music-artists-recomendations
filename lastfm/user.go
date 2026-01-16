package lastfm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (c *Client) UserGetTopArtists(user, period string, limit, page int) ([]TopArtist, error) {
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

	requestURL := baseURL + "?" + q.Encode()

	var body []byte

	if c.cache != nil {
		if cached, ok := c.cache.Get(requestURL); ok {
			body = []byte(cached)
		}
	}

	if body == nil {
		if c.limiter != nil {
			c.limiter.Wait()
		}
		resp, err := http.Get(requestURL)
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

		if c.cache != nil {
			c.cache.Set(requestURL, string(body))
		}
	}

	var result topArtistsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.TopArtists.Artist, nil
}
