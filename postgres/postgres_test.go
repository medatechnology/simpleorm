package postgres

import (
	"testing"
	"time"
)

// TestNewDefaultConfig tests the creation of a default PostgreSQL configuration
func TestNewDefaultConfig(t *testing.T) {
	config := NewDefaultConfig()

	if config.Host != DefaultHost {
		t.Errorf("Expected Host to be %s, got %s", DefaultHost, config.Host)
	}
	if config.Port != DefaultPort {
		t.Errorf("Expected Port to be %d, got %d", DefaultPort, config.Port)
	}
	if config.SSLMode != DefaultSSLMode {
		t.Errorf("Expected SSLMode to be %s, got %s", DefaultSSLMode, config.SSLMode)
	}
	if config.MaxOpenConns != DefaultMaxOpenConns {
		t.Errorf("Expected MaxOpenConns to be %d, got %d", DefaultMaxOpenConns, config.MaxOpenConns)
	}
	if config.MaxIdleConns != DefaultMaxIdleConns {
		t.Errorf("Expected MaxIdleConns to be %d, got %d", DefaultMaxIdleConns, config.MaxIdleConns)
	}
	if config.ConnMaxLifetime != DefaultConnMaxLifetime {
		t.Errorf("Expected ConnMaxLifetime to be %v, got %v", DefaultConnMaxLifetime, config.ConnMaxLifetime)
	}
	if config.ConnectTimeout != DefaultConnectTimeout {
		t.Errorf("Expected ConnectTimeout to be %v, got %v", DefaultConnectTimeout, config.ConnectTimeout)
	}
	if config.ApplicationName != DefaultApplicationName {
		t.Errorf("Expected ApplicationName to be %s, got %s", DefaultApplicationName, config.ApplicationName)
	}
}

// TestNewConfig tests the creation of a PostgreSQL configuration with custom values
func TestNewConfig(t *testing.T) {
	host := "testhost"
	port := 5433
	user := "testuser"
	password := "testpass"
	dbname := "testdb"

	config := NewConfig(host, port, user, password, dbname)

	if config.Host != host {
		t.Errorf("Expected Host to be %s, got %s", host, config.Host)
	}
	if config.Port != port {
		t.Errorf("Expected Port to be %d, got %d", port, config.Port)
	}
	if config.User != user {
		t.Errorf("Expected User to be %s, got %s", user, config.User)
	}
	if config.Password != password {
		t.Errorf("Expected Password to be %s, got %s", password, config.Password)
	}
	if config.DBName != dbname {
		t.Errorf("Expected DBName to be %s, got %s", dbname, config.DBName)
	}

	// Check that defaults are still applied
	if config.SSLMode != DefaultSSLMode {
		t.Errorf("Expected SSLMode to be %s, got %s", DefaultSSLMode, config.SSLMode)
	}
}

// TestConfigValidate tests the validation of PostgreSQL configuration
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    PostgresConfig
		shouldErr bool
	}{
		{
			name: "Valid config",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "testpass",
				DBName:   "testdb",
				SSLMode:  "disable",
			},
			shouldErr: false,
		},
		{
			name: "Missing user",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Password: "testpass",
				DBName:   "testdb",
			},
			shouldErr: true,
		},
		{
			name: "Missing database name",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "testpass",
			},
			shouldErr: true,
		},
		{
			name: "Invalid SSL mode",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "testpass",
				DBName:   "testdb",
				SSLMode:  "invalid",
			},
			shouldErr: true,
		},
		{
			name: "Auto-corrects invalid port",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     -1,
				User:     "testuser",
				Password: "testpass",
				DBName:   "testdb",
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestConfigToDSN tests the DSN generation
func TestConfigToDSN(t *testing.T) {
	config := PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		ConnectTimeout:  10 * time.Second,
		ApplicationName: "testapp",
	}

	dsn, err := config.ToDSN()
	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	expectedContents := []string{
		"postgres://",
		"testuser",
		"testpass",
		"localhost:5432",
		"/testdb",
		"sslmode=disable",
		"application_name=testapp",
		"connect_timeout=10",
	}

	for _, expected := range expectedContents {
		if !contains(dsn, expected) {
			t.Errorf("Expected DSN to contain '%s', got: %s", expected, dsn)
		}
	}
}

