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
	return NewClient(url, nil)
}

// decodeProxyReq reads a POST /v1/query body inside a test proxy handler and
// returns the method + params. It also locks the documented wire format:
// lowercase "method"/"params" keys, and no proxy-managed api_key/format.
func decodeProxyReq(t *testing.T, r *http.Request) (string, map[string]string) {
	t.Helper()
	if r.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", r.Method)
	}
	if r.URL.Path != "/v1/query" {
		t.Errorf("path = %q, want /v1/query", r.URL.Path)
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, `"method"`) || !strings.Contains(s, `"params"`) {
		t.Errorf("body missing lowercase method/params keys: %s", s)
	}
	var req struct {
		Method string            `json:"method"`
		Params map[string]string `json:"params"`
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if _, ok := req.Params["api_key"]; ok {
		t.Errorf("params must not contain api_key")
	}
	if _, ok := req.Params["format"]; ok {
		t.Errorf("params must not contain format")
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

func TestArtistGetInfo_NotFoundReturnsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":6,"message":"The artist you supplied could not be found"}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	info, err := c.ArtistGetInfo(context.Background(), "Nonexistent", "", "", false)
	if err != nil {
		t.Fatalf("404 should not be an error, got %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil *ArtistInfo, got nil")
	}
	if len(info.Tags.Tag) != 0 {
		t.Fatalf("got %d tags, want 0", len(info.Tags.Tag))
	}
}

func TestArtistGetInfo_UpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.ArtistGetInfo(context.Background(), "Radiohead", "", "", false)
	if err == nil {
		t.Fatal("expected error for 504 status")
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
