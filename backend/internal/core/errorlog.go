package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zapi/zapi-go/internal/config"
)

type ErrorEntry struct { Message string; Time string }
type ErrorLogger struct {
	mu      sync.RWMutex
	entries []ErrorEntry
	max     int
	file    *os.File
	fileDay string
}

var ErrLog = &ErrorLogger{max: 0}

func (e *ErrorLogger) getMax() int {
	if e.max <= 0 {
		e.max = config.Cfg.Log.ErrorMaxEntries
		if e.max <= 0 { e.max = 1000 }
	}
	return e.max
}

// openFile ensures the log file for today is open, rotating daily
func (e *ErrorLogger) openFile() error {
	today := time.Now().Format("2006-01-02")
	if e.file != nil && e.fileDay == today {
		return nil
	}
	// Close previous file
	if e.file != nil {
		e.file.Close()
		e.file = nil
	}
	// Create logs directory
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[ErrLog] Failed to create logs directory: %v\n", err)
		e.fileDay = ""
		return err
	}
	// Open today's file
	path := filepath.Join(logDir, "zapi-"+today+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ErrLog] Failed to open log file %s: %v\n", path, err)
		e.fileDay = ""
		return err
	}
	e.file = f
	e.fileDay = today

	// Clean old log files on daily rotation
	go e.CleanOldFiles()

	return nil
}

func (e *ErrorLogger) Error(msg string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	entry := ErrorEntry{Message: msg, Time: now}

	e.mu.Lock()
	e.entries = append(e.entries, entry)
	if len(e.entries) > e.getMax() {
		e.entries = e.entries[len(e.entries)-e.getMax():]
	}
	// Write to file
	if e.openFile() == nil && e.file != nil {
		fmt.Fprintf(e.file, "[%s] %s\n", now, msg)
		e.file.Sync()
	}
	e.mu.Unlock()
}

func (e *ErrorLogger) GetEntries(limit int) []ErrorEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if limit <= 0 || limit > len(e.entries) {
		limit = len(e.entries)
	}
	result := make([]ErrorEntry, limit)
	copy(result, e.entries[len(e.entries)-limit:])
	return result
}

func (e *ErrorLogger) Clear() {
	e.mu.Lock()
	e.entries = nil
	e.mu.Unlock()
}

// Close closes the log file (call on shutdown)
func (e *ErrorLogger) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.file != nil {
		e.file.Close()
		e.file = nil
		e.fileDay = ""
	}
}

// CleanOldFiles removes error log files older than maxDays
func (e *ErrorLogger) CleanOldFiles() {
	maxDays := config.Cfg.Log.ErrorMaxDays
	if maxDays <= 0 {
		maxDays = 30
	}
	cutoff := time.Now().AddDate(0, 0, -maxDays)
	logDir := "logs"
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".log" {
			continue
		}
		// Parse date from filename: zapi-2026-04-23.log
		if len(entry.Name()) < 15 {
			continue
		}
		dateStr := entry.Name()[5:15] // "2026-04-23"
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			os.Remove(filepath.Join(logDir, entry.Name()))
		}
	}
}
