package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/middleware"
	"github.com/zapi/zapi-go/internal/model"
)

func HandleListLogs(c *gin.Context) {
	q := model.DB.Model(&model.Log{})
	if df := c.Query("date_from"); df != "" {
		dfOut, _ := core.ParseDateFilters(df, "")
		if dfOut != "" { q = q.Where("created_at >= ?", dfOut) }
	}
	if dt := c.Query("date_to"); dt != "" {
		_, dtOut := core.ParseDateFilters("", dt)
		if dtOut != "" { q = q.Where("created_at < ?", dtOut) }
	}
	if m := c.Query("model"); m != "" {
		q = q.Where("model ILIKE ?", "%"+m+"%")
	}
	if s := c.Query("success"); s != "" {
		q = q.Where("success = ?", s == "1" || s == "true")
	}
	if uid := c.Query("user_id"); uid != "" {
		if id, err := strconv.ParseUint(uid, 10, 64); err == nil {
			q = q.Where("user_id = ?", id)
		}
	}
	if un := c.Query("username"); un != "" {
		var u model.User
		if model.DB.Where("username ILIKE ?", "%"+un+"%").First(&u).Error == nil {
			q = q.Where("user_id = ?", u.ID)
		} else {
			q = q.Where("1 = 0") // no match
		}
	}
	if cid := c.Query("channel_id"); cid != "" {
		if id, err := strconv.ParseUint(cid, 10, 64); err == nil {
			q = q.Where("channel_id = ?", id)
		}
	}
	if minp := c.Query("min_prompt_tokens"); minp != "" {
		if v, err := strconv.ParseInt(minp, 10, 64); err == nil { q = q.Where("prompt_tokens >= ?", v) }
	}
	if maxp := c.Query("max_prompt_tokens"); maxp != "" {
		if v, err := strconv.ParseInt(maxp, 10, 64); err == nil { q = q.Where("prompt_tokens <= ?", v) }
	}
	if minc := c.Query("min_completion_tokens"); minc != "" {
		if v, err := strconv.ParseInt(minc, 10, 64); err == nil { q = q.Where("completion_tokens >= ?", v) }
	}
	if maxc := c.Query("max_completion_tokens"); maxc != "" {
		if v, err := strconv.ParseInt(maxc, 10, 64); err == nil { q = q.Where("completion_tokens <= ?", v) }
	}
	var total int64
	q.Count(&total)
	offset, limit := middleware.GetPageParams(c)
	var logs []model.Log
	q.Order("id desc").Offset(offset).Limit(limit).Find(&logs)
	um := make(map[uint]string)
	var users []model.User
	model.DB.Select("id, username").Find(&users)
	for _, u := range users {
		um[u.ID] = u.Username
	}
	items := make([]gin.H, len(logs))
	for i, l := range logs {
		em := l.ErrorMsg
		if len(em) > 500 { em = em[:500] }
		items[i] = gin.H{
			"id": l.ID, "user_id": l.UserID, "username": um[l.UserID],
			"token_name": l.TokenName, "channel_name": l.ChannelName,
			"model": l.Model, "is_stream": l.IsStream,
			"prompt_tokens": l.PromptTokens, "completion_tokens": l.CompletionTokens,
			"cached_tokens": l.CachedTokens,
			"latency_ms": l.LatencyMs, "success": l.Success,
			"error_msg": em, "client_ip": l.ClientIP,
			"created_at": core.ToLocal(l.CreatedAt),
		}
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

// HandleMyLogs returns logs for the currently authenticated user only.
// It reuses the same filtering logic but forces user_id to the caller's ID.
func HandleMyLogs(c *gin.Context) {
	auth := middleware.GetAuth(c)
	if auth == nil {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	q := model.DB.Model(&model.Log{}).Where("user_id = ?", auth.UserID)
	if df := c.Query("date_from"); df != "" {
		dfOut, _ := core.ParseDateFilters(df, "")
		if dfOut != "" { q = q.Where("created_at >= ?", dfOut) }
	}
	if dt := c.Query("date_to"); dt != "" {
		_, dtOut := core.ParseDateFilters("", dt)
		if dtOut != "" { q = q.Where("created_at < ?", dtOut) }
	}
	if m := c.Query("model"); m != "" {
		q = q.Where("model ILIKE ?", "%"+m+"%")
	}
	if s := c.Query("success"); s != "" {
		q = q.Where("success = ?", s == "1" || s == "true")
	}
	if minp := c.Query("min_prompt_tokens"); minp != "" {
		if v, err := strconv.ParseInt(minp, 10, 64); err == nil { q = q.Where("prompt_tokens >= ?", v) }
	}
	if maxp := c.Query("max_prompt_tokens"); maxp != "" {
		if v, err := strconv.ParseInt(maxp, 10, 64); err == nil { q = q.Where("prompt_tokens <= ?", v) }
	}
	if minc := c.Query("min_completion_tokens"); minc != "" {
		if v, err := strconv.ParseInt(minc, 10, 64); err == nil { q = q.Where("completion_tokens >= ?", v) }
	}
	if maxc := c.Query("max_completion_tokens"); maxc != "" {
		if v, err := strconv.ParseInt(maxc, 10, 64); err == nil { q = q.Where("completion_tokens <= ?", v) }
	}
	var total int64
	q.Count(&total)
	offset, limit := middleware.GetPageParams(c)
	var logs []model.Log
	q.Order("id desc").Offset(offset).Limit(limit).Find(&logs)
	items := make([]gin.H, len(logs))
	for i, l := range logs {
		em := l.ErrorMsg
		if len(em) > 500 { em = em[:500] }
		items[i] = gin.H{
			"id": l.ID, "user_id": l.UserID,
			"token_name": l.TokenName, "channel_name": l.ChannelName,
			"model": l.Model, "is_stream": l.IsStream,
			"prompt_tokens": l.PromptTokens, "completion_tokens": l.CompletionTokens,
			"cached_tokens": l.CachedTokens,
			"latency_ms": l.LatencyMs, "success": l.Success,
			"error_msg": em, "client_ip": l.ClientIP,
			"created_at": core.ToLocal(l.CreatedAt),
		}
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}
