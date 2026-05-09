package handler

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/core/routing"
	"github.com/zapi/zapi-go/internal/middleware"
	"github.com/zapi/zapi-go/internal/model"
	"gorm.io/gorm"
)

// parsePagination extracts limit and offset from query params.
// Defaults: limit=0 (no limit, return all), offset=0.
// If limit is provided, it's capped at 500.
func parsePagination(c *gin.Context) (limit, offset int) {
	offset = 0
	limit = 0 // 0 means no limit (backward compatible)
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
			if limit > 500 { limit = 500 }
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

func HandleListUsers(c *gin.Context) {
	limit, offset := parsePagination(c)

	var total int64
	model.DB.Model(&model.User{}).Count(&total)

	var users []model.User
	qry := model.DB.Order("id")
	if limit > 0 {
		qry = qry.Offset(offset).Limit(limit)
	}
	qry.Find(&users)

	var tc []struct {
		UserID uint
		Cnt    int64
	}
	model.DB.Model(&model.Token{}).Select("user_id, count(*) as cnt").Group("user_id").Scan(&tc)
	tm := make(map[uint]int64)
	for _, r := range tc {
		tm[r.UserID] = r.Cnt
	}
	var groups []model.Group
	model.DB.Find(&groups)
	groupMap := make(map[uint]model.Group)
	for _, g := range groups {
		groupMap[g.ID] = g
	}
	result := make([]gin.H, len(users))
	for i, u := range users {
		gn := ""
		var groupAllowedModels string
		if u.GroupID != nil {
			if gg, ok := groupMap[*u.GroupID]; ok {
				gn = gg.Name
				groupAllowedModels = gg.AllowedModels
			}
		}
		// Determine effective models based on bind_mode
		var effectiveModels []string
		if u.BindMode == "custom" {
			if u.AllowedModels != "" {
				effectiveModels = core.GetGroupAuthedModels(gn, u.AllowedModels, routing.Pool)
			}
		} else {
			effectiveModels = core.GetGroupAuthedModels(gn, groupAllowedModels, routing.Pool)
		}
		result[i] = gin.H{
			"id": u.ID, "username": u.Username, "role": u.Role,
			"group_id": u.GroupID, "group_name": gn, "enabled": u.Enabled,
			"bind_mode": u.BindMode,
			"group_allowed_models": groupAllowedModels,
			"max_tokens": u.MaxTokens,
			"token_quota":      core.SafeInt(u.TokenQuota),
			"token_quota_used": core.SafeInt(u.TokenQuotaUsed),
			"allowed_models":   u.AllowedModels, "authed_models": effectiveModels,
			"rate_mode": u.RateMode, "rpm": u.RPM, "tpm": u.TPM, "model_rate_limits": u.ModelRateLimits,
			"token_count": tm[u.ID], "created_at": core.FmtTimeVal(u.CreatedAt),
		}
	}
	// Return paginated response only when limit is specified
	if limit > 0 {
		c.JSON(200, gin.H{"items": result, "total": total})
	} else {
		c.JSON(200, result)
	}
}

func HandleUpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "无效的用户ID"}})
		return
	}
	a := middleware.GetAuth(c)
	if a != nil && uint(id) == model.SuperAdminID && !a.IsSuper {
		c.JSON(403, gin.H{"error": gin.H{"message": "\u65e0\u6cd5\u4fee\u6539\u8d85\u7ea7\u7ba1\u7406\u5458"}})
		return
	}
	var user model.User
	if model.DB.First(&user, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "\u7528\u6237\u4e0d\u5b58\u5728"}})
		return
	}
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	if v, ok := req["enabled"].(bool); ok {
		user.Enabled = v
	}
	if v, ok := req["max_tokens"].(float64); ok {
		user.MaxTokens = int(v)
	}
	if v, ok := req["token_quota"].(float64); ok {
		if int64(v) < -1 {
			c.JSON(400, gin.H{"error": gin.H{"message": "token_quota \u53ea\u80fd\u4e3a -1(\u65e0\u9650) \u6216\u6b63\u6570"}})
			return
		}
		user.TokenQuota = int64(v)
	}
	if v, ok := req["token_quota_used"].(float64); ok {
		if int64(v) < 0 {
			c.JSON(400, gin.H{"error": gin.H{"message": "token_quota_used \u4e0d\u80fd\u4e3a\u8d1f\u6570"}})
			return
		}
		user.TokenQuotaUsed = int64(v)
	}
	if v, ok := req["allowed_models"].(string); ok {
		user.AllowedModels = v
	}
	if v, ok := req["bind_mode"].(string); ok {
		if v == "inherit" || v == "custom" {
			user.BindMode = v
		}
		if v == "inherit" {
			// When switching to inherit, clear user's own settings
			user.AllowedModels = ""
		}
	}
	if v, ok := req["rate_mode"].(string); ok {
		if v == "inherit" || v == "global" || v == "per_model" { user.RateMode = v }
		if v == "inherit" { user.RPM = 0; user.TPM = 0; user.ModelRateLimits = "" }
	}
	if user.RateMode != "inherit" {
		if v, ok := req["rpm"].(float64); ok { user.RPM = int(v) }
		if v, ok := req["tpm"].(float64); ok { user.TPM = int64(v) }
		if v, ok := req["model_rate_limits"].(string); ok {
			user.ModelRateLimits = v
		} else if arr, ok := req["model_rate_limits"].([]interface{}); ok {
			// Convert array format [{model,rpm,tpm,blocked}] to map {model: {rpm,tpm,blocked}} then to JSON string
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
					user.ModelRateLimits = string(b)
				}
			} else {
				user.ModelRateLimits = ""
			}
		}
	}
	if v, ok := req["group_id"].(float64); ok {
		if v > 0 {
			uid := uint(v)
			user.GroupID = &uid
		} else {
			user.GroupID = nil
		}
	}
	if v, ok := req["role"].(string); ok {
		if v != "admin" && v != "operator" && v != "user" {
			c.JSON(400, gin.H{"error": gin.H{"message": "\u65e0\u6548\u89d2\u8272"}})
			return
		}
		if user.ID == model.SuperAdminID && v != "admin" {
			c.JSON(400, gin.H{"error": gin.H{"message": "\u65e0\u6cd5\u66f4\u6539\u8d85\u7ea7\u7ba1\u7406\u5458\u89d2\u8272"}})
			return
		}
		if v == "admin" && a != nil && !a.IsSuper {
			c.JSON(403, gin.H{"error": gin.H{"message": "\u53ea\u6709\u8d85\u7ea7\u7ba1\u7406\u5458\u624d\u80fd\u6307\u5b9aadmin\u89d2\u8272"}})
			return
		}
		if user.Role == "admin" && v != "admin" && a != nil && !a.IsSuper {
			c.JSON(400, gin.H{"error": gin.H{"message": "\u65e0\u6cd5\u66f4\u6539\u5176\u4ed6\u7ba1\u7406\u5458\u89d2\u8272"}})
			return
		}
		user.Role = v
	}
	if v, ok := req["password"].(string); ok {
		if msg := core.ValidatePasswordStrength(v); msg != "" {
			c.JSON(400, gin.H{"error": gin.H{"message": msg}})
			return
		}
		hash, _ := core.HashPassword(v)
		user.PasswordHash = hash
	}
	model.DB.Save(&user)
	core.InvalidateUserCache(user.ID)
	core.InvalidateGroupCache(0) // invalidate all group caches since group membership may affect model permissions
	core.InvalidateAllTokenCache()
	c.JSON(200, gin.H{"success": true})
}

func HandleDeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "无效的用户ID"}})
		return
	}
	if uint(id) == model.SuperAdminID {
		c.JSON(403, gin.H{"error": gin.H{"message": "\u65e0\u6cd5\u5220\u9664\u8d85\u7ea7\u7ba1\u7406\u5458"}})
		return
	}
	var user model.User
	if model.DB.First(&user, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "\u7528\u6237\u4e0d\u5b58\u5728"}})
		return
	}
	auth4 := middleware.GetAuth(c); if user.Role == "admin" && auth4 != nil && !auth4.IsSuper {
		c.JSON(403, gin.H{"error": gin.H{"message": "\u65e0\u6cd5\u5220\u9664\u7ba1\u7406\u5458\u7528\u6237"}})
		return
	}
	core.InvalidateUserCache(user.ID)
	model.DB.Where("user_id = ?", user.ID).Delete(&model.Token{})
	model.DB.Delete(&user)
	c.JSON(200, gin.H{"success": true})
}

func HandleRechargeUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "无效的用户ID"}})
		return
	}
	var u model.User
	if model.DB.First(&u, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "\u7528\u6237\u4e0d\u5b58\u5728"}})
		return
	}
	var req struct{ Amount int64 `json:"amount"` }
	c.ShouldBindJSON(&req)
	if req.Amount <= 0 {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u5145\u503c\u6570\u91cf\u5fc5\u987b\u5927\u4e8e0"}})
		return
	}
	if u.TokenQuota == -1 {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u7528\u6237\u989d\u5ea6\u4e3a\u65e0\u9650\uff0c\u65e0\u987b\u5145\u503c"}})
		return
	}
	// Atomic update to avoid race condition on concurrent recharge
	model.DB.Model(&model.User{}).Where("id = ?", u.ID).UpdateColumn("token_quota", gorm.Expr("token_quota + ?", req.Amount))
	core.InvalidateUserCache(u.ID)
	// Re-read user data to return accurate values
	model.DB.First(&u, id)
	c.JSON(200, gin.H{"success": true, "token_quota": core.SafeInt(u.TokenQuota), "token_quota_used": core.SafeInt(u.TokenQuotaUsed)})
}

func HandleDeductUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "无效的用户ID"}})
		return
	}
	var u model.User
	if model.DB.First(&u, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "\u7528\u6237\u4e0d\u5b58\u5728"}})
		return
	}
	var req struct{ Amount int64 `json:"amount"` }
	c.ShouldBindJSON(&req)
	if req.Amount <= 0 {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u6263\u9664\u6570\u91cf\u5fc5\u987b\u5927\u4e8e0"}})
		return
	}
	if u.TokenQuota == -1 {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u7528\u6237\u989d\u5ea6\u4e3a\u65e0\u9650\uff0c\u65e0\u6cd5\u6263\u9664"}})
		return
	}
	if u.TokenQuota-req.Amount < u.TokenQuotaUsed {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u6263\u9664\u540e\u989d\u5ea6\u4e0d\u80fd\u4f4e\u4e8e\u5df2\u7528\u91cf"}})
		return
	}
	// Atomic update with WHERE guard to prevent over-deduction under concurrency
	result := model.DB.Model(&model.User{}).
		Where("id = ? AND token_quota - ? >= token_quota_used", u.ID, req.Amount).
		UpdateColumn("token_quota", gorm.Expr("token_quota - ?", req.Amount))
	if result.RowsAffected == 0 {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u6263\u9664\u540e\u989d\u5ea6\u4e0d\u80fd\u4f4e\u4e8e\u5df2\u7528\u91cf"}})
		return
	}
	core.InvalidateUserCache(u.ID)
	// Re-read user data to return accurate values
	model.DB.First(&u, id)
	c.JSON(200, gin.H{"success": true, "token_quota": core.SafeInt(u.TokenQuota), "token_quota_used": core.SafeInt(u.TokenQuotaUsed)})
}
