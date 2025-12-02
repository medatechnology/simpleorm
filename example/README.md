# SimpleORM PostgreSQL Example

This example project demonstrates how to use the SimpleORM library with PostgreSQL, including comprehensive transaction usage patterns.

## Features Demonstrated

### 1. **Database Setup and Connection**
- Creating and configuring PostgreSQL connection
- Setting up connection pooling
- Managing database lifecycle

### 2. **Basic CRUD Operations**
- **Insert**: Single and batch inserts using `DBRecord`
- **Select**: Simple queries, conditional queries, parameterized queries
- **Update**: Using raw SQL with parameterized queries
- **Delete**: Using raw SQL statements

### 3. **Query Conditions**
- Simple conditions (field, operator, value)
- Complex nested conditions (AND/OR logic)
- Parameterized SQL queries for safety

### 4. **TableStruct Interface**
- Defining domain models that implement `orm.TableStruct`
- Type-safe struct-based operations
- Converting between structs and `DBRecord`

### 5. **Transaction Usage** ⭐
The example includes comprehensive transaction demonstrations with **explicit transaction control** similar to sqlx:

#### **Explicit Transaction Control**
- Using `BeginTransaction()` to start a transaction
- Manual `Commit()` and `Rollback()` control
- Just like database/sql and sqlx!

#### **Transaction with Commit**
- Begin transaction with `db.BeginTransaction()`
- Perform multiple operations
- Explicitly commit with `tx.Commit()`

#### **Transaction with Rollback**
- Error handling with automatic rollback
- Verifying rollback behavior
- Data integrity protection

#### **Complex Multi-Step Transactions**
- Order creation with inventory management
- Multiple related operations in a single atomic transaction
- Stock deduction coordinated with order creation

#### **Deferred Rollback Pattern**
- Using Go's `defer` for safe transaction cleanup
- Ensuring rollback on early returns or panics
- Best practice transaction patterns

#### **Implicit Transactions**
- Using `ExecManySQL` for automatic transaction wrapping
- No manual Begin/Commit needed for simple batches

## Project Structure

```
example/
├── main.go           # Main application entry point and basic examples
├── models.go         # Domain models (User, Product, Order, OrderItem)
├── transactions.go   # Transaction usage examples
├── go.mod           # Go module definition
├── .env.example     # Environment configuration template
└── README.md        # This file
```

## Prerequisites

1. **Go 1.21 or higher**
   ```bash
   go version
   ```

2. **PostgreSQL 12 or higher**
   ```bash
   psql --version
   ```

3. **Running PostgreSQL instance**
   - Default: `localhost:5432`
   - Can be configured via environment variables

## Setup

### 1. Install PostgreSQL

**macOS (using Homebrew):**
```bash
brew install postgresql@15
brew services start postgresql@15
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
```

**Docker:**
```bash
docker run --name postgres-simpleorm \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=simpleorm_example \
  -p 5432:5432 \
  -d postgres:15
```

### 2. Create Database

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database
CREATE DATABASE simpleorm_example;

# Exit psql
\q
```

### 3. Configure Environment

Copy the example environment file:
```bash
cp .env.example .env
```

Edit `.env` with your PostgreSQL credentials:
```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=simpleorm_example
```

### 4. Install Dependencies

From the example directory:
```bash
cd example
go mod download
```

## Running the Example

```bash
# From the example directory
go run .
```

Or with custom environment variables:
```bash
DB_HOST=localhost DB_NAME=mydb DB_USER=myuser DB_PASSWORD=mypass go run .
```

## Expected Output

```
=== SimpleORM PostgreSQL Example ===

✓ Successfully connected to PostgreSQL

--- Setting Up Database Tables ---
✓ Database tables created successfully

--- Example 1: Basic Insert & Select ---
✓ Inserted user (ID: 1, Rows affected: 1)
✓ Batch inserted 3 users
✓ Retrieved 4 users from database

--- Example 2: Query with Conditions ---
✓ Found 3 users aged 30 or older
  - Bob Smith (age: 35)
  - Carol White (age: 42)
  - David Brown (age: 31)
✓ Complex condition matched 2 users
  - Bob Smith (bob@example.com)
  - Carol White (carol@example.com)
✓ Parameterized query returned 3 users

--- Example 3: Using TableStruct ---
✓ Inserted 4 products
✓ Retrieved 4 products:
  - Laptop: $999.99 (stock: 10)
  - Mouse: $29.99 (stock: 50)
  - Keyboard: $79.99 (stock: 30)
  - Monitor: $299.99 (stock: 15)

--- Example 4: Update & Delete with Raw SQL ---
✓ Updated user email (rows affected: 1)
✓ Bulk updated product stock (rows affected: 3)
✓ Deleted products with 0 stock (rows affected: 0)
✓ Verified update - Alice's new email: alice.j@example.com

--- Example 5: Transactions ---

  Transaction Example 1: Atomic Batch Operations
  ✓ Atomic batch completed - 3 total rows affected across 3 operations

  Transaction Example 2: Parameterized Batch Operations
  ✓ Parameterized batch completed - inserted 3 users atomically
  ✓ Verified: 3 transaction users in database

  Transaction Example 3: Complex Order Creation
  ✓ Order created successfully in atomic transaction
    - Order record created
    - Stock updated for 2 products
  ✓ Verified stock levels after transaction:
    - Laptop: 8 units
    - Mouse: 47 units

  Transaction Example 4: Rollback on Error
  ✓ Transaction correctly failed: failed to execute SQL: ...
  ✓ All changes have been rolled back
  ✓ Rollback verified - stock unchanged (still 120)
  ✓ Transaction atomicity confirmed: all-or-nothing behavior

