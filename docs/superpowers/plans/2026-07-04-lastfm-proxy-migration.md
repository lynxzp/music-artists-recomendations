# music-recomendations → lastfm-proxy Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route all Last.fm traffic from `music-recomendations` through the internal `lastfm-proxy` (`POST http://lastfm-proxy:8080/v1/query`) and delete its now-redundant local cache / rate-limiter / retry.

**Architecture:** Rewrite the `lastfm` package in place — same package name, same exported types, same three method signatures — so the `server.MusicClient` interface and every handler stay untouched. Only the client internals change from "GET Last.fm + local SQLite cache" to "POST proxy". The proxy owns the API key, caching, rate-limiting, retries, negative caching, and stale-if-error.

**Tech Stack:** Go 1.25, stdlib `net/http` + `encoding/json`, `httptest` for tests. No new dependencies; `modernc.org/sqlite` is removed.

---

## Repo & git conventions (READ FIRST)

- This service's `CLAUDE.md` says: **only the user commits.** Do **not** run `git commit`. After each task passes, **stage** with `git add`. Do not use `git -C`; run git from inside the relevant repo with relative paths.
- Two repos are touched:
  - **Service repo** `services/music-recomendations/` (its own local git repo): all Go code, Dockerfile, service `CLAUDE.md`, and these docs.
  - **Parent repo** `/Users/l/code/l/sergua.com/` (tracks orchestration): `docker-compose.yml` and the parent `CLAUDE.md`. Note `services/` is git-ignored by the parent, so service changes are staged in the service repo only.
