package handler

import (
	"encoding/json"
	"os"
	"sort"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/config"
	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/core/routing"
	"github.com/zapi/zapi-go/internal/model"
	"gopkg.in/yaml.v3"
)

var configMu sync.Mutex

var yamlMap = map[string][2]string{
	"jwt_expire_hours":         {"security", "jwt_expire_hours"},
	"cors_origins":             {"security", "cors_origins"},
	"proxy_timeout":            {"proxy", "timeout"},
	"proxy_max_connections":    {"proxy", "max_connections"},
	"proxy_max_keepalive":      {"proxy", "max_keepalive"},
	"proxy_keepalive_expiry":   {"proxy", "keepalive_expiry"},
	"proxy_retry_count":        {"proxy", "retry_count"},
	"proxy_max_fails":          {"proxy", "max_fails"},
	"proxy_fail_timeout":       {"proxy", "fail_timeout"},
	"cache_enabled":            {"cache", "enabled"},
	"cache_ttl":                {"cache", "ttl"},
	"cache_max_entries":        {"cache", "max_entries"},
	"log_batch_size":            {"log", "batch_size"},
	"log_batch_interval":       {"log", "batch_interval"},
	"log_retention_days":       {"log", "retention_days"},
	"log_cleanup_interval_hours": {"log", "cleanup_interval_hours"},
	"log_cleanup_batch_size":   {"log", "cleanup_batch_size"},
	"error_log_max_entries":    {"log", "error_max_entries"},
	"error_log_max_days":       {"log", "error_max_days"},
	"allow_register":           {"registration", "allow_register"},
	"default_max_tokens":       {"registration", "default_max_tokens"},
	"default_token_quota":      {"registration", "default_token_quota"},
	"default_group":            {"registration", "default_group"},
	"min_password_length":      {"registration", "min_password_length"},
	"timezone_offset":          {"server", "timezone_offset"},
	"heartbeat_enabled":        {"heartbeat", "enabled"},
	"heartbeat_interval":       {"heartbeat", "interval"},
	"heartbeat_timeout":        {"heartbeat", "timeout"},
}

func HandleGetSettings(c *gin.Context) {
	cfg := config.GetCfg()
	var gs []model.Group
	model.DB.Order("id").Find(&gs)
	gl := make([]gin.H, len(gs))
	for i, g := range gs {
		gl[i] = gin.H{"id": g.ID, "name": g.Name, "comment": g.Comment}
	}
	var chs []model.Channel
	model.DB.Where("enabled = ?", true).Find(&chs)
	am := map[string]bool{}
	for _, ch := range chs {
		for _, m := range core.SplitComma(ch.Models) {
			if m != "" {
				am[m] = true
			}
		}
		// Include model_mapping keys as available models (external model names)
		if ch.ModelMapping != "" {
			var mapping map[string]string
			if err := json.Unmarshal([]byte(ch.ModelMapping), &mapping); err == nil {
				for k := range mapping {
					if k != "" {
						am[k] = true
					}
				}
			}
		}
	}
	// Also include upstream group aliases as available models
	for _, ug := range routing.Upstreams.GetAllGroups() {
		if ug.Alias != "" {
			am[ug.Alias] = true
		}
	}
	models := make([]string, 0, len(am))
	for m := range am {
		models = append(models, m)
	}
	sort.Strings(models)
	c.JSON(200, gin.H{
		"jwt_expire_hours":           cfg.Security.JWTExpireHours,
		"cors_origins":               cfg.Security.CORSOrigins,
		"proxy_timeout":              cfg.Proxy.Timeout,
		"proxy_max_connections":      cfg.Proxy.MaxConnections,
		"proxy_max_keepalive":        cfg.Proxy.MaxKeepalive,
		"proxy_keepalive_expiry":     cfg.Proxy.KeepaliveExpiry,
		"proxy_retry_count":          cfg.Proxy.RetryCount,
		"proxy_max_fails":            cfg.Proxy.MaxFails,
		"proxy_fail_timeout":         cfg.Proxy.FailTimeout,
		"cache_enabled":              cfg.Cache.Enabled,
		"cache_ttl":                  cfg.Cache.TTL,
		"cache_max_entries":          cfg.Cache.MaxEntries,
		"log_batch_size":             cfg.Log.BatchSize,
		"log_batch_interval":         cfg.Log.BatchInterval,
		"log_retention_days":         cfg.Log.RetentionDays,
		"log_cleanup_interval_hours": cfg.Log.CleanupIntervalHours,
		"log_cleanup_batch_size":     cfg.Log.CleanupBatchSize,
		"error_log_max_entries":      cfg.Log.ErrorMaxEntries,
		"error_log_max_days":         cfg.Log.ErrorMaxDays,
		"allow_register":             cfg.Registration.AllowRegister,
		"default_max_tokens":         cfg.Registration.DefaultMaxTokens,
		"default_token_quota":        cfg.Registration.DefaultTokenQuota,
		"default_group":              cfg.Registration.DefaultGroup,
		"min_password_length":        cfg.Registration.MinPasswordLength,
		"timezone_offset":            cfg.Server.TimezoneOffset,
		"heartbeat_enabled":          cfg.Heartbeat.Enabled,
		"heartbeat_interval":         cfg.Heartbeat.Interval,
		"heartbeat_timeout":          cfg.Heartbeat.Timeout,
		// Read-only fields
		"server_host":               cfg.Server.Host,
		"server_port":               cfg.Server.Port,
		"database_url":              cfg.Database.URL,
		"db_pool_size":              cfg.Database.PoolSize,
		"db_max_overflow":           cfg.Database.MaxOverflow,
		// Auxiliary data
		"groups":                     gl,
		"all_models":                 models,
	})
}

