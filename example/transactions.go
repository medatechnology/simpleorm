package main

import (
	"fmt"
	"log"

	orm "github.com/medatechnology/simpleorm"
	"github.com/medatechnology/simpleorm/postgres"
)

// runTransactionExamples demonstrates transaction usage patterns
func runTransactionExamples(db postgres.PostgresDirectDB) {
	fmt.Println("\n--- Example 5: Transactions ---")

	// Transaction Example 1: Explicit transaction with Commit
	explicitTransactionExample(db)

	// Transaction Example 2: Transaction with Rollback on error
	rollbackOnErrorExample(db)

	// Transaction Example 3: Complex multi-step transaction
	complexOrderTransactionExample(db)

	// Transaction Example 4: Deferred rollback pattern
	deferredRollbackExample(db)

	// Transaction Example 5: Atomic batch operations (implicit transactions)
	atomicBatchExample(db)
}

// explicitTransactionExample demonstrates explicit transaction control with Begin/Commit
func explicitTransactionExample(db postgres.PostgresDirectDB) {
	fmt.Println("\n  Transaction Example 1: Explicit Transaction with Commit")

	// Begin a transaction
	tx, err := db.BeginTransaction()
	if err != nil {
		log.Printf("  ✗ Failed to begin transaction: %v", err)
		return
	}

	// Insert multiple users within the transaction
	user1 := orm.DBRecord{
		TableName: "users",
		Data: map[string]interface{}{
			"name":  "Transaction User A",
			"email": "tx_a@example.com",
			"age":   25,
		},
	}

	result1 := tx.InsertOneDBRecord(user1)
	if result1.Error != nil {
		tx.Rollback()
		log.Printf("  ✗ Insert failed: %v", result1.Error)
		return
	}

	user2 := orm.DBRecord{
		TableName: "users",
		Data: map[string]interface{}{
			"name":  "Transaction User B",
			"email": "tx_b@example.com",
			"age":   30,
		},
	}

	result2 := tx.InsertOneDBRecord(user2)
	if result2.Error != nil {
		tx.Rollback()
		log.Printf("  ✗ Insert failed: %v", result2.Error)
		return
	}

	// Update a product within the same transaction
	updateResult := tx.ExecOneSQL("UPDATE products SET stock = stock + 5 WHERE name = 'Laptop'")
	if updateResult.Error != nil {
		tx.Rollback()
		log.Printf("  ✗ Update failed: %v", updateResult.Error)
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("  ✗ Failed to commit transaction: %v", err)
		return
	}

	fmt.Printf("  ✓ Transaction committed successfully\n")
	fmt.Printf("    - Inserted 2 users\n")
	fmt.Printf("    - Updated product stock\n")
}

// rollbackOnErrorExample demonstrates automatic rollback on error
func rollbackOnErrorExample(db postgres.PostgresDirectDB) {
	fmt.Println("\n  Transaction Example 2: Rollback on Error")

	// Get current user count
	beforeCount, err := db.SelectOneSQL("SELECT COUNT(*) as count FROM users")
	if err != nil {
		log.Printf("  ✗ Failed to get initial count: %v", err)
		return
	}
	initialCount := beforeCount[0].Data["count"]

	// Begin transaction
	tx, err := db.BeginTransaction()
	if err != nil {
		log.Printf("  ✗ Failed to begin transaction: %v", err)
		return
	}

	// Insert a valid user
	validUser := orm.DBRecord{
		TableName: "users",
		Data: map[string]interface{}{
			"name":  "Valid User",
			"email": "valid@example.com",
			"age":   35,
		},
	}

	result := tx.InsertOneDBRecord(validUser)
	if result.Error != nil {
		tx.Rollback()
		log.Printf("  ✗ Insert failed: %v", result.Error)
		return
	}

	fmt.Printf("  ✓ Inserted valid user (within transaction)\n")

	// Try to insert a user with duplicate email (will fail)
	duplicateUser := orm.DBRecord{
		TableName: "users",
		Data: map[string]interface{}{
			"name":  "Duplicate User",
			"email": "alice@example.com", // This email already exists
			"age":   40,
		},
	}

	result2 := tx.InsertOneDBRecord(duplicateUser)
	if result2.Error != nil {
		// Rollback on error
		tx.Rollback()
		fmt.Printf("  ✓ Error detected: %v\n", result2.Error)
		fmt.Printf("  ✓ Transaction rolled back\n")

		// Verify rollback - count should be unchanged
		afterCount, err := db.SelectOneSQL("SELECT COUNT(*) as count FROM users")
		if err != nil {
			log.Printf("  ✗ Failed to verify rollback: %v", err)
			return
		}
		finalCount := afterCount[0].Data["count"]

		if initialCount == finalCount {
			fmt.Printf("  ✓ Rollback verified - user count unchanged (still %v)\n", finalCount)
		} else {
			log.Printf("  ✗ Rollback failed - count changed from %v to %v", initialCount, finalCount)
		}
		return
	}

	// This should not execute because of the error above
	tx.Commit()
}

