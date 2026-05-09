package routing

import (
	"sync"
	"testing"
)

func TestIncrDecrRequest(t *testing.T) {
	b := &Balancer{}

	// Basic: Incr then Decr should be 0
	b.IncrRequest(1)
	if got := b.getRequestCount(1); got != 1 {
		t.Errorf("after Incr: got %d, want 1", got)
	}
	b.DecrRequest(1)
	if got := b.getRequestCount(1); got != 0 {
		t.Errorf("after Decr: got %d, want 0", got)
	}

	// Multiple increments
	b.IncrRequest(1)
	b.IncrRequest(1)
	b.IncrRequest(1)
	if got := b.getRequestCount(1); got != 3 {
		t.Errorf("after 3 Incr: got %d, want 3", got)
	}
	b.DecrRequest(1)
	if got := b.getRequestCount(1); got != 2 {
		t.Errorf("after Decr: got %d, want 2", got)
	}
	b.DecrRequest(1)
	b.DecrRequest(1)
	if got := b.getRequestCount(1); got != 0 {
		t.Errorf("after all Decr: got %d, want 0", got)
	}

	// Different channel IDs are independent
	b.IncrRequest(1)
	b.IncrRequest(2)
	b.IncrRequest(2)
	if got := b.getRequestCount(1); got != 1 {
		t.Errorf("channel 1: got %d, want 1", got)
	}
	if got := b.getRequestCount(2); got != 2 {
		t.Errorf("channel 2: got %d, want 2", got)
	}

	// Non-existent channel returns 0
	if got := b.getRequestCount(999); got != 0 {
		t.Errorf("non-existent channel: got %d, want 0", got)
	}
}

func TestIncrDecrConcurrent(t *testing.T) {
	b := &Balancer{}
	const goroutines = 100
	const opsPerGoroutine = 1000

	var wg sync.WaitGroup
	// Half goroutines do Incr, half do Decr — net should be 0
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				b.IncrRequest(1)
			}
		}(i)
	}
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				b.DecrRequest(1)
			}
		}(i)
	}
	wg.Wait()

	got := b.getRequestCount(1)
	if got != 0 {
		t.Errorf("after %d Incr + %d Decr: got %d, want 0", goroutines*opsPerGoroutine, goroutines*opsPerGoroutine, got)
	}
}

func TestLeastRequestsStrategy(t *testing.T) {
	b := &Balancer{}
	ch1 := &ChannelInfo{ID: 1, Weight: 1}
	ch2 := &ChannelInfo{ID: 2, Weight: 1}
	ch3 := &ChannelInfo{ID: 3, Weight: 1}
	candidates := []*ChannelInfo{ch1, ch2, ch3}

	// All zero — should pick first
	sel := b.leastRequests(candidates)
	if sel.ID != 1 {
		t.Errorf("all zero: picked %d, want 1", sel.ID)
	}

	// ch1 has 5 requests, ch2 has 0, ch3 has 3
	for i := 0; i < 5; i++ {
		b.IncrRequest(1)
	}
	for i := 0; i < 3; i++ {
		b.IncrRequest(3)
	}
	sel = b.leastRequests(candidates)
	if sel.ID != 2 {
		t.Errorf("ch1=5 ch2=0 ch3=3: picked %d, want 2", sel.ID)
	}

	// ch2 gets a request, now ch2 and ch3 tied — should pick first of tied (ch2)
	b.IncrRequest(2)
	sel = b.leastRequests(candidates)
	// ch2=1, ch3=3, ch1=5 → ch2 wins
	if sel.ID != 2 {
		t.Errorf("ch1=5 ch2=1 ch3=3: picked %d, want 2", sel.ID)
	}

	// Cleanup
	for i := 0; i < 5; i++ {
		b.DecrRequest(1)
	}
	b.DecrRequest(2)
	for i := 0; i < 3; i++ {
		b.DecrRequest(3)
	}
}
