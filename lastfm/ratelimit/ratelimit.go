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
	sleepDuration := l.interval - time.Since(l.lastReq)
	if sleepDuration < 0 {
		sleepDuration = 0
	}
	l.lastReq = time.Now().Add(sleepDuration)
	l.mu.Unlock()

	if sleepDuration <= 0 {
		return nil
	}

	timer := time.NewTimer(sleepDuration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
