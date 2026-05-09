package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/middleware"
	"github.com/zapi/zapi-go/internal/model"
	"gorm.io/gorm"
)

func HandleListTokens(c *gin.Context) {
	limit, offset := parsePagination(c)

	var total int64
	model.DB.Model(&model.Token{}).Count(&total)

	var tokens []model.Token
	qry := model.DB.Order("id")
	if limit > 0 {
		qry = qry.Offset(offset).Limit(limit)
	}
	qry.Find(&tokens)

	userMap := make(map[uint]string)
	var users []model.User
	model.DB.Select("id, username").Find(&users)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}
	result := make([]gin.H, len(tokens))
	for i, t := range tokens {
		result[i] = gin.H{
			"id": t.ID, "user_id": t.UserID, "username": userMap[t.UserID],
			"name": t.Name, "key": t.Key, "models": t.Models,
			"enabled": t.Enabled, "quota_limit": core.SafeInt(t.QuotaLimit),
			"quota_used": core.SafeInt(t.QuotaUsed),
			"expires_at": core.FmtTimePtr(t.ExpiresAt), "created_at": core.FmtTimeVal(t.CreatedAt),
		}
	}
	if limit > 0 {
		c.JSON(200, gin.H{"items": result, "total": total})
	} else {
		c.JSON(200, result)
	}
}

func HandleListMyTokens(c *gin.Context) {
	u := getUserOrAdmin(c)
	var tokens []model.Token
	model.DB.Where("user_id = ?", u.ID).Order("id").Find(&tokens)
	result := make([]gin.H, len(tokens))
	for i, t := range tokens {
		result[i] = gin.H{
			"id": t.ID, "name": t.Name, "key": t.Key, "models": t.Models,
			"enabled": t.Enabled, "quota_limit": core.SafeInt(t.QuotaLimit),
			"quota_used": core.SafeInt(t.QuotaUsed),
			"expires_at": core.FmtTimePtr(t.ExpiresAt), "created_at": core.FmtTimeVal(t.CreatedAt),
		}
	}
	c.JSON(200, result)
}

func HandleCreateToken(c *gin.Context) {
	var req struct {
		Name       string `json:"name"`
		Models     string `json:"models"`
		QuotaLimit int64  `json:"quota_limit"`
		UserID     *uint  `json:"user_id"`
	}
	c.ShouldBindJSON(&req)
	if req.QuotaLimit < -1 {
		c.JSON(400, gin.H{"error": gin.H{"message": "quota_limit \u53ea\u80fd\u4e3a -1(\u65e0\u9650) \u6216\u6b63\u6570"}})
		return
	}
	var tu *model.User
	auth := middleware.GetAuth(c)
	isAdmin := auth != nil && auth.IsAdmin
	if isAdmin {
		uid := uint(1)
		if req.UserID != nil {
			uid = *req.UserID
		}
		var u model.User
		if model.DB.First(&u, uid).Error != nil {
			c.JSON(404, gin.H{"error": gin.H{"message": "\u7528\u6237\u4e0d\u5b58\u5728"}})
			return
		}
		tu = &u
	} else {
		tu = getUserOrAdmin(c)
	}
	var tc int64
	model.DB.Model(&model.Token{}).Where("user_id = ?", tu.ID).Count(&tc)
	if tu.MaxTokens > 0 && tc >= int64(tu.MaxTokens) {
		c.JSON(403, gin.H{"error": gin.H{"message": "\u4ee4\u724c\u6570\u91cf\u5df2\u8fbe\u4e0a\u9650"}})
		return
	}
	key := core.GenerateAPIKey()
	// Validate token models against effective allowed models (group or user)
	if req.Models != "" {
		var effectiveAllowed string
		if tu.BindMode == "custom" {
			effectiveAllowed = tu.AllowedModels
		} else if tu.GroupID != nil {
			var grp model.Group
			if model.DB.First(&grp, *tu.GroupID).Error == nil {
				effectiveAllowed = grp.AllowedModels
			}
		}
		if effectiveAllowed != "" {
			requested := core.GetModelSet(req.Models)
			allowed := core.GetModelSet(effectiveAllowed)
			for m := range requested {
				if !allowed[m] {
					c.JSON(403, gin.H{"error": gin.H{"message": fmt.Sprintf("无权使用模型: %s（超出分组/用户权限范围）", m)}})
					return
				}
			}
		}
	}
	t := model.Token{UserID: tu.ID, Name: req.Name, Key: key, Models: req.Models, QuotaLimit: req.QuotaLimit}
	model.DB.Create(&t)
	core.InvalidateTokenCache(key)
	c.JSON(200, gin.H{"success": true, "id": t.ID, "key": key})
}

