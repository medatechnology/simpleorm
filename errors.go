package orm

import (
	"fmt"

	"github.com/medatechnology/goutil/medaerror"
)

// ErrorContext provides additional context for errors
type ErrorContext struct {
	Operation string                 // The operation that failed (e.g., "SELECT", "INSERT")
	Table     string                 // The table involved (if applicable)
	Query     string                 // The SQL query (if applicable)
	Fields    map[string]interface{} // Additional context fields
}

// ORMError wraps an error with additional context
type ORMError struct {
	Err     error
	Context ErrorContext
}

// Error implements the error interface
func (e *ORMError) Error() string {
	msg := e.Err.Error()

	var parts []string
	if e.Context.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", e.Context.Operation))
	}
	if e.Context.Table != "" {
		parts = append(parts, fmt.Sprintf("table=%s", e.Context.Table))
	}

	if len(parts) > 0 {
		msg = fmt.Sprintf("%s [%s]", msg, joinStrings(parts, ", "))
	}

	return msg
}

// Unwrap returns the underlying error
func (e *ORMError) Unwrap() error {
	return e.Err
}

// WrapError wraps an error with context information
func WrapError(err error, operation, table string) error {
	if err == nil {
		return nil
	}

	return &ORMError{
		Err: err,
		Context: ErrorContext{
			Operation: operation,
			Table:     table,
		},
	}
}

// WrapErrorWithQuery wraps an error with context including the SQL query
func WrapErrorWithQuery(err error, operation, table, query string) error {
	if err == nil {
		return nil
	}

	return &ORMError{
		Err: err,
		Context: ErrorContext{
			Operation: operation,
			Table:     table,
			Query:     query,
		},
	}
}

// WrapErrorWithFields wraps an error with additional field context
func WrapErrorWithFields(err error, operation, table string, fields map[string]interface{}) error {
	if err == nil {
		return nil
	}

	return &ORMError{
		Err: err,
		Context: ErrorContext{
			Operation: operation,
			Table:     table,
			Fields:    fields,
		},
	}
}

// IsORMError checks if an error is an ORMError
func IsORMError(err error) bool {
	_, ok := err.(*ORMError)
	return ok
}

// GetErrorContext extracts the error context if the error is an ORMError
func GetErrorContext(err error) (ErrorContext, bool) {
	if ormErr, ok := err.(*ORMError); ok {
		return ormErr.Context, true
	}
	return ErrorContext{}, false
}

// NewError creates a new medaerror with a message and wraps it with ORM context
func NewError(message, operation, table string) error {
	return WrapError(
		medaerror.MedaError{Message: message},
		operation,
		table,
	)
}

// Helper function to join strings
func joinStrings(parts []string, separator string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += separator + parts[i]
	}
	return result
}

// Common error wrapping helpers for specific operations

// WrapSelectError wraps a SELECT operation error
func WrapSelectError(err error, table string) error {
	return WrapError(err, "SELECT", table)
}

// WrapInsertError wraps an INSERT operation error
func WrapInsertError(err error, table string) error {
	return WrapError(err, "INSERT", table)
}

// WrapUpdateError wraps an UPDATE operation error
func WrapUpdateError(err error, table string) error {
	return WrapError(err, "UPDATE", table)
}

// WrapDeleteError wraps a DELETE operation error
func WrapDeleteError(err error, table string) error {
	return WrapError(err, "DELETE", table)
}

// WrapConnectionError wraps a connection-related error
func WrapConnectionError(err error) error {
	return WrapError(err, "CONNECT", "")
}

// WrapTransactionError wraps a transaction-related error
func WrapTransactionError(err error, operation string) error {
	return WrapError(err, "TRANSACTION:"+operation, "")
}

// FormatError formats an error for logging with all available context
func FormatError(err error) string {
	if err == nil {
		return "no error"
	}

	if ormErr, ok := err.(*ORMError); ok {
		var parts []string
		parts = append(parts, fmt.Sprintf("Error: %s", ormErr.Err.Error()))

		if ormErr.Context.Operation != "" {
			parts = append(parts, fmt.Sprintf("Operation: %s", ormErr.Context.Operation))
		}
		if ormErr.Context.Table != "" {
			parts = append(parts, fmt.Sprintf("Table: %s", ormErr.Context.Table))
		}
		if ormErr.Context.Query != "" {
			parts = append(parts, fmt.Sprintf("Query: %s", ormErr.Context.Query))
		}
		if len(ormErr.Context.Fields) > 0 {
			parts = append(parts, fmt.Sprintf("Fields: %v", ormErr.Context.Fields))
		}

		return joinStrings(parts, " | ")
	}

	return err.Error()
}

// LogError logs an error with the default logger
func LogErrorWithContext(err error, fields ...Field) {
	if err == nil {
		return
	}

	logFields := make([]Field, 0, len(fields)+4)
	logFields = append(logFields, fields...)

	// Add context from ORMError if available
	if ormErr, ok := err.(*ORMError); ok {
		if ormErr.Context.Operation != "" {
			logFields = append(logFields, String("operation", ormErr.Context.Operation))
		}
		if ormErr.Context.Table != "" {
			logFields = append(logFields, String("table", ormErr.Context.Table))
		}
		if ormErr.Context.Query != "" {
			logFields = append(logFields, String("query", ormErr.Context.Query))
		}
	}

	logFields = append(logFields, Error(err))

	LogError(err.Error(), logFields...)
}
