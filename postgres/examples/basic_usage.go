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
	// Create a PostgreSQL configuration
	config := postgres.NewConfig(
		"localhost", // host
		5432,        // port
		"postgres",  // user
		"password",  // password
		"testdb",    // database name
	)

	// Customize the configuration using method chaining
	config.WithSSLMode("disable").
		WithConnectionPool(25, 5, 0, 0).
		WithApplicationName("simpleorm-example")

	// Create a new PostgreSQL database connection
	db, err := postgres.NewDatabase(*config)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	fmt.Println("Successfully connected to PostgreSQL!")

	// Example 1: Insert a single record
	fmt.Println("\n=== Example 1: Insert Single Record ===")
	record := orm.DBRecord{
		TableName: "users",
		Data: map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
			"age":   30,
		},
	}

	result := db.InsertOneDBRecord(record, false)
	if result.Error != nil {
		log.Printf("Failed to insert record: %v", result.Error)
	} else {
		fmt.Printf("Inserted record. Rows affected: %d\n", result.RowsAffected)
	}

	// Example 2: Insert multiple records (batch insert)
	fmt.Println("\n=== Example 2: Batch Insert Multiple Records ===")
	records := []orm.DBRecord{
		{
			TableName: "users",
			Data: map[string]interface{}{
				"name":  "Jane Smith",
				"email": "jane@example.com",
				"age":   28,
			},
		},
		{
			TableName: "users",
			Data: map[string]interface{}{
				"name":  "Bob Johnson",
				"email": "bob@example.com",
				"age":   35,
			},
		},
	}

	batchResults, err := db.InsertManyDBRecordsSameTable(records, false)
	if err != nil {
		log.Printf("Failed to batch insert: %v", err)
	} else {
		fmt.Printf("Batch inserted %d records\n", len(records))
		for i, res := range batchResults {
			if res.Error != nil {
				log.Printf("Batch %d failed: %v", i, res.Error)
			} else {
				fmt.Printf("Batch %d: Rows affected: %d\n", i, res.RowsAffected)
			}
		}
	}

	// Example 3: Select one record
	fmt.Println("\n=== Example 3: Select One Record ===")
	oneRecord, err := db.SelectOne("users")
	if err != nil {
		log.Printf("Failed to select one: %v", err)
	} else {
		fmt.Printf("Retrieved record: %+v\n", oneRecord.Data)
	}

	// Example 4: Select multiple records
	fmt.Println("\n=== Example 4: Select Multiple Records ===")
	allRecords, err := db.SelectMany("users")
	if err != nil {
		log.Printf("Failed to select many: %v", err)
	} else {
		fmt.Printf("Retrieved %d records\n", len(allRecords))
		for i, rec := range allRecords {
			fmt.Printf("  Record %d: %+v\n", i+1, rec.Data)
		}
	}

	// Example 5: Select with conditions
	fmt.Println("\n=== Example 5: Select With Conditions ===")
	condition := &orm.Condition{
		Field:    "age",
		Operator: ">",
		Value:    25,
	}

	filteredRecords, err := db.SelectManyWithCondition("users", condition)
	if err != nil {
		log.Printf("Failed to select with condition: %v", err)
	} else {
		fmt.Printf("Found %d records with age > 25\n", len(filteredRecords))
		for _, rec := range filteredRecords {
			fmt.Printf("  %s (age: %v)\n", rec.Data["name"], rec.Data["age"])
		}
	}

	// Example 6: Complex nested conditions
	fmt.Println("\n=== Example 6: Complex Nested Conditions ===")
	complexCondition := &orm.Condition{
		Logic: "OR",
		Nested: []orm.Condition{
			{
				Logic: "AND",
				Nested: []orm.Condition{
					{Field: "age", Operator: ">", Value: 30},
					{Field: "name", Operator: "LIKE", Value: "%John%"},
				},
			},
			{
				Field:    "email",
				Operator: "LIKE",
				Value:    "%@example.com",
			},
		},
	}

	complexRecords, err := db.SelectManyWithCondition("users", complexCondition)
	if err != nil {
		log.Printf("Failed to select with complex condition: %v", err)
	} else {
		fmt.Printf("Found %d records matching complex condition\n", len(complexRecords))
	}

	// Example 7: Raw SQL query
	fmt.Println("\n=== Example 7: Raw SQL Query ===")
	sql := "SELECT name, email FROM users WHERE age >= $1 ORDER BY age DESC"
	paramSQL := orm.ParametereizedSQL{
		Query:  sql,
		Values: []interface{}{30},
	}

	rawRecords, err := db.SelectOnlyOneSQLParameterized(paramSQL)
	if err != nil {
		log.Printf("Failed to execute raw SQL: %v", err)
	} else {
		fmt.Printf("Raw SQL result: %+v\n", rawRecords.Data)
	}

	// Example 8: Get database status
	fmt.Println("\n=== Example 8: Database Status ===")
	status, err := db.Status()
	if err != nil {
		log.Printf("Failed to get status: %v", err)
	} else {
		fmt.Printf("DBMS: %s\n", status.DBMS)
		fmt.Printf("Driver: %s\n", status.DBMSDriver)
		fmt.Printf("Version: %s\n", status.Version)
		fmt.Printf("Leader: %s\n", status.Leader)
		fmt.Printf("Nodes: %d\n", status.Nodes)
	}

	fmt.Println("\n=== Examples Complete ===")
}
