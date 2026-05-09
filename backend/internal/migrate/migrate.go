package migrate

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
)

// Migration represents a single database migration
type Migration struct {
	Version string
	Up      func(db *gorm.DB) error
}

// Migrations is the ordered list of all migrations
var Migrations = []Migration{
	{
		Version: "3.8.0",
		Up: func(db *gorm.DB) error {
			log.Println("[MIGRATE] 3.8.0: Adding rate limit columns...")
			db.Exec("ALTER TABLE groups ADD COLUMN IF NOT EXISTS rate_mode TEXT DEFAULT 'global'")
			db.Exec("ALTER TABLE groups ADD COLUMN IF NOT EXISTS rpm INTEGER DEFAULT 0")
			db.Exec("ALTER TABLE groups ADD COLUMN IF NOT EXISTS tpm BIGINT DEFAULT 0")
			db.Exec("ALTER TABLE groups ADD COLUMN IF NOT EXISTS model_rate_limits TEXT DEFAULT ''")
			db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS rate_mode TEXT DEFAULT 'inherit'")
			db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS rpm INTEGER DEFAULT 0")
			db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS tpm BIGINT DEFAULT 0")
			db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS model_rate_limits TEXT DEFAULT ''")
			return nil
		},
	},
	{
		Version: "3.8.1",
		Up: func(db *gorm.DB) error {
			log.Println("[MIGRATE] 3.8.1: Adding performance indexes...")
			db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_user_created ON logs(user_id, created_at)")
			db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_created ON logs(created_at)")
			db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_success ON logs(success)")
			db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_model ON logs(model)")
			return nil
		},
	},
	{
		Version: "4.2.0",
		Up: func(db *gorm.DB) error {
			log.Println("[MIGRATE] 4.2.0: Adding upstream group columns and table...")
			db.Exec("ALTER TABLE channels ADD COLUMN IF NOT EXISTS upstream_group_id INTEGER DEFAULT NULL")
			// backup column was removed in v4.2.0 — drop if exists
			db.Exec("ALTER TABLE channels DROP COLUMN IF EXISTS backup")
			return nil
		},
	},
	{
		Version: "4.3.0",
		Up: func(db *gorm.DB) error {
			log.Println("[MIGRATE] 4.3.0: Many-to-many upstream_group_channels + migrate data...")
			// Create join table
			db.Exec(`CREATE TABLE IF NOT EXISTS upstream_group_channels (
				upstream_group_id INTEGER NOT NULL,
				channel_id INTEGER NOT NULL,
				PRIMARY KEY (upstream_group_id, channel_id)
			)`)
			// Migrate old data: channels.upstream_group_id → upstream_group_channels rows
			db.Exec(`INSERT INTO upstream_group_channels (upstream_group_id, channel_id)
				SELECT upstream_group_id, id FROM channels
				WHERE upstream_group_id IS NOT NULL
				ON CONFLICT DO NOTHING`)
			// Drop old column (keep for safety, no data loss)
			// db.Exec("ALTER TABLE channels DROP COLUMN IF EXISTS upstream_group_id")
			return nil
		},
	},
	{
		Version: "4.4.0",
		Up: func(db *gorm.DB) error {
			log.Println("[MIGRATE] 4.4.0: Adding upstream_group_id to users...")
			db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS upstream_group_id INTEGER DEFAULT NULL")
			return nil
		},
	},
	{
		Version: "4.5.0",
		Up: func(db *gorm.DB) error {
			log.Println("[MIGRATE] 4.5.0: User upstream groups many-to-many...")
			// Create join table
			db.Exec(`CREATE TABLE IF NOT EXISTS user_upstream_groups (
				user_id INTEGER NOT NULL,
				upstream_group_id INTEGER NOT NULL,
				PRIMARY KEY (user_id, upstream_group_id)
			)`)
			// Migrate old data: users.upstream_group_id → user_upstream_groups rows
			db.Exec(`INSERT INTO user_upstream_groups (user_id, upstream_group_id)
				SELECT id, upstream_group_id FROM users
				WHERE upstream_group_id IS NOT NULL AND upstream_group_id != 0
				ON CONFLICT DO NOTHING`)
			// Drop old column (SQLite doesn't support DROP COLUMN before 3.35.0, so keep it)
			// db.Exec("ALTER TABLE users DROP COLUMN IF EXISTS upstream_group_id")
			return nil
		},
	},
	{
		Version: "4.6.0",
		Up: func(db *gorm.DB) error {
			log.Println("[MIGRATE] 4.6.0: Adding upstream_group_id to logs...")
			db.Exec("ALTER TABLE logs ADD COLUMN IF NOT EXISTS upstream_group_id INTEGER DEFAULT 0")
			db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_upstream_group_id ON logs(upstream_group_id)")
			return nil
		},
	},
	{
		Version: "4.7.0",
		Up: func(db *gorm.DB) error {
			log.Println("[MIGRATE] 4.7.0: Adding cached_tokens to logs...")
			if err := db.Exec("ALTER TABLE logs ADD COLUMN IF NOT EXISTS cached_tokens BIGINT NOT NULL DEFAULT 0").Error; err != nil {
				return err
			}
			return nil
		},
	},
	{
		Version: "4.8.0",
		Up: func(db *gorm.DB) error {
			log.Println("[MIGRATE] 4.8.0: Migrate rate limit semantics — 0 now means 'blocked', was 'unlimited'. Convert existing 0s to -1 (unlimited)...")
			// Groups: rpm=0/tpm=0 was "use global" (effectively unlimited) → now -1 (unlimited)
			db.Exec("UPDATE groups SET rpm = -1 WHERE rpm = 0")
			db.Exec("UPDATE groups SET tpm = -1 WHERE tpm = 0")
			// Users with rate_mode='global' and rpm=0/tpm=0 was "use global" → change to rate_mode='inherit' (use group)
			// Users with rate_mode='inherit' already inherit from group, rpm=0/tpm=0 is fine (means inherit)
			db.Exec("UPDATE users SET rate_mode = 'inherit', rpm = 0, tpm = 0 WHERE rate_mode = 'global' AND rpm = 0 AND tpm = 0")
			// Users with rate_mode='global' and rpm>0 or tpm>0 keep their values (those are intentional limits)
			// Users with rate_mode='global' and rpm=-1 or tpm=-1 keep their values (those are intentional unlimited)
			return nil
		},
	},
	{
		Version: "4.9.0",
		Up: func(db *gorm.DB) error {
			log.Println("[MIGRATE] 4.9.0: Adding group allowed_models, group_upstream_groups, user bind_mode...")
			// Add allowed_models to groups
			db.Exec("ALTER TABLE groups ADD COLUMN IF NOT EXISTS allowed_models TEXT DEFAULT ''")
			// Add bind_mode to users (inherit=use group's models/upstream_groups, custom=use own)
			db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS bind_mode TEXT DEFAULT 'inherit'")
			// Create group_upstream_groups join table
			db.Exec(`CREATE TABLE IF NOT EXISTS group_upstream_groups (
				group_id INTEGER NOT NULL,
				upstream_group_id INTEGER NOT NULL,
				PRIMARY KEY (group_id, upstream_group_id)
			)`)
			// Existing users with upstream_group_ids should be set to custom mode
			db.Exec(`UPDATE users SET bind_mode = 'custom' WHERE id IN (SELECT DISTINCT user_id FROM user_upstream_groups)`)

			// Auto-fill allowed_models for groups where it's empty (was "all allowed", now must be explicit)
			// Collect all models available to each group via its upstream groups' channels
			type GroupInfo struct {
				ID uint
			}
		var groups []GroupInfo
			db.Raw("SELECT id FROM groups WHERE allowed_models = '' OR allowed_models IS NULL").Scan(&groups)
			for _, g := range groups {
				// Find models from channels reachable through this group's upstream groups
				var models []string
				db.Raw(`SELECT DISTINCT c.models FROM channels c
					JOIN upstream_group_channels ugc ON c.id = ugc.channel_id
					JOIN group_upstream_groups gug ON ugc.upstream_group_id = gug.upstream_group_id
					WHERE gug.group_id = ? AND c.models != '' AND c.enabled = true`, g.ID).Scan(&models)
				if len(models) == 0 {
					// No upstream groups — try direct channel associations via allowed_groups
					// allowed_groups stores group names (comma-separated), not IDs
					var groupName string
					db.Raw("SELECT name FROM groups WHERE id = ?", g.ID).Scan(&groupName)
					if groupName != "" {
						db.Raw(`SELECT DISTINCT c.models FROM channels c
							WHERE c.allowed_groups != '' AND c.enabled = true AND (
								c.allowed_groups LIKE ? OR c.allowed_groups LIKE ? OR c.allowed_groups LIKE ? OR c.allowed_groups = ?
							)`,
							fmt.Sprintf("%%,%s,%%", groupName),
							fmt.Sprintf("%s,%%", groupName),
							fmt.Sprintf("%%,%s", groupName),
							groupName,
						).Scan(&models)
					}
				}
				// Deduplicate and flatten
				seen := map[string]bool{}
				var allModels []string
				for _, m := range models {
					for _, name := range strings.Split(m, ",") {
						name = strings.TrimSpace(name)
						if name != "" && !seen[name] {
							seen[name] = true
							allModels = append(allModels, name)
						}
					}
				}
				if len(allModels) > 0 {
					merged := strings.Join(allModels, ",")
					db.Exec("UPDATE groups SET allowed_models = ? WHERE id = ?", merged, g.ID)
					log.Printf("[MIGRATE] 4.9.0: Group %d auto-filled allowed_models: %s", g.ID, merged)
				}
				// No channels found = no models allowed (leave empty)
			}
			return nil
		},
	},
}

// schemaMigrations tracks applied migrations
type schemaMigration struct {
	Version string `gorm:"primaryKey"`
	Applied bool   `gorm:"default:false"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

// RunMigrations executes all pending migrations
func RunMigrations(db *gorm.DB) {
	db.AutoMigrate(&schemaMigration{})

	for _, m := range Migrations {
		var sm schemaMigration
		if db.Where("version = ?", m.Version).First(&sm).Error == nil && sm.Applied {
			continue // already applied
		}
		if err := m.Up(db); err != nil {
			log.Printf("[MIGRATE] ERROR %s: %v", m.Version, err)
			continue
		}
		db.Where("version = ?", m.Version).Assign("applied", true).FirstOrCreate(&schemaMigration{Version: m.Version})
		log.Printf("[MIGRATE] %s: OK", m.Version)
	}
}
