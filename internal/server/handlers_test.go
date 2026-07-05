package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"music-recomendations/lastfm"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockClient struct {
	artistGetSimilarFn  func(ctx context.Context, artist, mbid string, limit int, autocorrect bool) ([]lastfm.SimilarArtist, error)
	artistGetInfoFn     func(ctx context.Context, artist, mbid, username string, autocorrect bool) (*lastfm.ArtistInfo, error)
	userGetTopArtistsFn func(ctx context.Context, user, period string, limit, page int) ([]lastfm.TopArtist, error)
}

func (m *mockClient) ArtistGetSimilar(ctx context.Context, artist, mbid string, limit int, autocorrect bool) ([]lastfm.SimilarArtist, error) {
	return m.artistGetSimilarFn(ctx, artist, mbid, limit, autocorrect)
}

func (m *mockClient) ArtistGetInfo(ctx context.Context, artist, mbid, username string, autocorrect bool) (*lastfm.ArtistInfo, error) {
	return m.artistGetInfoFn(ctx, artist, mbid, username, autocorrect)
}

func (m *mockClient) UserGetTopArtists(ctx context.Context, user, period string, limit, page int) ([]lastfm.TopArtist, error) {
	return m.userGetTopArtistsFn(ctx, user, period, limit, page)
}

func newTestServer(client MusicClient) *Server {
	return &Server{
		client: client,
		config: Config{SimilarArtistsLimit: 10, TopArtistsLimit: 50},
		logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}
}

func TestHandleIndex(t *testing.T) {
	s := newTestServer(nil)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	s.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/html")
	}
}

