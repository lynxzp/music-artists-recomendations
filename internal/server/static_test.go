package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestHandleIndex_ServesHTML(t *testing.T) {
	s := &Server{logger: testLogger(t)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	s.handleIndex(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html", rec.Header().Get("Content-Type"))
	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), "<!DOCTYPE html>")
}

func TestStaticFiles_ServeCSS(t *testing.T) {
	s, err := New(Config{APIKey: "test", Logger: testLogger(t)})
	require.NoError(t, err)

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	ct := rec.Header().Get("Content-Type")
	assert.Contains(t, ct, "text/css")
}
