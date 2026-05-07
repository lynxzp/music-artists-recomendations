package retry

import (
	"context"
	"music-recomendations/lastfm/ratelimit"
	"net/http"
	"time"
)

// DefaultDelays provides exponential backoff delays for retries: 500ms, 1s.
var DefaultDelays = []time.Duration{
	500 * time.Millisecond,
	1 * time.Second,
}

// RetryTransport wraps an http.RoundTripper and adds retry logic with configurable delays.
type RetryTransport struct {
	Base    http.RoundTripper
	Delays  []time.Duration
	Limiter *ratelimit.Limiter
}

// RoundTrip executes the HTTP request with retry logic for 429 and 5xx status codes.
func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	delays := t.Delays
	if len(delays) == 0 {
		delays = DefaultDelays
	}

	var resp *http.Response
	var err error

	// wait applies the global rate limiter if configured.
	wait := func(ctx context.Context) error {
		if t.Limiter == nil {
			return nil
		}
		return t.Limiter.Wait(ctx)
	}

	// First attempt
	if err := wait(req.Context()); err != nil {
		return nil, err
	}
	resp, err = base.RoundTrip(req)
	if err == nil && !isRetryable(resp.StatusCode) {
		return resp, nil
	}

	// Retry attempts
	for _, delay := range delays {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		select {
		case <-time.After(delay):
		case <-req.Context().Done():
			return resp, req.Context().Err()
		}

		// Clone request for retry (Last.fm API uses GET only, no body to handle)
		retryReq, cloneErr := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), nil)
		if cloneErr != nil {
			return resp, err
		}
		retryReq.Header = req.Header.Clone()

		if err := wait(req.Context()); err != nil {
			return nil, err
		}
		resp, err = base.RoundTrip(retryReq)
		if err == nil && !isRetryable(resp.StatusCode) {
			return resp, nil
		}
	}

	return resp, err
}

// isRetryable returns true for status codes that should trigger a retry (429 and 5xx).
func isRetryable(statusCode int) bool {
	return statusCode == 429 || (statusCode >= 500 && statusCode <= 599)
}
