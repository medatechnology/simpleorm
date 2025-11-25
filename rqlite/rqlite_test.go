package rqlite

import (
	"testing"
	"time"

	orm "github.com/medatechnology/simpleorm"
)

// TestNewDatabase tests the creation of a new RQLiteDirectDB instance
func TestNewDatabase(t *testing.T) {
	config := RqliteDirectConfig{
		URL:         "http://localhost:4001",
		Consistency: "weak",
		Username:    "testuser",
		Password:    "testpass",
		Timeout:     30 * time.Second,
		RetryCount:  5,
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if db == nil {
		t.Fatal("Expected non-nil database instance")
	}

	if db.Config.URL != config.URL {
		t.Errorf("Expected URL to be %s, got %s", config.URL, db.Config.URL)
	}
	if db.Config.Consistency != config.Consistency {
		t.Errorf("Expected Consistency to be %s, got %s", config.Consistency, db.Config.Consistency)
	}
	if db.Config.Username != config.Username {
		t.Errorf("Expected Username to be %s, got %s", config.Username, db.Config.Username)
	}
	if db.Config.Password != config.Password {
		t.Errorf("Expected Password to be %s, got %s", config.Password, db.Config.Password)
	}
	if db.Config.Timeout != config.Timeout {
		t.Errorf("Expected Timeout to be %v, got %v", config.Timeout, db.Config.Timeout)
	}
	if db.Config.RetryCount != config.RetryCount {
		t.Errorf("Expected RetryCount to be %d, got %d", config.RetryCount, db.Config.RetryCount)
	}

	if db.HTTPClient == nil {
		t.Fatal("Expected non-nil HTTP client")
	}
	if db.HTTPClient.Timeout != config.Timeout {
		t.Errorf("Expected HTTP client timeout to be %v, got %v", config.Timeout, db.HTTPClient.Timeout)
	}
}

// TestNewDatabaseDefaultTimeout tests that default timeout is set when not specified
func TestNewDatabaseDefaultTimeout(t *testing.T) {
	config := RqliteDirectConfig{
		URL: "http://localhost:4001",
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// The HTTP client should use the default timeout
	if db.HTTPClient.Timeout != DEFAULT_TIMEOUT {
		t.Errorf("Expected HTTP client timeout to be %v (default), got %v", DEFAULT_TIMEOUT, db.HTTPClient.Timeout)
	}
}

// TestNewDatabaseDefaultRetryCount tests that default retry count is set when not specified
func TestNewDatabaseDefaultRetryCount(t *testing.T) {
	config := RqliteDirectConfig{
		URL: "http://localhost:4001",
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if db.Config.RetryCount != DEFAULT_MAX_RETRIES {
		t.Errorf("Expected retry count to be %d (default), got %d", DEFAULT_MAX_RETRIES, db.Config.RetryCount)
	}
}

// TestNewDatabaseTrailingSlash tests that trailing slashes are removed from URL
func TestNewDatabaseTrailingSlash(t *testing.T) {
	config := RqliteDirectConfig{
		URL: "http://localhost:4001/",
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedURL := "http://localhost:4001"
	if db.Config.URL != expectedURL {
		t.Errorf("Expected URL to be %s (without trailing slash), got %s", expectedURL, db.Config.URL)
	}
}

// TestIsConnected tests the IsConnected method
func TestIsConnected(t *testing.T) {
	config := RqliteDirectConfig{
		URL: "http://localhost:4001",
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !db.IsConnected() {
		t.Error("Expected IsConnected to return true when HTTPClient is initialized")
	}

	// Test with nil client
	db.HTTPClient = nil
	if db.IsConnected() {
		t.Error("Expected IsConnected to return false when HTTPClient is nil")
	}
}

// TestGetTableNameFromSQL tests the table name extraction helper
func TestGetTableNameFromSQL(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "Simple SELECT",
			sql:      "SELECT * FROM users",
			expected: "USERS",
		},
		{
			name:     "SELECT with WHERE",
			sql:      "SELECT id, name FROM products WHERE active = 1",
			expected: "PRODUCTS",
		},
		{
			name:     "SELECT with JOIN",
			sql:      "SELECT * FROM orders INNER JOIN customers ON orders.customer_id = customers.id",
			expected: "", // JOIN is a marker, so table name extraction stops
		},
		{
			name:     "Lowercase select",
			sql:      "select * from items",
			expected: "ITEMS",
		},
		{
			name:     "Mixed case with GROUP BY",
			sql:      "SELECT COUNT(*) FROM transactions GROUP BY user_id",
			expected: "TRANSACTIONS",
		},
		{
			name:     "With quoted identifiers",
			sql:      "SELECT * FROM \"my_table\"",
			expected: "MY_TABLE",
		},
		{
			name:     "With backticks",
			sql:      "SELECT * FROM `my_table`",
			expected: "MY_TABLE",
		},
		{
			name:     "No FROM clause",
			sql:      "INSERT INTO users (name) VALUES ('test')",
			expected: "unknown",
		},
		{
			name:     "Complex query with LIMIT",
			sql:      "SELECT * FROM events WHERE date > NOW() ORDER BY date DESC LIMIT 10",
			expected: "EVENTS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTableNameFromSQL(tt.sql)
			if result != tt.expected {
				t.Errorf("getTableNameFromSQL(%q) = %q; want %q", tt.sql, result, tt.expected)
			}
		})
	}
}

// TestGetOperationFromSQL tests the operation extraction helper
func TestGetOperationFromSQL(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "SELECT query",
			sql:      "SELECT * FROM users",
			expected: "SELECT",
		},
		{
			name:     "INSERT statement",
			sql:      "INSERT INTO users (name) VALUES ('test')",
			expected: "INSERT",
		},
		{
			name:     "UPDATE statement",
			sql:      "UPDATE users SET name = 'test' WHERE id = 1",
			expected: "UPDATE",
		},
		{
			name:     "DELETE statement",
			sql:      "DELETE FROM users WHERE id = 1",
			expected: "DELETE",
		},
		{
			name:     "CREATE TABLE",
			sql:      "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
			expected: "CREATE",
		},
		{
			name:     "DROP TABLE",
			sql:      "DROP TABLE users",
			expected: "DROP",
		},
		{
			name:     "ALTER TABLE",
			sql:      "ALTER TABLE users ADD COLUMN email TEXT",
			expected: "ALTER",
		},
		{
			name:     "REPLACE statement",
			sql:      "REPLACE INTO users (id, name) VALUES (1, 'test')",
			expected: "REPLACE",
		},
		{
			name:     "Lowercase select",
			sql:      "select * from users",
			expected: "SELECT",
		},
		{
			name:     "Mixed case UPDATE",
			sql:      "UpDaTe users SET name = 'test'",
			expected: "UPDATE",
		},
		{
			name:     "With leading spaces",
			sql:      "   INSERT INTO users (name) VALUES ('test')",
			expected: "INSERT",
		},
		{
			name:     "Unknown operation",
			sql:      "PRAGMA table_info(users)",
			expected: "EXEC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOperationFromSQL(tt.sql)
			if result != tt.expected {
				t.Errorf("getOperationFromSQL(%q) = %q; want %q", tt.sql, result, tt.expected)
			}
		})
	}
}

