package core

import (
	"os"
	"time"

	"github.com/zapi/zapi-go/internal/config"
	"github.com/zapi/zapi-go/internal/model"
)

var logCh = make(chan model.Log, 65536)

// StopChan signals all background goroutines to exit
var StopChan = make(chan os.Signal, 1)

// processLogEntry writes a single log entry directly to the database (fallback when channel is full)
func processLogEntry(l model.Log) {
	model.DB.Create(&l)
}

func StartLogWriter() {
	go func() {
		batch := make([]model.Log, 0, config.Cfg.Log.BatchSize)
		ticker := time.NewTicker(time.Duration(config.Cfg.Log.BatchInterval) * time.Second)
		for {
			select {
			case l := <-logCh:
				batch = append(batch, l)
				if len(batch) >= config.Cfg.Log.BatchSize { model.DB.CreateInBatches(batch, len(batch)); batch = make([]model.Log, 0, config.Cfg.Log.BatchSize) }
			case <-ticker.C:
				if len(batch) > 0 { model.DB.CreateInBatches(batch, len(batch)); batch = make([]model.Log, 0, config.Cfg.Log.BatchSize) }
			case <-StopChan:
				// Flush remaining logs before exit
				if len(batch) > 0 { model.DB.CreateInBatches(batch, len(batch)) }
				return
			}
		}
	}()
}

func AddLog(l model.Log) {
	select {
	case logCh <- l:
	default:
		// Buffer full — fall back to synchronous write
		processLogEntry(l)
	}
}

func StartLogCleanup() {
	go func() {
		for {
			select {
			case <-time.After(time.Duration(config.Cfg.Log.CleanupIntervalHours) * time.Hour):
				if config.Cfg.Log.RetentionDays > 0 {
					cutoff := time.Now().UTC().AddDate(0, 0, -config.Cfg.Log.RetentionDays)
					batchSize := config.Cfg.Log.CleanupBatchSize
					if batchSize <= 0 { batchSize = 5000 }
					for {
					result := model.DB.Where("created_at < ?", cutoff).Limit(batchSize).Delete(&model.Log{})
					if result.RowsAffected == 0 { break }
					time.Sleep(100 * time.Millisecond)
				}
				}
			case <-StopChan:
				return
			}
		}
	}()
}
