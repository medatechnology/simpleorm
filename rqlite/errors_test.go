package rqlite

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	orm "github.com/medatechnology/simpleorm"
)

// TestRQLiteError tests the RQLiteError type
func TestRQLiteError(t *testing.T) {
	baseErr := errors.New("database error")
	rqErr := &RQLiteError{
		Operation:  "SELECT",
		Table:      "users",
		Query:      "SELECT * FROM users",
		StatusCode: 500,
		Message:    "query failed",
		Detail:     "connection timeout",
		Err:        baseErr,
	}

	// Test Error() method
	errMsg := rqErr.Error()
	if errMsg == "" {
		t.Error("Error() should return non-empty string")
	}
	if !contains(errMsg, "SELECT") {
		t.Errorf("Error message should contain operation: %s", errMsg)
	}
	if !contains(errMsg, "users") {
		t.Errorf("Error message should contain table: %s", errMsg)
	}
	if !contains(errMsg, "500") {
		t.Errorf("Error message should contain status code: %s", errMsg)
	}

	// Test Unwrap() method
	unwrapped := rqErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("Unwrap() should return original error")
	}
}

// TestWrapRQLiteError tests error wrapping
func TestWrapRQLiteError(t *testing.T) {
	baseErr := errors.New("connection failed")

	wrapped := WrapRQLiteError(baseErr, "INSERT", "products", "INSERT INTO products VALUES (?)")

	if wrapped == nil {
		t.Fatal("WrapRQLiteError should not return nil for non-nil error")
	}

	var rqErr *RQLiteError
	if !errors.As(wrapped, &rqErr) {
		t.Fatal("Wrapped error should be of type *RQLiteError")
	}

	if rqErr.Operation != "INSERT" {
		t.Errorf("Expected operation 'INSERT', got '%s'", rqErr.Operation)
	}
	if rqErr.Table != "products" {
		t.Errorf("Expected table 'products', got '%s'", rqErr.Table)
	}

	// Test with nil error
	nilWrapped := WrapRQLiteError(nil, "SELECT", "users", "")
	if nilWrapped != nil {
		t.Error("WrapRQLiteError should return nil for nil error")
	}
}

// TestWrapRQLiteHTTPError tests HTTP error wrapping
func TestWrapRQLiteHTTPError(t *testing.T) {
	baseErr := errors.New("unauthorized")

	wrapped := WrapRQLiteHTTPError(baseErr, "SELECT", "users", "SELECT * FROM users", 401)

	if wrapped == nil {
		t.Fatal("WrapRQLiteHTTPError should not return nil")
	}

	var rqErr *RQLiteError
	if !errors.As(wrapped, &rqErr) {
		t.Fatal("Wrapped error should be of type *RQLiteError")
	}

	if rqErr.StatusCode != 401 {
		t.Errorf("Expected status code 401, got %d", rqErr.StatusCode)
	}
}

// TestIsUniqueViolation tests unique constraint detection
func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Unique constraint error",
			err:      errors.New("UNIQUE constraint failed: users.email"),
			expected: true,
		},
		{
			name:     "Not unique error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Wrapped unique error",
			err:      fmt.Errorf("query failed: %w", errors.New("UNIQUE constraint failed")),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUniqueViolation(tt.err)
			if result != tt.expected {
				t.Errorf("IsUniqueViolation() = %v; want %v", result, tt.expected)
			}
		})
	}
}

// TestIsForeignKeyViolation tests foreign key constraint detection
func TestIsForeignKeyViolation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Foreign key error",
			err:      errors.New("FOREIGN KEY constraint failed"),
			expected: true,
		},
		{
			name:     "Not FK error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsForeignKeyViolation(tt.err)
			if result != tt.expected {
				t.Errorf("IsForeignKeyViolation() = %v; want %v", result, tt.expected)
			}
		})
	}
}

// TestIsNotNullViolation tests NOT NULL constraint detection
func TestIsNotNullViolation(t *testing.T) {
	err := errors.New("NOT NULL constraint failed: users.name")
	if !IsNotNullViolation(err) {
		t.Error("Should detect NOT NULL violation")
	}

	nonNullErr := errors.New("some error")
	if IsNotNullViolation(nonNullErr) {
		t.Error("Should not detect NOT NULL violation for non-constraint error")
	}
}

