package core

import (
	"strconv"
	"sync"
	"time"

	"github.com/zapi/zapi-go/internal/config"
	"github.com/zapi/zapi-go/internal/model"
)

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

type localCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func newCache() *localCache {
	return &localCache{entries: make(map[string]cacheEntry)}
}

func (c *localCache) ttl() time.Duration {
	t := config.Cfg.Cache.TTL
	if t <= 0 { t = 30 }
	return time.Duration(t) * time.Second
}

func (c *localCache) maxSize() int {
	m := config.Cfg.Cache.MaxEntries
	if m <= 0 { m = 10000 }
	return m
}

func (c *localCache) evictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, v := range c.entries {
		if now.After(v.expiresAt) { delete(c.entries, k) }
	}
}

func (c *localCache) get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok { return nil, false }
	if time.Now().After(e.expiresAt) { return nil, false }
	return e.value, true
}

func (c *localCache) set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Evict expired entries if at capacity
	if len(c.entries) >= c.maxSize() {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expiresAt) { delete(c.entries, k) }
		}
		// If still full after evicting expired, remove oldest 10%
		if len(c.entries) >= c.maxSize() {
			removeCount := len(c.entries)/10 + 1
			i := 0
			for k := range c.entries {
				delete(c.entries, k)
				i++
				if i >= removeCount { break }
			}
		}
	}
	c.entries[key] = cacheEntry{value: value, expiresAt: time.Now().Add(c.ttl())}
}

func (c *localCache) invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *localCache) clear() {
	c.mu.Lock()
	c.entries = make(map[string]cacheEntry)
	c.mu.Unlock()
}

var tokenCache = newCache()
var userCache = newCache()
var groupCache = newCache()

func init() {
	go func() {
		for {
			time.Sleep(60 * time.Second)
			tokenCache.evictExpired()
			userCache.evictExpired()
			groupCache.evictExpired()
		}
	}()
}

// CachedLookupToken looks up a token by API key with cache
func CachedLookupToken(apiKey string) (*model.Token, bool) {
	if v, ok := tokenCache.get(apiKey); ok {
		if t, ok := v.(*model.Token); ok { return t, true }
	}
	var tk model.Token
	result := model.DB.Where("key = ? AND enabled = ?", apiKey, true).First(&tk)
	if result.Error != nil {
		return nil, false
	}
	tokenCache.set(apiKey, &tk)
	return &tk, true
}

// CachedLookupUser looks up a user by ID with cache
func CachedLookupUser(userID uint) (*model.User, bool) {
	key := userCacheKey(userID)
	if v, ok := userCache.get(key); ok {
		if u, ok := v.(*model.User); ok { return u, true }
	}
	var user model.User
	if model.DB.First(&user, userID).Error != nil {
		return nil, false
	}
	userCache.set(key, &user)
	return &user, true
}

// CachedLookupGroup looks up a group by ID with cache
func CachedLookupGroup(groupID uint) (*model.Group, bool) {
	key := groupCacheKey(groupID)
	if v, ok := groupCache.get(key); ok {
		if g, ok := v.(*model.Group); ok { return g, true }
	}
	var grp model.Group
	if model.DB.First(&grp, groupID).Error != nil {
		return nil, false
	}
	groupCache.set(key, &grp)
	return &grp, true
}

// ModelSetCache — pre-parsed model sets for Token/User to avoid repeated SplitComma on hot path
var (
	modelSetCache   = make(map[string]map[string]bool) // raw string → parsed set
	modelSetCacheMu sync.RWMutex
)

// GetModelSet returns a cached parsed model set for a comma-separated string.
// Thread-safe. Returns nil if string is empty.
func GetModelSet(s string) map[string]bool {
	if s == "" { return nil }
	modelSetCacheMu.RLock()
	m, ok := modelSetCache[s]
	modelSetCacheMu.RUnlock()
	if ok { return m }
	modelSetCacheMu.Lock()
	// double-check after acquiring write lock
	if m, ok = modelSetCache[s]; ok {
		modelSetCacheMu.Unlock()
		return m
	}
	m = make(map[string]bool)
	for _, v := range SplitComma(s) { m[v] = true }
	modelSetCache[s] = m
	modelSetCacheMu.Unlock()
	return m
}

// IsModelAllowed checks if a model name is in a comma-separated list, using cached parsed sets
func IsModelAllowed(modelName, commaList string) bool {
	if commaList == "" { return true } // empty = no restriction
	ms := GetModelSet(commaList)
	return ms[modelName]
}

func InvalidateTokenCache(apiKey string) { tokenCache.invalidate(apiKey) }
func InvalidateUserCache(userID uint)    { userCache.invalidate(userCacheKey(userID)) }
func InvalidateGroupCache(groupID uint)  { groupCache.invalidate(groupCacheKey(groupID)) }
func InvalidateAllTokenCache()           { tokenCache.clear() }
func InvalidateAllUserCache()            { userCache.clear() }

func userCacheKey(id uint) string { return "u:" + strconv.FormatUint(uint64(id), 10) }
func groupCacheKey(id uint) string { return "g:" + strconv.FormatUint(uint64(id), 10) }
