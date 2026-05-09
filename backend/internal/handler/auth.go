package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/config"
	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/core/routing"
	"github.com/zapi/zapi-go/internal/middleware"
	"github.com/zapi/zapi-go/internal/model"
)

func HandleCaptcha(c *gin.Context) {
	id, imgBytes := core.GenerateCaptcha()
	c.Header("X-Captcha-Id", id)
	c.Data(200, "image/png", imgBytes)
}

func HandleRegister(c *gin.Context) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		CaptchaID   string `json:"captcha_id"`
		CaptchaCode string `json:"captcha_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u8bf7\u6c42\u53c2\u6570\u9519\u8bef"}})
		return
	}
	if req.CaptchaID == "" || req.CaptchaCode == "" {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u8bf7\u8f93\u5165\u9a8c\u8bc1\u7801"}})
		return
	}
	if !core.VerifyCaptcha(req.CaptchaID, req.CaptchaCode) {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u9a8c\u8bc1\u7801\u9519\u8bef\u6216\u5df2\u8fc7\u671f"}})
		return
	}
	if !config.Cfg.Registration.AllowRegister {
		c.JSON(403, gin.H{"error": gin.H{"message": "\u6ce8\u518c\u5df2\u5173\u95ed\uff0c\u8bf7\u8054\u7cfb\u7ba1\u7406\u5458"}})
		return
	}
	username := strings.TrimSpace(req.Username)
	if len(username) < 2 || len(username) > 32 {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u7528\u6237\u540d\u9700 2-32 \u4e2a\u5b57\u7b26"}})
		return
	}
	if msg := core.ValidateUsername(username); msg != "" {
		c.JSON(400, gin.H{"error": gin.H{"message": msg}})
		return
	}
	if msg := core.ValidatePasswordStrength(req.Password); msg != "" {
		c.JSON(400, gin.H{"error": gin.H{"message": msg}})
		return
	}
	var existing model.User
	if model.DB.Where("username = ?", username).First(&existing).Error == nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u7528\u6237\u540d\u5df2\u5b58\u5728"}})
		return
	}
	var groupID *uint
	var grp model.Group
	if config.Cfg.Registration.DefaultGroup != "" && model.DB.Where("name = ?", config.Cfg.Registration.DefaultGroup).First(&grp).Error == nil {
		groupID = &grp.ID
	}
	user := model.User{
		Username:    username,
		Role:        "user",
		GroupID:     groupID,
		MaxTokens:   config.Cfg.Registration.DefaultMaxTokens,
		TokenQuota:  config.Cfg.Registration.DefaultTokenQuota,
	}
	hash, err := core.HashPassword(req.Password)
	if err != nil { c.JSON(500, gin.H{"error": gin.H{"message": "密码处理失败"}}); return }
	user.PasswordHash = hash
	if err := model.DB.Create(&user).Error; err != nil {
		c.JSON(500, gin.H{"error": gin.H{"message": "注册失败，请稍后重试"}})
		return
	}
	core.InvalidateUserCache(user.ID)
	jwtStr, err := core.CreateJWT(user.ID)
	if err != nil { c.JSON(500, gin.H{"error": gin.H{"message": "JWT创建失败"}}); return }
	c.JSON(200, gin.H{
		"success": true,
		"user":    gin.H{"id": user.ID, "username": user.Username, "role": user.Role},
		"token":   jwtStr,
	})
}

func HandleLogin(c *gin.Context) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		CaptchaID   string `json:"captcha_id"`
		CaptchaCode string `json:"captcha_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u8bf7\u6c42\u53c2\u6570\u9519\u8bef"}})
		return
	}
	username := strings.TrimSpace(req.Username)
	clientIP := c.ClientIP()
	if err := core.CheckLoginRate(strings.ToLower(username)); err != nil {
		c.JSON(429, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if err := core.CheckLoginRate("ip:" + clientIP); err != nil {
		c.JSON(429, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if req.CaptchaID == "" || req.CaptchaCode == "" {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u8bf7\u8f93\u5165\u9a8c\u8bc1\u7801"}})
		return
	}
	if !core.VerifyCaptcha(req.CaptchaID, req.CaptchaCode) {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u9a8c\u8bc1\u7801\u9519\u8bef\u6216\u5df2\u8fc7\u671f"}})
		return
	}
	var user model.User
	if model.DB.Where("username = ?", username).First(&user).Error != nil || !core.CheckPassword(user.PasswordHash, req.Password) {
		core.RecordLoginFailure(strings.ToLower(username))
		core.RecordLoginFailure("ip:" + clientIP)
		c.JSON(401, gin.H{"error": gin.H{"message": "\u7528\u6237\u540d\u6216\u5bc6\u7801\u9519\u8bef", "type": "invalid_request_error", "code": "invalid_credentials"}})
		return
	}
	core.RecordLoginSuccess(strings.ToLower(username))
	core.RecordLoginSuccess("ip:" + clientIP)
	if !user.Enabled {
		c.JSON(403, gin.H{"error": gin.H{"message": "\u8d26\u53f7\u5df2\u88ab\u7981\u7528"}})
		return
	}
	jwtStr, err := core.CreateJWT(user.ID)
	if err != nil { c.JSON(500, gin.H{"error": gin.H{"message": "JWT创建失败"}}); return }
	c.JSON(200, gin.H{
		"success": true,
		"user": gin.H{
			"id":              user.ID,
			"username":        user.Username,
			"role":            user.Role,
			"max_tokens":      user.MaxTokens,
			"allowed_models":  user.AllowedModels,
			"token_quota":     core.SafeInt(user.TokenQuota),
			"token_quota_used": core.SafeInt(user.TokenQuotaUsed),
			"is_super":        user.ID == model.SuperAdminID,
		},
		"token": jwtStr,
	})
}

func HandleGetMe(c *gin.Context) {
	a := middleware.GetAuth(c)
	if a != nil && a.IsAdmin {
		var user *model.User
		if u, ok := core.CachedLookupUser(model.SuperAdminID); ok { user = u } else {
			var u2 model.User; model.DB.First(&u2, model.SuperAdminID); user = &u2
		}
		var tc int64
		model.DB.Model(&model.Token{}).Where("user_id = ?", user.ID).Count(&tc)
		c.JSON(200, gin.H{
			"id": user.ID, "username": user.Username, "role": user.Role,
			"enabled": user.Enabled, "max_tokens": user.MaxTokens,
			"allowed_models": user.AllowedModels,
			"token_quota": core.SafeInt(user.TokenQuota),
			"token_quota_used": core.SafeInt(user.TokenQuotaUsed),
			"token_count": tc, "can_create_token": tc < int64(user.MaxTokens),
			"group_id": user.GroupID, "is_super": true,
		})
		return
	}
	u := c.MustGet("user").(*model.User)
	var tc int64
	model.DB.Model(&model.Token{}).Where("user_id = ?", u.ID).Count(&tc)
	gn := ""
	authedModels := []string{}
	if u.GroupID != nil {
		if g, ok := core.CachedLookupGroup(*u.GroupID); ok {
			gn = g.Name
			authedModels = core.GetGroupAuthedModels(gn, u.AllowedModels, routing.Pool)
		}
	}
	c.JSON(200, gin.H{
		"id":              u.ID,
		"username":        u.Username,
		"role":            u.Role,
		"enabled":         u.Enabled,
		"max_tokens":      u.MaxTokens,
		"allowed_models":  u.AllowedModels,
		"authed_models":   authedModels,
		"token_quota":     core.SafeInt(u.TokenQuota),
		"token_quota_used": core.SafeInt(u.TokenQuotaUsed),
		"token_count":     tc,
		"can_create_token": tc < int64(u.MaxTokens),
		"group_id":        u.GroupID,
		"group_name":      gn,
		"is_super":        u.ID == model.SuperAdminID,
	})
}

func HandleChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u8bf7\u6c42\u53c2\u6570\u9519\u8bef"}})
		return
	}
	a := middleware.GetAuth(c)
	var u *model.User
	if a != nil && a.IsAdmin {
		var user model.User
		model.DB.First(&user, model.SuperAdminID)
		u = &user
	} else {
		u = c.MustGet("user").(*model.User)
	}
	if !core.CheckPassword(u.PasswordHash, req.OldPassword) {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u539f\u5bc6\u7801\u9519\u8bef"}})
		return
	}
	if msg := core.ValidatePasswordStrength(req.NewPassword); msg != "" {
		c.JSON(400, gin.H{"error": gin.H{"message": msg}})
		return
	}
	if req.NewPassword == req.OldPassword {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u65b0\u5bc6\u7801\u4e0d\u80fd\u4e0e\u539f\u5bc6\u7801\u76f8\u540c"}})
		return
	}
	hash, err := core.HashPassword(req.NewPassword)
	if err != nil { c.JSON(500, gin.H{"error": gin.H{"message": "密码处理失败"}}); return }
	model.DB.Model(u).Update("password_hash", hash)
	core.InvalidateUserCache(u.ID)
	c.JSON(200, gin.H{"success": true})
}

// HandleRefreshToken — refresh a JWT token (returns a new token with extended expiry)
func HandleRefreshToken(c *gin.Context) {
	a := middleware.GetAuth(c)
	if a == nil {
		c.JSON(401, gin.H{"error": gin.H{"message": "未认证"}})
		return
	}
	// Verify user still exists and is enabled
	var user model.User
	if model.DB.First(&user, a.UserID).Error != nil {
		c.JSON(401, gin.H{"error": gin.H{"message": "用户不存在"}})
		return
	}
	if !user.Enabled {
		c.JSON(403, gin.H{"error": gin.H{"message": "账号已被禁用"}})
		return
	}
	jwtStr, err := core.CreateJWT(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": gin.H{"message": "JWT创建失败"}})
		return
	}
	c.JSON(200, gin.H{
		"token": jwtStr,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}
