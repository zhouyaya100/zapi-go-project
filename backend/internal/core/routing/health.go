package routing

import (
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState — circuit breaker states
type CircuitState int32

const (
	CircuitClosed   CircuitState = iota // normal
	CircuitOpen                         // tripped, all requests blocked
	CircuitHalfOpen                     // allowing one probe request
)

// GroupHealth — per-upstream-group health tracking
type GroupHealth struct {
	TotalReqs   atomic.Int64 // total requests routed through this group
	TotalFails  atomic.Int64 // total failed requests in this group
	TotalReqsCh atomic.Int64 // total requests for a specific channel in this group
}

// ChannelHealth — per-channel health tracking
type ChannelHealth struct {
	FailCount     atomic.Int64 // consecutive fail count
	TotalReqs     atomic.Int64 // total requests (global, across all groups)
	TotalFails    atomic.Int64 // total failed requests (global)
	TotalLatency  atomic.Int64 // cumulative latency ms (for average)
	CircuitState  atomic.Int32 // CircuitState as int32
	LastFailTime  atomic.Int64 // unix timestamp of last failure
	ActiveReqs    atomic.Int64 // currently in-flight requests

	// Per-group tracking: groupID → per-group stats for this channel
	groupMu    sync.RWMutex
	groupStats map[uint]*channelGroupStats
}

// channelGroupStats — per (channel, upstream-group) stats
type channelGroupStats struct {
	TotalReqs  atomic.Int64
	TotalFails atomic.Int64
}

// getGroupStats — get or create per-group stats for this channel
func (ch *ChannelHealth) getGroupStats(groupID uint) *channelGroupStats {
	ch.groupMu.RLock()
	s, ok := ch.groupStats[groupID]
	ch.groupMu.RUnlock()
	if ok {
		return s
	}
	ch.groupMu.Lock()
	defer ch.groupMu.Unlock()
	s, ok = ch.groupStats[groupID]
	if !ok {
		s = &channelGroupStats{}
		ch.groupStats[groupID] = s
	}
	return s
}

// HealthTracker — manages health state for all channels
type HealthTracker struct {
	mu      sync.RWMutex
	healths map[uint]*ChannelHealth // channel ID → health
	// Per-group request counters
	groupMu    sync.RWMutex
	groupStats map[uint]*GroupHealth // group ID → group health
}

var Health = &HealthTracker{
	healths:    make(map[uint]*ChannelHealth),
	groupStats: make(map[uint]*GroupHealth),
}

// Get — get or create health entry for a channel
func (h *HealthTracker) Get(id uint) *ChannelHealth {
	h.mu.RLock()
	ch, ok := h.healths[id]
	h.mu.RUnlock()
	if ok {
		return ch
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok = h.healths[id]
	if !ok {
		ch = &ChannelHealth{groupStats: make(map[uint]*channelGroupStats)}
		h.healths[id] = ch
	}
	return ch
}

// getGroupHealth — get or create group health entry
func (h *HealthTracker) getGroupHealth(groupID uint) *GroupHealth {
	if groupID == 0 {
		return nil
	}
	h.groupMu.RLock()
	gh, ok := h.groupStats[groupID]
	h.groupMu.RUnlock()
	if ok {
		return gh
	}
	h.groupMu.Lock()
	defer h.groupMu.Unlock()
	gh, ok = h.groupStats[groupID]
	if !ok {
		gh = &GroupHealth{}
		h.groupStats[groupID] = gh
	}
	return gh
}

// RecordSuccess — called after a successful proxy request
// groupID=0 means no upstream group (normal pool selection)
func (h *HealthTracker) RecordSuccess(id uint, latencyMs int, groupID ...uint) {
	ch := h.Get(id)
	ch.TotalReqs.Add(1)
	ch.TotalLatency.Add(int64(latencyMs))
	ch.FailCount.Store(0)
	// If half-open and success → close circuit
	ch.CircuitState.CompareAndSwap(int32(CircuitHalfOpen), int32(CircuitClosed))

	// Per-group tracking
	if len(groupID) > 0 && groupID[0] > 0 {
		gs := ch.getGroupStats(groupID[0])
		gs.TotalReqs.Add(1)
		gh := h.getGroupHealth(groupID[0])
		if gh != nil {
			gh.TotalReqs.Add(1)
		}
	}
}

// RecordFailure — called after a failed proxy request (real failure: connection refused, DNS, 5xx)
func (h *HealthTracker) RecordFailure(id uint, maxFails int, failTimeout int, groupID ...uint) CircuitState {
	ch := h.Get(id)
	ch.TotalReqs.Add(1)
	ch.TotalFails.Add(1)
	failCount := ch.FailCount.Add(1)
	ch.LastFailTime.Store(time.Now().Unix())

	// Per-group tracking
	if len(groupID) > 0 && groupID[0] > 0 {
		gs := ch.getGroupStats(groupID[0])
		gs.TotalReqs.Add(1)
		gs.TotalFails.Add(1)
		gh := h.getGroupHealth(groupID[0])
		if gh != nil {
			gh.TotalReqs.Add(1)
			gh.TotalFails.Add(1)
		}
	}

	// If half-open and failure → re-open circuit
	if ch.CircuitState.CompareAndSwap(int32(CircuitHalfOpen), int32(CircuitOpen)) {
		return CircuitOpen
	}

	// Check if we should trip the circuit
	if maxFails > 0 && int(failCount) >= maxFails {
		if ch.CircuitState.CompareAndSwap(int32(CircuitClosed), int32(CircuitOpen)) {
			return CircuitOpen
		}
	}
	return CircuitState(ch.CircuitState.Load())
}

// IsAvailable — check if a channel should receive requests
func (h *HealthTracker) IsAvailable(id uint, maxFails int, failTimeout int) bool {
	ch := h.Get(id)
	state := CircuitState(ch.CircuitState.Load())

	switch state {
	case CircuitClosed:
		return true
	case CircuitHalfOpen:
		return true // allow one probe
	case CircuitOpen:
		lastFail := ch.LastFailTime.Load()
		if lastFail == 0 {
			return true
		}
		elapsed := time.Now().Unix() - lastFail
		if int(elapsed) >= failTimeout {
			if ch.CircuitState.CompareAndSwap(int32(CircuitOpen), int32(CircuitHalfOpen)) {
				return true
			}
			return false
		}
		return false
	}
	return true
}

// ResetCircuit — manually reset circuit breaker for a channel
func (h *HealthTracker) ResetCircuit(id uint) {
	ch := h.Get(id)
	ch.CircuitState.Store(int32(CircuitClosed))
	ch.FailCount.Store(0)
}

// SyncFromHeartbeat — sync heartbeat failure state into the circuit breaker.
// Unlike RecordFailure, this does NOT increment FailCount — it sets it directly
// from the DB value and trips the circuit if needed. This prevents double-counting
// since heartbeat.go already incremented the DB FailCount before calling this.
func (h *HealthTracker) SyncFromHeartbeat(id uint, failCount int, maxFails int, failTimeout int) {
	ch := h.Get(id)
	ch.FailCount.Store(int64(failCount))
	if failCount > 0 {
		ch.LastFailTime.Store(time.Now().Unix())
	}
	// Trip circuit breaker if fail count reaches threshold
	state := CircuitState(ch.CircuitState.Load())
	if state == CircuitClosed && maxFails > 0 && failCount >= maxFails {
		ch.CircuitState.CompareAndSwap(int32(CircuitClosed), int32(CircuitOpen))
	}
	// If half-open and still failing, re-open
	if state == CircuitHalfOpen && failCount > 0 {
		ch.CircuitState.CompareAndSwap(int32(CircuitHalfOpen), int32(CircuitOpen))
	}
}

// GetStats — get global health stats for a channel
func (h *HealthTracker) GetStats(id uint) (totalReqs, totalFails, avgLatency int64, failCount int64, state CircuitState) {
	ch := h.Get(id)
	totalReqs = ch.TotalReqs.Load()
	totalFails = ch.TotalFails.Load()
	latSum := ch.TotalLatency.Load()
	if totalReqs > 0 {
		avgLatency = latSum / totalReqs
	}
	failCount = ch.FailCount.Load()
	state = CircuitState(ch.CircuitState.Load())
	return
}

// GetGroupChannelStats — get per-group stats for a specific channel
// Returns (totalReqs, totalFails) for the channel within the specified group
func (h *HealthTracker) GetGroupChannelStats(channelID uint, groupID uint) (totalReqs, totalFails int64) {
	ch := h.Get(channelID)
	ch.groupMu.RLock()
	s, ok := ch.groupStats[groupID]
	ch.groupMu.RUnlock()
	if !ok {
		return 0, 0
	}
	return s.TotalReqs.Load(), s.TotalFails.Load()
}

// GetGroupStats — get aggregate stats for a group
func (h *HealthTracker) GetGroupStats(groupID uint) (totalReqs, totalFails int64) {
	gh := h.getGroupHealth(groupID)
	if gh == nil {
		return 0, 0
	}
	return gh.TotalReqs.Load(), gh.TotalFails.Load()
}

// CircuitStateString — human-readable circuit state
func CircuitStateString(s CircuitState) string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// GetAllStats — get all channel health stats (for LB status API)
func (h *HealthTracker) GetAllStats() map[uint]*ChannelHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make(map[uint]*ChannelHealth, len(h.healths))
	for k, v := range h.healths {
		result[k] = v
	}
	return result
}
