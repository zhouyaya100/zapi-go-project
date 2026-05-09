package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/core/routing"
	"github.com/zapi/zapi-go/internal/model"
)

func HandleListUpstreamGroups(c *gin.Context) {
	limit, offset := parsePagination(c)

	var total int64
	model.DB.Model(&model.UpstreamGroup{}).Count(&total)

	var groups []model.UpstreamGroup
	qry := model.DB.Order("id")
	if limit > 0 {
		qry = qry.Offset(offset).Limit(limit)
	}
	qry.Find(&groups)

	result := make([]gin.H, len(groups))
	for i, ug := range groups {
		result[i] = buildUGResponse(&ug)
	}
	if limit > 0 {
		c.JSON(200, gin.H{"items": result, "total": total})
	} else {
		c.JSON(200, result)
	}
}

func HandleCreateUpstreamGroup(c *gin.Context) {
	var req struct {
		Name                string `json:"name"`
		Alias               string `json:"alias"`
		Strategy            string `json:"strategy"`
		AllowedGroups       string `json:"allowed_groups"`
		Enabled             *bool  `json:"enabled"`
		HealthCheckInterval int    `json:"health_check_interval"`
		MaxFails            int    `json:"max_fails"`
		FailTimeout         int    `json:"fail_timeout"`
		RetryOnFail         *bool  `json:"retry_on_fail"`
		ChannelIDs          []uint `json:"channel_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "请求参数错误"}})
		return
	}

	if req.Name == "" {
		c.JSON(400, gin.H{"error": gin.H{"message": "组名不能为空"}})
		return
	}
	if req.Alias == "" {
		c.JSON(400, gin.H{"error": gin.H{"message": "别名(模型名)不能为空"}})
		return
	}
	validStrategies := map[string]bool{"priority": true, "round_robin": true, "weighted": true, "least_latency": true, "least_requests": true}
	strategy := req.Strategy
	if !validStrategies[strategy] {
		strategy = "priority"
	}
	maxFails := req.MaxFails
	if maxFails <= 0 {
		maxFails = 5
	}
	failTimeout := req.FailTimeout
	if failTimeout <= 0 {
		failTimeout = 30
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	retryOnFail := true
	if req.RetryOnFail != nil {
		retryOnFail = *req.RetryOnFail
	}

	ug := model.UpstreamGroup{
		Name:                req.Name,
		Alias:               req.Alias,
		Strategy:            strategy,
		AllowedGroups:       req.AllowedGroups,
		Enabled:             enabled,
		HealthCheckInterval: req.HealthCheckInterval,
		MaxFails:            maxFails,
		FailTimeout:         failTimeout,
		RetryOnFail:         retryOnFail,
	}
	model.DB.Create(&ug)

	// Assign channels to this group via join table
	if len(req.ChannelIDs) > 0 {
		for _, chID := range req.ChannelIDs {
			model.DB.Create(&model.UpstreamGroupChannel{
				UpstreamGroupID: ug.ID,
				ChannelID:       chID,
			})
		}
	}

	refreshUpstreamIndex(ug.ID)

	c.JSON(200, gin.H{"success": true, "id": ug.ID})
}

func HandleUpdateUpstreamGroup(c *gin.Context) {
	id := c.Param("id")
	var ug model.UpstreamGroup
	if model.DB.First(&ug, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "上游组不存在"}})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "请求参数错误"}})
		return
	}
	if v, ok := req["name"].(string); ok {
		ug.Name = v
	}
	if v, ok := req["alias"].(string); ok {
		ug.Alias = v
	}
	if v, ok := req["strategy"].(string); ok {
		validStrategies := map[string]bool{"priority": true, "round_robin": true, "weighted": true, "least_latency": true, "least_requests": true}
		if validStrategies[v] {
			ug.Strategy = v
		}
	}
	if v, ok := req["allowed_groups"].(string); ok {
		ug.AllowedGroups = v
	}
	if v, ok := req["enabled"].(bool); ok {
		ug.Enabled = v
	}
	if v, ok := req["health_check_interval"].(float64); ok {
		ug.HealthCheckInterval = int(v)
	}
	if v, ok := req["max_fails"].(float64); ok {
		ug.MaxFails = int(v)
	}
	if v, ok := req["fail_timeout"].(float64); ok {
		ug.FailTimeout = int(v)
	}
	if v, ok := req["retry_on_fail"].(bool); ok {
		ug.RetryOnFail = v
	}
	model.DB.Save(&ug)

	// Handle channel_ids update — replace all channels in the group via join table
	if chIDsRaw, ok := req["channel_ids"]; ok {
		// Delete all existing links for this group
		model.DB.Where("upstream_group_id = ?", ug.ID).Delete(&model.UpstreamGroupChannel{})

		// Insert new links and collect channel IDs for pool refresh
		if chIDList, ok := chIDsRaw.([]interface{}); ok {
			for _, raw := range chIDList {
				if chID, ok := raw.(float64); ok {
					model.DB.Create(&model.UpstreamGroupChannel{
						UpstreamGroupID: ug.ID,
						ChannelID:       uint(chID),
					})
					var ch model.Channel
					if model.DB.First(&ch, uint(chID)).Error == nil {
						routing.Pool.UpdateChannel(&ch)
					}
				}
			}
		}
	}

	refreshUpstreamIndex(ug.ID)

	c.JSON(200, gin.H{"success": true})
}

func HandleDeleteUpstreamGroup(c *gin.Context) {
	id := c.Param("id")
	var ug model.UpstreamGroup
	if model.DB.First(&ug, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "上游组不存在"}})
		return
	}

	// Clear user associations referencing this group
	model.DB.Where("upstream_group_id = ?", ug.ID).Delete(&model.UserUpstreamGroup{})

	// Get affected channel IDs before deleting links
	var affectedChannels []model.UpstreamGroupChannel
	model.DB.Where("upstream_group_id = ?", ug.ID).Find(&affectedChannels)

	// Delete all join table links for this group
	model.DB.Where("upstream_group_id = ?", ug.ID).Delete(&model.UpstreamGroupChannel{})

	model.DB.Delete(&ug)
	routing.Upstreams.RemoveUpstream(ug.ID)

	// Refresh Pool for affected channels
	for _, link := range affectedChannels {
		var ch model.Channel
		if model.DB.First(&ch, link.ChannelID).Error == nil {
			routing.Pool.UpdateChannel(&ch)
		}
	}

	c.JSON(200, gin.H{"success": true})
}

func HandleGetUpstreamGroup(c *gin.Context) {
	id := c.Param("id")
	var ug model.UpstreamGroup
	if model.DB.First(&ug, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "上游组不存在"}})
		return
	}
	c.JSON(200, buildUGResponse(&ug))
}

// HandleAddChannelToGroup — add a channel to an upstream group via join table
func HandleAddChannelToGroup(c *gin.Context) {
	id := c.Param("id")
	var ug model.UpstreamGroup
	if model.DB.First(&ug, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "上游组不存在"}})
		return
	}
	var req struct {
		ChannelID uint `json:"channel_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "请求参数错误"}})
		return
	}
	var ch model.Channel
	if model.DB.First(&ch, req.ChannelID).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "渠道不存在"}})
		return
	}

	// Create join table entry (ignore if already exists)
	link := model.UpstreamGroupChannel{UpstreamGroupID: ug.ID, ChannelID: ch.ID}
	model.DB.Where("upstream_group_id = ? AND channel_id = ?", ug.ID, ch.ID).FirstOrCreate(&link)

	routing.Pool.UpdateChannel(&ch)
	refreshUpstreamIndex(ug.ID)

	c.JSON(200, gin.H{"success": true})
}