// TestIsConstraintViolation tests general constraint detection
func TestIsConstraintViolation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Unique constraint",
			err:      errors.New("UNIQUE constraint failed"),
			expected: true,
		},
		{
			name:     "Foreign key constraint",
			err:      errors.New("FOREIGN KEY constraint failed"),
			expected: true,
		},
		{
			name:     "NOT NULL constraint",
			err:      errors.New("NOT NULL constraint failed"),
			expected: true,
		},
		{
			name:     "CHECK constraint",
			err:      errors.New("CHECK constraint failed"),
			expected: true,
		},
		{
			name:     "PRIMARY KEY constraint",
			err:      errors.New("PRIMARY KEY constraint failed"),
			expected: true,
		},
		{
			name:     "Not a constraint error",
			err:      errors.New("syntax error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConstraintViolation(tt.err)
			if result != tt.expected {
				t.Errorf("IsConstraintViolation() = %v; want %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}

// TestIsDatabaseLocked tests database locked detection
func TestIsDatabaseLocked(t *testing.T) {
	lockedErr := errors.New("database is locked")
	if !IsDatabaseLocked(lockedErr) {
		t.Error("Should detect database locked error")
	}

	otherErr := errors.New("connection failed")
	if IsDatabaseLocked(otherErr) {
		t.Error("Should not detect database locked for other errors")
	}
}

// TestIsReadonlyError tests readonly database detection
func TestIsReadonlyError(t *testing.T) {
	readonlyErr := errors.New("attempt to write a readonly database")
	if !IsReadonlyError(readonlyErr) {
		t.Error("Should detect readonly database error")
	}
}

// TestIsTableNotFound tests table not found detection
func TestIsTableNotFound(t *testing.T) {
	tableErr := errors.New("no such table: users")
	if !IsTableNotFound(tableErr) {
		t.Error("Should detect table not found error")
	}

	otherErr := errors.New("syntax error")
	if IsTableNotFound(otherErr) {
		t.Error("Should not detect table not found for other errors")
	}
}

// TestIsColumnNotFound tests column not found detection
func TestIsColumnNotFound(t *testing.T) {
	columnErr := errors.New("no such column: email")
	if !IsColumnNotFound(columnErr) {
		t.Error("Should detect column not found error")
	}
}

// TestIsSyntaxError tests syntax error detection
func TestIsSyntaxError(t *testing.T) {
	syntaxErr := errors.New("syntax error near 'FROM'")
	if !IsSyntaxError(syntaxErr) {
		t.Error("Should detect syntax error")
	}
}

// TestIsAuthenticationError tests authentication error detection
func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "401 Unauthorized",
			err:      &RQLiteError{StatusCode: 401},
			expected: true,
		},
		{
			name:     "403 Forbidden",
			err:      &RQLiteError{StatusCode: 403},
			expected: true,
		},
		{
			name:     "ErrRQLiteUnauthorized",
			err:      ErrRQLiteUnauthorized,
			expected: true,
		},
		{
			name:     "500 Internal Server Error",
			err:      &RQLiteError{StatusCode: 500},
			expected: false,
		},
		{
			name:     "Other error",
			err:      errors.New("connection failed"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthenticationError(tt.err)
			if result != tt.expected {
				t.Errorf("IsAuthenticationError() = %v; want %v", result, tt.expected)
			}
		})
	}
}

// TestIsConnectionError tests connection error detection
func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Connection refused",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "Connection reset",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "No such host",
			err:      errors.New("no such host"),
			expected: true,
		},
		{
			name:     "I/O timeout",
			err:      errors.New("i/o timeout"),
			expected: true,
		},
		{
			name:     "Network unreachable",
			err:      errors.New("network is unreachable"),
			expected: true,
		},
		{
			name:     "Timeout error",
			err:      ErrRQLiteTimeout,
			expected: true,
		},
		{
			name:     "Other error",
			err:      errors.New("syntax error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectionError(tt.err)
			if result != tt.expected {
				t.Errorf("IsConnectionError() = %v; want %v", result, tt.expected)
			}
		})
	}
}

// TestIsNodeUnavailable tests node unavailable detection
func TestIsNodeUnavailable(t *testing.T) {
	unavailableErr := &RQLiteError{StatusCode: 503}
	if !IsNodeUnavailable(unavailableErr) {
		t.Error("Should detect node unavailable error")
	}

	directErr := ErrRQLiteNodeUnavailable
	if !IsNodeUnavailable(directErr) {
		t.Error("Should detect ErrRQLiteNodeUnavailable")
	}

	otherErr := &RQLiteError{StatusCode: 200}
	if IsNodeUnavailable(otherErr) {
		t.Error("Should not detect node unavailable for 200 status")
	}
}

