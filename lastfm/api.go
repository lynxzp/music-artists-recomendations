package lastfm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Client sends every Last.fm request through the lastfm-proxy service. The
// proxy owns the API key, caching, rate-limiting, retries, negative caching,
// and stale-if-error — so this client holds none of that.
type Client struct {
	proxyURL   string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient returns a client that posts to proxyURL (e.g. http://lastfm-proxy:8080).
// A trailing slash is trimmed so the endpoint join stays clean. A nil logger
// falls back to slog.Default().
func NewClient(proxyURL string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		proxyURL:   strings.TrimRight(proxyURL, "/"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
	}
}

// proxyRequest is the wire format for POST /v1/query. The JSON tags keep the
// keys lowercase to match the proxy's documented contract — do not rely on
// encoding/json's case-insensitive matching.
type proxyRequest struct {
	Method string            `json:"method"`
	Params map[string]string `json:"params"`
}

// query sends one request to the proxy and returns the raw Last.fm response
// body. A 200 or 404 returns the body (404 is the proxy's not-found /
// negative-cache disposition; its Last.fm error-6 envelope unmarshals to an
// empty result — so a genuine not-found yields empty data, not an error). Any
// other status is an error; the proxy already retries and serves stale, so
// there is nothing to retry here.
func (c *Client) query(ctx context.Context, method string, params map[string]string) ([]byte, error) {
	payload, err := json.Marshal(proxyRequest{Method: method, Params: params})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proxy request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.proxyURL+"/v1/query", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("proxy request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read proxy response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return body, nil
	case http.StatusNotFound:
		c.logger.Warn("lastfm lookup not found", "method", method, "params", params)
		return body, nil
	default:
		snippet := body
		if len(snippet) > 256 {
			snippet = snippet[:256]
		}
		return nil, fmt.Errorf("proxy returned status %d: %s", resp.StatusCode, snippet)
	}
}
