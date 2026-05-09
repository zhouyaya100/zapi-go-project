package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/middleware"
	"github.com/zapi/zapi-go/internal/model"
)

func HandleStats(c *gin.Context) {
	var tr, sr2, tp, tcm int64
	var avgLat float64
	var tct, tuct int64 // total_cached_tokens, total_uncached_tokens
	model.DB.Model(&model.Log{}).Select("count(id), sum(case when success then 1 else 0 end), coalesce(sum(prompt_tokens),0), coalesce(sum(completion_tokens),0), coalesce(avg(latency_ms),0), coalesce(sum(cached_tokens),0), coalesce(sum(prompt_tokens) - sum(cached_tokens),0)").Row().Scan(&tr, &sr2, &tp, &tcm, &avgLat, &tct, &tuct)
	var chTotal, chEnabled, uTotal, uEnabled, tkTotal, tkEnabled int64
	model.DB.Model(&model.Channel{}).Count(&chTotal); model.DB.Model(&model.Channel{}).Where("enabled = ?", true).Count(&chEnabled)
	model.DB.Model(&model.User{}).Count(&uTotal); model.DB.Model(&model.User{}).Where("enabled = ?", true).Count(&uEnabled)
	model.DB.Model(&model.Token{}).Count(&tkTotal); model.DB.Model(&model.Token{}).Where("enabled = ?", true).Count(&tkEnabled)
	// Recent 24h stats
	since := time.Now().UTC().Add(-24 * time.Hour)
	var r24hReq, r24hTok int64
	model.DB.Model(&model.Log{}).Where("created_at >= ?", since).Count(&r24hReq)
	model.DB.Model(&model.Log{}).Where("created_at >= ?", since).Select("coalesce(sum(prompt_tokens+completion_tokens),0)").Row().Scan(&r24hTok)
	c.JSON(200, gin.H{
		"total_requests": tr, "success_requests": sr2,
		"total_prompt_tokens": core.SafeInt(tp), "total_completion_tokens": core.SafeInt(tcm),
		"total_tokens": core.SafeInt(tp + tcm), "avg_latency_ms": int(avgLat),
		"total_cached_tokens": core.SafeInt(tct), "total_uncached_tokens": core.SafeInt(tuct),
		"channels": chTotal, "channels_enabled": chEnabled,
		"users": uTotal, "users_enabled": uEnabled,
		"tokens": tkTotal, "tokens_enabled": tkEnabled,
		"recent_24h_requests": r24hReq, "recent_24h_tokens": core.SafeInt(r24hTok),
	})
}

