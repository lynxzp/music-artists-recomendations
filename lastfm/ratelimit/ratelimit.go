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
		lastReq:  time.Now(),
	}
}

func (l *Limiter) Wait() {
	l.mu.Lock()
	sleepDuration := l.interval - time.Since(l.lastReq)
	if sleepDuration < 0 {
		sleepDuration = 0
	}
	l.lastReq = time.Now().Add(sleepDuration)
	l.mu.Unlock()

	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}
}
