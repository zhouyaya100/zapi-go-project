package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/core/routing"
	"github.com/zapi/zapi-go/internal/model"
)

func HandleListChannels(c *gin.Context) {
	limit, offset := parsePagination(c)
	reveal := c.Query("reveal") == "true"

	var total int64
	model.DB.Model(&model.Channel{}).Count(&total)

	var channels []model.Channel
	qry := model.DB.Order("id")
	if limit > 0 {
		qry = qry.Offset(offset).Limit(limit)
	}
	qry.Find(&channels)

	// Batch-load upstream group links for all channels (avoid N+1)
	var allLinks []model.UpstreamGroupChannel
	model.DB.Find(&allLinks)
	channelUGMap := make(map[uint][]uint)
	for _, link := range allLinks {
		channelUGMap[link.ChannelID] = append(channelUGMap[link.ChannelID], link.UpstreamGroupID)
	}

	result := make([]gin.H, len(channels))
	for i, ch := range channels {
		ugIDs := channelUGMap[ch.ID]
		if ugIDs == nil { ugIDs = []uint{} }

		apiKeyVal := core.MaskKey(ch.APIKey)
		if reveal {
			apiKeyVal = ch.APIKey
		}

		result[i] = gin.H{
			"id": ch.ID, "name": ch.Name, "type": ch.Type,
			"base_url": ch.BaseURL, "api_key": apiKeyVal,
			"api_key_length": len(ch.APIKey), "models": ch.Models,
			"model_mapping": ch.ModelMapping, "allowed_groups": ch.AllowedGroups,
			"weight": ch.Weight, "priority": ch.Priority, "enabled": ch.Enabled,
			"auto_ban": ch.AutoBan, "fail_count": ch.FailCount,
			"test_time": core.FmtTimePtr(ch.TestTime), "response_time": ch.ResponseTime,
			"upstream_group_ids": ugIDs,
			"created_at": core.FmtTimeVal(ch.CreatedAt),
		}
	}
	if limit > 0 {
		c.JSON(200, gin.H{"items": result, "total": total})
	} else {
		c.JSON(200, result)
	}
}

func HandleCreateChannel(c *gin.Context) {
	var ch model.Channel
	if err := c.ShouldBindJSON(&ch); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "请求参数错误"}})
		return
	}
	if ch.Type == "" {
		ch.Type = "openai"
	}
	ch.ModelMapping = core.NormalizeModelMapping(ch.ModelMapping)
	model.DB.Create(&ch)
	routing.Pool.UpdateChannel(&ch)
	c.JSON(200, gin.H{"success": true, "id": ch.ID})
}

func HandleUpdateChannel(c *gin.Context) {
	id := c.Param("id")
	var ch model.Channel
	if model.DB.First(&ch, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "渠道不存在"}})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "请求参数错误"}})
		return
	}
	if v, ok := req["name"].(string); ok {
		ch.Name = v
	}
	if v, ok := req["base_url"].(string); ok {
		ch.BaseURL = v
	}
	if v, ok := req["api_key"].(string); ok && !strings.HasPrefix(v, "***") {
		ch.APIKey = v
	}
	if v, ok := req["models"].(string); ok {
		ch.Models = v
	}
	if v, ok := req["model_mapping"].(string); ok {
		ch.ModelMapping = core.NormalizeModelMapping(v)
	}
	if v, ok := req["allowed_groups"].(string); ok {
		ch.AllowedGroups = v
	}
	if v, ok := req["weight"].(float64); ok {
		ch.Weight = int(v)
	}
	if v, ok := req["priority"].(float64); ok {
		ch.Priority = int(v)
	}
	if v, ok := req["enabled"].(bool); ok {
		ch.Enabled = v
	}
	if v, ok := req["auto_ban"].(bool); ok {
		ch.AutoBan = v
	}
	model.DB.Save(&ch)
	routing.Pool.UpdateChannel(&ch)
	c.JSON(200, gin.H{"success": true})
}

