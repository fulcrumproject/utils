package gormpg

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TestDB contains the database connection and utility functions for tests
type TestDB struct {
	DB         *gorm.DB
	MainConf   *Conf
	TestDBName string
}

// NewTestDB creates a new instance of TestDB
func NewTestDB(t *testing.T, cfg *Conf, migrate func(db *gorm.DB) error) *TestDB {
	// Generate a unique database name using properties.UUID without hyphens
	uuidStr := strings.Replace(uuid.New().String(), "-", "", -1)
	dbName := fmt.Sprintf("fulcrum_test_%s", uuidStr)

	// Connect to default fulcrum database to create the test database
	adminDB, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to postgres database: %v", err)
	}

	// Create the test database
	sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Fatalf("Failed to drop test database: %v", err)
	}

	sql = fmt.Sprintf("CREATE DATABASE %s", dbName)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to the test database
	testCfg := &Conf{
		DSN:       replaceDatabaseInDSN(cfg.DSN, dbName),
		LogLevel:  cfg.LogLevel,
		LogFormat: cfg.LogFormat,
	}
	db, err := NewConnection(testCfg)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Perform migrations
	if err := migrate(db); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return &TestDB{
		DB:         db,
		TestDBName: dbName,
		MainConf:   cfg,
	}
}

// replaceDatabaseInDSN replaces the database name in a PostgreSQL DSN string
// Format: "host=localhost user=fulcrum password=password dbname=fulcrum_db port=5432 sslmode=disable"
func replaceDatabaseInDSN(dsn, newDBName string) string {
	re := regexp.MustCompile(`dbname=\S+`)
	return re.ReplaceAllString(dsn, "dbname="+newDBName)
}

// Cleanup removes the test database
func (tdb *TestDB) Cleanup(t *testing.T) {
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		t.Errorf("Failed to get underlying *sql.DB: %v", err)
		return
	}

	// Close all database connections
	if err := sqlDB.Close(); err != nil {
		t.Errorf("Failed to close database connection: %v", err)
		return
	}

	adminDB, err := NewConnection(tdb.MainConf)
	if err != nil {
		t.Errorf("Failed to connect to postgres database: %v", err)
		return
	}

	// Force close all connections to the test database
	sql := fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()`,
		tdb.TestDBName,
	)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Errorf("Failed to terminate database connections: %v", err)
	}

	// Delete the test database
	sql = fmt.Sprintf("DROP DATABASE IF EXISTS %s", tdb.TestDBName)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Errorf("Failed to drop test database: %v", err)
	}
}
