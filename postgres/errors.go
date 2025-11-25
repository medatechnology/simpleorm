package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	orm "github.com/medatechnology/simpleorm"
	"github.com/medatechnology/goutil/medaerror"
)

// PostgreSQL-specific error codes
// See: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	// Class 23 - Integrity Constraint Violation
	ErrCodeUniqueViolation      = "23505"
	ErrCodeForeignKeyViolation  = "23503"
	ErrCodeNotNullViolation     = "23502"
	ErrCodeCheckViolation       = "23514"
	ErrCodeExclusionViolation   = "23P01"

	// Class 42 - Syntax Error or Access Rule Violation
	ErrCodeUndefinedTable       = "42P01"
	ErrCodeUndefinedColumn      = "42703"
	ErrCodeDuplicateTable       = "42P07"
	ErrCodeDuplicateColumn      = "42701"
	ErrCodeInvalidSchemaName    = "3F000"

	// Class 08 - Connection Exception
	ErrCodeConnectionException  = "08000"
	ErrCodeConnectionFailure    = "08006"
	ErrCodeSQLClientCannotConnect = "08001"

	// Class 57 - Operator Intervention
	ErrCodeAdminShutdown        = "57P01"
	ErrCodeCrashShutdown        = "57P02"
	ErrCodeCannotConnectNow     = "57P03"

	// Class 53 - Insufficient Resources
	ErrCodeInsufficientResources = "53000"
	ErrCodeDiskFull             = "53100"
	ErrCodeOutOfMemory          = "53200"
	ErrCodeTooManyConnections   = "53300"

	// Class 40 - Transaction Rollback
	ErrCodeDeadlockDetected     = "40P01"
	ErrCodeSerializationFailure = "40001"
)

// Custom PostgreSQL errors using medaerror
var (
	ErrPostgresNotConnected     medaerror.MedaError = medaerror.MedaError{Message: "PostgreSQL database is not connected"}
	ErrPostgresInvalidDSN       medaerror.MedaError = medaerror.MedaError{Message: "invalid PostgreSQL DSN connection string"}
	ErrPostgresConnectionFailed medaerror.MedaError = medaerror.MedaError{Message: "failed to connect to PostgreSQL database"}
	ErrPostgresQueryFailed      medaerror.MedaError = medaerror.MedaError{Message: "PostgreSQL query execution failed"}
	ErrPostgresTransactionFailed medaerror.MedaError = medaerror.MedaError{Message: "PostgreSQL transaction failed"}
	ErrPostgresInvalidConfig    medaerror.MedaError = medaerror.MedaError{Message: "invalid PostgreSQL configuration"}
	ErrPostgresTimeout          medaerror.MedaError = medaerror.MedaError{Message: "PostgreSQL operation timed out"}
	ErrPostgresNoAffectedRows   medaerror.MedaError = medaerror.MedaError{Message: "PostgreSQL query affected zero rows"}
)

// PostgreSQLError wraps PostgreSQL-specific errors with additional context
type PostgreSQLError struct {
	Operation string      // The operation that failed (e.g., "INSERT", "SELECT", "UPDATE")
	Table     string      // The table involved (if applicable)
	Query     string      // The SQL query that failed (if applicable)
	Code      string      // PostgreSQL error code
	Message   string      // Error message
	Detail    string      // Detailed error information
	Hint      string      // Hint for fixing the error
	Err       error       // Original error
}

// Error implements the error interface
func (e *PostgreSQLError) Error() string {
	var parts []string

	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", e.Operation))
	}
	if e.Table != "" {
		parts = append(parts, fmt.Sprintf("table=%s", e.Table))
	}
	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("code=%s", e.Code))
	}

	msg := e.Message
	if len(parts) > 0 {
		msg = fmt.Sprintf("%s [%s]", msg, strings.Join(parts, ", "))
	}

	if e.Detail != "" {
		msg = fmt.Sprintf("%s - Detail: %s", msg, e.Detail)
	}
	if e.Hint != "" {
		msg = fmt.Sprintf("%s - Hint: %s", msg, e.Hint)
	}

	return msg
}

// Unwrap returns the underlying error
func (e *PostgreSQLError) Unwrap() error {
	return e.Err
}

// WrapPostgreSQLError wraps a PostgreSQL error with additional context
func WrapPostgreSQLError(err error, operation, table, query string) error {
	if err == nil {
		return nil
	}

	pgErr := &PostgreSQLError{
		Operation: operation,
		Table:     table,
		Query:     query,
		Message:   err.Error(),
		Err:       err,
	}

	// Extract PostgreSQL-specific error information if available
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		pgErr.Code = string(pqErr.Code)
		pgErr.Message = pqErr.Message
		pgErr.Detail = pqErr.Detail
		pgErr.Hint = pqErr.Hint
	}

	return pgErr
}

// IsUniqueViolation checks if the error is a unique constraint violation
func IsUniqueViolation(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeUniqueViolation)
}

// IsForeignKeyViolation checks if the error is a foreign key constraint violation
func IsForeignKeyViolation(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeForeignKeyViolation)
}

