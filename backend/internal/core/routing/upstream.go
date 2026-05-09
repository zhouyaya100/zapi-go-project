package routing

import (
	"strconv"
	"strings"
	"sync"

	"github.com/zapi/zapi-go/internal/config"
	"github.com/zapi/zapi-go/internal/model"
)

// UpstreamGroupInfo — lightweight upstream group for routing (in-memory)
type UpstreamGroupInfo struct {
	ID            uint
	Name          string
	Alias         string            // exposed model name
	Strategy      string            // priority|round_robin|weighted|least_latency|least_requests
	AllowedGroups map[string]bool   // parsed group set
	Enabled       bool
	MaxFails      int
	FailTimeout   int               // seconds
	RetryOnFail   bool
	ChannelIDs    []uint            // ordered channel IDs in this group
}

// UpstreamIndex — maps alias (model name) → UpstreamGroupInfo
type UpstreamIndex struct {
	mu     sync.RWMutex
	groups map[uint]*UpstreamGroupInfo  // id -> info
	alias  map[string]*UpstreamGroupInfo // alias -> info (for fast lookup by model name)
}

var Upstreams = &UpstreamIndex{
	groups: make(map[uint]*UpstreamGroupInfo),
	alias:  make(map[string]*UpstreamGroupInfo),
}

// LoadUpstreams — rebuild index from DB
func LoadUpstreams(groups []model.UpstreamGroup, channelGroups []model.UpstreamGroupChannel) {
	// Pre-load group name mapping outside the lock
	groupNameMap := make(map[uint]string)
	var allGroups []model.Group
	model.DB.Select("id, name").Find(&allGroups)
	for _, g := range allGroups {
		groupNameMap[g.ID] = g.Name
	}
	Upstreams.mu.Lock()
	defer Upstreams.mu.Unlock()
	Upstreams.groups = make(map[uint]*UpstreamGroupInfo)
	Upstreams.alias = make(map[string]*UpstreamGroupInfo)

	// Build channel→group mapping from join table
	chByGroup := make(map[uint][]uint) // groupID → []channelID
	for _, cg := range channelGroups {
		chByGroup[cg.UpstreamGroupID] = append(chByGroup[cg.UpstreamGroupID], cg.ChannelID)
	}

	for i := range groups {
		g := &groups[i]
		if !g.Enabled {
			continue
		}
		allowed := make(map[string]bool)
		if g.AllowedGroups != "" {
			for _, idStr := range splitComma(g.AllowedGroups) {
				// Convert group ID to group name for matching
				if id, err := strconv.Atoi(idStr); err == nil {
					if name, ok := groupNameMap[uint(id)]; ok {
						allowed[name] = true
					}
				}
				// Also keep the raw string for backward compatibility
				allowed[idStr] = true
			}
		}
		info := &UpstreamGroupInfo{
			ID:            g.ID,
			Name:          g.Name,
			Alias:         g.Alias,
			Strategy:      g.Strategy,
			AllowedGroups: allowed,
			Enabled:       g.Enabled,
			MaxFails:      g.MaxFails,
			FailTimeout:   g.FailTimeout,
			RetryOnFail:   g.RetryOnFail,
			ChannelIDs:    chByGroup[g.ID],
		}
		Upstreams.groups[g.ID] = info
		if g.Alias != "" {
			Upstreams.alias[g.Alias] = info
		}
	}
}

// UpdateUpstream — add or update a single group in the index
func (idx *UpstreamIndex) UpdateUpstream(ug *model.UpstreamGroup, channelIDs []uint) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	// Check alias conflict: same alias used by a different group
	if ug.Alias != "" {
		if existing, ok := idx.alias[ug.Alias]; ok && existing.ID != ug.ID {
			// Alias conflict — log warning, but still update (last write wins)
			// The previous group's alias mapping is removed
		}
	}
	allowed := make(map[string]bool)
	if ug.AllowedGroups != "" {
		// Build groupID→groupName mapping for AllowedGroups resolution
		groupNameMap := make(map[uint]string)
		var allGroups []model.Group
		model.DB.Select("id, name").Find(&allGroups)
		for _, g := range allGroups {
			groupNameMap[g.ID] = g.Name
		}
		for _, idStr := range splitComma(ug.AllowedGroups) {
			if id, err := strconv.Atoi(idStr); err == nil {
				if name, ok := groupNameMap[uint(id)]; ok {
					allowed[name] = true
				}
			}
			allowed[idStr] = true
		}
	}
	info := &UpstreamGroupInfo{
		ID:            ug.ID,
		Name:          ug.Name,
		Alias:         ug.Alias,
		Strategy:      ug.Strategy,
		AllowedGroups: allowed,
		Enabled:       ug.Enabled,
		MaxFails:      ug.MaxFails,
		FailTimeout:   ug.FailTimeout,
		RetryOnFail:   ug.RetryOnFail,
		ChannelIDs:    channelIDs,
	}
	// Remove old alias mapping if name changed
	if old, ok := idx.groups[ug.ID]; ok && old.Alias != "" {
		delete(idx.alias, old.Alias)
	}
	if ug.Enabled {
		idx.groups[ug.ID] = info
		if ug.Alias != "" {
			idx.alias[ug.Alias] = info
		}
	} else {
		delete(idx.groups, ug.ID)
		if ug.Alias != "" {
			delete(idx.alias, ug.Alias)
		}
	}
}

