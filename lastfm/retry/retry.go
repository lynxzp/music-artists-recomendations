package retry

import (
	"net/http"
	"time"
)

// DefaultDelays provides exponential backoff delays: 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 256s, 512s, 1024s
var DefaultDelays = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
	32 * time.Second,
	64 * time.Second,
	128 * time.Second,
	256 * time.Second,
	512 * time.Second,
	1024 * time.Second,
}

// RetryTransport wraps an http.RoundTripper and adds retry logic with configurable delays.
type RetryTransport struct {
	Base   http.RoundTripper
	Delays []time.Duration
}

// RoundTrip executes the HTTP request with retry logic for 4xx and 5xx status codes.
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

	// First attempt
	resp, err = base.RoundTrip(req)
	if err == nil && !isRetryable(resp.StatusCode) {
		return resp, nil
	}

	// Retry attempts
	for _, delay := range delays {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		time.Sleep(delay)

		// Clone request for retry (Last.fm API uses GET only, no body to handle)
		retryReq, cloneErr := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), nil)
		if cloneErr != nil {
			return resp, err
		}
		retryReq.Header = req.Header.Clone()

		resp, err = base.RoundTrip(retryReq)
		if err == nil && !isRetryable(resp.StatusCode) {
			return resp, nil
		}
	}

	return resp, err
}

// isRetryable returns true for status codes that should trigger a retry (4xx and 5xx).
func isRetryable(statusCode int) bool {
	return statusCode >= 400 && statusCode <= 599
}
