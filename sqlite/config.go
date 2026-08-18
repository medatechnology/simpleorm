// Package sqlite provides a pure-Go SQLite implementation of the orm.Database
// interface for simpleorm. It uses modernc.org/sqlite (no cgo), so consumers
// keep CGO_ENABLED=0 static cross-compilation.
package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orm "github.com/medatechnology/simpleorm"
	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// Defaults for the sqlite connection pool.
const (
	DefaultBusyTimeout  = 5000 // ms
	DefaultMaxOpenConns = 10
	DefaultMaxIdleConns = 5
)

// SqliteConfig holds the connection settings for a local SQLite file.
type SqliteConfig struct {
	Path         string // database file path
	WAL          bool   // enable WAL journal mode
	BusyTimeout  int    // busy_timeout in ms
	MaxOpenConns int
	MaxIdleConns int
}

// NewDefaultConfig returns a SqliteConfig with safe defaults (WAL on).
func NewDefaultConfig(path string) *SqliteConfig {
	return &SqliteConfig{
		Path:         path,
		WAL:          true,
		BusyTimeout:  DefaultBusyTimeout,
		MaxOpenConns: DefaultMaxOpenConns,
		MaxIdleConns: DefaultMaxIdleConns,
	}
}

// dsn builds the modernc.org/sqlite connection string with pragmas.
func (c *SqliteConfig) dsn() string {
	params := []string{
		"_pragma=foreign_keys(1)",
		fmt.Sprintf("_pragma=busy_timeout(%d)", c.BusyTimeout),
	}
	if c.WAL {
		params = append(params,
			"_pragma=journal_mode(WAL)",
			"_pragma=synchronous(NORMAL)",
		)
	}
	return c.Path + "?" + strings.Join(params, "&")
}

// Validate checks the configuration.
func (c *SqliteConfig) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("sqlite: database path cannot be empty")
	}
	if c.BusyTimeout <= 0 {
		c.BusyTimeout = DefaultBusyTimeout
	}
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = DefaultMaxOpenConns
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = DefaultMaxIdleConns
	}
	return nil
}

// SqliteDirectDB is the sqlite driver surface: orm.Database plus Close.
type SqliteDirectDB interface {
	orm.Database
	Close() error
}

// sqliteDB is the exported driver: embeds the implementation so every
// orm.Database method is promoted, plus Close.
type sqliteDB struct {
	*sqliteImpl
	path string
}

// Close closes the underlying database.
func (s *sqliteDB) Close() error { return s.db.Close() }

// NewDatabase opens (creating if needed) the SQLite file and returns a driver
// instance implementing orm.Database.
func NewDatabase(cfg SqliteConfig) (SqliteDirectDB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if dir := filepath.Dir(cfg.Path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("sqlite: create dir: %w", err)
		}
	}
	impl, err := open(&cfg)
	if err != nil {
		return nil, err
	}
	return &sqliteDB{sqliteImpl: impl, path: cfg.Path}, nil
}
