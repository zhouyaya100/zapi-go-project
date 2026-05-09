package model

import (
	"time"
	"gorm.io/gorm"
)

const SuperAdminID = 1

type Channel struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	BaseURL         string     `gorm:"column:base_url" json:"base_url"`
	APIKey          string     `gorm:"column:api_key" json:"api_key"`
	Models          string     `json:"models"`
	ModelMapping    string     `gorm:"column:model_mapping" json:"model_mapping"`
	AllowedGroups   string     `gorm:"column:allowed_groups" json:"allowed_groups"`
	Weight          int        `json:"weight"`
	Priority        int        `json:"priority"`
	Enabled         bool       `json:"enabled"`
	AutoBan         bool       `gorm:"column:auto_ban" json:"auto_ban"`
	FailCount       int        `gorm:"column:fail_count" json:"fail_count"`
	TestTime        *time.Time `gorm:"column:test_time" json:"test_time"`
	ResponseTime    int        `gorm:"column:response_time" json:"response_time"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (Channel) TableName() string { return "channels" }

// UpstreamGroupChannel — many-to-many join table: channel can belong to multiple upstream groups
type UpstreamGroupChannel struct {
	UpstreamGroupID uint `gorm:"primaryKey;column:upstream_group_id" json:"upstream_group_id"`
	ChannelID       uint `gorm:"primaryKey;column:channel_id" json:"channel_id"`
}

func (UpstreamGroupChannel) TableName() string { return "upstream_group_channels" }

// UserUpstreamGroup — many-to-many: user can belong to multiple upstream groups
type UserUpstreamGroup struct {
	UserID         uint `gorm:"primaryKey;column:user_id" json:"user_id"`
	UpstreamGroupID uint `gorm:"primaryKey;column:upstream_group_id" json:"upstream_group_id"`
}

func (UserUpstreamGroup) TableName() string { return "user_upstream_groups" }

// GroupUpstreamGroup — many-to-many: group can bind multiple upstream groups
type GroupUpstreamGroup struct {
	GroupID         uint `gorm:"primaryKey;column:group_id" json:"group_id"`
	UpstreamGroupID uint `gorm:"primaryKey;column:upstream_group_id" json:"upstream_group_id"`
}

func (GroupUpstreamGroup) TableName() string { return "group_upstream_groups" }

type User struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Username       string    `gorm:"uniqueIndex" json:"username"`
	PasswordHash   string    `gorm:"column:password_hash" json:"-"`
	Role           string    `gorm:"default:user" json:"role"`
	GroupID           *uint     `gorm:"column:group_id" json:"group_id"`
	Enabled        bool      `gorm:"default:true" json:"enabled"`
	MaxTokens      int       `gorm:"column:max_tokens;default:3" json:"max_tokens"`
	TokenQuota     int64     `gorm:"column:token_quota;default:-1" json:"token_quota"`
	TokenQuotaUsed int64     `gorm:"column:token_quota_used;default:0" json:"token_quota_used"`
	AllowedModels  string    `gorm:"column:allowed_models" json:"allowed_models"`
	BindMode       string    `gorm:"column:bind_mode;default:'inherit'" json:"bind_mode"` // inherit=use group's models/upstream_groups, custom=use own (empty=none)
	RateMode       string    `gorm:"column:rate_mode;default:'inherit'" json:"rate_mode"` // inherit|global|per_model
	RPM            int       `gorm:"column:rpm;default:0" json:"rpm"`   // 0=inherit (use group), -1=unlimited, >0=limit
	TPM            int64     `gorm:"column:tpm;default:0" json:"tpm"`   // 0=inherit (use group), -1=unlimited, >0=limit
	ModelRateLimits string `gorm:"column:model_rate_limits;type:text" json:"model_rate_limits"` // JSON: {"gpt-4":{"rpm":5,"tpm":10000},"*":{"rpm":10}}
	CreatedAt      time.Time `json:"created_at"`
}

func (User) TableName() string { return "users" }

type Token struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"column:user_id;index" json:"user_id"`
	Name       string     `json:"name"`
	Key        string     `gorm:"uniqueIndex" json:"key"`
	Models     string     `json:"models"`
	Enabled    bool       `gorm:"default:true" json:"enabled"`
	QuotaLimit int64      `gorm:"column:quota_limit;default:-1" json:"quota_limit"`
	QuotaUsed  int64      `gorm:"column:quota_used;default:0" json:"quota_used"`
	ExpiresAt  *time.Time `gorm:"column:expires_at" json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (Token) TableName() string { return "tokens" }

type Group struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex" json:"name"`
	Comment   string    `json:"comment"`
	AllowedModels  string    `gorm:"column:allowed_models;type:text" json:"allowed_models"` // comma-separated, empty=no models allowed
	RateMode       string    `gorm:"column:rate_mode;default:'global'" json:"rate_mode"` // global|per_model
	RPM       int       `gorm:"column:rpm;default:-1" json:"rpm"`   // 0=blocked, -1=unlimited, >0=limit
	TPM       int64     `gorm:"column:tpm;default:-1" json:"tpm"`   // 0=blocked, -1=unlimited, >0=limit
	ModelRateLimits string `gorm:"column:model_rate_limits;type:text" json:"model_rate_limits"` // JSON: {"gpt-4":{"rpm":5,"tpm":10000},"*":{"rpm":10}}
	CreatedAt time.Time `json:"created_at"`
}

