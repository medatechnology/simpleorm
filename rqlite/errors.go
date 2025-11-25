package rqlite

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	orm "github.com/medatechnology/simpleorm"
	"github.com/medatechnology/goutil/medaerror"
)

// SQLite error codes (RQLite uses SQLite under the hood)
// See: https://www.sqlite.org/rescode.html
const (
	// Primary result codes
	ErrCodeSQLiteOK         = 0  // Successful result
	ErrCodeSQLiteError      = 1  // Generic error
	ErrCodeSQLiteBusy       = 5  // Database is locked
	ErrCodeSQLiteReadonly   = 8  // Attempt to write a readonly database
	ErrCodeSQLiteConstraint = 19 // Constraint violation
	ErrCodeSQLiteNotADB     = 26 // File opened that is not a database

	// Extended result codes for constraint violations
	ErrCodeSQLiteConstraintUnique      = 2067 // UNIQUE constraint failed
	ErrCodeSQLiteConstraintPrimaryKey  = 1555 // PRIMARY KEY constraint failed
	ErrCodeSQLiteConstraintNotNull     = 1299 // NOT NULL constraint failed
	ErrCodeSQLiteConstraintForeignKey  = 787  // FOREIGN KEY constraint failed
	ErrCodeSQLiteConstraintCheck       = 275  // CHECK constraint failed

	// HTTP status codes for RQLite-specific errors
	HTTPStatusUnauthorized      = http.StatusUnauthorized      // 401
	HTTPStatusForbidden         = http.StatusForbidden         // 403
	HTTPStatusNotFound          = http.StatusNotFound          // 404
	HTTPStatusBadRequest        = http.StatusBadRequest        // 400
	HTTPStatusInternalError     = http.StatusInternalServerError // 500
	HTTPStatusServiceUnavailable = http.StatusServiceUnavailable // 503
)

// Common error messages from SQLite/RQLite
const (
	ErrMsgUniqueConstraint     = "UNIQUE constraint failed"
	ErrMsgPrimaryKeyConstraint = "PRIMARY KEY constraint failed"
	ErrMsgNotNullConstraint    = "NOT NULL constraint failed"
	ErrMsgForeignKeyConstraint = "FOREIGN KEY constraint failed"
	ErrMsgCheckConstraint      = "CHECK constraint failed"
	ErrMsgDatabaseLocked       = "database is locked"
	ErrMsgReadonlyDatabase     = "attempt to write a readonly database"
	ErrMsgNoSuchTable          = "no such table"
	ErrMsgNoSuchColumn         = "no such column"
	ErrMsgSyntaxError          = "syntax error"
)

// Custom RQLite errors using medaerror
var (
	ErrRQLiteNotConnected       medaerror.MedaError = medaerror.MedaError{Message: "RQLite database is not connected"}
	ErrRQLiteInvalidURL         medaerror.MedaError = medaerror.MedaError{Message: "invalid RQLite URL"}
	ErrRQLiteConnectionFailed   medaerror.MedaError = medaerror.MedaError{Message: "failed to connect to RQLite server"}
	ErrRQLiteQueryFailed        medaerror.MedaError = medaerror.MedaError{Message: "RQLite query execution failed"}
	ErrRQLiteExecuteFailed      medaerror.MedaError = medaerror.MedaError{Message: "RQLite execute command failed"}
	ErrRQLiteInvalidConfig      medaerror.MedaError = medaerror.MedaError{Message: "invalid RQLite configuration"}
	ErrRQLiteTimeout            medaerror.MedaError = medaerror.MedaError{Message: "RQLite operation timed out"}
	ErrRQLiteUnauthorized       medaerror.MedaError = medaerror.MedaError{Message: "RQLite authentication failed"}
	ErrRQLiteNodeUnavailable    medaerror.MedaError = medaerror.MedaError{Message: "RQLite node is unavailable"}
	ErrRQLiteReadonly           medaerror.MedaError = medaerror.MedaError{Message: "RQLite node is in readonly mode"}
	ErrRQLiteDatabaseLocked     medaerror.MedaError = medaerror.MedaError{Message: "RQLite database is locked"}
	ErrRQLiteInvalidJSON        medaerror.MedaError = medaerror.MedaError{Message: "invalid JSON response from RQLite"}
	ErrRQLiteConsistencyFailure medaerror.MedaError = medaerror.MedaError{Message: "RQLite consistency requirement not met"}
)

