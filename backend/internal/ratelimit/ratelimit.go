package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

type bucket struct {
	count   int
	tokens  int64 // for TPM tracking
	resetAt time.Time
}

type shard struct {
	mu      sync.Mutex
	entries map[string]*bucket
}

type RateLimiter struct {
	shards [64]shard
}

var Limiter *RateLimiter

func Init() {
	Limiter = &RateLimiter{}
	for i := range Limiter.shards {
		Limiter.shards[i].entries = make(map[string]*bucket)
	}
	go func() {
		for {
			time.Sleep(time.Minute)
			now := time.Now()
			for i := range Limiter.shards {
				s := &Limiter.shards[i]
				s.mu.Lock()
				for k, v := range s.entries {
					if now.After(v.resetAt) {
						delete(s.entries, k)
					}
				}
				s.mu.Unlock()
			}
		}
	}()
}

func fnv32(key string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return h
}

func (r *RateLimiter) getShard(key string) *shard {
	return &r.shards[fnv32(key)%64]
}

// UserRateInfo carries per-user rate limit settings (already resolved from group/user hierarchy)
type UserRateInfo struct {
	UserID uint
	RPM    int   // 0 = blocked, -1 = unlimited, >0 = specific limit
	TPM    int64 // 0 = blocked, -1 = unlimited, >0 = specific limit
}

// Check enforces RPM limit + TPM pre-check (accumulated tokens so far).
// Call AccountTokens after the request completes to record actual usage.
// Value semantics: 0 = blocked (reject immediately), -1 = unlimited (skip check), >0 = limit.
func (r *RateLimiter) Check(apiKey string, userRate *UserRateInfo, promptTokens int64) string {
	if r == nil {
		return ""
	}
	now := time.Now()

	// RPM check: 0 = blocked, -1 = unlimited, >0 = limit
	if userRate != nil {
		if userRate.RPM == 0 {
			return "请求频率超限（额度为0），请联系管理员"
		}
		if userRate.RPM > 0 && apiKey != "" {
			s := r.getShard(apiKey)
			s.mu.Lock()
			b, ok := s.entries[apiKey]
			if !ok || now.After(b.resetAt) {
				s.entries[apiKey] = &bucket{count: 1, tokens: 0, resetAt: now.Add(time.Minute)}
			} else {
				b.count++
				if b.count > userRate.RPM {
					s.mu.Unlock()
					return fmt.Sprintf("请求频率超限（每分钟%d次），请稍后再试", userRate.RPM)
				}
			}
			s.mu.Unlock()
		}
	}

	// TPM check: 0 = blocked, -1 = unlimited, >0 = limit
	if userRate != nil {
		if userRate.TPM == 0 {
			return "Token用量超限（额度为0），请联系管理员"
		}
		if userRate.TPM > 0 {
			userKey := fmt.Sprintf("tpm:%d", userRate.UserID)
			s := r.getShard(userKey)
			s.mu.Lock()
			b, ok := s.entries[userKey]
			if !ok || now.After(b.resetAt) {
				s.entries[userKey] = &bucket{count: 0, tokens: promptTokens, resetAt: now.Add(time.Minute)}
				if promptTokens > userRate.TPM {
					s.mu.Unlock()
					return fmt.Sprintf("Token用量超限（每分钟%d tokens），请稍后再试", userRate.TPM)
				}
			} else {
				if b.tokens+promptTokens > userRate.TPM {
					s.mu.Unlock()
					return fmt.Sprintf("Token用量超限（每分钟%d tokens），请稍后再试", userRate.TPM)
				}
				b.tokens += promptTokens
			}
			s.mu.Unlock()
		}
	}

	return ""
}

// RefundTokens rolls back pre-charged prompt tokens when a request fails.
func (r *RateLimiter) RefundTokens(userID uint, tokens int64) {
	if r == nil || tokens <= 0 {
		return
	}
	now := time.Now()
	userKey := fmt.Sprintf("tpm:%d", userID)
	s := r.getShard(userKey)
	s.mu.Lock()
	if b, ok := s.entries[userKey]; ok && !now.After(b.resetAt) {
		b.tokens -= tokens
		if b.tokens < 0 {
			b.tokens = 0
		}
	}
	s.mu.Unlock()
}

// ReleaseRPM decrements the RPM counter when a request fails.
func (r *RateLimiter) ReleaseRPM(apiKey string) {
	if r == nil || apiKey == "" {
		return
	}
	now := time.Now()
	s := r.getShard(apiKey)
	s.mu.Lock()
	if b, ok := s.entries[apiKey]; ok && !now.After(b.resetAt) {
		b.count--
		if b.count < 0 {
			b.count = 0
		}
	}
	s.mu.Unlock()
}

// AccountTokens records actual token usage after a request completes.
// This adds completion tokens (not known at pre-check time) to the TPM bucket.
func (r *RateLimiter) AccountTokens(userID uint, completionTokens int64) {
	if r == nil || completionTokens <= 0 {
		return
	}
	now := time.Now()
	userKey := fmt.Sprintf("tpm:%d", userID)
	s := r.getShard(userKey)
	s.mu.Lock()
	b, ok := s.entries[userKey]
	if !ok || now.After(b.resetAt) {
		// Preserve pre-charged tokens from previous window if any
		oldTokens := int64(0)
		if ok {
			oldTokens = b.tokens
		}
		s.entries[userKey] = &bucket{count: 0, tokens: oldTokens + completionTokens, resetAt: now.Add(time.Minute)}
	} else {
		b.tokens += completionTokens
	}
	s.mu.Unlock()
}