=== All Examples Complete ===
```

## Code Examples

### Basic Insert
```go
record := orm.DBRecord{
    TableName: "users",
    Data: map[string]interface{}{
        "name":  "Alice Johnson",
        "email": "alice@example.com",
        "age":   28,
    },
}

result := db.InsertOneDBRecord(record, false)
if result.Error != nil {
    log.Printf("Insert failed: %v", result.Error)
}
```

### Query with Conditions
```go
condition := &orm.Condition{
    Field:    "age",
    Operator: ">=",
    Value:    30,
}

records, err := db.SelectManyWithCondition("users", condition)
```

### Using TableStruct
```go
type Product struct {
    ID    int     `json:"id" db:"id"`
    Name  string  `json:"name" db:"name"`
    Price float64 `json:"price" db:"price"`
}

func (p *Product) TableName() string {
    return "products"
}

// Insert using struct
product := &Product{Name: "Laptop", Price: 999.99}
result := db.InsertOneTableStruct(product, false)
```

### Explicit Transaction Control (like sqlx)
```go
// Begin a transaction
tx, err := db.BeginTransaction()
if err != nil {
    log.Fatalf("Failed to begin transaction: %v", err)
}

// Insert a record
user := orm.DBRecord{
    TableName: "users",
    Data: map[string]interface{}{
        "name":  "John Doe",
        "email": "john@example.com",
        "age":   30,
    },
}

result := tx.InsertOneDBRecord(user)
if result.Error != nil {
    tx.Rollback()
    log.Printf("Insert failed, rolled back: %v", result.Error)
    return
}

// Update product stock
updateResult := tx.ExecOneSQL("UPDATE products SET stock = stock - 1 WHERE id = 1")
if updateResult.Error != nil {
    tx.Rollback()
    log.Printf("Update failed, rolled back: %v", updateResult.Error)
    return
}

// Commit the transaction
if err := tx.Commit(); err != nil {
    log.Printf("Commit failed: %v", err)
    return
}

fmt.Println("Transaction committed successfully!")
```

### Deferred Rollback Pattern
```go
tx, err := db.BeginTransaction()
if err != nil {
    return err
}

// Ensure rollback if commit doesn't happen
committed := false
defer func() {
    if !committed {
        tx.Rollback()
    }
}()

// Perform operations
result := tx.ExecOneSQL("UPDATE products SET stock = stock + 10")
if result.Error != nil {
    return result.Error // defer will rollback
}

// More operations...

// Commit at the end
if err := tx.Commit(); err != nil {
    return err
}
committed = true
return nil
```

### Implicit Transaction (Batch Operations)
```go
// ExecManySQL automatically wraps in a transaction
sqls := []string{
    "UPDATE products SET stock = stock - 1 WHERE id = 1",
    "UPDATE products SET stock = stock - 2 WHERE id = 2",
    "INSERT INTO orders (user_id, total) VALUES (1, 1059.97)",
}

// All succeed or all fail together (automatic Begin/Commit/Rollback)
results, err := db.ExecManySQL(sqls)
if err != nil {
    log.Printf("Transaction failed (auto-rolled back): %v", err)
}
```

## Key Concepts

### DBRecord vs TableStruct

**DBRecord** (map-based):
- ✅ Flexible - works with any table
- ✅ Dynamic field access
- ❌ No compile-time type safety
- ✅ Best for dynamic queries

**TableStruct** (struct-based):
- ✅ Type-safe at compile time
- ✅ IDE autocomplete support
- ✅ Clear domain models
- ❌ Requires struct definition
- ✅ Best for known schemas

### Transaction Guarantees

The library provides ACID transaction guarantees:

- **Atomicity**: All operations succeed or all fail
- **Consistency**: Database constraints are enforced
- **Isolation**: Transactions don't interfere with each other
- **Durability**: Committed changes are permanent

### No UPDATE/DELETE from Structs

By design, the library **does not** provide `UpdateOneDBRecord` or `DeleteOneDBRecord` methods. This is intentional because:

1. **Ambiguity**: It's unclear which fields are WHERE conditions vs SET values
2. **Clarity**: Raw SQL makes the operation explicit
3. **Safety**: Forces developers to think about update/delete logic

**Instead, use:**
```go
// Update with raw SQL
updateSQL := orm.ParametereizedSQL{
    Query:  "UPDATE users SET email = $1 WHERE id = $2",
    Values: []interface{}{"new@email.com", 123},
}
result := db.ExecOneSQLParameterized(updateSQL)

// Delete with raw SQL
deleteSQL := "DELETE FROM users WHERE id = $1"
result := db.ExecOneSQLParameterized(orm.ParametereizedSQL{
    Query:  deleteSQL,
    Values: []interface{}{123},
})
```

## Troubleshooting

### Connection Refused
```
Failed to connect to PostgreSQL: connection refused
```
**Solution**: Ensure PostgreSQL is running:
```bash
# Check status
sudo systemctl status postgresql

# Start PostgreSQL
sudo systemctl start postgresql
```

### Database Does Not Exist
```
Failed to connect to PostgreSQL: database "simpleorm_example" does not exist
```
**Solution**: Create the database:
```bash
createdb -U postgres simpleorm_example
```

### Authentication Failed
```
Failed to connect to PostgreSQL: password authentication failed
```
**Solution**: Check your credentials in `.env` or PostgreSQL configuration.

### SSL Mode Error
```
pq: SSL is not enabled on the server
```
**Solution**: Use `sslmode=disable` in config (already set in example).

## Further Reading

- [SimpleORM Documentation](../README.md)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Go database/sql Tutorial](https://go.dev/doc/database/sql-injection)
- [ACID Transactions](https://en.wikipedia.org/wiki/ACID)

## License

This example is part of the SimpleORM project and follows the same license.