// RQLiteError wraps RQLite-specific errors with additional context
type RQLiteError struct {
	Operation  string // The operation that failed (e.g., "SELECT", "INSERT", "EXECUTE")
	Table      string // The table involved (if applicable)
	Query      string // The SQL query that failed (if applicable)
	StatusCode int    // HTTP status code (if applicable)
	Message    string // Error message
	Detail     string // Detailed error information from RQLite
	Err        error  // Original error
}

// Error implements the error interface
func (e *RQLiteError) Error() string {
	var parts []string

	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", e.Operation))
	}
	if e.Table != "" {
		parts = append(parts, fmt.Sprintf("table=%s", e.Table))
	}
	if e.StatusCode > 0 {
		parts = append(parts, fmt.Sprintf("status=%d", e.StatusCode))
	}

	msg := e.Message
	if len(parts) > 0 {
		msg = fmt.Sprintf("%s [%s]", msg, strings.Join(parts, ", "))
	}

	if e.Detail != "" {
		msg = fmt.Sprintf("%s - Detail: %s", msg, e.Detail)
	}

	return msg
}

// Unwrap returns the underlying error
func (e *RQLiteError) Unwrap() error {
	return e.Err
}

// WrapRQLiteError wraps an error with RQLite-specific context
func WrapRQLiteError(err error, operation, table, query string) error {
	if err == nil {
		return nil
	}

	rqErr := &RQLiteError{
		Operation: operation,
		Table:     table,
		Query:     query,
		Message:   err.Error(),
		Err:       err,
	}

	return rqErr
}

// WrapRQLiteHTTPError wraps an HTTP error with status code
func WrapRQLiteHTTPError(err error, operation, table, query string, statusCode int) error {
	if err == nil {
		return nil
	}

	rqErr := &RQLiteError{
		Operation:  operation,
		Table:      table,
		Query:      query,
		StatusCode: statusCode,
		Message:    err.Error(),
		Err:        err,
	}

	return rqErr
}

// IsUniqueViolation checks if the error is a UNIQUE constraint violation
func IsUniqueViolation(err error) bool {
	return containsErrorMessage(err, ErrMsgUniqueConstraint)
}

// IsPrimaryKeyViolation checks if the error is a PRIMARY KEY constraint violation
func IsPrimaryKeyViolation(err error) bool {
	return containsErrorMessage(err, ErrMsgPrimaryKeyConstraint)
}

// IsNotNullViolation checks if the error is a NOT NULL constraint violation
func IsNotNullViolation(err error) bool {
	return containsErrorMessage(err, ErrMsgNotNullConstraint)
}

// IsForeignKeyViolation checks if the error is a FOREIGN KEY constraint violation
func IsForeignKeyViolation(err error) bool {
	return containsErrorMessage(err, ErrMsgForeignKeyConstraint)
}

// IsCheckViolation checks if the error is a CHECK constraint violation
func IsCheckViolation(err error) bool {
	return containsErrorMessage(err, ErrMsgCheckConstraint)
}

// IsConstraintViolation checks if the error is any type of constraint violation
func IsConstraintViolation(err error) bool {
	return IsUniqueViolation(err) ||
		IsPrimaryKeyViolation(err) ||
		IsNotNullViolation(err) ||
		IsForeignKeyViolation(err) ||
		IsCheckViolation(err)
}

// IsDatabaseLocked checks if the error is due to database being locked
func IsDatabaseLocked(err error) bool {
	return containsErrorMessage(err, ErrMsgDatabaseLocked)
}

// IsReadonlyError checks if the error is due to attempting to write to a readonly database
func IsReadonlyError(err error) bool {
	return containsErrorMessage(err, ErrMsgReadonlyDatabase)
}

// IsTableNotFound checks if the error is due to a non-existent table
func IsTableNotFound(err error) bool {
	return containsErrorMessage(err, ErrMsgNoSuchTable)
}

// IsColumnNotFound checks if the error is due to a non-existent column
func IsColumnNotFound(err error) bool {
	return containsErrorMessage(err, ErrMsgNoSuchColumn)
}

// IsSyntaxError checks if the error is a SQL syntax error
func IsSyntaxError(err error) bool {
	return containsErrorMessage(err, ErrMsgSyntaxError)
}

// IsAuthenticationError checks if the error is an authentication failure
func IsAuthenticationError(err error) bool {
	var rqErr *RQLiteError
	if errors.As(err, &rqErr) {
		return rqErr.StatusCode == HTTPStatusUnauthorized || rqErr.StatusCode == HTTPStatusForbidden
	}
	return errors.Is(err, ErrRQLiteUnauthorized)
}

