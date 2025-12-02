package main

import (
	"fmt"
	"log"
	"os"

	orm "github.com/medatechnology/simpleorm"
	"github.com/medatechnology/simpleorm/postgres"
)

func main() {
	fmt.Println("=== SimpleORM PostgreSQL Example ===\n")

	// Database configuration
	config := postgres.NewConfig(
		getEnv("DB_HOST", "localhost"),
		getEnvInt("DB_PORT", 5432),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "simpleorm_example"),
	)

	// Configure connection pool and SSL
	config.WithSSLMode("disable").
		WithApplicationName("simpleorm-example")

	// Connect to database
	db, err := postgres.NewDatabase(*config)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	fmt.Println("✓ Successfully connected to PostgreSQL")

	// Run examples
	if err := setupDatabase(db); err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}

	runBasicExamples(db)
	runConditionExamples(db)
	runStructExamples(db)
	runTransactionExamples(db)
	runUpdateDeleteExamples(db)

	fmt.Println("\n=== All Examples Complete ===")
}

// setupDatabase creates the necessary tables for the examples
func setupDatabase(db postgres.PostgresDirectDB) error {
	fmt.Println("\n--- Setting Up Database Tables ---")

	// Drop existing tables if they exist
	dropSQL := []string{
		"DROP TABLE IF EXISTS order_items",
		"DROP TABLE IF EXISTS orders",
		"DROP TABLE IF EXISTS products",
		"DROP TABLE IF EXISTS users",
	}

	for _, sql := range dropSQL {
		result := db.ExecOneSQL(sql)
		if result.Error != nil {
			return fmt.Errorf("failed to drop table: %w", result.Error)
		}
	}

	// Create tables
	createSQL := []string{
		`CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			age INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			price DECIMAL(10, 2) NOT NULL,
			stock INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			total DECIMAL(10, 2),
			status VARCHAR(50) DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE order_items (
			id SERIAL PRIMARY KEY,
			order_id INTEGER REFERENCES orders(id),
			product_id INTEGER REFERENCES products(id),
			quantity INTEGER,
			price DECIMAL(10, 2)
		)`,
	}

	for _, sql := range createSQL {
		result := db.ExecOneSQL(sql)
		if result.Error != nil {
			return fmt.Errorf("failed to create table: %w", result.Error)
		}
	}

	fmt.Println("✓ Database tables created successfully")
	return nil
}

// runBasicExamples demonstrates basic CRUD operations
func runBasicExamples(db postgres.PostgresDirectDB) {
	fmt.Println("\n--- Example 1: Basic Insert & Select ---")

	// Insert a single record using DBRecord
	user := orm.DBRecord{
		TableName: "users",
		Data: map[string]interface{}{
			"name":  "Alice Johnson",
			"email": "alice@example.com",
			"age":   28,
		},
	}

	result := db.InsertOneDBRecord(user, false)
	if result.Error != nil {
		log.Printf("Insert failed: %v", result.Error)
		return
	}
	fmt.Printf("✓ Inserted user (ID: %d, Rows affected: %d)\n", result.LastInsertID, result.RowsAffected)

	// Batch insert multiple records
	users := []orm.DBRecord{
		{
			TableName: "users",
			Data: map[string]interface{}{
				"name":  "Bob Smith",
				"email": "bob@example.com",
				"age":   35,
			},
		},
		{
			TableName: "users",
			Data: map[string]interface{}{
				"name":  "Carol White",
				"email": "carol@example.com",
				"age":   42,
			},
		},
		{
			TableName: "users",
			Data: map[string]interface{}{
				"name":  "David Brown",
				"email": "david@example.com",
				"age":   31,
			},
		},
	}

	_, err := db.InsertManyDBRecordsSameTable(users, false)
	if err != nil {
		log.Printf("Batch insert failed: %v", err)
		return
	}
	fmt.Printf("✓ Batch inserted %d users\n", len(users))

	// Select all users
	allUsers, err := db.SelectMany("users")
	if err != nil {
		log.Printf("Select failed: %v", err)
		return
	}
	fmt.Printf("✓ Retrieved %d users from database\n", len(allUsers))
}

