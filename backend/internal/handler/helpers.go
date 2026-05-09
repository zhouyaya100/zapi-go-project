package handler

import (
	"github.com/gin-gonic/gin"
	"fmt"
	"strconv"
	"time"

	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/core/routing"
	"github.com/zapi/zapi-go/internal/middleware"
	"github.com/zapi/zapi-go/internal/ratelimit"
	"github.com/zapi/zapi-go/internal/model"
)

// getUserOrAdmin returns the user for JWT auth, or the super admin for admin-token auth
func getUserOrAdmin(c *gin.Context) *model.User {
	a := middleware.GetAuth(c)
	if a != nil && a.IsAdmin {
		if u, ok := core.CachedLookupUser(model.SuperAdminID); ok { return u }
		var user model.User
		model.DB.First(&user, model.SuperAdminID)
		return &user
	}
	u, ok := c.Get("user")
	if !ok {
		// Fallback: return super admin to avoid panic
		if u, ok := core.CachedLookupUser(model.SuperAdminID); ok { return u }
		var user model.User
		model.DB.First(&user, model.SuperAdminID)
		return &user
	}
	return u.(*model.User)
}

func HandleVersion(c *gin.Context) {
	c.JSON(200, gin.H{"version": "4.5.9"})
}

func HandleErrorLog(c *gin.Context) {
	entries := core.ErrLog.GetEntries(0)
	items := make([]gin.H, len(entries))
	for i, e := range entries {
		items[i] = gin.H{"time": e.Time, "type": "error", "message": e.Message}
	}
	c.JSON(200, gin.H{"items": items})
}

func HandleMyModels(c *gin.Context) {
	u := getUserOrAdmin(c)
	if u.ID == model.SuperAdminID || u.Role == "admin" {
		models := routing.Pool.GetAllEnabledModels()
		c.JSON(200, gin.H{"models": models})
		return
	}
	gn := ""
	if u.GroupID != nil {
		if g, ok := core.CachedLookupGroup(*u.GroupID); ok {
			gn = g.Name
		}
	}
	// Resolve effective allowed_models: inherit mode uses group's, custom mode uses user's
	effectiveModels := u.AllowedModels
	if u.BindMode != "custom" && u.GroupID != nil {
		var grp model.Group
		if model.DB.First(&grp, *u.GroupID).Error == nil {
			effectiveModels = grp.AllowedModels
		}
	}
	models := core.GetGroupAuthedModels(gn, effectiveModels, routing.Pool)
	c.JSON(200, gin.H{"models": models})
}

func HandleMyDashboard(c *gin.Context) {
	u := getUserOrAdmin(c)
	var tr, sr, tp, tcm int64
	model.DB.Model(&model.Log{}).Where("user_id = ?", u.ID).Select("count(id), sum(case when success then 1 else 0 end), coalesce(sum(prompt_tokens),0), coalesce(sum(completion_tokens),0)").Row().Scan(&tr, &sr, &tp, &tcm)
	var tc int64
	model.DB.Model(&model.Token{}).Where("user_id = ? AND enabled = ?", u.ID, true).Count(&tc)

	// Recent 24h
	since := time.Now().UTC().Add(-24 * time.Hour)
	var r24hReq, r24hTok int64
	model.DB.Model(&model.Log{}).Where("user_id = ? AND created_at >= ?", u.ID, since).Count(&r24hReq)
	model.DB.Model(&model.Log{}).Where("user_id = ? AND created_at >= ?", u.ID, since).Select("coalesce(sum(prompt_tokens+completion_tokens),0)").Row().Scan(&r24hTok)

	// Resolve rate limits using unified logic
	rl := ratelimit.ResolveRateLimits(u)

	// Authed models — resolve effective allowed_models based on bind_mode
	effectiveUserModels := u.AllowedModels
	if u.BindMode != "custom" && u.GroupID != nil {
		var grp model.Group
		if model.DB.First(&grp, *u.GroupID).Error == nil {
			effectiveUserModels = grp.AllowedModels
		}
	}
	authedModels := []string{}
	if rl.GroupName != "" {
		authedModels = core.GetGroupAuthedModels(rl.GroupName, effectiveUserModels, routing.Pool)
	}

	// Recent logs (last 20)
	var recentLogs []model.Log
	model.DB.Where("user_id = ?", u.ID).Order("id desc").Limit(20).Find(&recentLogs)
	logItems := make([]gin.H, len(recentLogs))
	for i, l := range recentLogs {
		logItems[i] = gin.H{"model": l.Model, "prompt_tokens": l.PromptTokens, "completion_tokens": l.CompletionTokens, "latency_ms": l.LatencyMs, "success": l.Success, "created_at": core.ToLocal(l.CreatedAt)}
	}

	// Model stats
	type msRow struct{ Model string; Count int64; AvgLatency float64 }
	var msRows []msRow
	model.DB.Model(&model.Log{}).Where("user_id = ?", u.ID).Select("model, count(*) as count, coalesce(avg(latency_ms),0) as avg_latency").Group("model").Order("count desc").Limit(10).Scan(&msRows)
	ms := make([]gin.H, len(msRows))
	for i, r := range msRows {
		ms[i] = gin.H{"model": r.Model, "count": r.Count, "avg_latency": int(r.AvgLatency)}
	}

	// Build model_limits for response
	modelLimits := make(map[string]interface{})
	if u.RateMode == "inherit" && (rl.EffectiveMode == "inherit_per_model") {
		// Inherit per_model: show all group model limits
		for m, e := range rl.ModelLimits {
			modelLimits[m] = gin.H{"rpm": e.RPM, "tpm": e.TPM}
		}
	} else {
		for _, m := range authedModels {
			mr, mt := rl.ResolveModelLimit(m)
			rpm := rl.RPM; if mr != 0 { rpm = mr }
			tpm := rl.TPM; if mt != 0 { tpm = mt }
			if rpm != 0 || tpm != 0 {
				modelLimits[m] = gin.H{"rpm": rpm, "tpm": tpm}
			}
		}
	}

	// Determine display rate_mode
	displayRateMode := u.RateMode
	if rl.EffectiveMode == "admin" { displayRateMode = "admin" }

	c.JSON(200, gin.H{
		"username":            u.Username,
		"token_count":         tc,
		"max_tokens":          u.MaxTokens,
		"token_quota":         core.SafeInt(u.TokenQuota),
		"token_quota_used":    core.SafeInt(u.TokenQuotaUsed),
		"group_name":          rl.GroupName,
		"authorized_models":   authedModels,
		"total_requests":      tr,
		"success_requests":    sr,
		"total_prompt_tokens":  core.SafeInt(tp),
		"total_completion_tokens": core.SafeInt(tcm),
		"total_tokens":        core.SafeInt(tp + tcm),
		"rate_mode":           displayRateMode,
		"rpm":                 rl.RPM,
		"tpm":                 rl.TPM,
		"model_limits":        modelLimits,
		"recent_24h_requests": r24hReq,
		"recent_24h_tokens":   core.SafeInt(r24hTok),
		"model_stats":         ms,
		"recent_logs":         logItems,
	})
}