// TestIsRetryable tests retryable error detection
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Database locked",
			err:      errors.New("database is locked"),
			expected: true,
		},
		{
			name:     "Connection error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "Node unavailable",
			err:      &RQLiteError{StatusCode: 503},
			expected: true,
		},
		{
			name:     "Syntax error (not retryable)",
			err:      errors.New("syntax error"),
			expected: false,
		},
		{
			name:     "Constraint violation (not retryable)",
			err:      errors.New("UNIQUE constraint failed"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryable() = %v; want %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}

// TestIsHTTPError tests HTTP error detection
func TestIsHTTPError(t *testing.T) {
	err404 := &RQLiteError{StatusCode: 404}
	if !IsHTTPError(err404, 404) {
		t.Error("Should detect HTTP 404 error")
	}

	if IsHTTPError(err404, 500) {
		t.Error("Should not detect HTTP 500 for 404 error")
	}

	otherErr := errors.New("generic error")
	if IsHTTPError(otherErr, 404) {
		t.Error("Should not detect HTTP error for non-RQLiteError")
	}
}

// TestGetRQLiteErrorDetails tests error detail extraction
func TestGetRQLiteErrorDetails(t *testing.T) {
	rqErr := &RQLiteError{
		Operation:  "UPDATE",
		Table:      "products",
		Query:      "UPDATE products SET price = 10",
		StatusCode: 500,
		Message:    "update failed",
		Detail:     "timeout occurred",
	}

	operation, table, query, message, detail, statusCode := GetRQLiteErrorDetails(rqErr)

	if operation != "UPDATE" {
		t.Errorf("Expected operation 'UPDATE', got '%s'", operation)
	}
	if table != "products" {
		t.Errorf("Expected table 'products', got '%s'", table)
	}
	if query != "UPDATE products SET price = 10" {
		t.Errorf("Expected query to match, got '%s'", query)
	}
	if message != "update failed" {
		t.Errorf("Expected message 'update failed', got '%s'", message)
	}
	if detail != "timeout occurred" {
		t.Errorf("Expected detail 'timeout occurred', got '%s'", detail)
	}
	if statusCode != 500 {
		t.Errorf("Expected status code 500, got %d", statusCode)
	}

	// Test with nil error
	op, tbl, qry, msg, det, code := GetRQLiteErrorDetails(nil)
	if op != "" || tbl != "" || qry != "" || msg != "" || det != "" || code != 0 {
		t.Error("GetRQLiteErrorDetails should return empty values for nil error")
	}

	// Test with non-RQLiteError
	genericErr := errors.New("generic error")
	_, _, _, message2, _, _ := GetRQLiteErrorDetails(genericErr)
	if message2 != "generic error" {
		t.Errorf("Expected message 'generic error', got '%s'", message2)
	}
}

// TestConvertRQLiteError tests error conversion
func TestConvertRQLiteError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedError string
	}{
		{
			name:          "Unique constraint",
			err:           errors.New("UNIQUE constraint failed"),
			expectedError: "unique constraint violation",
		},
		{
			name:          "Foreign key constraint",
			err:           errors.New("FOREIGN KEY constraint failed"),
			expectedError: "foreign key constraint violation",
		},
		{
			name:          "NOT NULL constraint",
			err:           errors.New("NOT NULL constraint failed"),
			expectedError: "NOT NULL constraint violation",
		},
		{
			name:          "Table not found",
			err:           errors.New("no such table: users"),
			expectedError: "table does not exist",
		},
		{
			name:          "Column not found",
			err:           errors.New("no such column: email"),
			expectedError: "column does not exist",
		},
		{
			name:          "Database locked",
			err:           errors.New("database is locked"),
			expectedError: "RQLite database is locked",
		},
		{
			name:          "Readonly database",
			err:           errors.New("attempt to write a readonly database"),
			expectedError: "RQLite node is in readonly mode",
		},
		{
			name:          "Connection error",
			err:           errors.New("connection refused"),
			expectedError: "failed to connect to RQLite server",
		},
		{
			name:          "ErrSQLNoRows",
			err:           orm.ErrSQLNoRows,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converted := ConvertRQLiteError(tt.err)

			if converted == nil {
				t.Fatal("ConvertRQLiteError should not return nil")
			}

			if tt.expectedError != "" && !strings.Contains(converted.Error(), tt.expectedError) {
				t.Errorf("Expected error to contain '%s', got '%s'", tt.expectedError, converted.Error())
			}
		})
	}

	// Test nil error
	nilConverted := ConvertRQLiteError(nil)
	if nilConverted != nil {
		t.Error("ConvertRQLiteError should return nil for nil error")
	}
}

// TestFormatRQLiteError tests error formatting
func TestFormatRQLiteError(t *testing.T) {
	rqErr := &RQLiteError{
		Operation:  "SELECT",
		Table:      "users",
		Query:      "SELECT * FROM users",
		StatusCode: 500,
		Message:    "query failed",
		Detail:     "connection timeout",
	}

	formatted := FormatRQLiteError(rqErr)

	if formatted == "" {
		t.Error("FormatRQLiteError should return non-empty string")
	}

	// Check that all components are present
	if !strings.Contains(formatted, "SELECT") {
		t.Errorf("Formatted error should contain operation: %s", formatted)
	}
	if !strings.Contains(formatted, "users") {
		t.Errorf("Formatted error should contain table: %s", formatted)
	}
	if !strings.Contains(formatted, "500") {
		t.Errorf("Formatted error should contain status code: %s", formatted)
	}
	if !strings.Contains(formatted, "query failed") {
		t.Errorf("Formatted error should contain message: %s", formatted)
	}

	// Test nil error
	nilFormatted := FormatRQLiteError(nil)
	if nilFormatted != "no error" {
		t.Errorf("Expected 'no error', got '%s'", nilFormatted)
	}

	// Test generic error
	genericErr := errors.New("generic error")
	genericFormatted := FormatRQLiteError(genericErr)
	if genericFormatted != "Message: generic error" {
		t.Errorf("Expected formatted generic error, got '%s'", genericFormatted)
	}
}

