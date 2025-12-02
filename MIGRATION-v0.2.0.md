# Migration Guide: v0.1.0 → v0.2.0

This guide helps you upgrade from SimpleORM v0.1.0 to v0.2.0, which adds explicit transaction support.

## TL;DR

**Good news!** v0.2.0 is **fully backward compatible**. No code changes are required to upgrade.

```bash
# Update dependency
go get github.com/medatechnology/simpleorm@v0.2.0
go mod tidy

# That's it! Your code still works.
```

## What's New in v0.2.0

### Explicit Transaction Support

The major feature in v0.2.0 is explicit transaction control:

```go
// New in v0.2.0
tx, err := db.BeginTransaction()
tx.InsertOneDBRecord(record)
tx.ExecOneSQL("UPDATE ...")
tx.Commit() // or tx.Rollback()
```

### Supported Databases

- ✅ PostgreSQL (stateful transactions)
- ✅ RQLite (buffered transactions)

## Breaking Changes

**None!** v0.2.0 is 100% backward compatible with v0.1.0.

## API Additions

### New Interface: `Transaction`

```go
type Transaction interface {
    Commit() error
    Rollback() error

    // All CRUD operations available
    ExecOneSQL(string) BasicSQLResult
    InsertOneDBRecord(DBRecord) BasicSQLResult
    SelectOneSQL(string) (DBRecords, error)
    // ... and more
}
```

### New Method on `Database` Interface

```go
type Database interface {
    // ... existing methods ...

    // New in v0.2.0
    BeginTransaction() (Transaction, error)
}
```

## Migration Steps

### Step 1: Update Dependency

```bash
go get github.com/medatechnology/simpleorm@v0.2.0
go mod tidy
```

### Step 2: Test Your Existing Code

Run your tests to ensure everything still works:

```bash
go test ./...
```

Your existing code should work without any changes!

### Step 3: (Optional) Adopt Transactions

You can optionally refactor code to use explicit transactions where beneficial.

## Code Migration Examples

### Before: Implicit Transactions (v0.1.0)

```go
// Old way - still works in v0.2.0!
results, err := db.ExecManySQL([]string{
    "UPDATE accounts SET balance = balance - 100 WHERE id = 1",
    "UPDATE accounts SET balance = balance + 100 WHERE id = 2",
})
if err != nil {
    // Transaction automatically rolled back
    log.Printf("Transfer failed: %v", err)
}
```

### After: Explicit Transactions (v0.2.0)

```go
// New way - more control
tx, err := db.BeginTransaction()
if err != nil {
    return err
}

committed := false
defer func() {
    if !committed {
        tx.Rollback()
    }
}()

result := tx.ExecOneSQL("UPDATE accounts SET balance = balance - 100 WHERE id = 1")
if result.Error != nil {
    return result.Error // defer will rollback
}

result = tx.ExecOneSQL("UPDATE accounts SET balance = balance + 100 WHERE id = 2")
if result.Error != nil {
    return result.Error // defer will rollback
}

if err := tx.Commit(); err != nil {
    return err
}

committed = true
```

**Both approaches work!** Choose based on your needs:
- **Implicit**: Simple, good for basic batching
- **Explicit**: More control, better for complex workflows

## When to Refactor to Transactions

### ✅ Good Candidates for Refactoring

**1. Related Operations That Must Succeed Together**

Before:
```go
db.ExecManySQL([]string{
    "INSERT INTO orders (user_id, total) VALUES (1, 100)",
    "INSERT INTO order_items (order_id, product_id) VALUES (1, 5)",
    "UPDATE products SET stock = stock - 1 WHERE id = 5",
})
```

After:
```go
tx, _ := db.BeginTransaction()
defer func() {
    if !committed {
        tx.Rollback()
    }
}()

tx.ExecOneSQL("INSERT INTO orders (user_id, total) VALUES (1, 100)")
tx.ExecOneSQL("INSERT INTO order_items (order_id, product_id) VALUES (1, 5)")
tx.ExecOneSQL("UPDATE products SET stock = stock - 1 WHERE id = 5")
tx.Commit()
```