// RemoveUpstream — remove a group from the index
func (idx *UpstreamIndex) RemoveUpstream(id uint) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if old, ok := idx.groups[id]; ok {
		if old.Alias != "" {
			delete(idx.alias, old.Alias)
		}
		delete(idx.groups, id)
	}
}

// ResolveUpstream — find upstream group by model name (alias)
// Returns nil if no group matches (falls back to normal channel routing)
func ResolveUpstream(modelName string, group *string) *UpstreamGroupInfo {
	Upstreams.mu.RLock()
	defer Upstreams.mu.RUnlock()
	ug, ok := Upstreams.alias[modelName]
	if !ok || !ug.Enabled {
		return nil
	}
	// Check group permission
	if group != nil {
		gname := *group
		if len(ug.AllowedGroups) > 0 && !ug.AllowedGroups[gname] {
			return nil
		}
	} else {
		if len(ug.AllowedGroups) > 0 {
			return nil
		}
	}
	return ug
}

// GetUpstreamInfo — get UpstreamGroupInfo by ID
func (idx *UpstreamIndex) GetUpstreamInfo(id uint) *UpstreamGroupInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.groups[id]
}

// GetUpstreamGroupChannelIDs — get all channel IDs belonging to upstream groups
// Used by pool.go to exclude group-managed channels from normal selection
// Cached with separate write lock to avoid data races
var (
	ugChannelIDsCache map[uint]bool
	ugChannelIDsMu    sync.RWMutex
	ugChannelIDsVer   int
)

func GetUpstreamGroupChannelIDs() map[uint]bool {
	Upstreams.mu.RLock()
	ver := len(Upstreams.groups) + len(Upstreams.alias) // simple version heuristic
	ugChannelIDsMu.RLock()
	if ugChannelIDsCache != nil && ugChannelIDsVer == ver {
		ugChannelIDsMu.RUnlock()
		Upstreams.mu.RUnlock()
		return ugChannelIDsCache
	}
	ugChannelIDsMu.RUnlock()
	ids := make(map[uint]bool)
	for _, ug := range Upstreams.groups {
		for _, cid := range ug.ChannelIDs {
			ids[cid] = true
		}
	}
	ugChannelIDsMu.Lock()
	ugChannelIDsCache = ids
	ugChannelIDsVer = ver
	ugChannelIDsMu.Unlock()
	Upstreams.mu.RUnlock()
	return ids
}

// splitComma is defined in pool.go — re-export not needed, same package

// GetAllGroups — get all upstream group infos
func (idx *UpstreamIndex) GetAllGroups() []*UpstreamGroupInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	result := make([]*UpstreamGroupInfo, 0, len(idx.groups))
	for _, ug := range idx.groups {
		result = append(result, ug)
	}
	return result
}

// GetMaxFailsForChannel — find the strictest (smallest) max_fails and fail_timeout
// across all upstream groups that contain this channel.
// Returns (maxFails, failTimeout); uses global proxy config defaults if channel is not in any group.
func (idx *UpstreamIndex) GetMaxFailsForChannel(channelID uint) (maxFails int, failTimeout int) {
	// Use global proxy config as defaults
	maxFails = config.Cfg.Proxy.MaxFails
	if maxFails <= 0 {
		maxFails = 5
	}
	failTimeout = config.Cfg.Proxy.FailTimeout
	if failTimeout <= 0 {
		failTimeout = 30
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	for _, ug := range idx.groups {
		for _, cid := range ug.ChannelIDs {
			if cid == channelID {
				if ug.MaxFails > 0 && ug.MaxFails < maxFails {
					maxFails = ug.MaxFails
				}
				if ug.FailTimeout > 0 && ug.FailTimeout < failTimeout {
					failTimeout = ug.FailTimeout
				}
				break // found in this group, check next group
			}
		}
	}
	return
}

// ParseAllowedGroups — helper for handler
func ParseAllowedGroups(s string) map[string]bool {
	m := make(map[string]bool)
	if s == "" {
		return m
	}
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			m[v] = true
		}
	}
	return m
}
