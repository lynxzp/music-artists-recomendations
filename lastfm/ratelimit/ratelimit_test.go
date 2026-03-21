package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	l := New(100 * time.Millisecond)
	if l == nil {
		t.Fatal("New returned nil")
	}
	if l.interval != 100*time.Millisecond {
		t.Errorf("interval = %v, want %v", l.interval, 100*time.Millisecond)
	}
}

func TestNewZeroInterval(t *testing.T) {
	l := New(0)
	start := time.Now()
	l.Wait()
	l.Wait()
	elapsed := time.Since(start)
	if elapsed > 50*time.Millisecond {
		t.Errorf("zero-interval Wait took %v, expected near-instant", elapsed)
	}
}

func TestWaitEnforcesInterval(t *testing.T) {
	interval := 100 * time.Millisecond
	l := New(interval)

	l.Wait()

	start := time.Now()
	l.Wait()
	elapsed := time.Since(start)

	if elapsed < interval-10*time.Millisecond {
		t.Errorf("second Wait took %v, expected at least %v", elapsed, interval)
	}
}

func TestWaitConcurrentSafe(t *testing.T) {
	l := New(1 * time.Millisecond)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.Wait()
		}()
	}
	wg.Wait()
}