func HandleUpdateSettings(c *gin.Context) {
	configMu.Lock()
	defer configMu.Unlock()
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u8bf7\u6c42\u53c2\u6570\u9519\u8bef"}})
		return
	}
	// Validation
	if v, ok := req["default_token_quota"]; ok {
		if f, ok := v.(float64); ok && int64(f) < -1 { c.JSON(400, gin.H{"error": gin.H{"message": "default_token_quota \u53ea\u80fd\u4e3a -1(\u65e0\u9650) \u6216\u6b63\u6570"}}); return }
	}
	if v, ok := req["jwt_expire_hours"]; ok {
		if f, ok := v.(float64); ok && int(f) < 1 { c.JSON(400, gin.H{"error": gin.H{"message": "JWT \u8fc7\u671f\u65f6\u95f4\u81f3\u5c11 1 \u5c0f\u65f6"}}); return }
	}
	if v, ok := req["proxy_timeout"]; ok {
		if f, ok := v.(float64); ok && int(f) < 5 { c.JSON(400, gin.H{"error": gin.H{"message": "\u4ee3\u7406\u8d85\u65f6\u81f3\u5c11 5 \u79d2"}}); return }
	}
	if v, ok := req["error_log_max_entries"]; ok {
		if f, ok := v.(float64); ok && int(f) < 100 { c.JSON(400, gin.H{"error": gin.H{"message": "\u9519\u8bef\u65e5\u5fd7\u6700\u5927\u6761\u6570\u81f3\u5c11 100"}}); return }
	}
	if v, ok := req["error_log_max_days"]; ok {
		if f, ok := v.(float64); ok && int(f) < 1 { c.JSON(400, gin.H{"error": gin.H{"message": "日志保留天数至少 1 天"}}); return }
	}
	if v, ok := req["proxy_max_fails"]; ok {
		if f, ok := v.(float64); ok && int(f) < 0 { c.JSON(400, gin.H{"error": gin.H{"message": "熔断阈值不能为负数"}}); return }
	}
	if v, ok := req["proxy_fail_timeout"]; ok {
		if f, ok := v.(float64); ok && int(f) < 1 { c.JSON(400, gin.H{"error": gin.H{"message": "熔断恢复时间至少 1 秒"}}); return }
	}
	// Update config in memory
	for key, val := range req {
		pair, ok := yamlMap[key]
		if !ok {
			continue
		}
		switch key {
		case "jwt_expire_hours":
			if v, ok := val.(float64); ok { config.Cfg.Security.JWTExpireHours = int(v) }
		case "cors_origins":
			if v, ok := val.(string); ok { config.Cfg.Security.CORSOrigins = v }
		case "proxy_timeout":
			if v, ok := val.(float64); ok { config.Cfg.Proxy.Timeout = int(v) }
		case "proxy_max_connections":
			if v, ok := val.(float64); ok { config.Cfg.Proxy.MaxConnections = int(v) }
		case "proxy_max_keepalive":
			if v, ok := val.(float64); ok { config.Cfg.Proxy.MaxKeepalive = int(v) }
		case "proxy_keepalive_expiry":
			if v, ok := val.(float64); ok { config.Cfg.Proxy.KeepaliveExpiry = int(v) }
		case "proxy_retry_count":
			if v, ok := val.(float64); ok { config.Cfg.Proxy.RetryCount = int(v) }
		case "proxy_max_fails":
			if v, ok := val.(float64); ok { config.Cfg.Proxy.MaxFails = int(v) }
		case "proxy_fail_timeout":
			if v, ok := val.(float64); ok { config.Cfg.Proxy.FailTimeout = int(v) }
		case "cache_enabled":
			if v, ok := val.(bool); ok { config.Cfg.Cache.Enabled = v }
		case "cache_ttl":
			if v, ok := val.(float64); ok { config.Cfg.Cache.TTL = int(v) }
		case "cache_max_entries":
			if v, ok := val.(float64); ok { config.Cfg.Cache.MaxEntries = int(v) }
		case "log_batch_size":
			if v, ok := val.(float64); ok { config.Cfg.Log.BatchSize = int(v) }
		case "log_batch_interval":
			if v, ok := val.(float64); ok { config.Cfg.Log.BatchInterval = int(v) }
		case "log_retention_days":
			if v, ok := val.(float64); ok { config.Cfg.Log.RetentionDays = int(v) }
		case "log_cleanup_interval_hours":
			if v, ok := val.(float64); ok { config.Cfg.Log.CleanupIntervalHours = int(v) }
		case "log_cleanup_batch_size":
			if v, ok := val.(float64); ok { config.Cfg.Log.CleanupBatchSize = int(v) }
		case "error_log_max_entries":
			if v, ok := val.(float64); ok { config.Cfg.Log.ErrorMaxEntries = int(v) }
		case "error_log_max_days":
			if v, ok := val.(float64); ok { config.Cfg.Log.ErrorMaxDays = int(v); core.ErrLog.CleanOldFiles() }
		case "allow_register":
			if v, ok := val.(bool); ok { config.Cfg.Registration.AllowRegister = v }
		case "default_max_tokens":
			if v, ok := val.(float64); ok { config.Cfg.Registration.DefaultMaxTokens = int(v) }
		case "default_token_quota":
			if v, ok := val.(float64); ok { config.Cfg.Registration.DefaultTokenQuota = int64(v) }
		case "default_group":
			if v, ok := val.(string); ok { config.Cfg.Registration.DefaultGroup = v }
		case "min_password_length":
			if v, ok := val.(float64); ok { config.Cfg.Registration.MinPasswordLength = int(v) }
		case "timezone_offset":
			if v, ok := val.(float64); ok { config.Cfg.Server.TimezoneOffset = int(v) }
		case "heartbeat_enabled":
			if v, ok := val.(bool); ok { config.Cfg.Heartbeat.Enabled = v }
		case "heartbeat_interval":
			if v, ok := val.(float64); ok { if int(v) >= 10 { config.Cfg.Heartbeat.Interval = int(v) } }
		case "heartbeat_timeout":
			if v, ok := val.(float64); ok { if int(v) >= 1 { config.Cfg.Heartbeat.Timeout = int(v) } }
		}
		_ = pair
	}
	// Write back to YAML
	data, err := yaml.Marshal(&config.Cfg)
	if err == nil {
		cfgPath := config.ConfigFilePath
		if cfgPath == "" {
			cfgPath = "config.yaml"
		}
		tmpFile := cfgPath + ".tmp"
		if writeErr := os.WriteFile(tmpFile, data, 0644); writeErr == nil {
			if renameErr := os.Rename(tmpFile, cfgPath); renameErr != nil {
				// Failed to rename — clean up tmp file and report error
				os.Remove(tmpFile)
				c.JSON(500, gin.H{"error": gin.H{"message": "配置文件写入失败"}})
				return
			}
		} else {
			c.JSON(500, gin.H{"error": gin.H{"message": "配置文件写入失败"}})
			return
		}
	}
	c.JSON(200, gin.H{"success": true})
}

func HandleGetPublicSettings(c *gin.Context) {
	var gs []model.Group
	model.DB.Order("id").Find(&gs)
	gl := make([]gin.H, len(gs))
	for i, g := range gs {
		gl[i] = gin.H{"id": g.ID, "name": g.Name, "comment": g.Comment}
	}
	c.JSON(200, gin.H{
		"allow_register":      config.Cfg.Registration.AllowRegister,
		"groups":              gl,
		"all_models":          routing.Pool.GetAllEnabledModels(),
	})
}

func HandleClearErrorLog(c *gin.Context) {
	core.ErrLog.Clear()
	c.JSON(200, gin.H{"success": true, "message": "\u9519\u8bef\u65e5\u5fd7\u5df2\u6e05\u7a7a"})
}