// IsConnectionError checks if the error is related to connection failure
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout errors
	if errors.Is(err, ErrRQLiteTimeout) {
		return true
	}

	// Check for connection-related errors in error message
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "network is unreachable") ||
		strings.Contains(errMsg, "i/o timeout")
}

// IsNodeUnavailable checks if the error is due to node being unavailable
func IsNodeUnavailable(err error) bool {
	var rqErr *RQLiteError
	if errors.As(err, &rqErr) {
		return rqErr.StatusCode == HTTPStatusServiceUnavailable
	}
	return errors.Is(err, ErrRQLiteNodeUnavailable)
}

// IsRetryable checks if the error is transient and the operation can be retried
func IsRetryable(err error) bool {
	return IsDatabaseLocked(err) ||
		IsConnectionError(err) ||
		IsNodeUnavailable(err)
}

// IsHTTPError checks if the error is an HTTP error with a specific status code
func IsHTTPError(err error, statusCode int) bool {
	var rqErr *RQLiteError
	if errors.As(err, &rqErr) {
		return rqErr.StatusCode == statusCode
	}
	return false
}

// containsErrorMessage checks if an error message contains a specific substring
func containsErrorMessage(err error, msg string) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, strings.ToLower(msg))
}

// GetRQLiteErrorDetails extracts detailed error information
func GetRQLiteErrorDetails(err error) (operation, table, query, message, detail string, statusCode int) {
	if err == nil {
		return
	}

	var rqErr *RQLiteError
	if errors.As(err, &rqErr) {
		operation = rqErr.Operation
		table = rqErr.Table
		query = rqErr.Query
		message = rqErr.Message
		detail = rqErr.Detail
		statusCode = rqErr.StatusCode
		return
	}

	message = err.Error()
	return
}

// ConvertRQLiteError converts an RQLite error to an appropriate ORM error
func ConvertRQLiteError(err error) error {
	if err == nil {
		return nil
	}

	// Check for specific error types
	if IsUniqueViolation(err) {
		return fmt.Errorf("unique constraint violation: %w", err)
	}
	if IsForeignKeyViolation(err) {
		return fmt.Errorf("foreign key constraint violation: %w", err)
	}
	if IsNotNullViolation(err) {
		return fmt.Errorf("NOT NULL constraint violation: %w", err)
	}
	if IsTableNotFound(err) {
		return fmt.Errorf("table does not exist: %w", err)
	}
	if IsColumnNotFound(err) {
		return fmt.Errorf("column does not exist: %w", err)
	}
	if IsDatabaseLocked(err) {
		return fmt.Errorf("%w: %v", ErrRQLiteDatabaseLocked, err)
	}
	if IsReadonlyError(err) {
		return fmt.Errorf("%w: %v", ErrRQLiteReadonly, err)
	}
	if IsConnectionError(err) {
		return fmt.Errorf("%w: %v", ErrRQLiteConnectionFailed, err)
	}
	if IsAuthenticationError(err) {
		return fmt.Errorf("%w: %v", ErrRQLiteUnauthorized, err)
	}
	if IsNodeUnavailable(err) {
		return fmt.Errorf("%w: %v", ErrRQLiteNodeUnavailable, err)
	}

	// Check for common ORM errors
	if errors.Is(err, orm.ErrSQLNoRows) {
		return orm.ErrSQLNoRows
	}
	if errors.Is(err, orm.ErrSQLMoreThanOneRow) {
		return orm.ErrSQLMoreThanOneRow
	}

	return err
}

// FormatRQLiteError formats an RQLite error for logging or display
func FormatRQLiteError(err error) string {
	if err == nil {
		return "no error"
	}

	operation, table, query, message, detail, statusCode := GetRQLiteErrorDetails(err)

	var parts []string
	if message != "" {
		parts = append(parts, fmt.Sprintf("Message: %s", message))
	}
	if operation != "" {
		parts = append(parts, fmt.Sprintf("Operation: %s", operation))
	}
	if table != "" {
		parts = append(parts, fmt.Sprintf("Table: %s", table))
	}
	if statusCode > 0 {
		parts = append(parts, fmt.Sprintf("HTTP Status: %d", statusCode))
	}
	if query != "" {
		parts = append(parts, fmt.Sprintf("Query: %s", query))
	}
	if detail != "" {
		parts = append(parts, fmt.Sprintf("Detail: %s", detail))
	}

	if len(parts) == 0 {
		return err.Error()
	}

	return strings.Join(parts, " | ")
}