// IsNotNullViolation checks if the error is a NOT NULL constraint violation
func IsNotNullViolation(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeNotNullViolation)
}

// IsCheckViolation checks if the error is a CHECK constraint violation
func IsCheckViolation(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeCheckViolation)
}

// IsConstraintViolation checks if the error is any type of constraint violation
func IsConstraintViolation(err error) bool {
	return IsUniqueViolation(err) ||
		   IsForeignKeyViolation(err) ||
		   IsNotNullViolation(err) ||
		   IsCheckViolation(err) ||
		   hasPostgreSQLErrorCode(err, ErrCodeExclusionViolation)
}

// IsUndefinedTable checks if the error is due to a non-existent table
func IsUndefinedTable(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeUndefinedTable)
}

// IsUndefinedColumn checks if the error is due to a non-existent column
func IsUndefinedColumn(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeUndefinedColumn)
}

// IsDuplicateTable checks if the error is due to attempting to create an existing table
func IsDuplicateTable(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeDuplicateTable)
}

// IsDuplicateColumn checks if the error is due to attempting to create an existing column
func IsDuplicateColumn(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeDuplicateColumn)
}

// IsConnectionError checks if the error is related to database connection
func IsConnectionError(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeConnectionException) ||
		   hasPostgreSQLErrorCode(err, ErrCodeConnectionFailure) ||
		   hasPostgreSQLErrorCode(err, ErrCodeSQLClientCannotConnect) ||
		   hasPostgreSQLErrorCode(err, ErrCodeCannotConnectNow)
}

// IsDeadlock checks if the error is due to a deadlock
func IsDeadlock(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeDeadlockDetected)
}

// IsSerializationFailure checks if the error is a serialization failure
func IsSerializationFailure(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeSerializationFailure)
}

// IsRetryable checks if the error is transient and the operation can be retried
func IsRetryable(err error) bool {
	return IsDeadlock(err) || IsSerializationFailure(err) || IsConnectionError(err)
}

// IsTooManyConnections checks if the error is due to too many connections
func IsTooManyConnections(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeTooManyConnections)
}

// IsInsufficientResources checks if the error is due to insufficient resources
func IsInsufficientResources(err error) bool {
	return hasPostgreSQLErrorCode(err, ErrCodeInsufficientResources) ||
		   hasPostgreSQLErrorCode(err, ErrCodeDiskFull) ||
		   hasPostgreSQLErrorCode(err, ErrCodeOutOfMemory) ||
		   IsTooManyConnections(err)
}

// hasPostgreSQLErrorCode checks if an error has a specific PostgreSQL error code
func hasPostgreSQLErrorCode(err error, code string) bool {
	if err == nil {
		return false
	}

	// Check if it's a pq.Error directly
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == code
	}

	// Check if it's wrapped in PostgreSQLError
	var pgErr *PostgreSQLError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}

	return false
}

// GetPostgreSQLErrorCode extracts the PostgreSQL error code from an error
func GetPostgreSQLErrorCode(err error) string {
	if err == nil {
		return ""
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code)
	}

	var pgErr *PostgreSQLError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}

	return ""
}

// GetPostgreSQLErrorDetail extracts the detailed error information
func GetPostgreSQLErrorDetail(err error) (code, message, detail, hint string) {
	if err == nil {
		return
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		code = string(pqErr.Code)
		message = pqErr.Message
		detail = pqErr.Detail
		hint = pqErr.Hint
		return
	}

	var pgErr *PostgreSQLError
	if errors.As(err, &pgErr) {
		code = pgErr.Code
		message = pgErr.Message
		detail = pgErr.Detail
		hint = pgErr.Hint
		return
	}

	message = err.Error()
	return
}

// ConvertPostgreSQLError converts a PostgreSQL error to an appropriate ORM error
func ConvertPostgreSQLError(err error) error {
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
	if IsUndefinedTable(err) {
		return fmt.Errorf("table does not exist: %w", err)
	}
	if IsUndefinedColumn(err) {
		return fmt.Errorf("column does not exist: %w", err)
	}
	if IsConnectionError(err) {
		return fmt.Errorf("%w: %v", ErrPostgresConnectionFailed, err)
	}
	if IsDeadlock(err) {
		return fmt.Errorf("deadlock detected: %w", err)
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

// FormatPostgreSQLError formats a PostgreSQL error for logging or display
func FormatPostgreSQLError(err error) string {
	if err == nil {
		return "no error"
	}

	code, message, detail, hint := GetPostgreSQLErrorDetail(err)

	var parts []string
	if message != "" {
		parts = append(parts, fmt.Sprintf("Message: %s", message))
	}
	if code != "" {
		parts = append(parts, fmt.Sprintf("Code: %s", code))
	}
	if detail != "" {
		parts = append(parts, fmt.Sprintf("Detail: %s", detail))
	}
	if hint != "" {
		parts = append(parts, fmt.Sprintf("Hint: %s", hint))
	}

	if len(parts) == 0 {
		return err.Error()
	}

	return strings.Join(parts, " | ")
}
