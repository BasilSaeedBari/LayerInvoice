package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// DB wraps *sql.DB to provide database access.
type DB struct {
	*sql.DB
}

// Open opens a connection to SQLite at the given path, configures it, and returns the DB struct.
func Open(dbPath string) (*DB, error) {
	// Open modernc SQLite driver
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection limits for SQLite to prevent lock contention
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Configure PRAGMAs
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA busy_timeout = 5000;",
	}

	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to configure sqlite pragma %q: %w", pragma, err)
		}
	}

	return &DB{db}, nil
}

// RunMigrations runs embedded sql migrations.
func (db *DB) RunMigrations(migrationFS embed.FS) error {
	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Disable goose verbosity in tests/production by default
	goose.SetVerbose(false)

	if err := goose.Up(db.DB, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// NewULID generates a new lexicographically sortable unique identifier.
func NewULID() string {
	entropy := rand.New(rand.NewSource(time.Now().UnixNano()))
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)
	if err != nil {
		// Fallback if entropy source fails (highly unlikely)
		return ulid.Make().String()
	}
	return id.String()
}