func HandleDashboard(c *gin.Context) {
	u := getUserOrAdmin(c)
	// Recent logs (all users for admin/operator, own logs for regular user)
	var recentLogs []model.Log
	if u.ID == model.SuperAdminID || u.Role == "admin" || u.Role == "operator" {
		model.DB.Order("id desc").Limit(10).Find(&recentLogs)
	} else {
		model.DB.Where("user_id = ?", u.ID).Order("id desc").Limit(10).Find(&recentLogs)
	}
	rl := make([]gin.H, len(recentLogs))
	for i, l := range recentLogs {
		rl[i] = gin.H{"model": l.Model, "latency_ms": l.LatencyMs, "success": l.Success, "created_at": core.ToLocal(l.CreatedAt)}
	}
	// Model stats from logs (all users for admin/operator)
	logQuery := model.DB.Model(&model.Log{})
	if !(u.ID == model.SuperAdminID || u.Role == "admin" || u.Role == "operator") {
		logQuery = logQuery.Where("user_id = ?", u.ID)
	}
	type msRow struct{ Model string; Count int64; AvgLatency float64 }
	var msRows []msRow
	logQuery.Select("model, count(*) as count, coalesce(avg(latency_ms),0) as avg_latency").Group("model").Order("count desc").Limit(10).Scan(&msRows)
	ms := make([]gin.H, len(msRows))
	for i, r := range msRows {
		ms[i] = gin.H{"model": r.Model, "count": r.Count, "avg_latency": int(r.AvgLatency)}
	}
	// User info
	var tc int64
	model.DB.Model(&model.Token{}).Where("user_id = ? AND enabled = ?", u.ID, true).Count(&tc)
	gn := ""
	if u.GroupID != nil {
		if g, ok := core.CachedLookupGroup(*u.GroupID); ok { gn = g.Name }
	}

	isAdminOrOp := u.ID == model.SuperAdminID || u.Role == "admin" || u.Role == "operator"

	// Personal stats (current user's own usage)
	var ptr, psr, ptp, ptcm int64
	model.DB.Model(&model.Log{}).Where("user_id = ?", u.ID).Select("count(id), sum(case when success then 1 else 0 end), coalesce(sum(prompt_tokens),0), coalesce(sum(completion_tokens),0)").Row().Scan(&ptr, &psr, &ptp, &ptcm)

	result := gin.H{
		"recent_logs": rl, "model_stats": ms,
		"rpm": -1, "tpm": -1, "rate_mode": "admin", "model_limits": map[string]interface{}{},
		"token_count": tc, "max_tokens": u.MaxTokens,
		"token_quota": core.SafeInt(u.TokenQuota), "token_quota_used": core.SafeInt(u.TokenQuotaUsed),
		"group_name": gn, "authorized_models": []string{},
		// Personal stats
		"total_requests": ptr, "success_requests": psr,
		"total_prompt_tokens": core.SafeInt(ptp), "total_completion_tokens": core.SafeInt(ptcm),
	}

	// For admin/operator: add platform-wide stats
	if isAdminOrOp {
		var pltr, plsr, pltp, pltcm int64
		model.DB.Model(&model.Log{}).Select("count(id), sum(case when success then 1 else 0 end), coalesce(sum(prompt_tokens),0), coalesce(sum(completion_tokens),0)").Row().Scan(&pltr, &plsr, &pltp, &pltcm)
		result["platform_total_requests"] = pltr
		result["platform_success_requests"] = plsr
		result["platform_total_prompt_tokens"] = core.SafeInt(pltp)
		result["platform_total_completion_tokens"] = core.SafeInt(pltcm)
		result["platform_total_tokens"] = core.SafeInt(pltp + pltcm)
	}

	c.JSON(200, result)
}