// TestExecuteResultToBasicSQLResult tests conversion from ExecuteResult to BasicSQLResult
func TestExecuteResultToBasicSQLResult(t *testing.T) {
	tests := []struct {
		name     string
		input    ExecuteResult
		expected orm.BasicSQLResult
	}{
		{
			name: "Successful execution",
			input: ExecuteResult{
				LastInsertID: 42,
				RowsAffected: 1,
				Time:         0.123,
				Error:        "",
			},
			expected: orm.BasicSQLResult{
				Error:        nil,
				Timing:       0.123,
				RowsAffected: 1,
				LastInsertID: 42,
			},
		},
		{
			name: "Execution with error string",
			input: ExecuteResult{
				LastInsertID: 0,
				RowsAffected: 0,
				Time:         0.056,
				Error:        "constraint violation",
			},
			expected: orm.BasicSQLResult{
				Error:        nil, // Note: executeResultToBasicSQLResult doesn't convert error string
				Timing:       0.056,
				RowsAffected: 0,
				LastInsertID: 0,
			},
		},
		{
			name: "Multiple rows affected",
			input: ExecuteResult{
				LastInsertID: 0,
				RowsAffected: 5,
				Time:         0.234,
				Error:        "",
			},
			expected: orm.BasicSQLResult{
				Error:        nil,
				Timing:       0.234,
				RowsAffected: 5,
				LastInsertID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executeResultToBasicSQLResult(tt.input)

			// Check timing
			if result.Timing != tt.expected.Timing {
				t.Errorf("Expected Timing to be %f, got %f", tt.expected.Timing, result.Timing)
			}

			// Check RowsAffected
			if result.RowsAffected != tt.expected.RowsAffected {
				t.Errorf("Expected RowsAffected to be %d, got %d", tt.expected.RowsAffected, result.RowsAffected)
			}

			// Check LastInsertID
			if result.LastInsertID != tt.expected.LastInsertID {
				t.Errorf("Expected LastInsertID to be %d, got %d", tt.expected.LastInsertID, result.LastInsertID)
			}

			// Note: executeResultToBasicSQLResult always returns nil error
			// Error handling is done at a higher level
			if result.Error != nil {
				t.Errorf("Expected no error, got %v", result.Error)
			}
		})
	}
}

