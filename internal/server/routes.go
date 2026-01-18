package server

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /api/artist/similar", s.handleArtistGetSimilar)
	mux.HandleFunc("GET /api/artist/info", s.handleArtistGetInfo)
	mux.HandleFunc("POST /api/append", s.handleAppendSimilarArtists)
	mux.HandleFunc("GET /api/user/top-artists", s.handleUserGetTopArtists)
}
