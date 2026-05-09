package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zapi/zapi-go/internal/config"
)

func SafeInt(n int64) interface{} {
	const maxJS = int64(1<<53 - 1)
	if n > maxJS { return fmt.Sprintf("%d", n) }
	return n
}

func MaskKey(key string) string {
	if len(key) <= 8 { return "***" }
	return key[:3] + "***" + key[len(key)-4:]
}

func SplitComma(s string) []string {
	if s == "" { return nil }
	var r []string
	for _, v := range strings.Split(s, ",") { v = strings.TrimSpace(v); if v != "" { r = append(r, v) } }
	return r
}

func ToLocal(t time.Time) string {
	if config.Cfg.Server.TimezoneOffset != 0 { t = t.Add(time.Duration(config.Cfg.Server.TimezoneOffset) * time.Hour) }
	return t.Format("2006-01-02 15:04:05")
}

func FmtTimePtr(t *time.Time) string {
	if t == nil { return "" }
	tt := *t
	if config.Cfg.Server.TimezoneOffset != 0 { tt = tt.Add(time.Duration(config.Cfg.Server.TimezoneOffset) * time.Hour) }
	return tt.Format("2006-01-02 15:04:05")
}

func FmtTimeVal(t time.Time) string {
	if config.Cfg.Server.TimezoneOffset != 0 { t = t.Add(time.Duration(config.Cfg.Server.TimezoneOffset) * time.Hour) }
	return t.Format("2006-01-02 15:04:05")
}

func TzDateExpr() string {
	offset := config.Cfg.Server.TimezoneOffset
	if offset == 0 { return "DATE(created_at)" }
	sign := "+"
	if offset < 0 { sign = "" }
	return fmt.Sprintf("DATE(created_at + INTERVAL '%s%d hours')", sign, offset)
}

func ParseDateFilters(df, dt string) (string, string) {
	dfOut, dtOut := "", ""
	if df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			t = t.Add(time.Duration(-config.Cfg.Server.TimezoneOffset) * time.Hour)
			dfOut = t.Format("2006-01-02 15:04:05")
		}
	}
	if dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			t = t.AddDate(0, 0, 1).Add(time.Duration(-config.Cfg.Server.TimezoneOffset) * time.Hour)
			dtOut = t.Format("2006-01-02 15:04:05")
		}
	}
	return dfOut, dtOut
}

func NormalizeModelMapping(s string) string {
	s = strings.TrimSpace(s)
	if s == "" { return "" }
	if strings.HasPrefix(s, "{") { return s }
	parts := strings.Split(s, ",")
	m := make(map[string]string)
	for _, p := range parts {
		p = strings.TrimSpace(p); if p == "" { continue }
		if strings.Contains(p, ":") {
			kv := strings.SplitN(p, ":", 2); m[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		} else if strings.Contains(p, "=") {
			kv := strings.SplitN(p, "=", 2); m[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	if len(m) == 0 { return "" }
	b, _ := json.Marshal(m); return string(b)
}

func GetGroupAuthedModels(groupName, userAllowedModels string, pool interface{ GetModelsForGroup(group *string) []string }) []string {
	var gn *string
	if groupName != "" { gn = &groupName }
	models := pool.GetModelsForGroup(gn)
	// Empty allowed_models means no permission, not "all models"
	// If user has no custom models (inherit mode), the group's allowed_models
	// filtering is handled by the routing layer; userAllowedModels="" here
	// means inherit from group — return models filtered by group's upstream groups
	if userAllowedModels != "" {
		userModels := make(map[string]bool)
		for _, m := range SplitComma(userAllowedModels) { userModels[m] = true }
		var filtered []string
		for _, m := range models { if userModels[m] { filtered = append(filtered, m) } }
		models = filtered
	}
	return models
}
