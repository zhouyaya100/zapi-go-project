package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/core/routing"
	"github.com/zapi/zapi-go/internal/model"
)

// HandleLBStatus — returns load balancing status for all upstream groups
func HandleLBStatus(c *gin.Context) {
	var groups []model.UpstreamGroup
	model.DB.Where("enabled = ?", true).Order("id").Find(&groups)

	type channelStatus struct {
		ID             uint   `json:"id"`
		Name           string `json:"name"`
		Weight         int    `json:"weight"`
		Priority       int    `json:"priority"`
		Status         string `json:"status"`
		ActiveReqs     int64  `json:"active_requests"`
		TotalReqs      int64  `json:"total_requests"`        // per-group total
		GlobalReqs     int64  `json:"global_total_requests"`  // global total
		SuccessRate    string `json:"success_rate"`           // per-group success rate
		GlobalSuccRate string `json:"global_success_rate"`    // global success rate
		AvgLatencyMs   int64  `json:"avg_latency_ms"`
		FailCount      int64  `json:"fail_count"`
		Circuit        string `json:"circuit"`
		Shared         bool   `json:"shared"`
		ResponseTime   int    `json:"response_time"`  // heartbeat response time (ms)
		HeartFailCount int    `json:"heart_fail_count"` // heartbeat fail count from DB
	}

	type groupStatus struct {
		ID       uint            `json:"id"`
		Name     string          `json:"name"`
		Strategy string          `json:"strategy"`
		Channels []channelStatus `json:"channels"`
	}

	// Build channel→group count map for "shared" flag
	channelGroupCount := map[uint]int{}
	{
		var allChIDs []uint
		model.DB.Table("upstream_group_channels").Pluck("channel_id", &allChIDs)
		for _, cid := range allChIDs {
			channelGroupCount[cid]++
		}
	}

	var result []groupStatus

	for _, ug := range groups {
		gs := groupStatus{
			ID:       ug.ID,
			Name:     ug.Name,
			Strategy: ug.Strategy,
		}

		// Get channel IDs for this group
		var chIDs []uint
		model.DB.Table("upstream_group_channels").Where("upstream_group_id = ?", ug.ID).Pluck("channel_id", &chIDs)

		var channels []model.Channel
		if len(chIDs) > 0 {
			model.DB.Where("id IN ?", chIDs).Order("priority DESC, weight DESC, id").Find(&channels)
		}

		gs.Channels = make([]channelStatus, 0)
		for _, ch := range channels {
			globalTotalReqs, globalTotalFails, avgLatency, failCount, circuitState := routing.Health.GetStats(ch.ID)
			groupTotalReqs, groupTotalFails := routing.Health.GetGroupChannelStats(ch.ID, ug.ID)

		status := "healthy"
		if !ch.Enabled {
			status = "disabled"
		} else if ch.FailCount > 0 {
			status = "unhealthy"
		} else if failCount > 0 {
			status = "unhealthy"
		}

			successRate := "100.0%"
			if groupTotalReqs > 0 {
				sr := float64(groupTotalReqs-groupTotalFails) / float64(groupTotalReqs) * 100
				successRate = fmt.Sprintf("%.1f%%", sr)
			}

			globalSuccRate := "100.0%"
			if globalTotalReqs > 0 {
				sr := float64(globalTotalReqs-globalTotalFails) / float64(globalTotalReqs) * 100
				globalSuccRate = fmt.Sprintf("%.1f%%", sr)
			}

			var activeReqs int64
			chHealth := routing.Health.Get(ch.ID)
			if chHealth != nil {
				activeReqs = chHealth.ActiveReqs.Load()
			}

			gs.Channels = append(gs.Channels, channelStatus{
				ID:             ch.ID,
				Name:           ch.Name,
				Weight:         ch.Weight,
				Priority:       ch.Priority,
				Status:         status,
				ActiveReqs:     activeReqs,
				TotalReqs:      groupTotalReqs,
				GlobalReqs:     globalTotalReqs,
				SuccessRate:    successRate,
				GlobalSuccRate: globalSuccRate,
				AvgLatencyMs:   avgLatency,
				FailCount:      failCount,
				Circuit:        routing.CircuitStateString(circuitState),
				Shared:         channelGroupCount[ch.ID] > 1,
				ResponseTime:   ch.ResponseTime,
				HeartFailCount: ch.FailCount,
			})
		}

		result = append(result, gs)
	}

	c.JSON(200, gin.H{"groups": result})
}

// HandleResetCircuit — manually reset a channel's circuit breaker
func HandleResetCircuit(c *gin.Context) {
	chID := c.Param("channel_id")
	var ch model.Channel
	if model.DB.First(&ch, chID).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "渠道不存在"}})
		return
	}
	routing.Health.ResetCircuit(ch.ID)
	c.JSON(200, gin.H{"success": true})
}