// runConditionExamples demonstrates various query conditions
func runConditionExamples(db postgres.PostgresDirectDB) {
	fmt.Println("\n--- Example 2: Query with Conditions ---")

	// Simple condition
	condition := &orm.Condition{
		Field:    "age",
		Operator: ">=",
		Value:    30,
	}

	records, err := db.SelectManyWithCondition("users", condition)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	fmt.Printf("✓ Found %d users aged 30 or older\n", len(records))
	for _, rec := range records {
		fmt.Printf("  - %s (age: %v)\n", rec.Data["name"], rec.Data["age"])
	}

	// Complex nested conditions: (age > 30 AND name LIKE '%Smith%') OR email LIKE '%carol%'
	complexCondition := &orm.Condition{
		Logic: "OR",
		Nested: []orm.Condition{
			{
				Logic: "AND",
				Nested: []orm.Condition{
					{Field: "age", Operator: ">", Value: 30},
					{Field: "name", Operator: "LIKE", Value: "%Smith%"},
				},
			},
			{Field: "email", Operator: "LIKE", Value: "%carol%"},
		},
	}

	complexRecords, err := db.SelectManyWithCondition("users", complexCondition)
	if err != nil {
		log.Printf("Complex query failed: %v", err)
		return
	}
	fmt.Printf("✓ Complex condition matched %d users\n", len(complexRecords))
	for _, rec := range complexRecords {
		fmt.Printf("  - %s (%v)\n", rec.Data["name"], rec.Data["email"])
	}

	// Parameterized SQL query
	paramSQL := orm.ParametereizedSQL{
		Query:  "SELECT * FROM users WHERE age BETWEEN $1 AND $2 ORDER BY age",
		Values: []interface{}{25, 35},
	}

	paramRecords, err := db.SelectOneSQLParameterized(paramSQL)
	if err != nil {
		log.Printf("Parameterized query failed: %v", err)
		return
	}
	fmt.Printf("✓ Parameterized query returned %d users\n", len(paramRecords))
}

// runStructExamples demonstrates using TableStruct interface
func runStructExamples(db postgres.PostgresDirectDB) {
	fmt.Println("\n--- Example 3: Using TableStruct ---")

	// Insert products using struct
	products := []orm.TableStruct{
		&Product{Name: "Laptop", Price: 999.99, Stock: 10},
		&Product{Name: "Mouse", Price: 29.99, Stock: 50},
		&Product{Name: "Keyboard", Price: 79.99, Stock: 30},
		&Product{Name: "Monitor", Price: 299.99, Stock: 15},
	}

	results, err := db.InsertManyTableStructs(products, false)
	if err != nil {
		log.Printf("Struct insert failed: %v", err)
		return
	}
	fmt.Printf("✓ Inserted %d products\n", len(results))

	// Query products
	allProducts, err := db.SelectMany("products")
	if err != nil {
		log.Printf("Product query failed: %v", err)
		return
	}
	fmt.Printf("✓ Retrieved %d products:\n", len(allProducts))
	for _, p := range allProducts {
		fmt.Printf("  - %s: $%.2f (stock: %v)\n", p.Data["name"], p.Data["price"], p.Data["stock"])
	}
}

// runUpdateDeleteExamples demonstrates UPDATE and DELETE using raw SQL
func runUpdateDeleteExamples(db postgres.PostgresDirectDB) {
	fmt.Println("\n--- Example 4: Update & Delete with Raw SQL ---")

	// UPDATE: Change a user's email
	updateSQL := orm.ParametereizedSQL{
		Query:  "UPDATE users SET email = $1 WHERE name = $2",
		Values: []interface{}{"alice.j@example.com", "Alice Johnson"},
	}

	updateResult := db.ExecOneSQLParameterized(updateSQL)
	if updateResult.Error != nil {
		log.Printf("Update failed: %v", updateResult.Error)
		return
	}
	fmt.Printf("✓ Updated user email (rows affected: %d)\n", updateResult.RowsAffected)

	// UPDATE multiple fields with raw SQL
	bulkUpdateSQL := "UPDATE products SET stock = stock + 5 WHERE price < 100"
	bulkResult := db.ExecOneSQL(bulkUpdateSQL)
	if bulkResult.Error != nil {
		log.Printf("Bulk update failed: %v", bulkResult.Error)
		return
	}
	fmt.Printf("✓ Bulk updated product stock (rows affected: %d)\n", bulkResult.RowsAffected)

	// DELETE: Remove products with 0 stock (none in this example)
	deleteSQL := "DELETE FROM products WHERE stock = 0"
	deleteResult := db.ExecOneSQL(deleteSQL)
	if deleteResult.Error != nil {
		log.Printf("Delete failed: %v", deleteResult.Error)
		return
	}
	fmt.Printf("✓ Deleted products with 0 stock (rows affected: %d)\n", deleteResult.RowsAffected)

	// Verify the update
	updatedUser, err := db.SelectOnlyOneSQL("SELECT * FROM users WHERE name = 'Alice Johnson'")
	if err != nil {
		log.Printf("Verification query failed: %v", err)
		return
	}
	fmt.Printf("✓ Verified update - Alice's new email: %s\n", updatedUser.Data["email"])
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}
