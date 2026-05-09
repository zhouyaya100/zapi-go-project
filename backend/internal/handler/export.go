package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/middleware"
	"github.com/zapi/zapi-go/internal/model"
	"github.com/xuri/excelize/v2"
)


func exportDetail(c *gin.Context, fmt_ string) {
	// Use model.DB directly since we can't easily type-alias GORM
	qry := model.DB.Model(&model.Log{})
	// Re-apply filters
	if df := c.Query("date_from"); df != "" {
		dfOut, _ := core.ParseDateFilters(df, "")
		if dfOut != "" { qry = qry.Where("created_at >= ?", dfOut) }
	}
	if dt := c.Query("date_to"); dt != "" {
		_, dtOut := core.ParseDateFilters("", dt)
		if dtOut != "" { qry = qry.Where("created_at < ?", dtOut) }
	}
	if fuid, ok := c.Get("_force_user_id"); ok {
		qry = qry.Where("user_id = ?", fuid)
	} else if uid := c.Query("user_id"); uid != "" {
		if id, err := strconv.ParseUint(uid, 10, 64); err == nil { qry = qry.Where("user_id = ?", id) }
	}
	if m := c.Query("model"); m != "" { qry = qry.Where("model ILIKE ?", "%"+m+"%") }
	if cid := c.Query("channel_id"); cid != "" {
		if id, err := strconv.ParseUint(cid, 10, 64); err == nil { qry = qry.Where("channel_id = ?", id) }
	}

	// Date range check for detail export
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	if dateFrom == "" && dateTo == "" {
		c.JSON(400, gin.H{"error": gin.H{"message": "明细导出需指定日期范围"}})
		return
	}
	// Check 93-day limit
	if dateFrom != "" && dateTo != "" {
		df, _ := time.Parse("2006-01-02", dateFrom)
		dt, _ := time.Parse("2006-01-02", dateTo)
		if !df.IsZero() && !dt.IsZero() && dt.Sub(df).Hours() > 93*24 {
			c.JSON(400, gin.H{"error": gin.H{"message": "日期范围不能超过93天"}})
			return
		}
	}

	var logs []model.Log
	qry.Order("id desc").Limit(50000).Find(&logs)
	um := make(map[uint]string)
	var users []model.User
	model.DB.Select("id, username").Find(&users)
	for _, u := range users { um[u.ID] = u.Username }

	rows := make([]map[string]interface{}, len(logs))
	for i, l := range logs {
		rows[i] = map[string]interface{}{
			"id": l.ID, "username": um[l.UserID], "token_name": l.TokenName,
			"channel_name": l.ChannelName, "model": l.Model, "is_stream": l.IsStream,
			"prompt_tokens": l.PromptTokens, "completion_tokens": l.CompletionTokens,
			"cached_tokens": l.CachedTokens, "uncached_tokens": l.PromptTokens - l.CachedTokens,
			"total_tokens": l.PromptTokens + l.CompletionTokens,
			"latency_ms": l.LatencyMs, "success": l.Success,
			"error_msg": l.ErrorMsg, "client_ip": l.ClientIP,
			"created_at": core.ToLocal(l.CreatedAt),
		}
	}
	nowStr := time.Now().Format("20060102_150405")
	if fmt_ == "xlsx" {
		writeXLSXDetail(c, rows, nowStr)
	} else {
		writeCSVDetail(c, rows, nowStr)
	}
}

