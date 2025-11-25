package orm

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	// LogLevelDebug for detailed diagnostic information
	LogLevelDebug LogLevel = iota
	// LogLevelInfo for general informational messages
	LogLevelInfo
	// LogLevelWarn for warning messages
	LogLevelWarn
	// LogLevelError for error messages
	LogLevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger is the interface for structured logging in SimpleORM
// Implementations can use any logging library (zap, logrus, zerolog, etc.)
type Logger interface {
	// Debug logs a debug-level message with optional fields
	Debug(msg string, fields ...Field)

	// Info logs an info-level message with optional fields
	Info(msg string, fields ...Field)

	// Warn logs a warning-level message with optional fields
	Warn(msg string, fields ...Field)

	// Error logs an error-level message with optional fields
	Error(msg string, fields ...Field)

	// With creates a new logger with the given fields pre-populated
	With(fields ...Field) Logger

	// SetLevel sets the minimum log level
	SetLevel(level LogLevel)
}

// Field represents a structured logging field (key-value pair)
type Field struct {
	Key   string
	Value interface{}
}

// F is a shorthand constructor for Field
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Common field constructors for convenience
func String(key, value string) Field         { return Field{key, value} }
func Int(key string, value int) Field        { return Field{key, value} }
func Int64(key string, value int64) Field    { return Field{key, value} }
func Float64(key string, value float64) Field { return Field{key, value} }
func Bool(key string, value bool) Field      { return Field{key, value} }
func Error(err error) Field                  { return Field{"error", err} }
func Duration(key string, d time.Duration) Field { return Field{key, d} }
func Any(key string, value interface{}) Field { return Field{key, value} }

// DefaultLogger is a simple implementation of the Logger interface
// It uses Go's standard log package with structured field formatting
type DefaultLogger struct {
	logger   *log.Logger
	minLevel LogLevel
	fields   []Field
}

// NewDefaultLogger creates a new default logger with the specified minimum level
func NewDefaultLogger(minLevel LogLevel) *DefaultLogger {
	return &DefaultLogger{
		logger:   log.New(os.Stdout, "", log.LstdFlags),
		minLevel: minLevel,
		fields:   make([]Field, 0),
	}
}

// NewNoopLogger creates a logger that doesn't log anything (useful for testing)
func NewNoopLogger() Logger {
	return &NoopLogger{}
}

// Debug logs a debug-level message
func (l *DefaultLogger) Debug(msg string, fields ...Field) {
	if l.minLevel <= LogLevelDebug {
		l.log(LogLevelDebug, msg, fields...)
	}
}

// Info logs an info-level message
func (l *DefaultLogger) Info(msg string, fields ...Field) {
	if l.minLevel <= LogLevelInfo {
		l.log(LogLevelInfo, msg, fields...)
	}
}

// Warn logs a warning-level message
func (l *DefaultLogger) Warn(msg string, fields ...Field) {
	if l.minLevel <= LogLevelWarn {
		l.log(LogLevelWarn, msg, fields...)
	}
}

// Error logs an error-level message
func (l *DefaultLogger) Error(msg string, fields ...Field) {
	if l.minLevel <= LogLevelError {
		l.log(LogLevelError, msg, fields...)
	}
}

// With creates a new logger with pre-populated fields
func (l *DefaultLogger) With(fields ...Field) Logger {
	newFields := make([]Field, 0, len(l.fields)+len(fields))
	newFields = append(newFields, l.fields...)
	newFields = append(newFields, fields...)

	return &DefaultLogger{
		logger:   l.logger,
		minLevel: l.minLevel,
		fields:   newFields,
	}
}

// SetLevel sets the minimum log level
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.minLevel = level
}

// log is the internal logging function
func (l *DefaultLogger) log(level LogLevel, msg string, fields ...Field) {
	// Combine pre-populated fields with current fields
	allFields := make([]Field, 0, len(l.fields)+len(fields))
	allFields = append(allFields, l.fields...)
	allFields = append(allFields, fields...)

	// Format the message with fields
	formatted := fmt.Sprintf("[%s] %s", level.String(), msg)
	if len(allFields) > 0 {
		formatted += " |"
		for _, field := range allFields {
			formatted += fmt.Sprintf(" %s=%v", field.Key, field.Value)
		}
	}

	l.logger.Println(formatted)
}

// NoopLogger is a logger that doesn't log anything
type NoopLogger struct{}

func (n *NoopLogger) Debug(msg string, fields ...Field) {}
func (n *NoopLogger) Info(msg string, fields ...Field)  {}
func (n *NoopLogger) Warn(msg string, fields ...Field)  {}
func (n *NoopLogger) Error(msg string, fields ...Field) {}
func (n *NoopLogger) With(fields ...Field) Logger       { return n }
func (n *NoopLogger) SetLevel(level LogLevel)           {}

// Global default logger - can be replaced by applications
var defaultLogger Logger = NewNoopLogger()

// SetDefaultLogger sets the global default logger used by SimpleORM
func SetDefaultLogger(logger Logger) {
	defaultLogger = logger
}

// GetDefaultLogger returns the current default logger
func GetDefaultLogger() Logger {
	return defaultLogger
}

// Convenience functions for logging with the default logger
func Debug(msg string, fields ...Field) {
	defaultLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...Field) {
	defaultLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...Field) {
	defaultLogger.Warn(msg, fields...)
}

func LogError(msg string, fields ...Field) {
	defaultLogger.Error(msg, fields...)
}