// HandleRemoveChannelFromGroup — remove a channel from an upstream group via join table
func HandleRemoveChannelFromGroup(c *gin.Context) {
	chID := c.Param("channel_id")
	groupID := c.Param("id")
	var ch model.Channel
	if model.DB.First(&ch, chID).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "渠道不存在"}})
		return
	}

	// Parse group ID
	var gid uint
	if f, err := strconv.ParseUint(groupID, 10, 64); err == nil {
		gid = uint(f)
	}

	// Delete the specific join table entry
	model.DB.Where("upstream_group_id = ? AND channel_id = ?", gid, ch.ID).Delete(&model.UpstreamGroupChannel{})

	routing.Pool.UpdateChannel(&ch)
	refreshUpstreamIndex(gid)

	c.JSON(200, gin.H{"success": true})
}

// refreshUpstreamIndex — reload a single upstream group's channel IDs into the routing index
func refreshUpstreamIndex(groupID uint) {
	var ug model.UpstreamGroup
	if model.DB.First(&ug, groupID).Error != nil {
		return
	}
	// Query join table for channel IDs
	var links []model.UpstreamGroupChannel
	model.DB.Where("upstream_group_id = ?", ug.ID).Find(&links)
	var chIDs []uint
	for _, link := range links {
		chIDs = append(chIDs, link.ChannelID)
	}
	routing.Upstreams.UpdateUpstream(&ug, chIDs)
}

// buildUGResponse — build response JSON for an upstream group with its channels
func buildUGResponse(ug *model.UpstreamGroup) gin.H {
	// Query join table for channel IDs
	var links []model.UpstreamGroupChannel
	model.DB.Where("upstream_group_id = ?", ug.ID).Find(&links)
	chIDs := make([]uint, 0, len(links))
	for _, link := range links {
		chIDs = append(chIDs, link.ChannelID)
	}

	// Load channels by IDs
	var channels []model.Channel
	if len(chIDs) > 0 {
		model.DB.Where("id IN ?", chIDs).Order("priority DESC, weight DESC, id").Find(&channels)
	}
	chItems := make([]gin.H, len(channels))
	for i, ch := range channels {
		chItems[i] = gin.H{
			"id": ch.ID, "name": ch.Name, "type": ch.Type,
			"weight": ch.Weight, "priority": ch.Priority,
			"enabled": ch.Enabled,
			"fail_count": ch.FailCount, "response_time": ch.ResponseTime,
		}
	}
	return gin.H{
		"id":                    ug.ID,
		"name":                  ug.Name,
		"alias":                 ug.Alias,
		"strategy":              ug.Strategy,
		"allowed_groups":        ug.AllowedGroups,
		"enabled":               ug.Enabled,
		"health_check_interval": ug.HealthCheckInterval,
		"max_fails":             ug.MaxFails,
		"fail_timeout":          ug.FailTimeout,
		"retry_on_fail":         ug.RetryOnFail,
		"channels":              chItems,
		"created_at":            ug.CreatedAt,
	}
}