func HandleMyUsage(c *gin.Context) {
	u := getUserOrAdmin(c)
	q := model.DB.Model(&model.Log{}).Where("user_id = ?", u.ID)
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1")); if page < 1 { page = 1 }
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50")); if pageSize < 1 { pageSize = 50 }; if pageSize > 500 { pageSize = 500 }
	order := c.DefaultQuery("order", "desc")
	group_by := c.DefaultQuery("group_by", "day")
	// Build summary
	var sReq, sSucc, sPT, sCT int64; var sAvgLat float64
	q.Select("count(id), sum(case when success then 1 else 0 end), coalesce(sum(prompt_tokens),0), coalesce(sum(completion_tokens),0), coalesce(avg(latency_ms),0)").Row().Scan(&sReq, &sSucc, &sPT, &sCT, &sAvgLat)
	summary := gin.H{"total_requests": sReq, "success_requests": sSucc, "total_prompt_tokens": core.SafeInt(sPT), "total_completion_tokens": core.SafeInt(sCT), "total_tokens": core.SafeInt(sPT+sCT), "avg_latency_ms": int(sAvgLat)}
	switch group_by {
	case "model":
		type row struct{ Model string; Requests int64; Success int64; PromptTokens int64; CompletionTokens int64; AvgLatencyMs float64 }
		var total int64
		model.DB.Model(&model.Log{}).Where("user_id = ?", u.ID).Group("model").Count(&total)
		var rows []row
		q.Select("model, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group("model").Order("requests desc").Offset((page-1)*pageSize).Limit(pageSize).Scan(&rows)
		items := make([]gin.H, len(rows))
		for i, r := range rows { fail:=r.Requests-r.Success; sr:=0.0; if r.Requests>0{sr=float64(r.Success)/float64(r.Requests)*100}; items[i] = gin.H{"key":r.Model,"requests":r.Requests,"success":r.Success,"fail":fail,"success_rate":fmt.Sprintf("%.1f%%",sr),"prompt_tokens":core.SafeInt(r.PromptTokens),"completion_tokens":core.SafeInt(r.CompletionTokens),"total_tokens":core.SafeInt(r.PromptTokens+r.CompletionTokens),"avg_latency_ms":int(r.AvgLatencyMs)} }
		c.JSON(200, gin.H{"summary":summary,"group_by":"model","items":items,"total":total,"page":page,"page_size":pageSize})
	default:
		type row struct{ Period string; Requests int64; Success int64; PromptTokens int64; CompletionTokens int64; AvgLatencyMs float64 }
		var total int64
		model.DB.Model(&model.Log{}).Where("user_id = ?", u.ID).Group(core.TzDateExpr()).Count(&total)
		var rows []row
		dq := q.Select(core.TzDateExpr()+" as period, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group(core.TzDateExpr())
		if order == "asc" { dq = dq.Order("period") } else { dq = dq.Order("period desc") }
		dq.Offset((page-1)*pageSize).Limit(pageSize).Scan(&rows)
		items := make([]gin.H, len(rows))
		for i, r := range rows {
			key := r.Period; if len(key) > 10 { key = key[:10] }; fail:=r.Requests-r.Success; sr:=0.0; if r.Requests>0{sr=float64(r.Success)/float64(r.Requests)*100}
			items[i] = gin.H{"key":key,"requests":r.Requests,"success":r.Success,"fail":fail,"success_rate":fmt.Sprintf("%.1f%%",sr),"prompt_tokens":core.SafeInt(r.PromptTokens),"completion_tokens":core.SafeInt(r.CompletionTokens),"total_tokens":core.SafeInt(r.PromptTokens+r.CompletionTokens),"avg_latency_ms":int(r.AvgLatencyMs)}
		}
		c.JSON(200, gin.H{"summary":summary,"group_by":"day","items":items,"total":total,"page":page,"page_size":pageSize})
	}
}
