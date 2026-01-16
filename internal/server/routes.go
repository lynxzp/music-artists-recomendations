package server

import "net/http"

func (s *Server) registerRoutes() {
	http.HandleFunc("GET /", s.handleIndex)
	http.HandleFunc("GET /api/artist/similar", s.handleArtistGetSimilar)
	http.HandleFunc("POST /api/append", s.handleAppendSimilarArtists)
	http.HandleFunc("GET /api/user/top-artists", s.handleUserGetTopArtists)
}
