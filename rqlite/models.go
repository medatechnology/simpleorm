package rqlite

import (
	"net/http"
	"time"
)

const (
	PREFIX_SURESQL_TABLE = "_"
	PREFIX_SQLITE_TABLE  = "sqlite_"
	SCHEMA_TABLE         = "sqlite_master"

	// RQLite API endpoints
	ENDPOINT_EXECUTE       = "/db/execute"
	ENDPOINT_QUERY         = "/db/query"
	ENDPOINT_UNIFIED       = "/db/request"
	ENDPOINT_LOAD          = "/db/load"
	ENDPOINT_BACKUP        = "/db/backup" // curl -s -XGET localhost:4001/db/backup -o bak.sqlite3 (returns might need to be saved as a file right away)
	ENDPOINT_BOOT          = "/boot"
	ENDPOINT_SNAPSHOT      = "/snapshot"
	ENDPOINT_STATUS        = "/status" // option are /status?pretty
	ENDPOINT_READY         = "/readyz" // option readyz?noleader or /readyz?sync&timeout=5s
	ENDPOINT_JOIN          = "/join"
	ENDPOINT_REMOVE        = "/remove"
	ENDPOINT_NODE          = "/nodes" // can have nodes?pretty&ver=2 for improved JSON format
	ENDPOINT_DEBUG         = "/debug"
	ENDPOINT_VARS          = "/debug/vars"
	ENDPOINT_PPROF_CMDLINE = "/debug/pprof/cmdline"
	ENDPOINT_PPROF_PROFILE = "/debug/pprof/profile"
	ENDPOINT_PPROF_SYMBOL  = "/debug/pprof/symbol"

	DEFAULT_MAX_POOL = 25
	// Backup options
	// like .dump in cli (sql file) : /db/backup?fmt=sql
	// curl -s -XGET localhost:4001/db/backup?fmt=sql -o bak.sql
	// vacuumed : /db/backup?vacuum
	// curl -s -XGET localhost:4001/db/backup?vacuum -o bak.sqlite3
	// compressed : /db/backup?compress
	// curl -s -XGET localhost:4001/db/backup?compress -o bak.sqlite3.gz
	// or commbined : /db/backup?compress&vacuumhttp
	// to check backup files, use pragma
	// curl -s -XPOST localhost:4001/db/execute -H "Content-Type: application/json" -d '["PRAGMA schema.integrity_check"]'

	// Restore (use /boot or /load)
	// curl -XPOST 'http://localhost:4001/boot' -H "Transfer-Encoding: chunked" \
	//    --upload-file largedb.sqlite
	// curl -XPOST localhost:4001/db/load -H "Content-type: text/plain" --data-binary @restore.dump
	// curl -v -XPOST localhost:4001/db/load -H "Content-type: application/octet-stream" --data-binary @restore.sqlite

	// Delete
	// Just pass in the ID of the node
	// curl -XDELETE http://host:4001/remove -d '{"id": "<node ID>"}'

	// Default timeouts
	DEFAULT_TIMEOUT       = 30 * time.Second
	DEFAULT_RETRY_TIMEOUT = 2 * time.Second
	DEFAULT_MAX_RETRIES   = 3
)

// RqliteDirectConfig holds configuration for direct RQLite connections
type RqliteDirectConfig struct {
	URL         string        // Base URL for the RQLite node (e.g. "http://localhost:4001")
	Consistency string        // Consistency level: "none", "weak", "strong"
	Username    string        // Optional username for authentication
	Password    string        // Optional password for authentication
	Timeout     time.Duration // HTTP client timeout
	RetryCount  int           // Number of retries for failed requests
}

// RQLiteDirectDB implements the orm.Database interface for direct HTTP access to RQLite
type RQLiteDirectDB struct {
	Config     RqliteDirectConfig
	HTTPClient *http.Client
}

// Response structures for RQLite API

// ExecuteResponse represents the response from a write operation
type ExecuteResponse struct {
	Results []ExecuteResult `json:"results"`
	Time    float64         `json:"time"`
}

// ExecuteResult represents a single result from a write operation
type ExecuteResult struct {
	LastInsertID int     `json:"last_insert_id"`
	RowsAffected int     `json:"rows_affected"`
	Time         float64 `json:"time"`
	Error        string  `json:"error,omitempty"`
}

// QueryResponse represents the response from a read operation
type QueryResponse struct {
	Results []QueryResult `json:"results"`
	Time    float64       `json:"time"`
}

// QueryResult represents a single result from a read operation
type QueryResult struct {
	Columns []string        `json:"columns"`
	Types   []string        `json:"types"`
	Values  [][]interface{} `json:"values"`
	Time    float64         `json:"time"`
	Error   string          `json:"error,omitempty"`
}
