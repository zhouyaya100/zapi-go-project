package core

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/zapi/zapi-go/internal/config"
	"github.com/zapi/zapi-go/internal/migrate"
	"github.com/zapi/zapi-go/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB() error {
	var err error
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), NowFunc: func() time.Time { return time.Now().UTC() }}
	if strings.Contains(config.Cfg.Database.URL, "sqlite") {
		dbPath := strings.TrimPrefix(config.Cfg.Database.URL, "sqlite:///")
		model.DB, err = gorm.Open(sqlite.Open(dbPath), cfg)
		if err != nil { return err }
		model.DB.Exec("PRAGMA journal_mode=WAL")
		model.DB.Exec("PRAGMA busy_timeout=5000")
	} else {
		model.DB, err = gorm.Open(postgres.Open(config.Cfg.Database.URL), cfg)
		if err != nil { return err }
		sqlDB, err := model.DB.DB()
		if err != nil { return fmt.Errorf("failed to get underlying sql.DB: %w", err) }
		sqlDB.SetMaxOpenConns(config.Cfg.Database.PoolSize + config.Cfg.Database.MaxOverflow)
		sqlDB.SetMaxIdleConns(config.Cfg.Database.PoolSize)
		sqlDB.SetConnMaxLifetime(time.Duration(config.Cfg.Database.PoolRecycle) * time.Second)
	}
	model.DB.AutoMigrate(&model.Channel{}, &model.User{}, &model.Token{}, &model.Group{}, &model.Log{}, &model.Notification{}, &model.UpstreamGroup{}, &model.UpstreamGroupChannel{}, &model.UserUpstreamGroup{}, &model.GroupUpstreamGroup{})
	// Verify critical tables exist
	if !model.DB.Migrator().HasTable(&model.UpstreamGroup{}) {
		log.Println("[DB] WARNING: upstream_groups table not created by AutoMigrate, creating manually...")
		model.DB.Migrator().CreateTable(&model.UpstreamGroup{})
	}
	// Run schema migrations (handles new columns, indexes, etc.)
	migrate.RunMigrations(model.DB)
	log.Println("[DB] Connected")
	return nil
}

func SeedDefaults() {
	var user model.User
	if model.DB.Where("id = ?", model.SuperAdminID).First(&user).Error != nil {
		hash, _ := HashPassword("Admin@123")
		model.DB.Create(&model.User{ID: model.SuperAdminID, Username: "admin", PasswordHash: hash, Role: "admin", Enabled: true, MaxTokens: 100, TokenQuota: -1})
		log.Println("[DB] Seeded default admin user")
	}
	var grp model.Group
	if model.DB.Where("name = ?", "Default").First(&grp).Error != nil {
		model.DB.Create(&model.Group{Name: "Default", Comment: "\u9ed8\u8ba4\u5206\u7ec4"})
		log.Println("[DB] Seeded default group")
	}
}
