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
		q := r.URL.Query()
		if q.Get("user") != "testuser" {
			t.Errorf("user param = %q, want %q", q.Get("user"), "testuser")
		}
		if q.Get("period") != "7day" {
			t.Errorf("period param = %q, want %q", q.Get("period"), "7day")
		}
		if q.Get("limit") != "5" {
			t.Errorf("limit param = %q, want %q", q.Get("limit"), "5")
		}
		if q.Get("page") != "2" {
			t.Errorf("page param = %q, want %q", q.Get("page"), "2")
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

func TestUserGetTopArtists_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.UserGetTopArtists(context.Background(), "testuser", "overall", 10, 1)
	if err == nil {
		t.Fatal("expected error for non-200 status")
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
		q := r.URL.Query()
		if q.Get("period") != "" {
			t.Errorf("period should be omitted for empty string, got %q", q.Get("period"))
		}
		if q.Get("limit") != "" {
			t.Errorf("limit should be omitted for 0, got %q", q.Get("limit"))
		}
		if q.Get("page") != "" {
			t.Errorf("page should be omitted for 0, got %q", q.Get("page"))
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