// complexOrderTransactionExample demonstrates a complex multi-step transaction
func complexOrderTransactionExample(db postgres.PostgresDirectDB) {
	fmt.Println("\n  Transaction Example 3: Complex Order Creation Transaction")

	// Get a user for the order
	userRecord, err := db.SelectOnlyOneSQL("SELECT id FROM users LIMIT 1")
	if err != nil {
		log.Printf("  ✗ Failed to get user: %v", err)
		return
	}
	userID := userRecord.Data["id"]

	// Begin transaction
	tx, err := db.BeginTransaction()
	if err != nil {
		log.Printf("  ✗ Failed to begin transaction: %v", err)
		return
	}

	// Ensure rollback on panic or error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("  ✗ Panic recovered, transaction rolled back: %v", r)
		}
	}()

	// Step 1: Create the order
	orderResult := tx.ExecOneSQLParameterized(orm.ParametereizedSQL{
		Query:  "INSERT INTO orders (user_id, total, status) VALUES ($1, $2, $3)",
		Values: []interface{}{userID, 1059.97, "confirmed"},
	})

	if orderResult.Error != nil {
		tx.Rollback()
		log.Printf("  ✗ Failed to create order: %v", orderResult.Error)
		return
	}

	fmt.Printf("  ✓ Created order (ID: %d)\n", orderResult.LastInsertID)

	// Step 2: Get the order ID (in real code, use RETURNING clause)
	orderRecords, err := tx.SelectOneSQLParameterized(orm.ParametereizedSQL{
		Query:  "SELECT id FROM orders WHERE user_id = $1 ORDER BY id DESC LIMIT 1",
		Values: []interface{}{userID},
	})

	if err != nil || len(orderRecords) == 0 {
		tx.Rollback()
		log.Printf("  ✗ Failed to get order ID: %v", err)
		return
	}

	orderID := orderRecords[0].Data["id"]

	// Step 3: Get products
	products, err := tx.SelectOneSQLParameterized(orm.ParametereizedSQL{
		Query:  "SELECT id, price, stock FROM products WHERE name IN ($1, $2) ORDER BY name",
		Values: []interface{}{"Laptop", "Mouse"},
	})

	if err != nil || len(products) < 2 {
		tx.Rollback()
		log.Printf("  ✗ Failed to get products: %v", err)
		return
	}

	laptopID := products[0].Data["id"]
	laptopPrice := products[0].Data["price"]
	laptopStock := products[0].Data["stock"]

	mouseID := products[1].Data["id"]
	mousePrice := products[1].Data["price"]
	mouseStock := products[1].Data["stock"]

	// Step 4: Check stock availability
	if toInt(laptopStock) < 1 || toInt(mouseStock) < 2 {
		tx.Rollback()
		log.Printf("  ✗ Insufficient stock")
		return
	}

	// Step 5: Create order items
	orderItems := []orm.DBRecord{
		{
			TableName: "order_items",
			Data: map[string]interface{}{
				"order_id":   orderID,
				"product_id": laptopID,
				"quantity":   1,
				"price":      laptopPrice,
			},
		},
		{
			TableName: "order_items",
			Data: map[string]interface{}{
				"order_id":   orderID,
				"product_id": mouseID,
				"quantity":   2,
				"price":      mousePrice,
			},
		},
	}

	itemResults, err := tx.InsertManyDBRecordsSameTable(orderItems)
	if err != nil {
		tx.Rollback()
		log.Printf("  ✗ Failed to create order items: %v", err)
		return
	}

	fmt.Printf("  ✓ Created %d order items\n", len(orderItems))

	// Step 6: Update product stock
	stockUpdates := []orm.ParametereizedSQL{
		{
			Query:  "UPDATE products SET stock = stock - $1 WHERE id = $2",
			Values: []interface{}{1, laptopID},
		},
		{
			Query:  "UPDATE products SET stock = stock - $1 WHERE id = $2",
			Values: []interface{}{2, mouseID},
		},
	}

	stockResults, err := tx.ExecManySQLParameterized(stockUpdates)
	if err != nil {
		tx.Rollback()
		log.Printf("  ✗ Failed to update stock: %v", err)
		return
	}

	totalStockUpdated := 0
	for _, res := range stockResults {
		totalStockUpdated += res.RowsAffected
	}

	fmt.Printf("  ✓ Updated stock for %d products\n", totalStockUpdated)

	// Commit the entire transaction
	if err := tx.Commit(); err != nil {
		log.Printf("  ✗ Failed to commit transaction: %v", err)
		return
	}

	fmt.Printf("  ✓ Order transaction completed successfully\n")
	fmt.Printf("    - Order created and confirmed\n")
	fmt.Printf("    - %d items added to order\n", len(itemResults))
	fmt.Printf("    - Inventory updated atomically\n")
}

