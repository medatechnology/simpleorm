# 🎉 SimpleORM v0.2.0 Released - Explicit Transaction Support!

**Release Date:** December 2, 2025
**Version:** v0.2.0
**GitHub:** https://github.com/medatechnology/simpleorm/releases/tag/v0.2.0

We're excited to announce SimpleORM v0.2.0, bringing **explicit transaction support** to both PostgreSQL and RQLite with an API similar to sqlx!

## 🚀 What's New

### Explicit Transaction Control

The headline feature: full transaction support with `BeginTransaction()`, `Commit()`, and `Rollback()`!

```go
// Begin a transaction
tx, err := db.BeginTransaction()
if err != nil {
    log.Fatal(err)
}

// Perform operations
tx.InsertOneDBRecord(user)
tx.ExecOneSQL("UPDATE products SET stock = stock - 1")

// Commit or rollback
if err := tx.Commit(); err != nil {
    tx.Rollback()
    log.Fatal(err)
}
```

### Same API, Two Implementations

- **PostgreSQL**: Stateful transactions using `*sql.Tx`
- **RQLite**: Buffered transactions using `/db/request` endpoint

Both provide **ACID guarantees** and **atomic execution**!

## ✨ Key Features

### 1. Deferred Rollback Pattern

```go
func transferMoney(db orm.Database, from, to int, amount float64) error {
    tx, _ := db.BeginTransaction()

    committed := false
    defer func() {
        if !committed {
            tx.Rollback()
        }
    }()

    // Perform operations...
    tx.ExecOneSQL("UPDATE accounts SET balance = balance - ? WHERE id = ?")
    tx.ExecOneSQL("UPDATE accounts SET balance = balance + ? WHERE id = ?")

    if err := tx.Commit(); err != nil {
        return err
    }

    committed = true
    return nil
}
```

### 2. Full CRUD Operations

All database operations available within transactions:

```go
tx.InsertOneDBRecord(record)
tx.InsertManyDBRecordsSameTable(records)
tx.ExecOneSQL(sql)
tx.ExecOneSQLParameterized(paramSQL)
tx.SelectOneSQL(sql)
tx.SelectOnlyOneSQLParameterized(paramSQL)
```

### 3. Network Efficiency (RQLite)

RQLite transactions buffer operations and send them atomically in a single HTTP request:

```go
// 100 operations = 1 HTTP request!
tx, _ := rqliteDB.BeginTransaction()
for i := 0; i < 100; i++ {
    tx.InsertOneDBRecord(records[i])
}
tx.Commit() // Single /db/request with all 100 inserts
```

## 📦 Installation

```bash
go get github.com/medatechnology/simpleorm@v0.2.0
```

## ✅ Backward Compatibility

**100% backward compatible!** No code changes required to upgrade.

All existing code continues to work:

```go
// v0.1.0 code still works in v0.2.0!
results, err := db.ExecManySQL([]string{
    "UPDATE accounts SET balance = balance - 100",
    "UPDATE accounts SET balance = balance + 100",
})
```

## 📚 Documentation

### New Documentation

- **[RQLITE-TRANSACTIONS.md](./RQLITE-TRANSACTIONS.md)** - Comprehensive 300+ line guide
  - Architecture differences
  - Best practices
  - Performance comparison
  - Detailed examples

- **[MIGRATION-v0.2.0.md](./MIGRATION-v0.2.0.md)** - Migration guide from v0.1.0
  - Step-by-step upgrade instructions
  - When to refactor to transactions
  - Common issues and solutions

- **[CHANGELOG.md](./CHANGELOG.md)** - Complete change log

### Updated Documentation

- **README.md** - Added comprehensive transaction section
- **example/README.md** - Updated with transaction examples

### Examples

- `example/transactions.go` - 5 PostgreSQL transaction patterns
- `example/rqlite_transactions_example.go` - 4 RQLite transaction patterns

## 🔍 Architecture Highlights

### PostgreSQL: Stateful Transactions

```go
tx, _ := postgresDB.BeginTransaction()
tx.InsertOneDBRecord(user)    // → Sent to DB immediately
tx.ExecOneSQL("UPDATE ...")   // → Sent to DB immediately
tx.Commit()                    // → COMMIT on server
```

**Benefits:**
- Server maintains transaction state
- SELECTs see uncommitted changes
- True database isolation

### RQLite: Buffered Transactions

```go
tx, _ := rqliteDB.BeginTransaction()
tx.InsertOneDBRecord(user)    // → Buffered locally
tx.ExecOneSQL("UPDATE ...")   // → Buffered locally
tx.Commit()                    // → All sent to /db/request atomically
```

**Benefits:**
- Single HTTP request (network efficient!)
- Better for high-latency networks
- Automatic transaction parameter handling

## 📊 Performance

### PostgreSQL
- Similar to standard `database/sql` transactions
- N+2 network calls for N operations (BEGIN + N ops + COMMIT)

### RQLite
- **Improved:** 1 HTTP call for N operations
- Significantly better for bulk operations
- Ideal for distributed deployments

