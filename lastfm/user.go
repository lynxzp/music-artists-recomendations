package lastfm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

	params := map[string]string{"user": user}
	if period != "" {
		params["period"] = period
	}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	body, err := c.query(ctx, "user.gettopartists", params)
	if err != nil {
		return nil, err
	}

	var result topArtistsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, '%s'", err, string(body))
	}
	return result.TopArtists.Artist, nil
}