- **Build/test cannot pass until the whole Go cluster (Tasks 1–4) is in place** — Go compiles the module as a whole and Tasks 1–4 are interdependent (deleting the cache breaks `server.go` until it's rewritten). **Task 5 is the single green checkpoint.** Do not run `go build`/`go test` at the end of Tasks 1–4; just write the files.

File map (service repo, paths relative to `services/music-recomendations/`):

- Rewrite: `lastfm/api.go` (new `Client` + `query` helper; drop `BaseURL`, `newHTTPClient`, old constructors)
- Rewrite: `lastfm/artist.go` (methods call `query`; types + `AppendSimilarArtists` unchanged)
- Rewrite: `lastfm/user.go` (method calls `query`)
- Delete: `lastfm/cache/`, `lastfm/ratelimit/`, `lastfm/retry/` (dirs incl. their tests)
- Rewrite: `lastfm/artist_test.go`, `lastfm/user_test.go` (proxy-shaped test server)
- Rewrite: `internal/server/server.go` (`Config.ProxyURL`; `New` returns `*Server`; no `cache`/`Close`)
- Rewrite: `internal/service/music-recomendations.go` (`ProxyURL` from env; no `err`/`Close`)
- Edit: `internal/server/static_test.go` (drop `APIKey`/`require`)
- Edit: `Dockerfile` (drop `/app/data`)
- Edit (parent repo): `docker-compose.yml`, `CLAUDE.md`

---

## Task 1: Rewrite the `lastfm` client core and methods

**Files:**
- Rewrite: `lastfm/api.go`
- Rewrite: `lastfm/artist.go`
- Rewrite: `lastfm/user.go`

> Do NOT build/test after this task — the module won't compile until Task 5.

- [ ] **Step 1: Replace `lastfm/api.go` with the proxy client**

```go
package lastfm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
}

// NewClient returns a client that posts to proxyURL (e.g. http://lastfm-proxy:8080).
// A trailing slash is trimmed so the endpoint join stays clean.
func NewClient(proxyURL string) *Client {
	return &Client{
		proxyURL:   strings.TrimRight(proxyURL, "/"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
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
// negative-cache disposition; its error-6 envelope unmarshals to an empty
// result — preserving the old "not found → empty" behavior). Any other status
// is an error; the proxy already retries and serves stale, so there is nothing
// to retry here.
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
	case http.StatusOK, http.StatusNotFound:
		return body, nil
	default:
		snippet := body
		if len(snippet) > 256 {
			snippet = snippet[:256]
		}
		return nil, fmt.Errorf("proxy returned status %d: %s", resp.StatusCode, snippet)
	}
}
```

- [ ] **Step 2: Replace `lastfm/artist.go` (methods use `query`; types & `AppendSimilarArtists` unchanged)**

```go
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
```

- [ ] **Step 3: Replace `lastfm/user.go`**

```go
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
```

- [ ] **Step 4: Stage**

Run (CWD = `services/music-recomendations/`):
```bash
git add lastfm/api.go lastfm/artist.go lastfm/user.go
```

---

## Task 2: Delete the redundant subpackages

**Files:**
- Delete: `lastfm/cache/` `lastfm/ratelimit/` `lastfm/retry/`

The proxy now owns caching, rate-limiting, and retries. These packages have no remaining importers after Task 1 (only `api.go` and `server.go` referenced them; `api.go` is done and `server.go` is Task 4).

- [ ] **Step 1: Remove the directories**

Run (CWD = `services/music-recomendations/`):
```bash
git rm -r lastfm/cache lastfm/ratelimit lastfm/retry
```
Expected: the removals are staged (`git rm` stages deletions). If the files are untracked in this local repo, use `rm -rf lastfm/cache lastfm/ratelimit lastfm/retry` instead.

---

## Task 3: Rewrite the `lastfm` package tests

**Files:**
- Rewrite: `lastfm/artist_test.go` (holds the shared `newTestClient` + `decodeProxyReq` helpers)
- Rewrite: `lastfm/user_test.go`

The old tests asserted params on the **URL query** and treated any non-200 as an error. The proxy client sends params in the **POST body** and treats 404 as not-found (empty, no error). These rewrites reflect that and lock the lowercase wire format.

> Do NOT build/test after this task — the module won't compile until Task 5.

- [ ] **Step 1: Replace `lastfm/artist_test.go`**

```go
package lastfm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(url string) *Client {
	return &Client{
		proxyURL:   strings.TrimRight(url, "/"),
		httpClient: &http.Client{},
	}
}

// decodeProxyReq reads a POST /v1/query body inside a test proxy handler and
// returns the method + params. It also locks the documented wire format:
// lowercase "method"/"params" keys, and no proxy-managed api_key/format.
func decodeProxyReq(t *testing.T, r *http.Request) (string, map[string]string) {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, `"method"`) || !strings.Contains(s, `"params"`) {
		t.Errorf("body missing lowercase method/params keys: %s", s)
	}
	if strings.Contains(s, "api_key") || strings.Contains(s, "format") {
		t.Errorf("body must not send api_key/format: %s", s)
	}
	var req struct {
		Method string            `json:"method"`
		Params map[string]string `json:"params"`
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	return req.Method, req.Params
}

func TestAppendSimilarArtists(t *testing.T) {
	tests := []struct {
		name   string
		a      []SimilarArtist
		b      []SimilarArtist
		weight float64
		want   map[string]float64 // expected normalized matches
	}{
		{
			name: "no duplicates weight 1",
			a: []SimilarArtist{
				{Name: "Artist1", Match: 100},
				{Name: "Artist2", Match: 50},
			},
			b: []SimilarArtist{
				{Name: "Artist3", Match: 80},
				{Name: "Artist4", Match: 40},
			},
			weight: 1.0,
			want: map[string]float64{
				"Artist1": 100,
				"Artist2": 50,
				"Artist3": 80,
				"Artist4": 40,
			},
		},
		{
			name: "with duplicates sums matches",
			a: []SimilarArtist{
				{Name: "Artist1", Match: 50},
				{Name: "Artist2", Match: 50},
			},
			b: []SimilarArtist{
				{Name: "Artist2", Match: 50},
				{Name: "Artist3", Match: 25},
			},
			weight: 1.0,
			want: map[string]float64{
				"Artist1": 50,
				"Artist2": 100, // 50 + 50
				"Artist3": 25,
			},
		},
		{
			name: "weight scales b values",
			a: []SimilarArtist{
				{Name: "Artist1", Match: 100},
			},
			b: []SimilarArtist{
				{Name: "Artist2", Match: 100},
			},
			weight: 0.5,
			want: map[string]float64{
				"Artist1": 100, // max
				"Artist2": 50,  // 100 * 0.5
			},
		},
		{
			name: "empty b returns a unchanged",
			a: []SimilarArtist{
				{Name: "Artist1", Match: 80},
				{Name: "Artist2", Match: 40},
			},
			b:      []SimilarArtist{},
			weight: 1.0,
			want: map[string]float64{
				"Artist1": 80,
				"Artist2": 40,
			},
		},
		{
			name: "nil a with non-empty b",
			a:    nil,
			b: []SimilarArtist{
				{Name: "Artist1", Match: 80},
				{Name: "Artist2", Match: 40},
			},
			weight: 0.5,
			want: map[string]float64{
				"Artist1": 40,
				"Artist2": 20,
			},
		},
		{
			name:   "nil a and nil b",
			a:      nil,
			b:      nil,
			weight: 1.0,
			want:   map[string]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendSimilarArtists(tt.a, tt.b, tt.weight)

			if len(got) != len(tt.want) {
				if !(got == nil && len(tt.want) == 0) {
					t.Errorf("got %d artists, want %d", len(got), len(tt.want))
					return
				}
			}

			for _, artist := range got {
				expected, ok := tt.want[artist.Name]
				if !ok {
					t.Errorf("unexpected artist %q in result", artist.Name)
					continue
				}
				if artist.Match != expected {
					t.Errorf("artist %q: got match %.2f, want %.2f", artist.Name, artist.Match, expected)
				}
			}
		})
	}
}

func TestArtistGetSimilar_EmptyInputs(t *testing.T) {
	c := newTestClient("http://unused")
	_, err := c.ArtistGetSimilar(context.Background(), "", "", 10, false)
	if err == nil {
		t.Fatal("expected error for empty artist and mbid")
	}
}

func TestArtistGetSimilar_ValidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, params := decodeProxyReq(t, r)
		if method != "artist.getsimilar" {
			t.Errorf("method = %q, want %q", method, "artist.getsimilar")
		}
		if params["artist"] != "Radiohead" {
			t.Errorf("artist param = %q, want %q", params["artist"], "Radiohead")
		}
		if params["autocorrect"] != "1" {
			t.Errorf("autocorrect param = %q, want %q", params["autocorrect"], "1")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"similarartists":{"artist":[{"name":"Artist1","mbid":"123","match":"0.9","url":"http://example.com/1"},{"name":"Artist2","mbid":"456","match":"0.5","url":"http://example.com/2"}]}}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	artists, err := c.ArtistGetSimilar(context.Background(), "Radiohead", "", 10, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artists) != 2 {
		t.Fatalf("got %d artists, want 2", len(artists))
	}
	if artists[0].Name != "Artist1" {
		t.Errorf("artists[0].Name = %q, want %q", artists[0].Name, "Artist1")
	}
	if artists[0].Match != 0.9 {
		t.Errorf("artists[0].Match = %v, want 0.9", artists[0].Match)
	}
}

func TestArtistGetSimilar_NotFoundReturnsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":6,"message":"The artist you supplied could not be found"}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	artists, err := c.ArtistGetSimilar(context.Background(), "Nonexistent", "", 10, false)
	if err != nil {
		t.Fatalf("404 should not be an error, got %v", err)
	}
	if len(artists) != 0 {
		t.Fatalf("got %d artists, want 0", len(artists))
	}
}

func TestArtistGetSimilar_UpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"upstream_error"}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.ArtistGetSimilar(context.Background(), "Radiohead", "", 10, false)
	if err == nil {
		t.Fatal("expected error for 502 status")
	}
}

func TestArtistGetSimilar_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.ArtistGetSimilar(context.Background(), "Radiohead", "", 10, false)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestArtistGetInfo_EmptyInputs(t *testing.T) {
	c := newTestClient("http://unused")
	_, err := c.ArtistGetInfo(context.Background(), "", "", "", false)
	if err == nil {
		t.Fatal("expected error for empty artist and mbid")
	}
}

func TestArtistGetInfo_ValidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, params := decodeProxyReq(t, r)
		if method != "artist.getinfo" {
			t.Errorf("method = %q, want %q", method, "artist.getinfo")
		}
		if params["artist"] != "Radiohead" {
			t.Errorf("artist param = %q, want %q", params["artist"], "Radiohead")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"artist":{"name":"Radiohead","mbid":"abc","url":"http://example.com","tags":{"tag":[{"name":"alternative","url":"http://example.com/tag1"},{"name":"rock","url":"http://example.com/tag2"}]}}}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	info, err := c.ArtistGetInfo(context.Background(), "Radiohead", "", "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "Radiohead" {
		t.Errorf("Name = %q, want %q", info.Name, "Radiohead")
	}
	if len(info.Tags.Tag) != 2 {
		t.Fatalf("got %d tags, want 2", len(info.Tags.Tag))
	}
	if info.Tags.Tag[0].Name != "alternative" {
		t.Errorf("tag[0].Name = %q, want %q", info.Tags.Tag[0].Name, "alternative")
	}
}

func TestArtistGetInfo_UpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.ArtistGetInfo(context.Background(), "Radiohead", "", "", false)
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestArtistGetInfo_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{invalid`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.ArtistGetInfo(context.Background(), "Radiohead", "", "", false)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
```

- [ ] **Step 2: Replace `lastfm/user_test.go`**

```go
package lastfm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserGetTopArtists_EmptyUser(t *testing.T) {
	c := newTestClient("http://unused")
	_, err := c.UserGetTopArtists(context.Background(), "", "overall", 10, 1)
	if err == nil {
		t.Fatal("expected error for empty user")
	}
}

func TestUserGetTopArtists_ValidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, params := decodeProxyReq(t, r)
		if method != "user.gettopartists" {
			t.Errorf("method = %q, want %q", method, "user.gettopartists")
		}
		if params["user"] != "testuser" {
			t.Errorf("user param = %q, want %q", params["user"], "testuser")
		}
		if params["period"] != "7day" {
			t.Errorf("period param = %q, want %q", params["period"], "7day")
		}
		if params["limit"] != "5" {
			t.Errorf("limit param = %q, want %q", params["limit"], "5")
		}
		if params["page"] != "2" {
			t.Errorf("page param = %q, want %q", params["page"], "2")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"topartists":{"artist":[{"name":"Radiohead","mbid":"abc","playcount":"500","url":"http://example.com/1"},{"name":"Portishead","mbid":"def","playcount":"300","url":"http://example.com/2"}]}}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	artists, err := c.UserGetTopArtists(context.Background(), "testuser", "7day", 5, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artists) != 2 {
		t.Fatalf("got %d artists, want 2", len(artists))
	}
	if artists[0].Name != "Radiohead" {
		t.Errorf("artists[0].Name = %q, want %q", artists[0].Name, "Radiohead")
	}
	if artists[0].Playcount != "500" {
		t.Errorf("artists[0].Playcount = %q, want %q", artists[0].Playcount, "500")
	}
}

