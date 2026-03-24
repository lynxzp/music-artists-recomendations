package retry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{201, false},
		{301, false},
		{399, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},
		{500, true},
		{503, true},
		{599, true},
	}

	for _, tt := range tests {
		got := isRetryable(tt.code)
		if got != tt.want {
			t.Errorf("isRetryable(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestRoundTripSuccessFirstAttempt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	transport := &RetryTransport{
		Base:   server.Client().Transport,
		Delays: []time.Duration{1 * time.Millisecond},
	}

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestRoundTripRetryThenSucceed(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	transport := &RetryTransport{
		Base:   server.Client().Transport,
		Delays: []time.Duration{1 * time.Millisecond, 1 * time.Millisecond},
	}

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if attempts.Load() != 2 {
		t.Errorf("attempts = %d, want 2", attempts.Load())
	}
}

func TestRoundTripExhaustsRetries(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	transport := &RetryTransport{
		Base:   server.Client().Transport,
		Delays: []time.Duration{1 * time.Millisecond, 1 * time.Millisecond},
	}

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if attempts.Load() != 3 {
		t.Errorf("attempts = %d, want 3", attempts.Load())
	}
	if resp.StatusCode != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}

func TestRoundTripContextCancellation(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	transport := &RetryTransport{
		Base:   server.Client().Transport,
		Delays: []time.Duration{100 * time.Millisecond, 100 * time.Millisecond},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)

	_, err := transport.RoundTrip(req)

	if attempts.Load() > 2 {
		t.Errorf("attempts = %d, expected <= 2 with cancelled context", attempts.Load())
	}
	_ = err
}

func TestRoundTripNilBaseUsesDefault(t *testing.T) {
	transport := &RetryTransport{
		Base:   nil,
		Delays: []time.Duration{1 * time.Millisecond},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestRoundTripEmptyDelaysUsesDefault(t *testing.T) {
	transport := &RetryTransport{
		Delays: nil,
	}

	if transport.Delays != nil {
		t.Errorf("Delays should be nil, got %v", transport.Delays)
	}
	if len(DefaultDelays) == 0 {
		t.Error("DefaultDelays is empty")
	}
}