func HandleDeleteChannel(c *gin.Context) {
	id := c.Param("id")
	var ch model.Channel
	if model.DB.First(&ch, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "渠道不存在"}})
		return
	}
	// Delete all join table links for this channel
	model.DB.Where("channel_id = ?", ch.ID).Delete(&model.UpstreamGroupChannel{})
	model.DB.Delete(&ch)
	routing.Pool.RemoveChannel(ch.ID)
	c.JSON(200, gin.H{"success": true})
}

// HandleFetchModels fetches model list from upstream by channel ID
// GET /api/channels/:id/fetch-models
func HandleFetchModels(c *gin.Context) {
	id := c.Param("id")
	var ch model.Channel
	if model.DB.First(&ch, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "渠道不存在"}})
		return
	}
	models, err := fetchUpstreamModels(ch.BaseURL, ch.APIKey)
	if err != nil {
		c.JSON(200, gin.H{"success": false, "models": []string{}, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "models": models})
}

// HandleFetchModelsByCred fetches model list from upstream using provided credentials
// POST /api/channels/0/fetch-models  {base_url, api_key}
func HandleFetchModelsByCred(c *gin.Context) {
	var req struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "请求参数错误"}})
		return
	}
	if req.BaseURL == "" || req.APIKey == "" {
		c.JSON(400, gin.H{"error": gin.H{"message": "接口地址和API密钥不能为空"}})
		return
	}
	models, err := fetchUpstreamModels(req.BaseURL, req.APIKey)
	if err != nil {
		c.JSON(200, gin.H{"success": false, "models": []string{}, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "models": models})
}