func HandleUsageStats(c *gin.Context) {
	groupBy := c.DefaultQuery("group_by", "day")
	order := c.DefaultQuery("order", "desc")
	page, pageSize := middleware.GetPageParams(c)
	q := model.DB.Model(&model.Log{})
	df := c.Query("date_from")
	dt := c.Query("date_to")
	dfOut, dtOut := core.ParseDateFilters(df, dt)
	if dfOut != "" {
		q = q.Where("created_at >= ?", dfOut)
	}
	if dtOut != "" {
		q = q.Where("created_at < ?", dtOut)
	}
	if m := c.Query("model"); m != "" {
		q = q.Where("model ILIKE ?", "%"+m+"%")
	}

	// Summary (using same filtered query)
	var tr, sr2, tp, tcm int64
	var avgLat float64
	var tct, tuct int64
	q.Select("count(id), sum(case when success then 1 else 0 end), coalesce(sum(prompt_tokens),0), coalesce(sum(completion_tokens),0), coalesce(avg(latency_ms),0), coalesce(sum(cached_tokens),0), coalesce(sum(prompt_tokens) - sum(cached_tokens),0)").Row().Scan(&tr, &sr2, &tp, &tcm, &avgLat, &tct, &tuct)
	summary := gin.H{
		"total_requests": tr, "success_requests": sr2,
		"total_prompt_tokens": core.SafeInt(tp), "total_completion_tokens": core.SafeInt(tcm),
		"total_tokens": core.SafeInt(tp + tcm), "avg_latency_ms": int(avgLat),
		"total_cached_tokens": core.SafeInt(tct), "total_uncached_tokens": core.SafeInt(tuct),
	}

	switch groupBy {
	case "day":
		type row struct {
			Period          string
			Requests        int64
			Success         int64
			PromptTokens    int64
			CompletionTokens int64
			CachedTokens    int64
			AvgLatencyMs    float64
		}
		var rows []row
		dq := q.Select(core.TzDateExpr() + " as period, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(sum(cached_tokens),0) as cached_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group(core.TzDateExpr())
		if order == "asc" {
			dq = dq.Order("period")
		} else {
			dq = dq.Order("period desc")
		}
		dq.Scan(&rows)
		items := make([]gin.H, len(rows))
		for i, r := range rows {
			key := r.Period
			if len(key) > 10 {
				key = key[:10]
			}
			items[i] = gin.H{"key": key, "requests": r.Requests, "success": r.Success, "prompt_tokens": core.SafeInt(r.PromptTokens), "completion_tokens": core.SafeInt(r.CompletionTokens), "cached_tokens": core.SafeInt(r.CachedTokens), "uncached_tokens": core.SafeInt(r.PromptTokens - r.CachedTokens), "fail": r.Requests-r.Success, "success_rate": fmt.Sprintf("%.1f%%", float64(r.Success)/float64(max(r.Requests,1))*100), "total_tokens": core.SafeInt(r.PromptTokens+r.CompletionTokens), "avg_latency_ms": int(r.AvgLatencyMs)}
		}
		c.JSON(200, gin.H{"summary": summary, "items": items, "total": len(rows), "page": page, "page_size": pageSize})

	case "model":
		type row struct {
			Model           string
			Requests        int64
			Success         int64
			PromptTokens    int64
			CompletionTokens int64
			CachedTokens    int64
			AvgLatencyMs    float64
		}
		var rows []row
		q.Select("model, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(sum(cached_tokens),0) as cached_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group("model").Order("requests desc").Scan(&rows)
		items := make([]gin.H, len(rows))
		for i, r := range rows {
			items[i] = gin.H{"key": r.Model, "requests": r.Requests, "success": r.Success, "prompt_tokens": core.SafeInt(r.PromptTokens), "completion_tokens": core.SafeInt(r.CompletionTokens), "cached_tokens": core.SafeInt(r.CachedTokens), "uncached_tokens": core.SafeInt(r.PromptTokens - r.CachedTokens), "fail": r.Requests-r.Success, "success_rate": fmt.Sprintf("%.1f%%", float64(r.Success)/float64(max(r.Requests,1))*100), "total_tokens": core.SafeInt(r.PromptTokens+r.CompletionTokens), "avg_latency_ms": int(r.AvgLatencyMs)}
		}
		c.JSON(200, gin.H{"summary": summary, "items": items, "total": len(rows), "page": page, "page_size": pageSize})

	case "user":
		type row struct {
			UserID          uint
			Requests        int64
			Success         int64
			PromptTokens    int64
			CompletionTokens int64
			CachedTokens    int64
			AvgLatencyMs    float64
		}
		var rows []row
		q.Select("user_id, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(sum(cached_tokens),0) as cached_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group("user_id").Order("requests desc").Scan(&rows)
		um := make(map[uint]string)
		var users []model.User
		model.DB.Select("id, username").Find(&users)
		for _, u := range users {
			um[u.ID] = u.Username
		}
		items := make([]gin.H, len(rows))
		for i, r := range rows {
			key := um[r.UserID]
			if key == "" {
				key = fmt.Sprintf("user:%d", r.UserID)
			}
			items[i] = gin.H{"key": key, "requests": r.Requests, "success": r.Success, "prompt_tokens": core.SafeInt(r.PromptTokens), "completion_tokens": core.SafeInt(r.CompletionTokens), "cached_tokens": core.SafeInt(r.CachedTokens), "uncached_tokens": core.SafeInt(r.PromptTokens - r.CachedTokens), "fail": r.Requests-r.Success, "success_rate": fmt.Sprintf("%.1f%%", float64(r.Success)/float64(max(r.Requests,1))*100), "total_tokens": core.SafeInt(r.PromptTokens+r.CompletionTokens), "avg_latency_ms": int(r.AvgLatencyMs)}
		}
		c.JSON(200, gin.H{"summary": summary, "items": items, "total": len(rows), "page": page, "page_size": pageSize})

	case "channel":
		type row struct {
			ChannelID       uint
			Requests        int64
			Success         int64
			PromptTokens    int64
			CompletionTokens int64
			CachedTokens    int64
			AvgLatencyMs    float64
		}
		var rows []row
		q.Select("channel_id, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(sum(cached_tokens),0) as cached_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group("channel_id").Order("requests desc").Scan(&rows)
		cm := make(map[uint]string)
		var chs []model.Channel
		model.DB.Select("id, name").Find(&chs)
		for _, ch := range chs {
			cm[ch.ID] = ch.Name
		}
		items := make([]gin.H, len(rows))
		for i, r := range rows {
			key := cm[r.ChannelID]
			if key == "" {
				key = fmt.Sprintf("channel:%d", r.ChannelID)
			}
			items[i] = gin.H{"key": key, "requests": r.Requests, "success": r.Success, "prompt_tokens": core.SafeInt(r.PromptTokens), "completion_tokens": core.SafeInt(r.CompletionTokens), "cached_tokens": core.SafeInt(r.CachedTokens), "uncached_tokens": core.SafeInt(r.PromptTokens - r.CachedTokens), "fail": r.Requests-r.Success, "success_rate": fmt.Sprintf("%.1f%%", float64(r.Success)/float64(max(r.Requests,1))*100), "total_tokens": core.SafeInt(r.PromptTokens+r.CompletionTokens), "avg_latency_ms": int(r.AvgLatencyMs)}
		}
		c.JSON(200, gin.H{"summary": summary, "items": items, "total": len(rows), "page": page, "page_size": pageSize})

	case "detail":
		type row struct {
			UserID          uint
			Model           string
			ChannelID       uint
			Requests        int64
			Success         int64
			PromptTokens    int64
			CompletionTokens int64
			CachedTokens    int64
			AvgLatencyMs    float64
		}
		var total int64
		countQ := model.DB.Model(&model.Log{})
		if dfOut != "" { countQ = countQ.Where("created_at >= ?", dfOut) }
		if dtOut != "" { countQ = countQ.Where("created_at < ?", dtOut) }
		countQ.Count(&total)
		var rows []row
		q.Select("user_id, model, channel_id, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(sum(cached_tokens),0) as cached_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group("user_id, model, channel_id").Order("requests desc").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows)
		um := make(map[uint]string)
		cm := make(map[uint]string)
		var users []model.User; model.DB.Select("id, username").Find(&users); for _, u := range users { um[u.ID] = u.Username }
		var chs []model.Channel; model.DB.Select("id, name").Find(&chs); for _, ch := range chs { cm[ch.ID] = ch.Name }
		items := make([]gin.H, len(rows))
		for i, r := range rows {
			uName := um[r.UserID]; if uName == "" { uName = "-" }
			cName := cm[r.ChannelID]; if cName == "" { cName = "-" }
			items[i] = gin.H{
				"key": uName + " / " + r.Model + " / " + cName,
				"user": uName, "model": r.Model, "channel": cName,
				"requests": r.Requests, "success": r.Success,
				"prompt_tokens": core.SafeInt(r.PromptTokens), "completion_tokens": core.SafeInt(r.CompletionTokens),
				"cached_tokens": core.SafeInt(r.CachedTokens), "uncached_tokens": core.SafeInt(r.PromptTokens - r.CachedTokens),
				"total_tokens": core.SafeInt(r.PromptTokens+r.CompletionTokens), "avg_latency_ms": int(r.AvgLatencyMs),
			}
		}
		c.JSON(200, gin.H{"summary": summary, "items": items, "total": total, "page": page, "page_size": pageSize})

	default:
		c.JSON(400, gin.H{"error": gin.H{"message": "\u65e0\u6548\u7684group_by"}})
	}
}
