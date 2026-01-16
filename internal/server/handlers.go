package server

import (
	"encoding/json"
	"music-recomendations/lastfm"
	"net/http"
	"strconv"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	html := indexHTML(s.config.SimilarArtistsLimit, s.config.TopArtistsLimit)
	w.Write([]byte(html))
}

func (s *Server) handleArtistGetSimilar(w http.ResponseWriter, r *http.Request) {
	artist := r.URL.Query().Get("artist")
	mbid := r.URL.Query().Get("mbid")
	limitStr := r.URL.Query().Get("limit")
	autocorrectStr := r.URL.Query().Get("autocorrect")

	var limit int
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid limit parameter", http.StatusBadRequest)
			return
		}
	}

	autocorrect := autocorrectStr == "true" || autocorrectStr == "1"

	artists, err := s.client.ArtistGetSimilar(artist, mbid, limit, autocorrect)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	limitStr := r.URL.Query().Get("limit")
	pageStr := r.URL.Query().Get("page")

	var limit int
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid limit parameter", http.StatusBadRequest)
			return
		}
	}

	var page int
	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			http.Error(w, "invalid page parameter", http.StatusBadRequest)
			return
		}
	}

	artists, err := s.client.UserGetTopArtists(user, period, limit, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