// TestExecuteResultsToBasicSQLResults tests conversion from multiple ExecuteResults
func TestExecuteResultsToBasicSQLResults(t *testing.T) {
	inputs := []ExecuteResult{
		{
			LastInsertID: 1,
			RowsAffected: 1,
			Time:         0.1,
			Error:        "",
		},
		{
			LastInsertID: 2,
			RowsAffected: 1,
			Time:         0.2,
			Error:        "",
		},
		{
			LastInsertID: 0,
			RowsAffected: 0,
			Time:         0.15,
			Error:        "duplicate key",
		},
	}

	results := executeResultsToBasicSQLResults(inputs)

	if len(results) != len(inputs) {
		t.Fatalf("Expected %d results, got %d", len(inputs), len(results))
	}

	// Check first result
	if results[0].LastInsertID != 1 {
		t.Errorf("Expected first result LastInsertID to be 1, got %d", results[0].LastInsertID)
	}
	if results[0].Error != nil {
		t.Errorf("Expected first result to have no error, got %v", results[0].Error)
	}

	// Check second result
	if results[1].LastInsertID != 2 {
		t.Errorf("Expected second result LastInsertID to be 2, got %d", results[1].LastInsertID)
	}

	// Check third result timing
	if results[2].Timing != 0.15 {
		t.Errorf("Expected third result Timing to be 0.15, got %f", results[2].Timing)
	}
	// Note: executeResultsToBasicSQLResults doesn't convert error strings
	// Error handling is done at a higher level
}

// TestQueryResultToDBRecord tests conversion from QueryResult to DBRecord
func TestQueryResultToDBRecord(t *testing.T) {
	tests := []struct {
		name        string
		input       QueryResult
		tableName   string
		expectError bool
		checkLen    int
	}{
		{
			name: "Valid query result with data",
			input: QueryResult{
				Columns: []string{"id", "name", "age"},
				Types:   []string{"integer", "text", "integer"},
				Values: [][]interface{}{
					{float64(1), "Alice", float64(30)},
					{float64(2), "Bob", float64(25)},
				},
				Time:  0.123,
				Error: "",
			},
			tableName:   "users",
			expectError: false,
			checkLen:    2,
		},
		{
			name: "Empty result set",
			input: QueryResult{
				Columns: []string{"id", "name"},
				Types:   []string{"integer", "text"},
				Values:  [][]interface{}{},
				Time:    0.05,
				Error:   "",
			},
			tableName:   "users",
			expectError: true, // Should return ErrSQLNoRows
			checkLen:    0,
		},
		{
			name: "Single row result",
			input: QueryResult{
				Columns: []string{"count"},
				Types:   []string{"integer"},
				Values: [][]interface{}{
					{float64(42)},
				},
				Time:  0.01,
				Error: "",
			},
			tableName:   "aggregation",
			expectError: false,
			checkLen:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := queryResultToDBRecord(tt.input, tt.tableName)

			if tt.expectError && err == nil {
				t.Error("Expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if len(records) != tt.checkLen {
				t.Errorf("Expected %d records, got %d", tt.checkLen, len(records))
			}

			// If we have records, verify their structure
			if len(records) > 0 {
				if records[0].TableName != tt.tableName {
					t.Errorf("Expected table name to be %s, got %s", tt.tableName, records[0].TableName)
				}
				if len(records[0].Data) != len(tt.input.Columns) {
					t.Errorf("Expected %d columns in data, got %d", len(tt.input.Columns), len(records[0].Data))
				}
			}
		})
	}
}