// TestConfigToSimpleDSN tests the simple DSN generation
func TestConfigToSimpleDSN(t *testing.T) {
	config := PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		ConnectTimeout:  10 * time.Second,
		ApplicationName: "testapp",
	}

	dsn, err := config.ToSimpleDSN()
	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	expectedContents := []string{
		"host=localhost",
		"port=5432",
		"user=testuser",
		"password=testpass",
		"dbname=testdb",
		"sslmode=disable",
		"application_name=testapp",
		"connect_timeout=10",
	}

	for _, expected := range expectedContents {
		if !contains(dsn, expected) {
			t.Errorf("Expected DSN to contain '%s', got: %s", expected, dsn)
		}
	}
}

// TestConfigClone tests the cloning of configuration
func TestConfigClone(t *testing.T) {
	original := NewConfig("localhost", 5432, "user", "pass", "db")
	original.ExtraParams = map[string]string{"key": "value"}

	clone := original.Clone()

	// Modify clone
	clone.Host = "newhost"
	clone.ExtraParams["key2"] = "value2"

	// Original should be unchanged
	if original.Host == clone.Host {
		t.Errorf("Clone modified the original Host")
	}
	if _, ok := original.ExtraParams["key2"]; ok {
		t.Errorf("Clone modified the original ExtraParams")
	}
}

// TestConfigMethodChaining tests the fluent interface
func TestConfigMethodChaining(t *testing.T) {
	config := NewDefaultConfig().
		WithSSLMode("require").
		WithConnectionPool(50, 10, 10*time.Minute, 5*time.Minute).
		WithTimeouts(20*time.Second, 60*time.Second).
		WithApplicationName("myapp").
		WithSearchPath("public,custom").
		WithTimezone("UTC").
		WithExtraParam("statement_timeout", "30000")

	if config.SSLMode != "require" {
		t.Errorf("Expected SSLMode to be 'require', got %s", config.SSLMode)
	}
	if config.MaxOpenConns != 50 {
		t.Errorf("Expected MaxOpenConns to be 50, got %d", config.MaxOpenConns)
	}
	if config.MaxIdleConns != 10 {
		t.Errorf("Expected MaxIdleConns to be 10, got %d", config.MaxIdleConns)
	}
	if config.ApplicationName != "myapp" {
		t.Errorf("Expected ApplicationName to be 'myapp', got %s", config.ApplicationName)
	}
	if config.SearchPath != "public,custom" {
		t.Errorf("Expected SearchPath to be 'public,custom', got %s", config.SearchPath)
	}
	if config.Timezone != "UTC" {
		t.Errorf("Expected Timezone to be 'UTC', got %s", config.Timezone)
	}
	if config.ExtraParams["statement_timeout"] != "30000" {
		t.Errorf("Expected extra param 'statement_timeout' to be '30000', got %s", config.ExtraParams["statement_timeout"])
	}
}