func (Group) TableName() string { return "groups" }

type Log struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"column:user_id;index" json:"user_id"`
	TokenID          uint      `gorm:"column:token_id;index" json:"token_id"`
		TokenName        string    `gorm:"column:token_name" json:"token_name"`
	ChannelID        uint      `gorm:"column:channel_id;index" json:"channel_id"`
	ChannelName      string    `gorm:"column:channel_name" json:"channel_name"`
	UpstreamGroupID  uint      `gorm:"column:upstream_group_id;default:0;index" json:"upstream_group_id"`
	Model            string    `gorm:"index" json:"model"`
	IsStream         bool      `gorm:"column:is_stream" json:"is_stream"`
	PromptTokens     int64     `gorm:"column:prompt_tokens" json:"prompt_tokens"`
	CompletionTokens int64     `gorm:"column:completion_tokens" json:"completion_tokens"`
	CachedTokens     int64     `gorm:"column:cached_tokens;default:0" json:"cached_tokens"`
	LatencyMs        int       `gorm:"column:latency_ms" json:"latency_ms"`
	Success          bool      `json:"success"`
	ErrorMsg         string    `gorm:"column:error_msg;type:text" json:"error_msg"`
	ClientIP         string    `gorm:"column:client_ip" json:"client_ip"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`
}

func (Log) TableName() string { return "logs" }

type Notification struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SenderID   uint      `gorm:"column:sender_id;index" json:"sender_id"`
	ReceiverID *uint     `gorm:"column:receiver_id;index" json:"receiver_id"`
	Title      string    `json:"title"`
	Content    string    `gorm:"type:text" json:"content"`
	Category   string    `gorm:"default:info" json:"category"`
	Read       bool      `gorm:"default:false" json:"read"`
	CreatedAt  time.Time `json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }

// UpstreamGroup — load-balanced channel group
type UpstreamGroup struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	Name                 string    `gorm:"uniqueIndex" json:"name"`
	Alias                string    `json:"alias"`                                              // exposed model name
	Strategy             string    `gorm:"default:'priority'" json:"strategy"`                 // priority|round_robin|weighted|least_latency|least_requests
	AllowedGroups        string    `gorm:"column:allowed_groups" json:"allowed_groups"`        // comma-separated group names
	Enabled              bool      `gorm:"default:true" json:"enabled"`
	HealthCheckInterval  int       `gorm:"column:health_check_interval;default:0" json:"health_check_interval"` // 0=follow global heartbeat
	MaxFails             int       `gorm:"column:max_fails;default:5" json:"max_fails"`
	FailTimeout          int       `gorm:"column:fail_timeout;default:30" json:"fail_timeout"`  // seconds
	RetryOnFail          bool      `gorm:"column:retry_on_fail;default:true" json:"retry_on_fail"`
	CreatedAt            time.Time `json:"created_at"`
}

func (UpstreamGroup) TableName() string { return "upstream_groups" }

// DB global instance
var DB *gorm.DB
