package core

import (
	"time"

	"github.com/zapi/zapi-go/internal/model"
)

// channelUpdate represents a deferred DB update for a channel's fail_count / enabled status
type channelUpdate struct {
	ChannelID uint
	FailCount int
	Enabled   bool
}

var channelUpdateCh = make(chan channelUpdate, 16384)

func StartChannelUpdateWriter() {
	go func() {
		batch := make(map[uint]channelUpdate) // channelID → latest update (coalesce)
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case u := <-channelUpdateCh:
				batch[u.ChannelID] = u // last-write-wins coalescing
			case <-ticker.C:
				if len(batch) > 0 {
					flushChannelUpdates(batch)
					batch = make(map[uint]channelUpdate)
				}
			case <-StopChan:
				if len(batch) > 0 {
					flushChannelUpdates(batch)
				}
				return
			}
		}
	}()
}

func flushChannelUpdates(batch map[uint]channelUpdate) {
	for _, u := range batch {
		model.DB.Model(&model.Channel{}).Where("id = ?", u.ChannelID).
			Updates(map[string]interface{}{"fail_count": u.FailCount, "enabled": u.Enabled})
	}
}

// AsyncChannelUpdate enqueues a channel fail_count/enabled update for batched DB write
func AsyncChannelUpdate(channelID uint, failCount int, enabled bool) {
	select {
	case channelUpdateCh <- channelUpdate{ChannelID: channelID, FailCount: failCount, Enabled: enabled}:
	default:
		// Channel full — write synchronously as fallback
		model.DB.Model(&model.Channel{}).Where("id = ?", channelID).
			Updates(map[string]interface{}{"fail_count": failCount, "enabled": enabled})
	}
}
