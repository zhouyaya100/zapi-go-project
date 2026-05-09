package handler

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/model"
)

func HandleListGroups(c *gin.Context) {
	var gs []model.Group
	model.DB.Order("id").Find(&gs)
	var uc []struct {
		GroupID uint
		Cnt     int64
	}
	model.DB.Model(&model.User{}).Select("group_id, count(*) as cnt").Group("group_id").Scan(&uc)
	um := make(map[uint]int64)
	for _, r := range uc {
		um[r.GroupID] = r.Cnt
	}
	// Load all group-upstream-group associations
	var gugs []model.GroupUpstreamGroup
	model.DB.Find(&gugs)
	gugMap := make(map[uint][]uint) // group_id -> []upstream_group_id
	for _, gug := range gugs {
		gugMap[gug.GroupID] = append(gugMap[gug.GroupID], gug.UpstreamGroupID)
	}
	// Load upstream group names
	var ugs []model.UpstreamGroup
	model.DB.Find(&ugs)
	ugMap := make(map[uint]string)
	for _, ug := range ugs {
		ugMap[ug.ID] = ug.Name
	}
	result := make([]gin.H, len(gs))
	for i, g := range gs {
		ugIDs := gugMap[g.ID]
		ugNames := make([]string, 0, len(ugIDs))
		for _, id := range ugIDs {
			if name, ok := ugMap[id]; ok {
				ugNames = append(ugNames, name)
			}
		}
		result[i] = gin.H{
			"id": g.ID, "name": g.Name, "comment": g.Comment,
			"allowed_models": g.AllowedModels,
			"upstream_group_ids": ugIDs, "upstream_group_names": ugNames,
			"rate_mode": g.RateMode, "rpm": g.RPM, "tpm": g.TPM, "model_rate_limits": g.ModelRateLimits,
			"user_count": um[g.ID], "created_at": core.FmtTimeVal(g.CreatedAt),
		}
	}
	c.JSON(200, result)
}