**2. Operations Requiring SELECTs Within Transaction**

Before:
```go
// Can't easily read within transaction
user, _ := db.SelectOnlyOneSQL("SELECT * FROM users WHERE id = 1")
// Separate operation
db.ExecOneSQL("UPDATE users SET balance = ... WHERE id = 1")
```

After (PostgreSQL):
```go
tx, _ := db.BeginTransaction()
user, _ := tx.SelectOnlyOneSQL("SELECT * FROM users WHERE id = 1")
balance := user.Data["balance"].(float64)
// Use balance in UPDATE
tx.ExecOneSQL(fmt.Sprintf("UPDATE users SET balance = %f WHERE id = 1", balance+100))
tx.Commit()
```

**3. Error Recovery Scenarios**

Before:
```go
results, err := db.ExecManySQL(sqls)
// Hard to know which operation failed
```

After:
```go
tx, _ := db.BeginTransaction()
for i, sql := range sqls {
    result := tx.ExecOneSQL(sql)
    if result.Error != nil {
        tx.Rollback()
        return fmt.Errorf("operation %d failed: %w", i, result.Error)
    }
}
tx.Commit()
```

### ❌ Don't Refactor These

**1. Single Operations**
```go
// Don't wrap single operations
// Before (good):
db.InsertOneDBRecord(record, false)

// After (unnecessary):
tx, _ := db.BeginTransaction()
tx.InsertOneDBRecord(record)
tx.Commit() // Overkill!
```

**2. Read-Only Queries**
```go
// Don't use transactions for pure reads
// Good:
users, _ := db.SelectMany("users")

// Unnecessary:
tx, _ := db.BeginTransaction()
users, _ := tx.SelectOneSQL("SELECT * FROM users")
tx.Commit() // Why?
```

**3. Independent Operations**
```go
// These don't need to be atomic
db.InsertOneDBRecord(log1, false)
db.InsertOneDBRecord(log2, false)
db.InsertOneDBRecord(log3, false)
```

## Database-Specific Considerations

### PostgreSQL

PostgreSQL transactions work like traditional RDBMS transactions:

```go
tx, _ := postgresDB.BeginTransaction()
tx.InsertOneDBRecord(user)        // Sent to DB immediately
result, _ := tx.SelectOneSQL(...) // Sees uncommitted changes
tx.Commit()                        // Commits on server
```

**Benefits:**
- See uncommitted changes within transaction
- True database isolation
- Standard ACID semantics

### RQLite

RQLite uses buffered transactions for network efficiency:

```go
tx, _ := rqliteDB.BeginTransaction()
tx.InsertOneDBRecord(user)        // Buffered locally
result, _ := tx.SelectOneSQL(...) // Executes immediately (won't see buffered insert!)
tx.Commit()                        // Sends all buffered ops to /db/request
```

**Benefits:**
- Single HTTP request for all operations
- Better for high-latency networks
- Still provides atomic execution

**Limitation:**
- SELECT within transaction won't see buffered changes
- **Workaround:** Do SELECTs before or after transaction

## Testing Your Migration

### 1. Unit Tests

Ensure your existing tests pass:

```bash
go test ./...
```

### 2. Integration Tests

If you have integration tests that rely on transaction behavior:

```go
func TestTransaction(t *testing.T) {
    tx, err := db.BeginTransaction()
    if err != nil {
        t.Fatal(err)
    }

    // Test operations
    result := tx.InsertOneDBRecord(testRecord)
    if result.Error != nil {
        t.Fatal(result.Error)
    }

    // Verify rollback
    tx.Rollback()

    // Verify record was not inserted
    _, err = db.SelectOnlyOneSQL("SELECT * FROM test WHERE id = 1")
    if err != orm.ErrSQLNoRows {
        t.Fatal("Record should not exist after rollback")
    }
}
```

### 3. Performance Testing

Benchmark transaction performance if needed:

```go
func BenchmarkExplicitTransaction(b *testing.B) {
    for i := 0; i < b.N; i++ {
        tx, _ := db.BeginTransaction()
        tx.InsertOneDBRecord(record)
        tx.Commit()
    }
}

func BenchmarkImplicitTransaction(b *testing.B) {
    for i := 0; i < b.N; i++ {
        db.InsertOneDBRecord(record, false)
    }
}
```

