package lastfm

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(url string) *Client {
	return &Client{
		apiKey:     "test-key",
		baseURL:    url,
		httpClient: &http.Client{},
	}
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
			name:   "empty b returns a unchanged",
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
	_, err := c.ArtistGetSimilar("", "", 10, false)
	if err == nil {
		t.Fatal("expected error for empty artist and mbid")
	}
}

func TestArtistGetSimilar_ValidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"similarartists":{"artist":[{"name":"Artist1","mbid":"123","match":"0.9","url":"http://example.com/1"},{"name":"Artist2","mbid":"456","match":"0.5","url":"http://example.com/2"}]}}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	artists, err := c.ArtistGetSimilar("Radiohead", "", 10, true)
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

func TestArtistGetSimilar_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.ArtistGetSimilar("Radiohead", "", 10, false)
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestArtistGetSimilar_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.ArtistGetSimilar("Radiohead", "", 10, false)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestArtistGetInfo_EmptyInputs(t *testing.T) {
	c := newTestClient("http://unused")
	_, err := c.ArtistGetInfo("", "", false)
	if err == nil {
		t.Fatal("expected error for empty artist and mbid")
	}
}

func TestArtistGetInfo_ValidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"artist":{"name":"Radiohead","mbid":"abc","url":"http://example.com","tags":{"tag":[{"name":"alternative","url":"http://example.com/tag1"},{"name":"rock","url":"http://example.com/tag2"}]}}}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	info, err := c.ArtistGetInfo("Radiohead", "", true)
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

func TestArtistGetInfo_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.ArtistGetInfo("Radiohead", "", false)
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestArtistGetInfo_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{invalid`))
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.ArtistGetInfo("Radiohead", "", false)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
