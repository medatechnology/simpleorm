//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"

	"github.com/medatechnology/simpleorm/postgres"
	orm "github.com/medatechnology/simpleorm"
)

func main() {
	fmt.Println("=== PostgreSQL Error Handling Examples ===\n")

	// Setup (using a test database - adjust credentials as needed)
	config := postgres.NewConfig("localhost", 5432, "postgres", "password", "testdb")
	config.WithSSLMode("disable")

	db, err := postgres.NewDatabase(*config)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Example 1: Handling connection errors
	fmt.Println("Example 1: Connection Error Handling")
	badConfig := postgres.NewConfig("nonexistent-host", 5432, "user", "pass", "db")
	_, err = postgres.NewDatabase(*badConfig)
	if err != nil {
		fmt.Printf("Expected connection error: %v\n", err)
		if postgres.IsConnectionError(err) {
			fmt.Println("  -> Detected as connection error")
		}
	}
	fmt.Println()

	// Example 2: Handling "no rows" errors
	fmt.Println("Example 2: No Rows Error")
	condition := &orm.Condition{
		Field:    "id",
		Operator: "=",
		Value:    99999, // Assuming this ID doesn't exist
	}
	_, err = db.SelectOneWithCondition("users", condition)
	if err != nil {
		if err == orm.ErrSQLNoRows {
			fmt.Println("No matching records found (expected)")
		} else {
			fmt.Printf("Other error: %v\n", err)
		}
	}
	fmt.Println()

	// Example 3: Handling unique constraint violations
	fmt.Println("Example 3: Unique Constraint Violation")
	// First insert
	record1 := orm.DBRecord{
		TableName: "users",
		Data: map[string]interface{}{
			"email": "unique@example.com",
			"name":  "Test User",
		},
	}
	result := db.InsertOneDBRecord(record1, false)
	if result.Error != nil {
		fmt.Printf("First insert error: %v\n", result.Error)
	} else {
		fmt.Println("First insert successful")

		// Try to insert duplicate email (assuming email has UNIQUE constraint)
		record2 := orm.DBRecord{
			TableName: "users",
			Data: map[string]interface{}{
				"email": "unique@example.com", // Same email
				"name":  "Another User",
			},
		}
		result2 := db.InsertOneDBRecord(record2, false)
		if result2.Error != nil {
			fmt.Printf("Duplicate insert error (expected): %v\n", result2.Error)
			if postgres.IsUniqueViolation(result2.Error) {
				fmt.Println("  -> Detected as unique constraint violation")
				code := postgres.GetPostgreSQLErrorCode(result2.Error)
				fmt.Printf("  -> Error code: %s\n", code)
			}
		}
	}
	fmt.Println()

	// Example 4: Handling invalid SQL
	fmt.Println("Example 4: Invalid SQL Handling")
	invalidSQL := orm.ParametereizedSQL{
		Query:  "SELECT * FROM nonexistent_table",
		Values: []interface{}{},
	}
	_, err = db.SelectOnlyOneSQLParameterized(invalidSQL)
	if err != nil {
		fmt.Printf("Invalid SQL error: %v\n", err)
		if postgres.IsUndefinedTable(err) {
			fmt.Println("  -> Detected as undefined table error")
		}
	}
	fmt.Println()

	// Example 5: Checking for various constraint violations
	fmt.Println("Example 5: Constraint Violation Detection")

	// Simulate different errors (in real scenarios these would come from actual DB operations)
	// For demonstration, we'll just show the helper functions

	fmt.Println("Available error detection helpers:")
	fmt.Println("  - IsUniqueViolation(err)")
	fmt.Println("  - IsForeignKeyViolation(err)")
	fmt.Println("  - IsNotNullViolation(err)")
	fmt.Println("  - IsCheckViolation(err)")
	fmt.Println("  - IsConstraintViolation(err)  // Any constraint")
	fmt.Println("  - IsUndefinedTable(err)")
	fmt.Println("  - IsUndefinedColumn(err)")
	fmt.Println("  - IsConnectionError(err)")
	fmt.Println("  - IsDeadlock(err)")
	fmt.Println("  - IsSerializationFailure(err)")
	fmt.Println("  - IsRetryable(err)  // Transient errors that can be retried")
	fmt.Println()

	// Example 6: Formatted error details
	fmt.Println("Example 6: Detailed Error Information")
	// Using the error from Example 4
	if err != nil {
		formatted := postgres.FormatPostgreSQLError(err)
		fmt.Printf("Formatted error: %s\n", formatted)

		code, message, detail, hint := postgres.GetPostgreSQLErrorDetail(err)
		fmt.Printf("Error breakdown:\n")
		fmt.Printf("  Code: %s\n", code)
		fmt.Printf("  Message: %s\n", message)
		fmt.Printf("  Detail: %s\n", detail)
		fmt.Printf("  Hint: %s\n", hint)
	}
	fmt.Println()

	// Example 7: Error wrapping for better context
	fmt.Println("Example 7: Error Wrapping")
	testErr := db.InsertOneDBRecord(orm.DBRecord{
		TableName: "invalid_table",
		Data:      map[string]interface{}{"col": "val"},
	}, false)

	if testErr.Error != nil {
		// The error is automatically wrapped with operation context
		fmt.Printf("Wrapped error with context: %v\n", testErr.Error)
	}
	fmt.Println()

	// Example 8: Configuration validation errors
	fmt.Println("Example 8: Configuration Validation")
	invalidConfigs := []postgres.PostgresConfig{
		{Host: "localhost", Port: 5432, Password: "pass", DBName: "db"}, // Missing user
		{Host: "localhost", Port: 5432, User: "user", Password: "pass"}, // Missing dbname
		{Host: "localhost", Port: 5432, User: "user", Password: "pass", DBName: "db", SSLMode: "invalid"}, // Invalid SSL mode
	}

	for i, cfg := range invalidConfigs {
		err := cfg.Validate()
		if err != nil {
			fmt.Printf("Config %d validation error: %v\n", i+1, err)
		}
	}
	fmt.Println()

	// Example 9: Retry logic for transient errors
	fmt.Println("Example 9: Retry Pattern for Transient Errors")
	fmt.Println("Example retry implementation:")
	fmt.Println(`
	func retryableOperation(db orm.Database, maxRetries int) error {
		for attempt := 0; attempt < maxRetries; attempt++ {
			result := db.InsertOneDBRecord(record, false)

			if result.Error == nil {
				return nil // Success
			}

			if postgres.IsRetryable(result.Error) {
				log.Printf("Attempt %d failed with retryable error: %v", attempt+1, result.Error)
				time.Sleep(time.Second * time.Duration(attempt+1))
				continue
			}

			// Non-retryable error
			return result.Error
		}
		return fmt.Errorf("max retries exceeded")
	}
	`)

	fmt.Println("\n=== Error Handling Examples Complete ===")
}
