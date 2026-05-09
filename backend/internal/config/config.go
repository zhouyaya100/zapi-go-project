package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	Security     SecurityConfig     `yaml:"security"`
	Proxy        ProxyConfig        `yaml:"proxy"`
	Cache        CacheConfig        `yaml:"cache"`
	RateLimit    RateLimitConfig    `yaml:"rate_limit"`
	Log          LogConfig          `yaml:"log"`
	Registration RegistrationConfig `yaml:"registration"`
	Heartbeat    HeartbeatConfig    `yaml:"heartbeat"`
}

type ServerConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	TimezoneOffset int    `yaml:"timezone_offset"`
}
type DatabaseConfig struct {
	URL         string `yaml:"url"`
	PoolSize    int    `yaml:"pool_size"`
	MaxOverflow int    `yaml:"max_overflow"`
	PoolRecycle int    `yaml:"pool_recycle"`
}
type SecurityConfig struct {
	AdminToken     string `yaml:"admin_token"`
	SecretKey      string `yaml:"secret_key"`
	JWTExpireHours int    `yaml:"jwt_expire_hours"`
	CORSOrigins    string `yaml:"cors_origins"`
}
type ProxyConfig struct {
	Timeout         int `yaml:"timeout"`
	MaxConnections  int `yaml:"max_connections"`
	MaxKeepalive    int `yaml:"max_keepalive"`
	KeepaliveExpiry int `yaml:"keepalive_expiry"`
	RetryCount      int `yaml:"retry_count"`
	MaxFails        int `yaml:"max_fails"`        // global default: consecutive failures before circuit breaker trips (0=disabled)
	FailTimeout     int `yaml:"fail_timeout"`     // global default: seconds before circuit breaker half-open (auto-recover)
}
type CacheConfig struct {
	Enabled    bool `yaml:"enabled"`
	TTL        int  `yaml:"ttl"`
	MaxEntries int  `yaml:"max_entries"`
}
type RateLimitConfig struct {
}
type LogConfig struct {
	BatchSize            int `yaml:"batch_size"`
	BatchInterval        int `yaml:"batch_interval"`
	RetentionDays        int `yaml:"retention_days"`
	CleanupIntervalHours int `yaml:"cleanup_interval_hours"`
	CleanupBatchSize     int `yaml:"cleanup_batch_size"`
	ErrorMaxEntries      int `yaml:"error_max_entries"`
	ErrorMaxDays         int `yaml:"error_max_days"`
}
type RegistrationConfig struct {
	AllowRegister     bool   `yaml:"allow_register"`
	DefaultMaxTokens  int    `yaml:"default_max_tokens"`
	DefaultTokenQuota int64  `yaml:"default_token_quota"`
	DefaultGroup      string `yaml:"default_group"`
	MinPasswordLength int    `yaml:"min_password_length"`
}
type HeartbeatConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
	Timeout  int  `yaml:"timeout"`
	Retries  int  `yaml:"retries"`
}

var cfgMu sync.RWMutex

// ConfigFilePath stores the absolute path of the loaded config file for safe writes
var ConfigFilePath string

func GetCfg() Config { cfgMu.RLock(); defer cfgMu.RUnlock(); return Cfg }
func SetCfg(f func()) { cfgMu.Lock(); defer cfgMu.Unlock(); f() }

var Cfg = Config{
	Server:       ServerConfig{Host: "0.0.0.0", Port: 65000, TimezoneOffset: 8},
	Database:     DatabaseConfig{URL: "sqlite:///zapi.db", PoolSize: 20, MaxOverflow: 10, PoolRecycle: 3600},
	Security:     SecurityConfig{CORSOrigins: "*", JWTExpireHours: 24},
	Proxy:        ProxyConfig{Timeout: 60, MaxConnections: 100, MaxKeepalive: 20, KeepaliveExpiry: 60, RetryCount: 1, MaxFails: 5, FailTimeout: 30},
	Cache:        CacheConfig{Enabled: true, TTL: 300, MaxEntries: 1000},
	RateLimit:    RateLimitConfig{},
	Log:          LogConfig{BatchSize: 50, BatchInterval: 5, RetentionDays: 30, CleanupIntervalHours: 24, CleanupBatchSize: 500, ErrorMaxEntries: 1000},
	Registration: RegistrationConfig{AllowRegister: true, DefaultMaxTokens: 3, DefaultTokenQuota: -1, DefaultGroup: "Default", MinPasswordLength: 8},
	Heartbeat:    HeartbeatConfig{Enabled: true, Interval: 60, Timeout: 10, Retries: 3},
}

func Load(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	ConfigFilePath = absPath
	data, err := os.ReadFile(absPath)
	if err != nil { return fmt.Errorf("read config: %w", err) }
	var parseErr error
	SetCfg(func() {
		if parseErr = yaml.Unmarshal(data, &Cfg); parseErr != nil { return }
		if v := os.Getenv("ZAPI_ADMIN_TOKEN"); v != "" { Cfg.Security.AdminToken = v }
		if v := os.Getenv("ZAPI_SECRET_KEY"); v != "" { Cfg.Security.SecretKey = v }
		if v := os.Getenv("ZAPI_DB_URL"); v != "" { Cfg.Database.URL = v }
		if Cfg.Security.SecretKey == "" { Cfg.Security.SecretKey = fmt.Sprintf("sk-secret-%d", time.Now().UnixNano()) }
	})
	if parseErr != nil { return fmt.Errorf("parse config: %w", parseErr) }
	return nil
}

func TimezoneDuration() time.Duration {
	return time.Duration(Cfg.Server.TimezoneOffset) * time.Hour
}
