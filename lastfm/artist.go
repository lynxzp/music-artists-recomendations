package lastfm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const baseURL = "https://ws.audioscrobbler.com/2.0/"

type SimilarArtist struct {
	Name  string  `json:"name"`
	MBID  string  `json:"mbid"`
	Match float64 `json:"match,string"`
	URL   string  `json:"url"`
}

type similarArtistsResponse struct {
	SimilarArtists struct {
		Artist []SimilarArtist `json:"artist"`
	} `json:"similarartists"`
}

func (c *Client) ArtistGetSimilar(artist, mbid string, limit int, autocorrect bool) ([]SimilarArtist, error) {
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
		resp, err := c.httpClient.Get(requestURL)
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

	var result similarArtistsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.SimilarArtists.Artist, nil
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