func TestUserGetTopArtists_UpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.UserGetTopArtists(context.Background(), "testuser", "overall", 10, 1)
	if err == nil {
		t.Fatal("expected error for 503 status")
	}
}

func TestUserGetTopArtists_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`corrupted`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.UserGetTopArtists(context.Background(), "testuser", "overall", 10, 1)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestUserGetTopArtists_OptionalParamsOmittedWhenZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params := decodeProxyReq(t, r)
		if _, ok := params["period"]; ok {
			t.Errorf("period should be omitted for empty string, got %q", params["period"])
		}
		if _, ok := params["limit"]; ok {
			t.Errorf("limit should be omitted for 0, got %q", params["limit"])
		}
		if _, ok := params["page"]; ok {
			t.Errorf("page should be omitted for 0, got %q", params["page"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"topartists":{"artist":[]}}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.UserGetTopArtists(context.Background(), "testuser", "", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 3: Stage**

Run (CWD = `services/music-recomendations/`):
```bash
git add lastfm/artist_test.go lastfm/user_test.go
```

---

## Task 4: Rewire `server`, `service`, and the static test

**Files:**
- Rewrite: `internal/server/server.go`
- Rewrite: `internal/service/music-recomendations.go`
- Edit: `internal/server/static_test.go`

> Do NOT build/test after this task — run the checkpoint in Task 5.

- [ ] **Step 1: Replace `internal/server/server.go`**

```go
package server

import (
	"context"
	"log/slog"
	"music-recomendations/lastfm"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const defaultProxyURL = "http://lastfm-proxy:8080"

type Config struct {
	ProxyURL            string
	SimilarArtistsLimit int
	TopArtistsLimit     int
	Logger              *slog.Logger
}

type MusicClient interface {
	ArtistGetSimilar(ctx context.Context, artist, mbid string, limit int, autocorrect bool) ([]lastfm.SimilarArtist, error)
	ArtistGetInfo(ctx context.Context, artist, mbid, username string, autocorrect bool) (*lastfm.ArtistInfo, error)
	UserGetTopArtists(ctx context.Context, user, period string, limit, page int) ([]lastfm.TopArtist, error)
}

type Server struct {
	client     MusicClient
	config     Config
	logger     *slog.Logger
	httpServer *http.Server
}

func New(cfg Config) *Server {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}

	proxyURL := cfg.ProxyURL
	if proxyURL == "" {
		proxyURL = defaultProxyURL
	}

	return &Server{
		client: lastfm.NewClient(proxyURL),
		config: cfg,
		logger: logger,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	addr := "0.0.0.0:8080"
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to signal shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		s.logger.Info("starting server", "addr", addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-shutdownChan:
		s.logger.Info("received shutdown signal", "signal", sig.String())
	case err := <-serverErr:
		if err != nil {
			return err
		}
	}

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.logger.Info("shutting down server")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("server shutdown error", "error", err)
		return err
	}

	s.logger.Info("server stopped")
	return nil
}
```

Notes: `Server.cache`, `Server.Close()`, and the `error` return from `New` are gone (the only error source was `cache.New`; `lastfm.NewClient` cannot fail). The `fmt`, `music-recomendations/lastfm/cache`, and `music-recomendations/lastfm/ratelimit` imports are gone. `handlers.go` and `routes.go` are unchanged (handlers only use `s.config.SimilarArtistsLimit`/`TopArtistsLimit` and `s.client`).

- [ ] **Step 2: Replace `internal/service/music-recomendations.go`**

```go
package service

import (
	"log/slog"
	"music-recomendations/internal/server"
	"os"
)

type Config struct {
	ProxyURL            string
	SimilarArtistsLimit int
	TopArtistsLimit     int
}

func Run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	c := Config{
		SimilarArtistsLimit: 500,
		TopArtistsLimit:     500,
		ProxyURL:            os.Getenv("LASTFM_PROXY_URL"),
	}

	srv := server.New(server.Config{
		ProxyURL:            c.ProxyURL,
		SimilarArtistsLimit: c.SimilarArtistsLimit,
		TopArtistsLimit:     c.TopArtistsLimit,
		Logger:              logger,
	})

	return srv.Start()
}
```

- [ ] **Step 3: Fix `internal/server/static_test.go`**

Change the `TestStaticFiles_ServeCSS` opening from:
```go
	s, err := New(Config{APIKey: "test", Logger: testLogger(t)})
	require.NoError(t, err)
```
to:
```go
	s := New(Config{Logger: testLogger(t)})
```

Then remove the now-unused `require` import (line `"github.com/stretchr/testify/require"`). Leave the `assert` import — it is still used elsewhere in the file.

- [ ] **Step 4: Stage**

Run (CWD = `services/music-recomendations/`):
```bash
git add internal/server/server.go internal/service/music-recomendations.go internal/server/static_test.go
```

---

## Task 5: Tidy modules and verify the whole build (GREEN CHECKPOINT)

**Files:**
- Modify: `go.mod`, `go.sum` (via `go mod tidy`)

- [ ] **Step 1: Tidy dependencies**

Run (CWD = `services/music-recomendations/`):
```bash
go mod tidy
```
Expected: `modernc.org/sqlite` and its transitive deps (`modernc.org/libc`, `memory`, `mathutil`, `ncruces/go-strftime`, `remyoudompheng/bigfft`, `google/uuid`, `dustin/go-humanize`, `mattn/go-isatty`) drop out of `go.mod`/`go.sum`. `testify` and its deps (`davecgh/go-spew`, `pmezard/go-difflib`, `gopkg.in/yaml.v3`) remain.

- [ ] **Step 2: Vet**

Run: `go vet ./...`
Expected: no output (exit 0).

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: no output (exit 0). If it fails on an import of `lastfm/cache`, `lastfm/ratelimit`, `lastfm/retry`, or `fmt` in `server.go`, a prior task was left incomplete — fix before proceeding.

- [ ] **Step 4: Run all tests with the race detector**

Run: `go test -race ./...`
Expected: all packages `ok` (this matches `make test`). The `internal/server` handler tests are unchanged and must still pass (they mock `MusicClient`, whose signatures are unchanged).

- [ ] **Step 5: Stage**

Run (CWD = `services/music-recomendations/`):
```bash
git add go.mod go.sum
```

---

## Task 6: Trim the Dockerfile

**Files:**
- Edit: `Dockerfile`

The local SQLite cache is gone, so the `/app/data` directory it provisioned is dead scaffolding.

- [ ] **Step 1: Remove the two `/app/data` lines**

Delete this line (currently line 9):
```dockerfile
RUN mkdir -p /app/data && chown -R scratchuser:scratchuser /app/data
```
And this line (currently line 15):
```dockerfile
COPY --from=builder --chown=10001:10001 /app/data /app/data
```

The resulting `Dockerfile` is:
```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache ca-certificates
RUN adduser -D -u 10001 scratchuser
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o music-recomendations .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/music-recomendations /music-recomendations
COPY --from=builder /etc/passwd /etc/passwd
USER scratchuser
EXPOSE 8080
ENTRYPOINT ["/music-recomendations"]
```

- [ ] **Step 2: Stage**

Run (CWD = `services/music-recomendations/`):
```bash
git add Dockerfile
```

---

## Task 7: Update `docker-compose.yml` (parent repo)

**Files:**
- Edit (parent repo): `docker-compose.yml`

- [ ] **Step 1: Rewrite the `music-recomendations` service env + volumes**

In the `music-recomendations:` service block, replace:
```yaml
    environment:
      - API_KEY=${LASTFM_API_KEY}
      - CACHE_PATH=/app/data/cache.db
    volumes:
      - music-recomendations-cache:/app/data
```
with:
```yaml
    environment:
      - LASTFM_PROXY_URL=http://lastfm-proxy:8080
```
(The service no longer needs the Last.fm key or a data volume — the proxy holds the key and owns the cache.)

- [ ] **Step 2: Remove the now-unused named volume**

In the top-level `volumes:` section, delete the line:
```yaml
  music-recomendations-cache: {}
```
Leaving a declared-but-unmounted volume would be dead config.

- [ ] **Step 3: Sanity-check compose syntax**

Run (CWD = parent repo root `/Users/l/code/l/sergua.com/`):
```bash
docker compose config >/dev/null && echo OK
```
Expected: `OK`. (If `docker compose` is unavailable in this environment, skip — the edit is a mechanical two-block change.)

- [ ] **Step 4: Stage**

Run (CWD = parent repo root):
```bash
git add docker-compose.yml
```

---

## Task 8: Update documentation

**Files:**
- Edit: `CLAUDE.md` (service repo)
- Edit (parent repo): `CLAUDE.md`

- [ ] **Step 1: Parent `CLAUDE.md` — Last.fm key consumer**

In `/Users/l/code/l/sergua.com/CLAUDE.md`, "Environment Variables" section, change:
```
- `LASTFM_API_KEY` - Last.fm API key; consumed by music-recomendations and by lastfm-proxy
```
to:
```
- `LASTFM_API_KEY` - Last.fm API key; consumed only by lastfm-proxy (the sole holder of the key; music-recomendations now reaches Last.fm through the proxy)
```

- [ ] **Step 2: Service `CLAUDE.md` — Configuration section**

In `services/music-recomendations/CLAUDE.md`, replace the "## Configuration" block:
```
Environment variables:

​```bash
export API_KEY=your_lastfm_api_key
export CACHE_PATH=./cache.db  # optional, defaults to ./cache.db
​```
```
with:
```
Environment variables:

​```bash
export LASTFM_PROXY_URL=http://lastfm-proxy:8080  # optional, this is the default
​```

All Last.fm access goes through the internal `lastfm-proxy` service, which holds
the API key and owns caching, rate-limiting, retries, and negative caching. This
service keeps no local cache and no API key.
```

- [ ] **Step 3: Service `CLAUDE.md` — Architecture section**

Replace the `lastfm/` sub-bullets under "## Architecture":
```
- `lastfm/` - Last.fm API client package
  - `api.go` - Client configuration and HTTP client setup
  - `artist.go` - Artist.getSimilar and Artist.getInfo endpoints
  - `user.go` - User.getTopArtists endpoint
  - `cache/cache.go` - SQLite-based response cache (7-day TTL)
  - `ratelimit/ratelimit.go` - Token bucket rate limiter (1 req/sec)
  - `retry/retry.go` - HTTP transport with exponential backoff retry
```
with:
```
- `lastfm/` - Thin client for the lastfm-proxy service
  - `api.go` - `Client` (proxy URL + `http.Client`) and the shared `query` helper (POST `/v1/query`)
  - `artist.go` - Artist.getSimilar and Artist.getInfo (types + `AppendSimilarArtists`)
  - `user.go` - User.getTopArtists
```
Also, in the same section, update the `server.go` bullet from `graceful shutdown, `MusicClient` interface` — keep it, but note `New` no longer returns an error and there is no cache to close (see Packages and Functions below).

- [ ] **Step 4: Service `CLAUDE.md` — Packages and Functions**

Under "### `lastfm`", replace the `Constants`, `Types` (`Client`), and `Functions` (constructors) entries and delete the `lastfm/cache`, `lastfm/ratelimit`, `lastfm/retry` subsections:
```
### `lastfm`

**Types:**
- `Client` — `proxyURL`, `httpClient`
- `proxyRequest` (unexported) — `Method`, `Params` (JSON-tagged lowercase for the proxy wire format)
- `SimilarArtist`, `ArtistTag`, `ArtistInfo`, `TopArtist` — response models (unchanged)
- `similarArtistsResponse`, `artistInfoResponse`, `topArtistsResponse` (unexported)

**Functions:**
- `NewClient(proxyURL string) *Client` — client posting to the lastfm-proxy at `proxyURL`
- `(*Client) query(ctx, method, params) ([]byte, error)` (unexported) — POST `/v1/query`; 200/404 → body, else error
- `(*Client) ArtistGetSimilar(...) ([]SimilarArtist, error)` — unchanged signature
- `(*Client) ArtistGetInfo(...) (*ArtistInfo, error)` — unchanged signature
- `(*Client) UserGetTopArtists(...) ([]TopArtist, error)` — unchanged signature
- `AppendSimilarArtists(a, b []SimilarArtist, weight float64) []SimilarArtist` — unchanged
```
Then under "### `internal/server`" update `Config`, `Server`, and the constructor:
```
- `Config` — `ProxyURL`, `SimilarArtistsLimit`, `TopArtistsLimit`, `Logger`
- `Server` (unexported) — `client MusicClient`, `config`, `logger`, `httpServer`
- `New(cfg Config) *Server` — builds the proxy-backed client (defaults `ProxyURL` to `http://lastfm-proxy:8080`); no longer returns an error
```
Remove the `(*Server) Close()` bullet. Under "### `internal/service`" change `Config` to `ProxyURL`, `SimilarArtistsLimit`, `TopArtistsLimit`.

- [ ] **Step 5: Stage**

Run for the service doc (CWD = `services/music-recomendations/`):
```bash
git add CLAUDE.md docs/superpowers
```
Run for the parent doc (CWD = parent repo root):
```bash
git add CLAUDE.md
```

---

## Task 9: Final verification & handoff

- [ ] **Step 1: Re-run the full service check**

Run (CWD = `services/music-recomendations/`):
```bash
go vet ./... && go build ./... && go test -race ./...
```
Expected: clean vet, clean build, all tests `ok`.

- [ ] **Step 2: Confirm no lingering references**

Run (CWD = `services/music-recomendations/`):
```bash
grep -rn 'API_KEY\|CACHE_PATH\|/lastfm/cache\|/lastfm/ratelimit\|/lastfm/retry\|BaseURL' --include='*.go' . || echo "clean"
```
Expected: `clean` (no matches).

- [ ] **Step 3: Report status to the user (do NOT commit)**

Summarize what changed and what remains **user-gated** (per this repo's model):
- `make build && make push` for `lynxzp/music-recomendations` (watchtower deploys within ~60s).
- Bringing the stack up with the edited `docker-compose.yml` (`docker compose up -d music-recomendations`) — and dropping the old `music-recomendations-cache` named volume on the host if desired (`docker volume rm <stack>_music-recomendations-cache`).
- `make lint` (golangci-lint) if the linter is installed locally.
- Committing the staged changes (only the user commits).

---

## Self-Review

**Spec coverage:** Client contract (T1) · delete cache/ratelimit/retry (T2) · tests with lowercase-wire-format lock + 404→empty + params-in-body (T3) · server/service/static rewiring, `New` without error, no `Close`/`cache` (T4) · `go mod tidy` + build/vet/test green (T5) · Dockerfile `/app/data` removal (T6) · compose env/volume/volumes cleanup (T7) · both `CLAUDE.md` files + `LASTFM_API_KEY` consumer note (T8) · final verification + user-gated deploy (T9). All spec sections map to a task.

**Placeholder scan:** none — every code step shows full content; every command shows expected output.

**Type consistency:** `proxyRequest{Method,Params}` (T1) matches the test decode struct and `decodeProxyReq` (T3); `Client{proxyURL,httpClient}` (T1) matches `newTestClient` (T3); `NewClient(proxyURL)` (T1) matches `server.New` (T4); `Config.ProxyURL` used identically in `server` (T4) and `service` (T4); method signatures unchanged, so `MusicClient` (T4) and `handlers_test.go` mock stay valid.
