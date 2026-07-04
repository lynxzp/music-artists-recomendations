package lastfm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

	params := map[string]string{}
	if artist != "" {
		params["artist"] = artist
	}
	if mbid != "" {
		params["mbid"] = mbid
	}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	if autocorrect {
		params["autocorrect"] = "1"
	}

	body, err := c.query(ctx, "artist.getsimilar", params)
	if err != nil {
		return nil, err
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

	params := map[string]string{}
	if artist != "" {
		params["artist"] = artist
	}
	if mbid != "" {
		params["mbid"] = mbid
	}
	if username != "" {
		params["username"] = username
	}
	if autocorrect {
		params["autocorrect"] = "1"
	}

	body, err := c.query(ctx, "artist.getinfo", params)
	if err != nil {
		return nil, err
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
