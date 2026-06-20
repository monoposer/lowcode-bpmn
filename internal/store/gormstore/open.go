package gormstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Driver identifies a supported SQL backend.
type Driver string

const (
	DriverPostgres Driver = "postgres"
	DriverMySQL    Driver = "mysql"
	DriverSQLite   Driver = "sqlite"
)

// Open connects to the database using the given driver and DSN.
func Open(driver Driver, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case DriverPostgres:
		dialector = postgres.Open(dsn)
	case DriverMySQL:
		dialector = mysql.Open(dsn)
	case DriverSQLite:
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported DB driver %q (use postgres, mysql, or sqlite)", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// ParseDriver normalizes a driver name from configuration.
func ParseDriver(raw string) (Driver, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "postgres", "postgresql", "pg":
		return DriverPostgres, nil
	case "mysql", "mariadb":
		return DriverMySQL, nil
	case "sqlite", "sqlite3":
		return DriverSQLite, nil
	default:
		return "", fmt.Errorf("unsupported DB driver %q", raw)
	}
}

// NewStore constructs a Store and applies schema migrations.
func NewStore(ctx context.Context, db *gorm.DB) (*Store, error) {
	s := &Store{db: db, dialect: db.Dialector.Name()}
	if s.dialect == "postgres" {
		if err := applyVersionedMigrations(db); err != nil {
			return nil, err
		}
	}
	if err := s.autoMigrate(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) autoMigrate(ctx context.Context) error {
	db := s.db.WithContext(ctx)
	if err := db.AutoMigrate(
		&BpmnProcess{},
		&BpmnInstance{},
		&BpmnActivity{},
		&BpmnJob{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return s.ensureIndexes(ctx)
}

func (s *Store) ensureIndexes(ctx context.Context) error {
	if s.dialect != "postgres" {
		return nil
	}
	db := s.db.WithContext(ctx)
	m := db.Migrator()
	if m.HasIndex(&BpmnActivity{}, "idx_bpmn_activities_user_task") {
		return nil
	}
	return db.Exec(`
		CREATE INDEX idx_bpmn_activities_user_task
		ON bpmn_activities (process_instance_id, status)
		WHERE element_kind = 'userTask' AND status = 'active'
	`).Error
}
