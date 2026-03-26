package lastfm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type SimilarArtist struct {
	Name  string  `json:"name"`
	MBID  string  `json:"mbid"`
	Match float64 `json:"match,string"`
	URL   string  `json:"url"`
}

type ArtistTag struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ArtistInfo struct {
	Name  string `json:"name"`
	MBID  string `json:"mbid"`
	URL   string `json:"url"`
	Stats struct {
		UserPlaycount string `json:"userplaycount"`
	} `json:"stats"`
	Tags struct {
		Tag []ArtistTag `json:"tag"`
	} `json:"tags"`
}

type similarArtistsResponse struct {
	SimilarArtists struct {
		Artist []SimilarArtist `json:"artist"`
	} `json:"similarartists"`
}

func (c *Client) ArtistGetSimilar(ctx context.Context, artist, mbid string, limit int, autocorrect bool) ([]SimilarArtist, error) {
	if artist == "" && mbid == "" {
		return nil, fmt.Errorf("either Artist or MBID must be provided")
	}

	q := url.Values{}
	q.Set("method", "artist.getsimilar")
	q.Set("api_key", c.apiKey)
	q.Set("format", "json")

	if artist != "" {
		q.Set("artist", artist)
	}
	if mbid != "" {
		q.Set("mbid", mbid)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if autocorrect {
		q.Set("autocorrect", "1")
	}

	requestURL := c.baseURL + "?" + q.Encode()

	var body []byte

	if c.cache != nil {
		if cached, ok := c.cache.Get(requestURL); ok {
			body = []byte(cached)
		}
	}

	if body == nil {
		if c.limiter != nil {
			if err := c.limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter wait cancelled: %w", err)
			}
		}
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
			c.cache.Set(requestURL, string(body))
		}
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	var result similarArtistsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.SimilarArtists.Artist, nil
}

type artistInfoResponse struct {
	Artist ArtistInfo `json:"artist"`
}

func (c *Client) ArtistGetInfo(ctx context.Context, artist, mbid, username string, autocorrect bool) (*ArtistInfo, error) {
	if artist == "" && mbid == "" {
		return nil, fmt.Errorf("either Artist or MBID must be provided")
	}

	q := url.Values{}
	q.Set("method", "artist.getinfo")
	q.Set("api_key", c.apiKey)
	q.Set("format", "json")

	if artist != "" {
		q.Set("artist", artist)
	}
	if mbid != "" {
		q.Set("mbid", mbid)
	}
	if username != "" {
		q.Set("username", username)
	}
	if autocorrect {
		q.Set("autocorrect", "1")
	}

	requestURL := c.baseURL + "?" + q.Encode()

	var body []byte

	if c.cache != nil {
		if cached, ok := c.cache.Get(requestURL); ok {
			body = []byte(cached)
		}
	}

	if body == nil {
		if c.limiter != nil {
			if err := c.limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter wait cancelled: %w", err)
			}
		}
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
			c.cache.Set(requestURL, string(body))
		}
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	var result artistInfoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result.Artist, nil
}

// AppendSimilarArtists merges two SimilarArtist slices. For artists present in both,
// match values are summed. The weight parameter scales the match values from b before merging.
func AppendSimilarArtists(a, b []SimilarArtist, weight float64) []SimilarArtist {
	if len(b) == 0 {
		return a
	}

	// Build map from b keyed by artist name, applying weight to match values
	bMap := make(map[string]SimilarArtist, len(b))
	for _, artist := range b {
		artist.Match *= weight
		bMap[artist.Name] = artist
	}

	// Track which artists from b were processed
	processed := make(map[string]bool, len(b))

	// Iterate through a, summing match values for duplicates
	for i := range a {
		if bArtist, exists := bMap[a[i].Name]; exists {
			a[i].Match += bArtist.Match
			processed[a[i].Name] = true
		}
	}

	// Append unprocessed artists from b (using weighted values from bMap)
	for _, artist := range b {
		if !processed[artist.Name] {
			a = append(a, bMap[artist.Name])
		}
	}

	return a
}