// deferredRollbackExample demonstrates the defer pattern for safe rollback
func deferredRollbackExample(db postgres.PostgresDirectDB) {
	fmt.Println("\n  Transaction Example 4: Deferred Rollback Pattern")

	tx, err := db.BeginTransaction()
	if err != nil {
		log.Printf("  ✗ Failed to begin transaction: %v", err)
		return
	}

	// Use defer to ensure rollback if commit doesn't happen
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
			fmt.Printf("  ✓ Transaction rolled back (defer pattern)\n")
		}
	}()

	// Perform operations
	result := tx.ExecOneSQL("UPDATE products SET stock = stock + 10 WHERE name = 'Monitor'")
	if result.Error != nil {
		log.Printf("  ✗ Update failed: %v", result.Error)
		return // defer will rollback
	}

	// Simulate an error condition
	checkResult, err := tx.SelectOnlyOneSQL("SELECT stock FROM products WHERE name = 'Monitor'")
	if err != nil {
		log.Printf("  ✗ Check failed: %v", err)
		return // defer will rollback
	}

	newStock := checkResult.Data["stock"]
	fmt.Printf("  ✓ Updated monitor stock to %v\n", newStock)

	// Intentionally NOT committing to demonstrate defer rollback
	// In real code, you would: committed = true; tx.Commit()
	fmt.Printf("  ℹ Simulating early return without commit...\n")
}

// atomicBatchExample demonstrates implicit transactions via ExecManySQL
func atomicBatchExample(db postgres.PostgresDirectDB) {
	fmt.Println("\n  Transaction Example 5: Atomic Batch Operations (Implicit Transaction)")

	// ExecManySQL automatically wraps operations in a transaction
	sqls := []string{
		"UPDATE products SET stock = stock - 1 WHERE name = 'Keyboard'",
		"UPDATE products SET stock = stock - 1 WHERE name = 'Mouse'",
	}

	results, err := db.ExecManySQL(sqls)
	if err != nil {
		log.Printf("  ✗ Atomic batch failed (auto-rollback): %v", err)
		return
	}

	totalAffected := 0
	for _, result := range results {
		if result.Error != nil {
			log.Printf("  ✗ Operation failed: %v", result.Error)
			return
		}
		totalAffected += result.RowsAffected
	}

	fmt.Printf("  ✓ Atomic batch completed - %d rows affected across %d operations\n", totalAffected, len(results))
	fmt.Printf("  ℹ Note: ExecManySQL uses implicit transaction (no manual Begin/Commit needed)\n")
}

// Helper function to convert interface{} to int
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