// fetchUpstreamModels calls the upstream /v1/models endpoint and returns model IDs
func fetchUpstreamModels(baseURL, apiKey string) ([]string, error) {
	base := strings.TrimRight(baseURL, "/")
	modelsURL := base + "/models"
	if strings.HasSuffix(base, "/v1") {
		modelsURL = base + "/models"
	} else if !strings.Contains(base, "/v1") {
		modelsURL = base + "/v1/models"
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", modelsURL, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %s", err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("上游返回 HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析模型列表失败: %s", err.Error())
	}
	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	return models, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func HandleTestChannel(c *gin.Context) {
	id := c.Param("id")
	var ch model.Channel
	if model.DB.First(&ch, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "渠道不存在"}})
		return
	}
	testModel := ""
	if ch.Models != "" {
		testModel = strings.TrimSpace(strings.Split(ch.Models, ",")[0])
	}
	if testModel == "" {
		c.JSON(200, gin.H{"success": false, "latency_ms": 0, "model": "-", "status": "无模型配置"})
		return
	}
	modelToUse := testModel
	if ch.ModelMapping != "" {
		var mapping map[string]string
		if json.Unmarshal([]byte(ch.ModelMapping), &mapping) == nil {
			if mapped, ok := mapping[testModel]; ok { modelToUse = mapped }
		}
	}
	base := strings.TrimRight(ch.BaseURL, "/")
	testURL := base + "/v1/chat/completions"
	if strings.HasSuffix(base, "/v1") {
		testURL = base[:len(base)-3] + "/v1/chat/completions"
	}
	testBody, _ := json.Marshal(map[string]interface{}{"model": modelToUse, "messages": []map[string]string{{"role": "user", "content": "Hi"}}, "max_tokens": 5})
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("POST", testURL, bytes.NewReader(testBody))
	req.Header.Set("Authorization", "Bearer "+ch.APIKey)
	req.Header.Set("Content-Type", "application/json")
	start := time.Now()
	resp, err := client.Do(req)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		now := time.Now().UTC()
		// Atomic fail count increment to avoid race
		model.DB.Exec("UPDATE channels SET fail_count = fail_count + 1, test_time = ?, response_time = 0 WHERE id = ?", now, id)
		model.DB.First(&ch, id)
		mf, _ := routing.Upstreams.GetMaxFailsForChannel(ch.ID)
		autoBanThreshold := mf
		if autoBanThreshold <= 0 { autoBanThreshold = 5 }
		shouldDisable := ch.AutoBan && ch.FailCount >= autoBanThreshold
		if shouldDisable {
			model.DB.Model(&ch).Update("enabled", false)
			ch.Enabled = false
		}
		routing.Pool.UpdateFailCount(ch.ID, ch.FailCount, ch.Enabled)
		core.ErrLog.Error(fmt.Sprintf("渠道测试失败: [%s] ID:%s 模型:%s 错误:%s", ch.Name, id, modelToUse, err.Error()))
		// Friendly error messages based on error type
		friendlyMsg := "连接失败"
		errStr := err.Error()
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
			friendlyMsg = "连接超时：上游服务器未在15秒内响应，请检查网络或接口地址"
		} else if strings.Contains(errStr, "connection refused") {
			friendlyMsg = "连接被拒绝：上游服务器未运行或端口错误"
		} else if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "lookup") {
			friendlyMsg = "DNS解析失败：域名不存在或无法解析"
		} else if strings.Contains(errStr, "TLS") || strings.Contains(errStr, "certificate") {
			friendlyMsg = "TLS/证书错误：SSL握手失败"
		} else if strings.Contains(errStr, "i/o timeout") {
			friendlyMsg = "网络超时：请检查服务器地址和网络连通性"
		}
		c.JSON(200, gin.H{"success": false, "latency_ms": 0, "model": modelToUse, "status": friendlyMsg, "error": errStr})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		now := time.Now().UTC()
		// Atomic reset fail count
		model.DB.Model(&model.Channel{}).Where("id = ?", id).Updates(map[string]interface{}{
			"fail_count":    0,
			"enabled":       true,
			"test_time":     &now,
			"response_time": latency,
		})
		routing.Pool.UpdateFailCount(ch.ID, 0, true)
		c.JSON(200, gin.H{"success": true, "latency_ms": latency, "model": modelToUse, "status": "OK"})
	} else {
		errorBody, _ := io.ReadAll(resp.Body)
		errorStr := string(errorBody); if len(errorStr) > 300 { errorStr = errorStr[:300] }
		now := time.Now().UTC()
		// Atomic fail count increment when enabled
		if ch.Enabled {
			model.DB.Exec("UPDATE channels SET fail_count = fail_count + 1, test_time = ?, response_time = ? WHERE id = ?", now, latency, id)
			model.DB.First(&ch, id)
			mf, _ := routing.Upstreams.GetMaxFailsForChannel(ch.ID)
			autoBanThreshold := mf
			if autoBanThreshold <= 0 { autoBanThreshold = 5 }
			if ch.AutoBan && ch.FailCount >= autoBanThreshold {
				model.DB.Model(&ch).Update("enabled", false)
				ch.Enabled = false
			}
		} else {
			model.DB.Model(&ch).Updates(map[string]interface{}{
				"test_time":     &now,
				"response_time": latency,
			})
		}
		routing.Pool.UpdateFailCount(ch.ID, ch.FailCount, ch.Enabled)
		core.ErrLog.Error(fmt.Sprintf("渠道测试失败: [%s] ID:%s HTTP:%d 模型:%s", ch.Name, id, resp.StatusCode, modelToUse))
		// Friendly HTTP error messages
		httpMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		switch resp.StatusCode {
		case 401:
			httpMsg = "认证失败(401)：API Key无效或已过期"
		case 403:
			httpMsg = "权限不足(403)：该Key无权访问此模型"
		case 404:
			httpMsg = "接口不存在(404)：请检查Base URL地址"
		case 429:
			httpMsg = "请求过多(429)：上游速率限制，请稍后重试"
		case 500:
			httpMsg = "上游服务器错误(500)"
		case 502:
			httpMsg = "网关错误(502)：上游服务不可用"
		case 503:
			httpMsg = "服务不可用(503)：上游暂时过载或维护中"
		}
		c.JSON(200, gin.H{"success": false, "latency_ms": latency, "model": modelToUse, "status": httpMsg, "error": errorStr})
	}
}
