package core

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zapi/zapi-go/internal/config"
	"github.com/zapi/zapi-go/internal/model"
	"github.com/zapi/zapi-go/internal/core/routing"
)

var (
	heartbeatNotified   = make(map[uint]bool)   // channelID -> already notified
	heartbeatFaultStart = make(map[uint]time.Time) // channelID -> fault start time
	heartbeatMu         sync.Mutex
)

func heartbeatTimeout() time.Duration {
	t := config.Cfg.Heartbeat.Timeout
	if t <= 0 { t = 10 }
	return time.Duration(t) * time.Second
}

func StartHeartbeat() {
	if !config.Cfg.Heartbeat.Enabled { return }
	go func() {
		for {
			select {
			case <-time.After(time.Duration(config.Cfg.Heartbeat.Interval) * time.Second):
				checkAllChannels()
			case <-StopChan:
				return
			}
		}
	}()
	log.Println("[HEARTBEAT] Started (lightweight: GET /v1/models)")
}

// buildModelsURL constructs the /v1/models URL from a channel's BaseURL
func buildModelsURL(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	// If base ends with /v1, just append /models
	if strings.HasSuffix(base, "/v1") {
		return base + "/models"
	}
	// If base ends with /v1/, trim and append
	if strings.HasSuffix(base, "/v1/") {
		return strings.TrimRight(base, "/") + "/models"
	}
	// Otherwise append /v1/models
	return base + "/v1/models"
}

func checkAllChannels() {
	var channels []model.Channel
	model.DB.Find(&channels)

	// Get all admin+operator user IDs for notifications
	var adminIDs []uint
	model.DB.Model(&model.User{}).Where("role IN ?", []string{"admin", "operator"}).Pluck("id", &adminIDs)

	for i := range channels {
		ch := &channels[i]
		testURL := buildModelsURL(ch.BaseURL)

		client := &http.Client{Timeout: heartbeatTimeout()}
		req, _ := http.NewRequest("GET", testURL, nil)
		req.Header.Set("Authorization", "Bearer "+ch.APIKey)
		start := time.Now(); resp, err := client.Do(req); latency := int(time.Since(start).Milliseconds())
		now := time.Now().UTC(); ch.TestTime = &now

	if err != nil {
		// Connection failed — this is a real failure (DNS, connection refused, etc.)
		ch.FailCount++; ch.ResponseTime = 0
		mf, ft := routing.Upstreams.GetMaxFailsForChannel(ch.ID)
		autoBanThreshold := mf
		if autoBanThreshold <= 0 { autoBanThreshold = 5 }
		if ch.AutoBan && ch.FailCount >= autoBanThreshold { ch.Enabled = false }
		model.DB.Save(ch); routing.Pool.UpdateChannel(ch)
		// Sync heartbeat failure to circuit breaker so IsAvailable() can skip this channel
		routing.Health.SyncFromHeartbeat(ch.ID, ch.FailCount, mf, ft)
		handleChannelFault(ch, adminIDs)
		continue
	}
		resp.Body.Close(); ch.ResponseTime = latency

		if resp.StatusCode < 500 {
			// Any non-5xx response proves the upstream server is alive and responding:
			// 200 = OK, 401/403 = auth issue, 404 = endpoint not supported, 429 = rate limited
			// Only 5xx indicates the server itself is unhealthy
			wasDisabled := ch.FailCount > 0 || !ch.Enabled
			ch.FailCount = 0
			// Auto-re-enable if channel was disabled by auto-ban
			if !ch.Enabled && wasDisabled { ch.Enabled = true }
			model.DB.Save(ch); routing.Pool.UpdateChannel(ch)
			routing.Health.RecordSuccess(ch.ID, latency)
			handleChannelRecovery(ch, latency, adminIDs)
		} else {
			// Non-2xx/non-auth response (5xx, etc.) — server is alive but unhealthy
			ch.FailCount++
			mf, ft := routing.Upstreams.GetMaxFailsForChannel(ch.ID)
			autoBanThreshold := mf
			if autoBanThreshold <= 0 { autoBanThreshold = 5 }
			if ch.AutoBan && ch.FailCount >= autoBanThreshold { ch.Enabled = false }
			model.DB.Save(ch); routing.Pool.UpdateChannel(ch)
			// Sync heartbeat failure to circuit breaker
			routing.Health.SyncFromHeartbeat(ch.ID, ch.FailCount, mf, ft)
			handleChannelFault(ch, adminIDs)
		}
	}
}

func handleChannelFault(ch *model.Channel, adminIDs []uint) {
	heartbeatMu.Lock()
	defer heartbeatMu.Unlock()

	// Notify when: enabled and failCount >= 2, or already disabled and not yet notified
	shouldAlert := (ch.Enabled && ch.FailCount >= 2) || (!ch.Enabled && !heartbeatNotified[ch.ID])
	if shouldAlert && !heartbeatNotified[ch.ID] {
		status := "运行中"
		if !ch.Enabled { status = "已禁用" }
		title := fmt.Sprintf("渠道故障: %s", ch.Name)
		content := fmt.Sprintf("渠道 [%s] (ID:%d) 健康检测失败，当前状态: %s，失败次数: %d，地址: %s", ch.Name, ch.ID, status, ch.FailCount, ch.BaseURL)
		sendHeartbeatNotifications(adminIDs, "fault", title, content)
		heartbeatNotified[ch.ID] = true
		heartbeatFaultStart[ch.ID] = time.Now()
		ErrLog.Error(fmt.Sprintf("心跳故障: 渠道[%s] ID:%d 失败次数:%d", ch.Name, ch.ID, ch.FailCount))
	}
}

func handleChannelRecovery(ch *model.Channel, latency int, adminIDs []uint) {
	heartbeatMu.Lock()
	defer heartbeatMu.Unlock()

	if heartbeatNotified[ch.ID] {
		duration := ""
		if startTime, ok := heartbeatFaultStart[ch.ID]; ok {
			delta := time.Since(startTime)
			mins := int(delta.Minutes())
			if mins >= 60 {
				duration = fmt.Sprintf("，断线时长约 %d小时%d分钟", mins/60, mins%60)
			} else {
				duration = fmt.Sprintf("，断线时长约 %d分钟", mins)
			}
		}
		title := fmt.Sprintf("渠道恢复: %s", ch.Name)
		content := fmt.Sprintf("渠道 [%s] (ID:%d) 已恢复正常，当前延迟: %dms%s，地址: %s", ch.Name, ch.ID, latency, duration, ch.BaseURL)
		sendHeartbeatNotifications(adminIDs, "recovery", title, content)
		delete(heartbeatNotified, ch.ID)
		delete(heartbeatFaultStart, ch.ID); ErrLog.Error(fmt.Sprintf("心跳恢复: 渠道[%s] ID:%d", ch.Name, ch.ID))
	}
}

func sendHeartbeatNotifications(adminIDs []uint, category, title, content string) {
	for _, uid := range adminIDs {
		notif := model.Notification{
			SenderID:   0, // System notification
			ReceiverID: &uid,
			Category:   category,
			Title:      title,
			Content:    content,
		}
		model.DB.Create(&notif)
	}
}