func HandleUpdateToken(c *gin.Context) {
	id := c.Param("id")
	var tk model.Token
	if model.DB.First(&tk, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "\u4ee4\u724c\u4e0d\u5b58\u5728"}})
		return
	}
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	auth2 := middleware.GetAuth(c)
	isAdmin2 := auth2 != nil && auth2.IsAdmin
	if !isAdmin2 {
		u := getUserOrAdmin(c)
		if tk.UserID != u.ID {
			c.JSON(403, gin.H{"error": gin.H{"message": "\u65e0\u6743\u64cd\u4f5c\u6b64\u4ee4\u724c"}})
			return
		}
	}
	if v, ok := req["name"].(string); ok {
		tk.Name = v
	}
	if v, ok := req["models"].(string); ok {
		// Validate token models against effective allowed models (group or user)
		if v != "" {
			var tkUser model.User
			if model.DB.First(&tkUser, tk.UserID).Error == nil {
				var effectiveAllowed string
				if tkUser.BindMode == "custom" {
					effectiveAllowed = tkUser.AllowedModels
				} else if tkUser.GroupID != nil {
					var grp model.Group
					if model.DB.First(&grp, *tkUser.GroupID).Error == nil {
						effectiveAllowed = grp.AllowedModels
					}
				}
				if effectiveAllowed != "" {
					requested := core.GetModelSet(v)
					allowed := core.GetModelSet(effectiveAllowed)
					for m := range requested {
						if !allowed[m] {
							c.JSON(403, gin.H{"error": gin.H{"message": fmt.Sprintf("无权使用模型: %s（超出分组/用户权限范围）", m)}})
							return
						}
					}
				}
			}
		}
		tk.Models = v
	}
	if v, ok := req["enabled"].(bool); ok {
		tk.Enabled = v
	}
	if v, ok := req["quota_limit"].(float64); ok {
		if int64(v) < -1 {
			c.JSON(400, gin.H{"error": gin.H{"message": "quota_limit \u53ea\u80fd\u4e3a -1(\u65e0\u9650) \u6216\u6b63\u6570"}})
			return
		}
		tk.QuotaLimit = int64(v)
	}
	model.DB.Save(&tk)
	core.InvalidateTokenCache(tk.Key)
	c.JSON(200, gin.H{"success": true})
}

func HandleRechargeToken(c *gin.Context) {
	id := c.Param("id")
	var tk model.Token
	if model.DB.First(&tk, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "\u4ee4\u724c\u4e0d\u5b58\u5728"}})
		return
	}
	var req struct {
		Amount int64 `json:"amount"`
	}
	c.ShouldBindJSON(&req)
	if req.Amount <= 0 {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u5145\u503c\u6570\u91cf\u5fc5\u987b\u5927\u4e8e0"}})
		return
	}
	if tk.QuotaLimit == -1 {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u4ee4\u724c\u989d\u5ea6\u4e3a\u65e0\u9650\uff0c\u65e0\u9700\u5145\u503c"}})
		return
	}
	// Atomic update to avoid race condition on concurrent recharge
	model.DB.Model(&model.Token{}).Where("id = ?", tk.ID).UpdateColumn("quota_limit", gorm.Expr("quota_limit + ?", req.Amount))
	core.InvalidateTokenCache(tk.Key)
	// Re-read token data to return accurate values
	model.DB.First(&tk, id)
	c.JSON(200, gin.H{"success": true, "quota_limit": core.SafeInt(tk.QuotaLimit), "quota_used": core.SafeInt(tk.QuotaUsed)})
}

func HandleDeleteToken(c *gin.Context) {
	id := c.Param("id")
	var tk model.Token
	if model.DB.First(&tk, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "\u4ee4\u724c\u4e0d\u5b58\u5728"}})
		return
	}
	auth3 := middleware.GetAuth(c)
	if auth3 == nil || !auth3.IsAdmin {
		u := getUserOrAdmin(c)
		if tk.UserID != u.ID {
			c.JSON(403, gin.H{"error": gin.H{"message": "\u65e0\u6743\u64cd\u4f5c\u6b64\u4ee4\u724c"}})
			return
		}
	}
	core.InvalidateTokenCache(tk.Key)
	model.DB.Delete(&tk)
	c.JSON(200, gin.H{"success": true})
}
