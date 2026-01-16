package ratelimit

import (
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
	}
}

func (l *Limiter) Wait() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lastReq.IsZero() {
		l.lastReq = time.Now()
		return
	}

	elapsed := time.Since(l.lastReq)
	if elapsed < l.interval {
		time.Sleep(l.interval - elapsed)
	}
	l.lastReq = time.Now()
}
