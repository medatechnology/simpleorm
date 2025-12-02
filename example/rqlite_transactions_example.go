// +build ignore

package main

import (
	"fmt"
	"log"

	orm "github.com/medatechnology/simpleorm"
	"github.com/medatechnology/simpleorm/rqlite"
)

// This example demonstrates RQLite transaction support
// Note: RQLite transactions work differently from PostgreSQL:
// - Operations are buffered locally
// - All operations are sent atomically via /db/request endpoint on Commit()
// - SELECT queries within transactions execute immediately (limitation of HTTP-based architecture)

func main() {
	fmt.Println("=== SimpleORM RQLite Transaction Example ===\n")

	// Connect to RQLite
	config := rqlite.RqliteDirectConfig{
		URL:         "http://localhost:4001",
		Consistency: "strong", // strong, weak, or none
		Timeout:     0,        // use default
		RetryCount:  3,
	}

	db, err := rqlite.NewDatabase(config)
	if err != nil {
		log.Fatalf("Failed to connect to RQLite: %v", err)
	}

	fmt.Println("✓ Connected to RQLite")

	// Setup: Create tables
	setupTables(db)

	// Example 1: Basic transaction with commit
	example1BasicTransaction(db)

	// Example 2: Transaction with rollback
	example2TransactionRollback(db)

	// Example 3: Complex multi-step transaction
	example3ComplexTransaction(db)

	// Example 4: Deferred rollback pattern
	example4DeferredRollback(db)

	fmt.Println("\n=== All RQLite Transaction Examples Complete ===")
}

