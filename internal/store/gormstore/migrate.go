package gormstore

import (
	"embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type schemaMigration struct {
	Version   string    `gorm:"primaryKey;column:version"`
	AppliedAt time.Time `gorm:"column:applied_at;not null"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

// applyVersionedMigrations runs embedded SQL files not yet recorded in schema_migrations.
func applyVersionedMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.TrimSuffix(name, ".up.sql")
		var count int64
		if err := db.Model(&schemaMigration{}).Where("version = ?", version).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		raw, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if err := db.Exec(string(raw)).Error; err != nil {
			return fmt.Errorf("migration %s: %w", version, err)
		}
		if err := db.Create(&schemaMigration{Version: version, AppliedAt: time.Now().UTC()}).Error; err != nil {
			return err
		}
	}
	return nil
}
