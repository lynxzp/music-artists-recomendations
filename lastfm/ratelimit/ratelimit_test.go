package ratelimit

import (
	"context"
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
	if err := l.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := l.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed > 50*time.Millisecond {
		t.Errorf("zero-interval Wait took %v, expected near-instant", elapsed)
	}
}

func TestWaitEnforcesInterval(t *testing.T) {
	interval := 100 * time.Millisecond
	l := New(interval)

	if err := l.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	if err := l.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
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
			if err := l.Wait(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestWaitCancelledSlotNotFreed(t *testing.T) {
	interval := 200 * time.Millisecond
	l := New(interval)

	// Establish baseline.
	if err := l.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Second caller reserves a slot, then cancels.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = l.Wait(ctx)

	// Third caller should still wait the full interval from the reserved slot,
	// not from the rolled-back time.
	start := time.Now()
	if err := l.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)

	if elapsed < interval-20*time.Millisecond {
		t.Errorf("Wait() took %v after cancelled Wait, expected at least ~%v (slot should not be freed)", elapsed, interval)
	}
}

func TestWaitContextCancelled(t *testing.T) {
	l := New(5 * time.Second)
	if err := l.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := l.Wait(ctx)
	elapsed := time.Since(start)

	if err != context.Canceled {
		t.Errorf("Wait() error = %v, want context.Canceled", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("Wait() took %v, expected prompt return after cancel", elapsed)
	}
}
