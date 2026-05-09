package core

import (
	"time"
	"github.com/zapi/zapi-go/internal/model"
	"gorm.io/gorm"
)

type quotaEntry struct { TokenID uint; UserID uint; TokensUsed int64; TokenKey string }
var quotaCh = make(chan quotaEntry, 65536)

// processQuotaEntry flushes a single quota entry directly to the database (fallback when channel is full)
func processQuotaEntry(e quotaEntry) {
	model.DB.Model(&model.Token{}).Where("id = ?", e.TokenID).UpdateColumn("quota_used", gorm.Expr("quota_used + ?", e.TokensUsed))
	if e.UserID > 0 {
		model.DB.Model(&model.User{}).Where("id = ?", e.UserID).UpdateColumn("token_quota_used", gorm.Expr("token_quota_used + ?", e.TokensUsed))
		InvalidateUserCache(e.UserID)
	}
	if e.TokenKey != "" {
		InvalidateTokenCache(e.TokenKey)
	}
}

func StartQuotaDeductor() {
	go func() {
		tokenBatch := make(map[uint]int64); userBatch := make(map[uint]int64)
		// Track API keys to invalidate per-token
		tokenKeysToInvalidate := make(map[string]bool)
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case e := <-quotaCh:
				tokenBatch[e.TokenID] += e.TokensUsed
				if e.UserID > 0 { userBatch[e.UserID] += e.TokensUsed }
				if e.TokenKey != "" {
					tokenKeysToInvalidate[e.TokenKey] = true
				}
			case <-ticker.C:
				for tid, amt := range tokenBatch {
					model.DB.Model(&model.Token{}).Where("id = ?", tid).UpdateColumn("quota_used", gorm.Expr("quota_used + ?", amt))
				}
				for uid, amt := range userBatch {
					model.DB.Model(&model.User{}).Where("id = ?", uid).UpdateColumn("token_quota_used", gorm.Expr("token_quota_used + ?", amt))
					InvalidateUserCache(uid)
				}
				// Invalidate only the specific token caches that were modified
				for key := range tokenKeysToInvalidate {
					InvalidateTokenCache(key)
				}
				tokenBatch = make(map[uint]int64); userBatch = make(map[uint]int64)
				tokenKeysToInvalidate = make(map[string]bool)
			case <-StopChan:
				// Flush remaining quota before exit
				for tid, amt := range tokenBatch {
					model.DB.Model(&model.Token{}).Where("id = ?", tid).UpdateColumn("quota_used", gorm.Expr("quota_used + ?", amt))
				}
				for uid, amt := range userBatch {
					model.DB.Model(&model.User{}).Where("id = ?", uid).UpdateColumn("token_quota_used", gorm.Expr("token_quota_used + ?", amt))
				}
				return
			}
		}
	}()
}

func DeductQuota(tokenID, userID uint, tokens int64, tokenKey string) {
	if tokens <= 0 { return }
	entry := quotaEntry{TokenID: tokenID, UserID: userID, TokensUsed: tokens, TokenKey: tokenKey}
	select {
	case quotaCh <- entry:
	default:
		// Buffer full — fall back to synchronous deduction
		processQuotaEntry(entry)
	}
}