## Common Issues and Solutions

### Issue 1: RQLite SELECT Not Seeing Buffered Changes

**Problem:**
```go
tx, _ := rqliteDB.BeginTransaction()
tx.InsertOneDBRecord(user)
users, _ := tx.SelectOneSQL("SELECT * FROM users") // Doesn't see buffered insert
tx.Commit()
```

**Solution:**
Do SELECTs before transaction or after Commit:
```go
// Option 1: Before
users, _ := db.SelectOneSQL("SELECT * FROM users")

tx, _ := db.BeginTransaction()
tx.InsertOneDBRecord(newUser)
tx.Commit()

// Option 2: After
tx, _ := db.BeginTransaction()
tx.InsertOneDBRecord(newUser)
tx.Commit()

users, _ := db.SelectOneSQL("SELECT * FROM users") // Now sees new user
```

### Issue 2: Forgetting to Call Commit

**Problem:**
```go
tx, _ := db.BeginTransaction()
tx.InsertOneDBRecord(user)
// Forgot tx.Commit()!
```

**Solution:**
Use deferred rollback pattern:
```go
tx, _ := db.BeginTransaction()

committed := false
defer func() {
    if !committed {
        tx.Rollback()
        log.Println("Transaction rolled back (not committed)")
    }
}()

tx.InsertOneDBRecord(user)

// Must set committed = true after successful Commit
if err := tx.Commit(); err != nil {
    return err
}
committed = true
```

### Issue 3: Error Handling Differences

**Problem:**
With `ExecManySQL`, errors happen immediately. With buffered transactions (RQLite), errors happen on Commit.

**Solution:**
Always check Commit errors:
```go
// RQLite
tx, _ := rqliteDB.BeginTransaction()
tx.ExecOneSQL("UPDATE ...") // result.Error is nil (buffered)

// Error happens here!
if err := tx.Commit(); err != nil {
    log.Printf("Transaction failed: %v", err)
}
```

## FAQ

### Q: Do I need to change my code?

**A:** No! v0.2.0 is fully backward compatible. Transactions are opt-in.

### Q: Should I switch to explicit transactions?

**A:** Only if you need:
- Fine-grained commit/rollback control
- Deferred rollback patterns
- Reading data within transaction context (PostgreSQL only)

Otherwise, `ExecManySQL` and `ExecManySQLParameterized` still work great!

### Q: What's the performance difference?

**A:**
- **PostgreSQL**: Similar performance (both use `*sql.Tx` internally)
- **RQLite**: Explicit transactions may be *faster* (fewer HTTP calls)

### Q: Can I mix old and new styles?

**A:** Yes! Use transactions where beneficial, keep existing code elsewhere.

```go
// Old style for simple operations
db.InsertOneDBRecord(log, false)

// New style for complex workflows
tx, _ := db.BeginTransaction()
tx.InsertOneDBRecord(order)
tx.InsertManyDBRecords(items)
tx.Commit()
```

### Q: Do transactions work the same for PostgreSQL and RQLite?

**A:** The API is identical, but the implementation differs:
- **PostgreSQL**: Stateful (server-side state)
- **RQLite**: Buffered (client-side buffering)

See [RQLITE-TRANSACTIONS.md](./RQLITE-TRANSACTIONS.md) for details.

## Support

If you encounter issues during migration:

1. Check [CHANGELOG.md](./CHANGELOG.md) for all changes
2. Review [RQLITE-TRANSACTIONS.md](./RQLITE-TRANSACTIONS.md) for transaction details
3. See examples in `example/` directory
4. Open an issue on GitHub

## Summary

```bash
# Migration Steps
✅ Update dependency: go get github.com/medatechnology/simpleorm@v0.2.0
✅ Run tests: go test ./...
✅ (Optional) Refactor to transactions where beneficial
✅ Review RQLITE-TRANSACTIONS.md for best practices

# No breaking changes!
# Fully backward compatible!
# Transactions are opt-in!
```

Happy upgrading! 🎉
