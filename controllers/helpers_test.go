package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// setupControllersDB points database.Instance at a fresh in-memory SQLite DB with
// the given models migrated.
func setupControllersDB(t *testing.T, tables ...interface{}) {
	t.Helper()

	dbSQL, err := sql.Open("sqlite", "file:"+uuid.NewString()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { dbSQL.Close() })
	dbSQL.SetMaxOpenConns(1)

	instance, err := gorm.Open(sqlite.Dialector{Conn: dbSQL}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	if len(tables) == 0 {
		tables = []interface{}{&models.OAuthClient{}}
	}
	if err := instance.AutoMigrate(tables...); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	database.Instance = instance
}
