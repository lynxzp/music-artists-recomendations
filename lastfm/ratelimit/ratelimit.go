package ratelimit

import (
	"context"
	"sync"
	"time"
)

type Limiter struct {
	mu       sync.Mutex
	lastReq  time.Time
	interval time.Duration
}

func New(interval time.Duration) *Limiter {
	return &Limiter{
		interval: interval,
		lastReq:  time.Now(),
	}
}

func (l *Limiter) Wait(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	sleepDuration := l.interval - time.Since(l.lastReq)
	if sleepDuration > 0 {
		select {
		case <-time.After(sleepDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	l.lastReq = time.Now()
	return nil
}