func setupTables(db *rqlite.RQLiteDirectDB) {
	fmt.Println("\n--- Setting Up Tables ---")

	sqls := []string{
		"DROP TABLE IF EXISTS orders",
		"DROP TABLE IF EXISTS products",
		"DROP TABLE IF EXISTS users",
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			balance REAL DEFAULT 0
		)`,
		`CREATE TABLE products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			price REAL NOT NULL,
			stock INTEGER DEFAULT 0
		)`,
		`CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			product_id INTEGER,
			quantity INTEGER,
			total REAL
		)`,
	}

	for _, sql := range sqls {
		result := db.ExecOneSQL(sql)
		if result.Error != nil {
			log.Printf("Warning: %v", result.Error)
		}
	}

	// Insert some test data
	testData := []string{
		"INSERT INTO users (name, email, balance) VALUES ('Alice', 'alice@example.com', 1000.00)",
		"INSERT INTO users (name, email, balance) VALUES ('Bob', 'bob@example.com', 500.00)",
		"INSERT INTO products (name, price, stock) VALUES ('Laptop', 999.99, 10)",
		"INSERT INTO products (name, price, stock) VALUES ('Mouse', 29.99, 50)",
	}

	for _, sql := range testData {
		db.ExecOneSQL(sql)
	}

	fmt.Println("✓ Tables created and test data inserted")
}

func example1BasicTransaction(db *rqlite.RQLiteDirectDB) {
	fmt.Println("\n--- Example 1: Basic Transaction with Commit ---")

	// Begin transaction
	tx, err := db.BeginTransaction()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	// Insert a new user (buffered)
	user := orm.DBRecord{
		TableName: "users",
		Data: map[string]interface{}{
			"name":    "Charlie",
			"email":   "charlie@example.com",
			"balance": 750.50,
		},
	}

	result := tx.InsertOneDBRecord(user)
	if result.Error != nil {
		tx.Rollback()
		log.Printf("Insert failed: %v", result.Error)
		return
	}

	// Update another user's balance (buffered)
	updateResult := tx.ExecOneSQL("UPDATE users SET balance = balance + 100 WHERE name = 'Bob'")
	if updateResult.Error != nil {
		tx.Rollback()
		log.Printf("Update failed: %v", updateResult.Error)
		return
	}

	// Commit transaction (sends all operations atomically to /db/request)
	if err := tx.Commit(); err != nil {
		log.Printf("Commit failed: %v", err)
		return
	}

	fmt.Println("✓ Transaction committed successfully")
	fmt.Println("  - Inserted new user: Charlie")
	fmt.Println("  - Updated Bob's balance")

	// Verify results
	users, _ := db.SelectOneSQL("SELECT name, balance FROM users WHERE name IN ('Bob', 'Charlie')")
	fmt.Println("✓ Verified transaction results:")
	for _, u := range users {
		fmt.Printf("  - %s: balance = %.2f\n", u.Data["name"], u.Data["balance"])
	}
}

func example2TransactionRollback(db *rqlite.RQLiteDirectDB) {
	fmt.Println("\n--- Example 2: Transaction with Rollback ---")

	// Get current user count
	beforeCount, _ := db.SelectOneSQL("SELECT COUNT(*) as count FROM users")
	initialCount := beforeCount[0].Data["count"]

	// Begin transaction
	tx, err := db.BeginTransaction()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	// Insert a valid user (buffered)
	validUser := orm.DBRecord{
		TableName: "users",
		Data: map[string]interface{}{
			"name":    "David",
			"email":   "david@example.com",
			"balance": 200.00,
		},
	}

	result := tx.InsertOneDBRecord(validUser)
	if result.Error != nil {
		tx.Rollback()
		return
	}

	fmt.Println("✓ Buffered insert for David")

	// Simulate an error condition - decide to rollback
	simulatedError := true
	if simulatedError {
		tx.Rollback()
		fmt.Println("✓ Transaction rolled back (operations discarded before sending)")

		// Verify rollback - count should be unchanged
		afterCount, _ := db.SelectOneSQL("SELECT COUNT(*) as count FROM users")
		finalCount := afterCount[0].Data["count"]

		if initialCount == finalCount {
			fmt.Printf("✓ Rollback verified - user count unchanged (still %v)\n", finalCount)
		} else {
			log.Printf("✗ Unexpected: count changed from %v to %v", initialCount, finalCount)
		}
	}
}

func example3ComplexTransaction(db *rqlite.RQLiteDirectDB) {
	fmt.Println("\n--- Example 3: Complex Multi-Step Transaction ---")

	// Begin transaction
	tx, err := db.BeginTransaction()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	// Step 1: Get user and product info (SELECTs execute immediately)
	users, err := tx.SelectOneSQLParameterized(orm.ParametereizedSQL{
		Query:  "SELECT id, balance FROM users WHERE name = ?",
		Values: []interface{}{"Alice"},
	})
	if err != nil || len(users) == 0 {
		tx.Rollback()
		log.Printf("Failed to get user: %v", err)
		return
	}

	userID := users[0].Data["id"]
	userBalance := users[0].Data["balance"]

	products, err := tx.SelectOneSQLParameterized(orm.ParametereizedSQL{
		Query:  "SELECT id, price, stock FROM products WHERE name = ?",
		Values: []interface{}{"Laptop"},
	})
	if err != nil || len(products) == 0 {
		tx.Rollback()
		log.Printf("Failed to get product: %v", err)
		return
	}

	productID := products[0].Data["id"]
	productPrice := products[0].Data["price"]
	productStock := products[0].Data["stock"]

	fmt.Printf("✓ User balance: %.2f, Product price: %.2f, Stock: %v\n",
		userBalance, productPrice, productStock)

	// Step 2: Check if user has enough balance
	balance := toFloat(userBalance)
	price := toFloat(productPrice)
	stock := toInt(productStock)

	if balance < price {
		tx.Rollback()
		fmt.Println("✗ Insufficient balance - transaction rolled back")
		return
	}

	if stock < 1 {
		tx.Rollback()
		fmt.Println("✗ Insufficient stock - transaction rolled back")
		return
	}

	// Step 3: Create order, update stock, update balance (all buffered)
	order := orm.DBRecord{
		TableName: "orders",
		Data: map[string]interface{}{
			"user_id":    userID,
			"product_id": productID,
			"quantity":   1,
			"total":      price,
		},
	}

	tx.InsertOneDBRecord(order)

	tx.ExecOneSQLParameterized(orm.ParametereizedSQL{
		Query:  "UPDATE products SET stock = stock - 1 WHERE id = ?",
		Values: []interface{}{productID},
	})

	tx.ExecOneSQLParameterized(orm.ParametereizedSQL{
		Query:  "UPDATE users SET balance = balance - ? WHERE id = ?",
		Values: []interface{}{price, userID},
	})

	// Step 4: Commit all operations atomically
	if err := tx.Commit(); err != nil {
		log.Printf("Transaction failed: %v", err)
		return
	}

	fmt.Println("✓ Complex transaction completed successfully")
	fmt.Println("  - Order created")
	fmt.Println("  - Product stock updated")
	fmt.Println("  - User balance updated")

	// Verify results
	updatedUser, _ := db.SelectOneSQL("SELECT name, balance FROM users WHERE id = " + fmt.Sprint(userID))
	if len(updatedUser) > 0 {
		fmt.Printf("✓ Alice's new balance: %.2f\n", updatedUser[0].Data["balance"])
	}
}

func example4DeferredRollback(db *rqlite.RQLiteDirectDB) {
	fmt.Println("\n--- Example 4: Deferred Rollback Pattern ---")

	tx, err := db.BeginTransaction()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	// Ensure rollback if not committed
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
			fmt.Println("✓ Transaction rolled back (defer pattern)")
		}
	}()

	// Buffer an update
	tx.ExecOneSQL("UPDATE products SET stock = stock + 5 WHERE name = 'Mouse'")

	// Verify the change will happen (check current stock)
	current, _ := db.SelectOneSQL("SELECT stock FROM products WHERE name = 'Mouse'")
	if len(current) > 0 {
		fmt.Printf("  Current mouse stock: %v (update buffered: +5)\n", current[0].Data["stock"])
	}

	// Intentionally NOT committing to demonstrate defer
	fmt.Println("  Simulating early return without commit...")
	// committed = true; tx.Commit() // Would commit if uncommented
}

// Helper functions
func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}