func HandleCreateGroup(c *gin.Context) {
	var req struct {
		Name            string   `json:"name"`
		Comment         string   `json:"comment"`
		AllowedModels   string   `json:"allowed_models"`
		RateMode        string   `json:"rate_mode"`
		RPM             int      `json:"rpm"`
		TPM             int64    `json:"tpm"`
		ModelRateLimits string   `json:"model_rate_limits"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "请求参数错误"}})
		return
	}
	if req.Name == "" {
		c.JSON(400, gin.H{"error": gin.H{"message": "分组名不能为空"}})
		return
	}
	var existing model.Group
	if model.DB.Where("name = ?", req.Name).First(&existing).Error == nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "分组名已存在"}})
		return
	}
	// Defaults: rate_mode="global", RPM=-1 (unlimited), TPM=-1 (unlimited)
	rateMode := req.RateMode
	if rateMode == "" {
		rateMode = "global"
	}
	rpm := req.RPM
	if rpm == 0 && req.RateMode == "" {
		rpm = -1
	}
	tpm := req.TPM
	if tpm == 0 && req.RateMode == "" {
		tpm = -1
	}
	g := model.Group{Name: req.Name, Comment: req.Comment, AllowedModels: req.AllowedModels, RateMode: rateMode, RPM: rpm, TPM: tpm, ModelRateLimits: req.ModelRateLimits}
	model.DB.Create(&g)
	// Auto-derive group_upstream_groups from allowed_models
	syncGroupUpstreamGroups(g.ID, req.AllowedModels)
	c.JSON(200, gin.H{"success": true, "id": g.ID})
}

func HandleUpdateGroup(c *gin.Context) {
	id := c.Param("id")
	var g model.Group
	if model.DB.First(&g, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "分组不存在"}})
		return
	}
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	if v, ok := req["name"]; ok {
		if s, ok := v.(string); ok && s != "" {
			// Check duplicate name
			var existing model.Group
			if model.DB.Where("name = ? AND id != ?", s, g.ID).First(&existing).Error == nil {
				c.JSON(400, gin.H{"error": gin.H{"message": fmt.Sprintf("分组 '%s' 已存在", s)}})
				return
			}
			g.Name = s
		}
	}
	if v, ok := req["comment"]; ok {
		if s, ok := v.(string); ok { g.Comment = s }
	}
	if v, ok := req["allowed_models"]; ok {
		if s, ok := v.(string); ok {
			g.AllowedModels = s
			// Auto-derive group_upstream_groups from allowed_models
			syncGroupUpstreamGroups(g.ID, s)
			// Invalidate caches for users in this group
			var affectedUsers []model.User
			model.DB.Where("group_id = ?", g.ID).Find(&affectedUsers)
			for _, u := range affectedUsers {
				core.InvalidateUserCache(u.ID)
			}
			core.InvalidateAllTokenCache()
		}
	}
	if v, ok := req["rate_mode"].(string); ok {
		if v == "global" || v == "per_model" { g.RateMode = v }
	}
	if v, ok := req["rpm"].(float64); ok {
		g.RPM = int(v)
	}
	if v, ok := req["tpm"].(float64); ok {
		g.TPM = int64(v)
	}
	if v, ok := req["model_rate_limits"].(string); ok {
		g.ModelRateLimits = v
	} else if arr, ok := req["model_rate_limits"].([]interface{}); ok {
		mrlMap := make(map[string]interface{})
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				modelName, _ := m["model"].(string)
				if modelName == "" {
					continue
				}
				entry := map[string]interface{}{}
				if rpm, ok := m["rpm"].(float64); ok {
					entry["rpm"] = int(rpm)
				} else {
					entry["rpm"] = 0
				}
				if tpm, ok := m["tpm"].(float64); ok {
					entry["tpm"] = int64(tpm)
				} else {
					entry["tpm"] = 0
				}
				if blocked, ok := m["blocked"].(bool); ok {
					entry["blocked"] = blocked
				}
				mrlMap[modelName] = entry
			}
		}
		if len(mrlMap) > 0 {
			if b, err := json.Marshal(mrlMap); err == nil {
				g.ModelRateLimits = string(b)
			}
		} else {
			g.ModelRateLimits = ""
		}
	}
	model.DB.Save(&g)
	core.InvalidateGroupCache(g.ID)
	c.JSON(200, gin.H{"success": true})
}

func HandleDeleteGroup(c *gin.Context) {
	id := c.Param("id")
	var g model.Group
	if model.DB.First(&g, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "分组不存在"}})
		return
	}
	// Clear users referencing this group + invalidate their caches
	var affectedUsers []model.User
	model.DB.Where("group_id = ?", g.ID).Find(&affectedUsers)
	for _, u := range affectedUsers { core.InvalidateUserCache(u.ID) }
	core.InvalidateAllTokenCache()
	model.DB.Model(&model.User{}).Where("group_id = ?", g.ID).Update("group_id", nil)
	// Delete group-upstream-group associations
	model.DB.Where("group_id = ?", g.ID).Delete(&model.GroupUpstreamGroup{})
	core.InvalidateGroupCache(g.ID)
	model.DB.Delete(&g)
	c.JSON(200, gin.H{"success": true})
}

// syncGroupUpstreamGroups auto-derives group_upstream_groups from allowed_models.
// If an allowed_model matches an upstream group alias, the group is associated with that upstream group.
func syncGroupUpstreamGroups(groupID uint, allowedModels string) {
	// Clear existing associations
	model.DB.Where("group_id = ?", groupID).Delete(&model.GroupUpstreamGroup{})

	// Build alias -> upstream_group_id map
	var ugs []model.UpstreamGroup
	model.DB.Find(&ugs)
	aliasMap := make(map[string]uint) // alias -> upstream_group_id
	for _, ug := range ugs {
		if ug.Alias != "" {
			aliasMap[ug.Alias] = ug.ID
		}
	}

	if allowedModels == "" {
		// No models allowed = no upstream groups
		return
	}

	// Check each model in allowed_models against aliases
	modelSet := core.GetModelSet(allowedModels)
	for alias, ugID := range aliasMap {
		if modelSet[alias] {
			model.DB.Create(&model.GroupUpstreamGroup{GroupID: groupID, UpstreamGroupID: ugID})
		}
	}
}