## 🎯 When to Use Transactions

### ✅ Use Transactions For:

**Related Operations**
```go
// Order creation + inventory update
tx.ExecOneSQL("INSERT INTO orders ...")
tx.ExecOneSQL("UPDATE products SET stock = stock - 1")
tx.Commit()
```

**Money Transfers**
```go
// Deduct + Add must both succeed
tx.ExecOneSQL("UPDATE accounts SET balance = balance - 100 WHERE id = 1")
tx.ExecOneSQL("UPDATE accounts SET balance = balance + 100 WHERE id = 2")
tx.Commit()
```

**Batch Operations**
```go
// Insert multiple related records atomically
tx.InsertOneDBRecord(order)
tx.InsertManyDBRecords(orderItems)
tx.Commit()
```

### ❌ Don't Use Transactions For:

- Single, independent operations
- Read-only queries
- Operations where partial success is acceptable

## 🛠️ Implementation Details

### Files Added

- `orm.go` - Added `Transaction` interface and `BeginTransaction()` method
- `postgres/transaction.go` - PostgreSQL implementation (370 lines)
- `rqlite/transaction.go` - RQLite implementation (360 lines)
- `rqlite/helper.go` - Added `execRequestUnified()` for `/db/request` endpoint

### Code Statistics

- **Total Lines Added:** ~1,500 lines
- **Documentation:** ~800 lines
- **Examples:** ~400 lines
- **Tests:** Ready for your contributions!

## 🔄 Migration Path

### Quick Upgrade

```bash
# Update dependency
go get github.com/medatechnology/simpleorm@v0.2.0
go mod tidy

# Run tests
go test ./...

# Done! (optional: adopt transactions where beneficial)
```

### Gradual Adoption

You can adopt transactions gradually:

```go
// Keep using ExecManySQL for simple batches
db.ExecManySQL(sqls)

// Use transactions for complex workflows
tx, _ := db.BeginTransaction()
// ... complex operations ...
tx.Commit()
```

## 💡 Best Practices

### 1. Always Use Deferred Rollback

```go
tx, _ := db.BeginTransaction()

committed := false
defer func() {
    if !committed {
        tx.Rollback()
    }
}()

// Operations...

tx.Commit()
committed = true
```

### 2. Handle Commit Errors

```go
if err := tx.Commit(); err != nil {
    // Transaction failed, already rolled back
    return fmt.Errorf("transaction failed: %w", err)
}
```

### 3. Keep Transactions Short

```go
// ✅ Good: Short, focused transaction
tx, _ := db.BeginTransaction()
tx.ExecOneSQL("UPDATE ...")
tx.ExecOneSQL("UPDATE ...")
tx.Commit()

// ❌ Avoid: Long-running operations in transaction
tx, _ := db.BeginTransaction()
processLargeFile() // Don't do this!
tx.ExecOneSQL("UPDATE ...")
tx.Commit()
```

## 🌟 Community

### Contributing

We welcome contributions! Areas we'd love help with:

- Additional database implementations (MySQL, SQLite, etc.)
- Performance benchmarks
- More examples and use cases
- Documentation improvements

### Feedback

Found a bug or have a feature request?
- **Issues:** https://github.com/medatechnology/simpleorm/issues
- **Discussions:** https://github.com/medatechnology/simpleorm/discussions

## 📅 What's Next?

### Planned for v0.3.0

- Prepared statements support
- Connection pooling improvements
- Query builder enhancements
- More database adapters

Stay tuned!

## 🙏 Acknowledgments

Thanks to everyone who contributed ideas, reported issues, and tested early versions!

## 📝 Complete Changelog

See [CHANGELOG.md](./CHANGELOG.md) for the complete list of changes.

## 🔗 Links

- **Repository:** https://github.com/medatechnology/simpleorm
- **Documentation:** See README.md
- **Transaction Guide:** See RQLITE-TRANSACTIONS.md
- **Migration Guide:** See MIGRATION-v0.2.0.md
- **Examples:** See `example/` directory

---

## Quick Start

### PostgreSQL

```go
import (
    orm "github.com/medatechnology/simpleorm"
    "github.com/medatechnology/simpleorm/postgres"
)

config := postgres.NewConfig("localhost", 5432, "user", "pass", "dbname")
db, _ := postgres.NewDatabase(*config)

tx, _ := db.BeginTransaction()
tx.InsertOneDBRecord(record)
tx.Commit()
```

### RQLite

```go
import (
    orm "github.com/medatechnology/simpleorm"
    "github.com/medatechnology/simpleorm/rqlite"
)

config := rqlite.RqliteDirectConfig{
    URL: "http://localhost:4001",
    Consistency: "strong",
}
db, _ := rqlite.NewDatabase(config)

tx, _ := db.BeginTransaction()
tx.InsertOneDBRecord(record)
tx.Commit()
```

---

**Happy coding with SimpleORM v0.2.0!** 🎉

*Released with ❤️ by the SimpleORM team*
