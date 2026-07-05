package server

import (
	"encoding/json"
	"music-recomendations/lastfm"
	"net/http"
	"strings"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		s.logger.Error("failed to read index.html", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write(data); err != nil {
		s.logger.Error("failed to write index response", "error", err)
	}
}

func (s *Server) handleArtistGetSimilar(w http.ResponseWriter, r *http.Request) {
	artist := r.URL.Query().Get("artist")

	if !isValidArtistName(artist) {
		s.logger.Warn("rejected invalid artist parameter", "artist", artist)
		http.Error(w, "invalid artist parameter", http.StatusBadRequest)
		return
	}

	artists, err := s.client.ArtistGetSimilar(r.Context(), artist, "", s.config.SimilarArtistsLimit, true)
	if err != nil {
		s.logger.Error("failed to get similar artists", "artist", artist, "error", err)
		http.Error(w, "failed to fetch similar artists", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data": map[string]interface{}{
			"artists": artists,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("failed to encode similar artists response", "artist", artist, "error", err)
	}
}

func (s *Server) handleArtistGetInfo(w http.ResponseWriter, r *http.Request) {
	artist := r.URL.Query().Get("artist")
	user := r.URL.Query().Get("user")

	if !isValidArtistName(artist) {
		s.logger.Warn("rejected invalid artist parameter", "artist", artist)
		http.Error(w, "invalid artist parameter", http.StatusBadRequest)
		return
	}
	if user != "" && !isValidUsername(user) {
		s.logger.Warn("rejected invalid user parameter", "user", user)
		http.Error(w, "invalid user parameter", http.StatusBadRequest)
		return
	}

	info, err := s.client.ArtistGetInfo(r.Context(), artist, "", user, true)
	if err != nil {
		s.logger.Error("failed to get artist info", "artist", artist, "error", err)
		http.Error(w, "failed to fetch artist info", http.StatusInternalServerError)
		return
	}

	// Lowercase tag names before sending to frontend
	for i := range info.Tags.Tag {
		info.Tags.Tag[i].Name = strings.ToLower(info.Tags.Tag[i].Name)
	}

	response := map[string]interface{}{
		"data": map[string]interface{}{
			"artist": info,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("failed to encode artist info response", "artist", artist, "error", err)
	}
}

type appendRequest struct {
	A      []lastfm.SimilarArtist `json:"a"`
	B      []lastfm.SimilarArtist `json:"b"`
	Weight float64                `json:"weight"`
}

func (s *Server) handleAppendSimilarArtists(w http.ResponseWriter, r *http.Request) {
	var req appendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn("rejected invalid append request body", "error", err)
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	result := lastfm.AppendSimilarArtists(req.A, req.B, req.Weight)

	response := map[string]interface{}{
		"data": map[string]interface{}{
			"artists": result,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("failed to encode append response", "error", err)
	}
}

func (s *Server) handleUserGetTopArtists(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	period := r.URL.Query().Get("period")

	if !isValidUsername(user) {
		s.logger.Warn("rejected invalid user parameter", "user", user)
		http.Error(w, "invalid user parameter", http.StatusBadRequest)
		return
	}
	if !isValidPeriod(period) {
		s.logger.Warn("rejected invalid period parameter", "user", user, "period", period)
		http.Error(w, "invalid period parameter", http.StatusBadRequest)
		return
	}

	artists, err := s.client.UserGetTopArtists(r.Context(), user, period, s.config.TopArtistsLimit, 0)
	if err != nil {
		s.logger.Error("failed to get top artists", "user", user, "period", period, "error", err)
		http.Error(w, "failed to fetch top artists", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data": map[string]interface{}{
			"artists": artists,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("failed to encode top artists response", "user", user, "period", period, "error", err)
	}
}
