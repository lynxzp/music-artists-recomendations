package server

import (
	"encoding/json"
	"music-recomendations/lastfm"
	"net/http"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	html := indexHTML()
	w.Write([]byte(html))
}

func (s *Server) handleArtistGetSimilar(w http.ResponseWriter, r *http.Request) {
	artist := r.URL.Query().Get("artist")

	if !isValidArtistName(artist) {
		http.Error(w, "invalid artist parameter", http.StatusBadRequest)
		return
	}

	artists, err := s.client.ArtistGetSimilar(artist, "", s.config.SimilarArtistsLimit, true)
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
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleArtistGetInfo(w http.ResponseWriter, r *http.Request) {
	artist := r.URL.Query().Get("artist")

	if !isValidArtistName(artist) {
		http.Error(w, "invalid artist parameter", http.StatusBadRequest)
		return
	}

	info, err := s.client.ArtistGetInfo(artist, "", true)
	if err != nil {
		s.logger.Error("failed to get artist info", "artist", artist, "error", err)
		http.Error(w, "failed to fetch artist info", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data": map[string]interface{}{
			"artist": info,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type appendRequest struct {
	A      []lastfm.SimilarArtist `json:"a"`
	B      []lastfm.SimilarArtist `json:"b"`
	Weight float64                `json:"weight"`
}

func (s *Server) handleAppendSimilarArtists(w http.ResponseWriter, r *http.Request) {
	var req appendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleUserGetTopArtists(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	period := r.URL.Query().Get("period")

	if !isValidUsername(user) {
		http.Error(w, "invalid user parameter", http.StatusBadRequest)
		return
	}
	if !isValidPeriod(period) {
		http.Error(w, "invalid period parameter", http.StatusBadRequest)
		return
	}

	artists, err := s.client.UserGetTopArtists(user, period, s.config.TopArtistsLimit, 0)
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
	json.NewEncoder(w).Encode(response)
}