func TestHandleArtistGetSimilar_Valid(t *testing.T) {
	mock := &mockClient{
		artistGetSimilarFn: func(ctx context.Context, artist, mbid string, limit int, autocorrect bool) ([]lastfm.SimilarArtist, error) {
			return []lastfm.SimilarArtist{
				{Name: "Artist1", Match: 0.9},
			}, nil
		},
	}
	s := newTestServer(mock)
	req := httptest.NewRequest("GET", "/api/artist/similar?artist=Radiohead", nil)
	w := httptest.NewRecorder()

	s.handleArtistGetSimilar(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	artists := data["artists"].([]interface{})
	if len(artists) != 1 {
		t.Errorf("got %d artists, want 1", len(artists))
	}
}

func TestHandleArtistGetSimilar_InvalidArtist(t *testing.T) {
	s := newTestServer(nil)
	req := httptest.NewRequest("GET", "/api/artist/similar?artist=bad%0Aname", nil)
	w := httptest.NewRecorder()

	s.handleArtistGetSimilar(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleArtistGetSimilar_EmptyArtist(t *testing.T) {
	mock := &mockClient{
		artistGetSimilarFn: func(ctx context.Context, artist, mbid string, limit int, autocorrect bool) ([]lastfm.SimilarArtist, error) {
			return nil, fmt.Errorf("either Artist or MBID must be provided")
		},
	}
	s := newTestServer(mock)
	req := httptest.NewRequest("GET", "/api/artist/similar?artist=", nil)
	w := httptest.NewRecorder()

	s.handleArtistGetSimilar(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleArtistGetSimilar_ClientError(t *testing.T) {
	mock := &mockClient{
		artistGetSimilarFn: func(ctx context.Context, artist, mbid string, limit int, autocorrect bool) ([]lastfm.SimilarArtist, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	s := newTestServer(mock)
	req := httptest.NewRequest("GET", "/api/artist/similar?artist=Radiohead", nil)
	w := httptest.NewRecorder()

	s.handleArtistGetSimilar(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleArtistGetInfo_Valid(t *testing.T) {
	mock := &mockClient{
		artistGetInfoFn: func(ctx context.Context, artist, mbid, username string, autocorrect bool) (*lastfm.ArtistInfo, error) {
			info := &lastfm.ArtistInfo{Name: "Radiohead"}
			info.Tags.Tag = []lastfm.ArtistTag{
				{Name: "Alternative Rock"},
				{Name: "ELECTRONIC"},
			}
			return info, nil
		},
	}
	s := newTestServer(mock)
	req := httptest.NewRequest("GET", "/api/artist/info?artist=Radiohead", nil)
	w := httptest.NewRecorder()

	s.handleArtistGetInfo(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	artist := data["artist"].(map[string]interface{})
	tags := artist["tags"].(map[string]interface{})
	tagList := tags["tag"].([]interface{})

	tag0 := tagList[0].(map[string]interface{})
	if tag0["name"] != "alternative rock" {
		t.Errorf("tag[0].name = %q, want %q", tag0["name"], "alternative rock")
	}
	tag1 := tagList[1].(map[string]interface{})
	if tag1["name"] != "electronic" {
		t.Errorf("tag[1].name = %q, want %q", tag1["name"], "electronic")
	}
}

func TestHandleArtistGetInfo_InvalidArtist(t *testing.T) {
	s := newTestServer(nil)
	req := httptest.NewRequest("GET", "/api/artist/info?artist=bad%0Aname", nil)
	w := httptest.NewRecorder()

	s.handleArtistGetInfo(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleArtistGetInfo_EmptyArtist(t *testing.T) {
	mock := &mockClient{
		artistGetInfoFn: func(ctx context.Context, artist, mbid, username string, autocorrect bool) (*lastfm.ArtistInfo, error) {
			return nil, fmt.Errorf("either Artist or MBID must be provided")
		},
	}
	s := newTestServer(mock)
	req := httptest.NewRequest("GET", "/api/artist/info?artist=", nil)
	w := httptest.NewRecorder()

	s.handleArtistGetInfo(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleArtistGetInfo_ClientError(t *testing.T) {
	mock := &mockClient{
		artistGetInfoFn: func(ctx context.Context, artist, mbid, username string, autocorrect bool) (*lastfm.ArtistInfo, error) {
			return nil, fmt.Errorf("api error")
		},
	}
	s := newTestServer(mock)
	req := httptest.NewRequest("GET", "/api/artist/info?artist=Radiohead", nil)
	w := httptest.NewRecorder()

	s.handleArtistGetInfo(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleAppendSimilarArtists_Valid(t *testing.T) {
	s := newTestServer(nil)
	body := `{"a":[{"name":"A1","match":"50"}],"b":[{"name":"A2","match":"80"}],"weight":0.5}`
	req := httptest.NewRequest("POST", "/api/append", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleAppendSimilarArtists(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	artists := data["artists"].([]interface{})
	if len(artists) != 2 {
		t.Errorf("got %d artists, want 2", len(artists))
	}
}

func TestHandleAppendSimilarArtists_InvalidJSON(t *testing.T) {
	s := newTestServer(nil)
	req := httptest.NewRequest("POST", "/api/append", strings.NewReader(`{invalid`))
	w := httptest.NewRecorder()

	s.handleAppendSimilarArtists(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleAppendSimilarArtists_EmptyBody(t *testing.T) {
	s := newTestServer(nil)
	req := httptest.NewRequest("POST", "/api/append", strings.NewReader(""))
	w := httptest.NewRecorder()

	s.handleAppendSimilarArtists(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleAppendSimilarArtists_BothEmpty(t *testing.T) {
	s := newTestServer(nil)
	body := `{"a":[],"b":[],"weight":1.0}`
	req := httptest.NewRequest("POST", "/api/append", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleAppendSimilarArtists(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleAppendSimilarArtists_ZeroWeight(t *testing.T) {
	s := newTestServer(nil)
	body := `{"a":[{"name":"A1","match":"100"}],"b":[{"name":"A2","match":"100"}],"weight":0}`
	req := httptest.NewRequest("POST", "/api/append", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleAppendSimilarArtists(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	artists := data["artists"].([]interface{})
	// Match field has json:"match,string" tag, so it serializes as a JSON string
	for _, a := range artists {
		artist := a.(map[string]interface{})
		if artist["name"] == "A2" {
			if match, ok := artist["match"].(string); ok {
				if match != "0" {
					t.Errorf("A2 match = %q, want %q", match, "0")
				}
			} else {
				t.Errorf("A2 match is not a string: %T", artist["match"])
			}
		}
	}
}

func TestHandleUserGetTopArtists_Valid(t *testing.T) {
	mock := &mockClient{
		userGetTopArtistsFn: func(ctx context.Context, user, period string, limit, page int) ([]lastfm.TopArtist, error) {
			return []lastfm.TopArtist{
				{Name: "Radiohead", Playcount: "500"},
			}, nil
		},
	}
	s := newTestServer(mock)
	req := httptest.NewRequest("GET", "/api/user/top-artists?user=john&period=7day", nil)
	w := httptest.NewRecorder()

	s.handleUserGetTopArtists(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	artists := data["artists"].([]interface{})
	if len(artists) != 1 {
		t.Errorf("got %d artists, want 1", len(artists))
	}
}

func TestHandleUserGetTopArtists_InvalidUser(t *testing.T) {
	s := newTestServer(nil)
	req := httptest.NewRequest("GET", "/api/user/top-artists?user=bad%0Aname&period=7day", nil)
	w := httptest.NewRecorder()

	s.handleUserGetTopArtists(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleUserGetTopArtists_InvalidPeriod(t *testing.T) {
	s := newTestServer(nil)
	req := httptest.NewRequest("GET", "/api/user/top-artists?user=john&period=2month", nil)
	w := httptest.NewRecorder()

	s.handleUserGetTopArtists(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleUserGetTopArtists_EmptyUser(t *testing.T) {
	mock := &mockClient{
		userGetTopArtistsFn: func(ctx context.Context, user, period string, limit, page int) ([]lastfm.TopArtist, error) {
			return nil, fmt.Errorf("user must be provided")
		},
	}
	s := newTestServer(mock)
	req := httptest.NewRequest("GET", "/api/user/top-artists?user=&period=overall", nil)
	w := httptest.NewRecorder()

	s.handleUserGetTopArtists(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleUserGetTopArtists_ClientError(t *testing.T) {
	mock := &mockClient{
		userGetTopArtistsFn: func(ctx context.Context, user, period string, limit, page int) ([]lastfm.TopArtist, error) {
			return nil, fmt.Errorf("timeout")
		},
	}
	s := newTestServer(mock)
	req := httptest.NewRequest("GET", "/api/user/top-artists?user=john&period=overall", nil)
	w := httptest.NewRecorder()

	s.handleUserGetTopArtists(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