func exportGrouped(c *gin.Context, groupBy string, fmt_ string) {
	qry := model.DB.Model(&model.Log{})
	// Apply same filters
	if df := c.Query("date_from"); df != "" {
		dfOut, _ := core.ParseDateFilters(df, "")
		if dfOut != "" { qry = qry.Where("created_at >= ?", dfOut) }
	}
	if dt := c.Query("date_to"); dt != "" {
		_, dtOut := core.ParseDateFilters("", dt)
		if dtOut != "" { qry = qry.Where("created_at < ?", dtOut) }
	}
	if fuid, ok := c.Get("_force_user_id"); ok {
		qry = qry.Where("user_id = ?", fuid)
	} else if uid := c.Query("user_id"); uid != "" {
		if id, err := strconv.ParseUint(uid, 10, 64); err == nil { qry = qry.Where("user_id = ?", id) }
	}
	if m := c.Query("model"); m != "" { qry = qry.Where("model ILIKE ?", "%"+m+"%") }
	if cid := c.Query("channel_id"); cid != "" {
		if id, err := strconv.ParseUint(cid, 10, 64); err == nil { qry = qry.Where("channel_id = ?", id) }
	}

	nowStr := time.Now().Format("20060102_150405")

	switch groupBy {
	case "day":
		type row struct{ Period string; Requests int64; Success int64; PromptTokens int64; CompletionTokens int64; CachedTokens int64; AvgLatencyMs float64 }
		var rows []row
		qry.Select(core.TzDateExpr()+" as period, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(sum(cached_tokens),0) as cached_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group(core.TzDateExpr()).Order("period desc").Scan(&rows)
		items := make([]map[string]interface{}, len(rows))
		for i, r := range rows {
			key := r.Period; if len(key) > 10 { key = key[:10] }
			fail := r.Requests - r.Success; sr := 0.0; if r.Requests > 0 { sr = float64(r.Success) / float64(r.Requests) * 100 }
			items[i] = map[string]interface{}{"key": key, "requests": r.Requests, "success": r.Success, "fail": fail, "success_rate": fmt.Sprintf("%.1f%%", sr), "prompt_tokens": core.SafeInt(r.PromptTokens), "completion_tokens": core.SafeInt(r.CompletionTokens), "cached_tokens": core.SafeInt(r.CachedTokens), "uncached_tokens": core.SafeInt(r.PromptTokens-r.CachedTokens), "total_tokens": core.SafeInt(r.PromptTokens+r.CompletionTokens), "avg_latency_ms": int(r.AvgLatencyMs)}
		}
		if fmt_ == "xlsx" { writeXLSXGrouped(c, items, groupBy, nowStr) } else { writeCSVGrouped(c, items, groupBy, nowStr) }
	case "user":
		type row struct{ UserID uint; Requests int64; Success int64; PromptTokens int64; CompletionTokens int64; CachedTokens int64; AvgLatencyMs float64 }
		var rows []row
		qry.Select("user_id, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(sum(cached_tokens),0) as cached_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group("user_id").Order("requests desc").Scan(&rows)
		um := make(map[uint]string); var users []model.User; model.DB.Select("id, username").Find(&users); for _, u := range users { um[u.ID] = u.Username }
		items := make([]map[string]interface{}, len(rows))
		for i, r := range rows {
			key := um[r.UserID]; if key == "" { key = fmt.Sprintf("user:%d", r.UserID) }
			fail := r.Requests - r.Success; sr := 0.0; if r.Requests > 0 { sr = float64(r.Success) / float64(r.Requests) * 100 }
			items[i] = map[string]interface{}{"key": key, "requests": r.Requests, "success": r.Success, "fail": fail, "success_rate": fmt.Sprintf("%.1f%%", sr), "prompt_tokens": core.SafeInt(r.PromptTokens), "completion_tokens": core.SafeInt(r.CompletionTokens), "cached_tokens": core.SafeInt(r.CachedTokens), "uncached_tokens": core.SafeInt(r.PromptTokens-r.CachedTokens), "total_tokens": core.SafeInt(r.PromptTokens+r.CompletionTokens), "avg_latency_ms": int(r.AvgLatencyMs)}
		}
		if fmt_ == "xlsx" { writeXLSXGrouped(c, items, groupBy, nowStr) } else { writeCSVGrouped(c, items, groupBy, nowStr) }
	case "model":
		type row struct{ Model string; Requests int64; Success int64; PromptTokens int64; CompletionTokens int64; CachedTokens int64; AvgLatencyMs float64 }
		var rows []row
		qry.Select("model, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(sum(cached_tokens),0) as cached_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group("model").Order("requests desc").Scan(&rows)
		items := make([]map[string]interface{}, len(rows))
		for i, r := range rows {
			fail := r.Requests - r.Success; sr := 0.0; if r.Requests > 0 { sr = float64(r.Success) / float64(r.Requests) * 100 }
			items[i] = map[string]interface{}{"key": r.Model, "requests": r.Requests, "success": r.Success, "fail": fail, "success_rate": fmt.Sprintf("%.1f%%", sr), "prompt_tokens": core.SafeInt(r.PromptTokens), "completion_tokens": core.SafeInt(r.CompletionTokens), "cached_tokens": core.SafeInt(r.CachedTokens), "uncached_tokens": core.SafeInt(r.PromptTokens-r.CachedTokens), "total_tokens": core.SafeInt(r.PromptTokens+r.CompletionTokens), "avg_latency_ms": int(r.AvgLatencyMs)}
		}
		if fmt_ == "xlsx" { writeXLSXGrouped(c, items, groupBy, nowStr) } else { writeCSVGrouped(c, items, groupBy, nowStr) }
	case "channel":
		type row struct{ ChannelID uint; Requests int64; Success int64; PromptTokens int64; CompletionTokens int64; CachedTokens int64; AvgLatencyMs float64 }
		var rows []row
		qry.Select("channel_id, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(sum(cached_tokens),0) as cached_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group("channel_id").Order("requests desc").Scan(&rows)
		cm := make(map[uint]string); var chs []model.Channel; model.DB.Select("id, name").Find(&chs); for _, ch := range chs { cm[ch.ID] = ch.Name }
		items := make([]map[string]interface{}, len(rows))
		for i, r := range rows {
			key := cm[r.ChannelID]; if key == "" { key = fmt.Sprintf("%d", r.ChannelID) }
			fail := r.Requests - r.Success; sr := 0.0; if r.Requests > 0 { sr = float64(r.Success) / float64(r.Requests) * 100 }
			items[i] = map[string]interface{}{"key": key, "requests": r.Requests, "success": r.Success, "fail": fail, "success_rate": fmt.Sprintf("%.1f%%", sr), "prompt_tokens": core.SafeInt(r.PromptTokens), "completion_tokens": core.SafeInt(r.CompletionTokens), "cached_tokens": core.SafeInt(r.CachedTokens), "uncached_tokens": core.SafeInt(r.PromptTokens-r.CachedTokens), "total_tokens": core.SafeInt(r.PromptTokens+r.CompletionTokens), "avg_latency_ms": int(r.AvgLatencyMs)}
		}
		if fmt_ == "xlsx" { writeXLSXGrouped(c, items, groupBy, nowStr) } else { writeCSVGrouped(c, items, groupBy, nowStr) }
	case "summary":
		type row struct{ UserID uint; Model string; ChannelID uint; Requests int64; Success int64; PromptTokens int64; CompletionTokens int64; CachedTokens int64; AvgLatencyMs float64 }
		var rows []row
		qry.Select("user_id, model, channel_id, count(*) as requests, sum(case when success then 1 else 0 end) as success, coalesce(sum(prompt_tokens),0) as prompt_tokens, coalesce(sum(completion_tokens),0) as completion_tokens, coalesce(sum(cached_tokens),0) as cached_tokens, coalesce(avg(latency_ms),0) as avg_latency_ms").Group("user_id, model, channel_id").Order("requests desc").Scan(&rows)
		um := make(map[uint]string); var users []model.User; model.DB.Select("id, username").Find(&users); for _, u := range users { um[u.ID] = u.Username }
		cm := make(map[uint]string); var chs []model.Channel; model.DB.Select("id, name").Find(&chs); for _, ch := range chs { cm[ch.ID] = ch.Name }
		items := make([]map[string]interface{}, len(rows))
		for i, r := range rows {
			uname := um[r.UserID]; if uname == "" { uname = "-" }
			cname := cm[r.ChannelID]; if cname == "" { cname = "-" }
			fail := r.Requests - r.Success; sr := 0.0; if r.Requests > 0 { sr = float64(r.Success) / float64(r.Requests) * 100 }
			items[i] = map[string]interface{}{"key": fmt.Sprintf("%s / %s / %s", uname, r.Model, cname), "user": uname, "model": r.Model, "channel": cname, "requests": r.Requests, "success": r.Success, "fail": fail, "success_rate": fmt.Sprintf("%.1f%%", sr), "prompt_tokens": core.SafeInt(r.PromptTokens), "completion_tokens": core.SafeInt(r.CompletionTokens), "cached_tokens": core.SafeInt(r.CachedTokens), "uncached_tokens": core.SafeInt(r.PromptTokens-r.CachedTokens), "total_tokens": core.SafeInt(r.PromptTokens+r.CompletionTokens), "avg_latency_ms": int(r.AvgLatencyMs)}
		}
		if fmt_ == "xlsx" { writeXLSXSummary(c, items, nowStr) } else { writeCSVSummary(c, items, nowStr) }
	default:
		c.JSON(400, gin.H{"error": gin.H{"message": "无效的 group_by 参数"}})
	}
}

func HandleExportCSV(c *gin.Context) { handleExport(c, "csv") }
func HandleExportXLSX(c *gin.Context) { handleExport(c, "xlsx") }

func handleExport(c *gin.Context, fmt_ string) {
	groupBy := c.DefaultQuery("group_by", "detail")
	if groupBy == "" || groupBy == "detail" {
		exportDetail(c, fmt_)
	} else {
		exportGrouped(c, groupBy, fmt_)
	}
}

// User export
func HandleMyExportCSV(c *gin.Context) { handleMyExport(c, "csv") }
func HandleMyExportXLSX(c *gin.Context) { handleMyExport(c, "xlsx") }

func handleMyExport(c *gin.Context, fmt_ string) {
	u := getUserOrAdmin(c)
	// Force user_id filter via context instead of RawQuery manipulation
	c.Set("_force_user_id", u.ID)
	groupBy := c.DefaultQuery("group_by", "detail")
	if groupBy == "" || groupBy == "detail" {
		exportDetail(c, fmt_)
	} else {
		exportGrouped(c, groupBy, fmt_)
	}
}

// CSV writers
func writeCSVDetail(c *gin.Context, rows []map[string]interface{}, nowStr string) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	w.Write([]string{"ID", "用户名", "令牌名", "渠道名", "模型", "流式", "输入Token", "输出Token", "缓存Token", "未命中Token", "总Token", "延迟(ms)", "成功", "错误信息", "客户端IP", "时间"})
	for _, r := range rows {
		w.Write([]string{fmt.Sprintf("%v", r["id"]), fmt.Sprintf("%v", r["username"]), fmt.Sprintf("%v", r["token_name"]), fmt.Sprintf("%v", r["channel_name"]), fmt.Sprintf("%v", r["model"]), fmt.Sprintf("%v", r["is_stream"]), fmt.Sprintf("%v", r["prompt_tokens"]), fmt.Sprintf("%v", r["completion_tokens"]), fmt.Sprintf("%v", r["cached_tokens"]), fmt.Sprintf("%v", r["uncached_tokens"]), fmt.Sprintf("%v", r["total_tokens"]), fmt.Sprintf("%v", r["latency_ms"]), fmt.Sprintf("%v", r["success"]), fmt.Sprintf("%v", r["error_msg"]), fmt.Sprintf("%v", r["client_ip"]), fmt.Sprintf("%v", r["created_at"])})
	}
	w.Flush()
	filename := url.QueryEscape(fmt.Sprintf("用量报表_%s.csv", nowStr))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", filename))
	c.Data(200, "text/csv; charset=utf-8", buf.Bytes())
}

func writeCSVGrouped(c *gin.Context, items []map[string]interface{}, groupBy string, nowStr string) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	labels := map[string]string{"day": "日期", "user": "用户", "model": "模型", "channel": "渠道"}
	w.Write([]string{labels[groupBy], "请求数", "成功数", "失败数", "成功率", "输入Token", "输出Token", "缓存Token", "未命中Token", "总Token", "平均延迟(ms)"})
	for _, r := range items {
		w.Write([]string{fmt.Sprintf("%v", r["key"]), fmt.Sprintf("%v", r["requests"]), fmt.Sprintf("%v", r["success"]), fmt.Sprintf("%v", r["fail"]), fmt.Sprintf("%v", r["success_rate"]), fmt.Sprintf("%v", r["prompt_tokens"]), fmt.Sprintf("%v", r["completion_tokens"]), fmt.Sprintf("%v", r["cached_tokens"]), fmt.Sprintf("%v", r["uncached_tokens"]), fmt.Sprintf("%v", r["total_tokens"]), fmt.Sprintf("%v", r["avg_latency_ms"])})
	}
	w.Flush()
	filename := url.QueryEscape(fmt.Sprintf("用量报表_%s.csv", nowStr))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", filename))
	c.Data(200, "text/csv; charset=utf-8", buf.Bytes())
}

func writeCSVSummary(c *gin.Context, items []map[string]interface{}, nowStr string) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	w.Write([]string{"用户", "模型", "渠道", "请求数", "成功数", "失败数", "成功率", "输入Token", "输出Token", "缓存Token", "未命中Token", "总Token", "平均延迟(ms)"})
	for _, r := range items {
		w.Write([]string{fmt.Sprintf("%v", r["user"]), fmt.Sprintf("%v", r["model"]), fmt.Sprintf("%v", r["channel"]), fmt.Sprintf("%v", r["requests"]), fmt.Sprintf("%v", r["success"]), fmt.Sprintf("%v", r["fail"]), fmt.Sprintf("%v", r["success_rate"]), fmt.Sprintf("%v", r["prompt_tokens"]), fmt.Sprintf("%v", r["completion_tokens"]), fmt.Sprintf("%v", r["cached_tokens"]), fmt.Sprintf("%v", r["uncached_tokens"]), fmt.Sprintf("%v", r["total_tokens"]), fmt.Sprintf("%v", r["avg_latency_ms"])})
	}
	w.Flush()
	filename := url.QueryEscape(fmt.Sprintf("用量报表_%s.csv", nowStr))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", filename))
	c.Data(200, "text/csv; charset=utf-8", buf.Bytes())
}

// XLSX writers
func xlsxCol(i int) string {
	n := i + 1
	var col string
	for n > 0 {
		n--
		col = string(rune('A'+n%26)) + col
		n /= 26
	}
	return col
}

func writeXLSXDetail(c *gin.Context, rows []map[string]interface{}, nowStr string) {
	f := excelize.NewFile()
	sheet := "请求明细"
	f.SetSheetName("Sheet1", sheet)
	headers := []string{"ID", "用户名", "令牌名", "渠道名", "模型", "流式", "输入Token", "输出Token", "缓存Token", "未命中Token", "总Token", "延迟(ms)", "成功", "错误信息", "客户端IP", "时间"}
	for i, h := range headers { f.SetCellValue(sheet, xlsxCol(i)+"1", h) }
	for i, r := range rows {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r["id"])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r["username"])
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r["token_name"])
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r["channel_name"])
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r["model"])
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), r["is_stream"])
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), r["prompt_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), r["completion_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(8), row), r["cached_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(9), row), r["uncached_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(10), row), r["total_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(11), row), r["latency_ms"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(12), row), r["success"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(13), row), r["error_msg"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(14), row), r["client_ip"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(15), row), r["created_at"])
	}
	buf, _ := f.WriteToBuffer()
	filename := url.QueryEscape(fmt.Sprintf("用量报表_%s.xlsx", nowStr))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", filename))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func writeXLSXGrouped(c *gin.Context, items []map[string]interface{}, groupBy string, nowStr string) {
	f := excelize.NewFile()
	sheet := "用量汇总"
	f.SetSheetName("Sheet1", sheet)
	labels := map[string]string{"day": "日期", "user": "用户", "model": "模型", "channel": "渠道"}
	headers := []string{labels[groupBy], "请求数", "成功数", "失败数", "成功率", "输入Token", "输出Token", "缓存Token", "未命中Token", "总Token", "平均延迟(ms)"}
	for i, h := range headers { f.SetCellValue(sheet, xlsxCol(i)+"1", h) }
	for i, r := range items {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r["key"])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r["requests"])
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r["success"])
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r["fail"])
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r["success_rate"])
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), r["prompt_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), r["completion_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(7), row), r["cached_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(8), row), r["uncached_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(9), row), r["total_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(10), row), r["avg_latency_ms"])
	}
	buf, _ := f.WriteToBuffer()
	filename := url.QueryEscape(fmt.Sprintf("用量报表_%s.xlsx", nowStr))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", filename))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func writeXLSXSummary(c *gin.Context, items []map[string]interface{}, nowStr string) {
	f := excelize.NewFile()
	sheet := "用量汇总"
	f.SetSheetName("Sheet1", sheet)
	headers := []string{"用户", "模型", "渠道", "请求数", "成功数", "失败数", "成功率", "输入Token", "输出Token", "缓存Token", "未命中Token", "总Token", "平均延迟(ms)"}
	for i, h := range headers { f.SetCellValue(sheet, xlsxCol(i)+"1", h) }
	for i, r := range items {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r["user"])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r["model"])
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r["channel"])
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r["requests"])
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r["success"])
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), r["fail"])
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), r["success_rate"])
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), r["prompt_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), r["completion_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(9), row), r["cached_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(10), row), r["uncached_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(11), row), r["total_tokens"])
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", xlsxCol(12), row), r["avg_latency_ms"])
	}
	buf, _ := f.WriteToBuffer()
	filename := url.QueryEscape(fmt.Sprintf("用量报表_%s.xlsx", nowStr))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", filename))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// Suppress unused import
var _ = middleware.GetAuth