// TestConstants verifies that all expected constants are defined
func TestConstants(t *testing.T) {
	// Test timeout constants
	if DEFAULT_TIMEOUT == 0 {
		t.Error("DEFAULT_TIMEOUT should be defined and non-zero")
	}
	if DEFAULT_MAX_RETRIES == 0 {
		t.Error("DEFAULT_MAX_RETRIES should be defined and non-zero")
	}

	// Test endpoint constants
	if ENDPOINT_EXECUTE == "" {
		t.Error("ENDPOINT_EXECUTE should be defined")
	}
	if ENDPOINT_QUERY == "" {
		t.Error("ENDPOINT_QUERY should be defined")
	}
	if ENDPOINT_STATUS == "" {
		t.Error("ENDPOINT_STATUS should be defined")
	}

	// Verify endpoint values
	expectedEndpoints := map[string]string{
		"execute": "/db/execute",
		"query":   "/db/query",
		"status":  "/status",
	}

	if ENDPOINT_EXECUTE != expectedEndpoints["execute"] {
		t.Errorf("Expected ENDPOINT_EXECUTE to be %s, got %s", expectedEndpoints["execute"], ENDPOINT_EXECUTE)
	}
	if ENDPOINT_QUERY != expectedEndpoints["query"] {
		t.Errorf("Expected ENDPOINT_QUERY to be %s, got %s", expectedEndpoints["query"], ENDPOINT_QUERY)
	}
	if ENDPOINT_STATUS != expectedEndpoints["status"] {
		t.Errorf("Expected ENDPOINT_STATUS to be %s, got %s", expectedEndpoints["status"], ENDPOINT_STATUS)
	}
}

// TestRqliteDirectConfigDefaults tests default values in configuration
func TestRqliteDirectConfigDefaults(t *testing.T) {
	config := RqliteDirectConfig{
		URL: "http://localhost:4001",
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify defaults were applied to HTTP client
	if db.HTTPClient.Timeout != DEFAULT_TIMEOUT {
		t.Errorf("Expected HTTP client timeout to be %v, got %v", DEFAULT_TIMEOUT, db.HTTPClient.Timeout)
	}

	if db.Config.RetryCount != DEFAULT_MAX_RETRIES {
		t.Errorf("Expected default retry count of %d, got %d", DEFAULT_MAX_RETRIES, db.Config.RetryCount)
	}
}

// TestHTTPClientConfiguration tests HTTP client configuration
func TestHTTPClientConfiguration(t *testing.T) {
	config := RqliteDirectConfig{
		URL:     "http://localhost:4001",
		Timeout: 45 * time.Second,
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if db.HTTPClient == nil {
		t.Fatal("Expected HTTP client to be initialized")
	}

	// Verify timeout
	if db.HTTPClient.Timeout != 45*time.Second {
		t.Errorf("Expected HTTP client timeout to be 45s, got %v", db.HTTPClient.Timeout)
	}

	// Verify transport is configured
	if db.HTTPClient.Transport == nil {
		t.Error("Expected HTTP client transport to be configured")
	}
}
