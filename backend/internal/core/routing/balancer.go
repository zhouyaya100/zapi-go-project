package routing

import (
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
)

// candidatePool — reuse []*ChannelInfo slices to reduce GC pressure on hot path
var candidatePool = sync.Pool{
	New: func() interface{} { s := make([]*ChannelInfo, 0, 8); return &s },
}

func getCandidates() *[]*ChannelInfo { return candidatePool.Get().(*[]*ChannelInfo) }
func putCandidates(s *[]*ChannelInfo) { *s = (*s)[:0]; candidatePool.Put(s) }

// Balancer — selects a channel from an upstream group based on strategy
type Balancer struct {
	rrCounters   sync.Map      // group ID (uint) → *atomic.Int64 (per-group round-robin counter)
	reqCounters  sync.Map      // channel ID (uint) → *atomic.Int64 (thread-safe)
}

var LB = &Balancer{}

// SelectFromGroup — pick a channel from the group's channels using the group's strategy
// candidates: pre-filtered (enabled, not circuit-broken, group-permission-checked)
func (b *Balancer) SelectFromGroup(ug *UpstreamGroupInfo, candidates []*ChannelInfo, excludeIDs map[uint]bool) *ChannelInfo {
	if len(candidates) == 0 {
		return nil
	}
	// Filter out excludeIDs using pooled slice
	eligible := getCandidates()
	defer putCandidates(eligible)
	for _, ch := range candidates {
		if excludeIDs != nil && excludeIDs[ch.ID] {
			continue
		}
		*eligible = append(*eligible, ch)
	}
	if len(*eligible) == 0 {
		return nil
	}

	switch ug.Strategy {
	case "round_robin":
		return b.roundRobin(ug.ID, *eligible)
	case "weighted":
		return b.weightedRoundRobin(*eligible)
	case "least_latency":
		return b.leastLatency(*eligible)
	case "least_requests":
		return b.leastRequests(*eligible)
	default: // "priority"
		return b.priorityWeighted(*eligible)
	}
}

// priorityWeighted — same as current pool.go Select(): sort by priority desc, then weight-random within same priority
func (b *Balancer) priorityWeighted(candidates []*ChannelInfo) *ChannelInfo {
	// Sort by priority desc
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority > candidates[j].Priority
	})
	// Pick from highest priority group
	topPri := candidates[0].Priority
	var top []*ChannelInfo
	for _, ch := range candidates {
		if ch.Priority == topPri {
			top = append(top, ch)
		}
	}
	return weightedRandom(top)
}

// roundRobin — per-group atomic counter modulo, lock-free
func (b *Balancer) roundRobin(groupID uint, candidates []*ChannelInfo) *ChannelInfo {
	v, _ := b.rrCounters.LoadOrStore(groupID, &atomic.Int64{})
	n := v.(*atomic.Int64).Add(1)
	idx := int(n-1) % len(candidates)
	if idx < 0 {
		idx = 0
	}
	return candidates[idx]
}

// weightedRoundRobin — weighted random selection (weight-proportional)
func (b *Balancer) weightedRoundRobin(candidates []*ChannelInfo) *ChannelInfo {
	return weightedRandom(candidates)
}

// leastLatency — pick the channel with lowest average latency
func (b *Balancer) leastLatency(candidates []*ChannelInfo) *ChannelInfo {
	best := candidates[0]
	_, _, bestLatency, _, _ := Health.GetStats(best.ID)
	for _, ch := range candidates[1:] {
		_, _, lat, _, _ := Health.GetStats(ch.ID)
		if lat < bestLatency {
			best = ch
			bestLatency = lat
		} else if lat == bestLatency && ch.Weight > best.Weight {
			best = ch
		}
	}
	return best
}

// leastRequests — pick the channel with fewest active requests
func (b *Balancer) leastRequests(candidates []*ChannelInfo) *ChannelInfo {
	best := candidates[0]
	bestCount := b.getRequestCount(best.ID)
	for _, ch := range candidates[1:] {
		cnt := b.getRequestCount(ch.ID)
		if cnt < bestCount {
			best = ch
			bestCount = cnt
		}
	}
	return best
}

// weightedRandom — pick one candidate proportionally by weight
func weightedRandom(candidates []*ChannelInfo) *ChannelInfo {
	weights := make([]int, len(candidates))
	total := 0
	for i, c := range candidates {
		w := c.Weight
		if w < 1 {
			w = 1
		}
		weights[i] = w
		total += w
	}
	r := rand.Intn(total)
	cum := 0
	for i, w := range weights {
		cum += w
		if r < cum {
			return candidates[i]
		}
	}
	return candidates[0]
}

// getRequestCount — get active request count for a channel (atomic read, thread-safe)
func (b *Balancer) getRequestCount(id uint) int64 {
	v, _ := b.reqCounters.LoadOrStore(id, &atomic.Int64{})
	return v.(*atomic.Int64).Load()
}

// IncrRequest — increment active request count (call before proxying, thread-safe)
func (b *Balancer) IncrRequest(id uint) {
	v, _ := b.reqCounters.LoadOrStore(id, &atomic.Int64{})
	v.(*atomic.Int64).Add(1)
}

// DecrRequest — decrement active request count (call after proxying, thread-safe)
func (b *Balancer) DecrRequest(id uint) {
	v, _ := b.reqCounters.LoadOrStore(id, &atomic.Int64{})
	v.(*atomic.Int64).Add(-1)
}