// TestParseDSN tests parsing of DSN strings
func TestParseDSN(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
		checks  func(*PostgresConfig) error
	}{
		{
			name:    "URL format",
			dsn:     "postgres://user:pass@localhost:5432/mydb?sslmode=require",
			wantErr: false,
			checks: func(c *PostgresConfig) error {
				if c.User != "user" {
					t.Errorf("Expected User to be 'user', got %s", c.User)
				}
				if c.Password != "pass" {
					t.Errorf("Expected Password to be 'pass', got %s", c.Password)
				}
				if c.Host != "localhost" {
					t.Errorf("Expected Host to be 'localhost', got %s", c.Host)
				}
				if c.Port != 5432 {
					t.Errorf("Expected Port to be 5432, got %d", c.Port)
				}
				if c.DBName != "mydb" {
					t.Errorf("Expected DBName to be 'mydb', got %s", c.DBName)
				}
				if c.SSLMode != "require" {
					t.Errorf("Expected SSLMode to be 'require', got %s", c.SSLMode)
				}
				return nil
			},
		},
		{
			name:    "Key-value format",
			dsn:     "host=localhost port=5432 user=user password=pass dbname=mydb sslmode=disable",
			wantErr: false,
			checks: func(c *PostgresConfig) error {
				if c.User != "user" {
					t.Errorf("Expected User to be 'user', got %s", c.User)
				}
				if c.Host != "localhost" {
					t.Errorf("Expected Host to be 'localhost', got %s", c.Host)
				}
				if c.DBName != "mydb" {
					t.Errorf("Expected DBName to be 'mydb', got %s", c.DBName)
				}
				return nil
			},
		},
		{
			name:    "Invalid DSN - missing user",
			dsn:     "postgres://localhost:5432/mydb",
			wantErr: true,
			checks:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseDSN(tt.dsn)
			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.wantErr && tt.checks != nil {
				tt.checks(config)
			}
		})
	}
}

// TestConvertToPostgreSQLPlaceholders tests placeholder conversion
func TestConvertToPostgreSQLPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single placeholder",
			input:    "SELECT * FROM users WHERE id = ?",
			expected: "SELECT * FROM users WHERE id = $1",
		},
		{
			name:     "Multiple placeholders",
			input:    "INSERT INTO users (name, email, age) VALUES (?, ?, ?)",
			expected: "INSERT INTO users (name, email, age) VALUES ($1, $2, $3)",
		},
		{
			name:     "Placeholder in string literal (single quote)",
			input:    "SELECT * FROM users WHERE name = 'What?' AND id = ?",
			expected: "SELECT * FROM users WHERE name = 'What?' AND id = $1",
		},
		{
			name:     "No placeholders",
			input:    "SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "Complex query",
			input:    "UPDATE users SET name = ?, email = ? WHERE id = ? AND status = ?",
			expected: "UPDATE users SET name = $1, email = $2 WHERE id = $3 AND status = $4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToPostgreSQLPlaceholders(tt.input)
			if result != tt.expected {
				t.Errorf("Expected: %s\nGot:      %s", tt.expected, result)
			}
		})
	}
}

// TestExtractTableNameFromSQL tests table name extraction
func TestExtractTableNameFromSQL(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "Simple SELECT",
			sql:      "SELECT * FROM users",
			expected: "users",
		},
		{
			name:     "SELECT with WHERE",
			sql:      "SELECT * FROM users WHERE id = 1",
			expected: "users",
		},
		{
			name:     "INSERT INTO",
			sql:      "INSERT INTO users (name) VALUES ('John')",
			expected: "users",
		},
		{
			name:     "UPDATE",
			sql:      "UPDATE users SET name = 'Jane'",
			expected: "users",
		},
		{
			name:     "DELETE FROM",
			sql:      "DELETE FROM users WHERE id = 1",
			expected: "users",
		},
		{
			name:     "Quoted table name",
			sql:      "SELECT * FROM \"users\"",
			expected: "users",
		},
		{
			name:     "Unknown query type",
			sql:      "TRUNCATE TABLE users",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTableNameFromSQL(tt.sql)
			if result != tt.expected {
				t.Errorf("Expected: %s, Got: %s", tt.expected, result)
			}
		})
	}
}

// TestErrorHelpers tests PostgreSQL error helper functions
func TestErrorHelpers(t *testing.T) {
	t.Run("GetPostgreSQLErrorCode on nil", func(t *testing.T) {
		code := GetPostgreSQLErrorCode(nil)
		if code != "" {
			t.Errorf("Expected empty string for nil error, got: %s", code)
		}
	})

	t.Run("FormatPostgreSQLError on nil", func(t *testing.T) {
		formatted := FormatPostgreSQLError(nil)
		if formatted != "no error" {
			t.Errorf("Expected 'no error' for nil, got: %s", formatted)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
