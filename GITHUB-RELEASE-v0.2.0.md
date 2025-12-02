# SimpleORM v0.2.0 - Explicit Transaction Support 🎉

We're excited to announce SimpleORM v0.2.0, bringing **explicit transaction support** to both PostgreSQL and RQLite!

## 🚀 Highlights

### Explicit Transactions (Just like sqlx!)

```go
// Begin a transaction
tx, err := db.BeginTransaction()

// Perform operations
tx.InsertOneDBRecord(user)
tx.ExecOneSQL("UPDATE products SET stock = stock - 1")

// Commit or rollback
if err := tx.Commit(); err != nil {
    tx.Rollback()
}
```

### ✨ Key Features

- ✅ **Same API for PostgreSQL and RQLite**
- ✅ **Full CRUD operations within transactions**
- ✅ **Deferred rollback pattern support**
- ✅ **ACID guarantees for both databases**
- ✅ **Network-efficient (RQLite uses single HTTP call)**
- ✅ **100% backward compatible**

## 📦 Installation

```bash
go get github.com/medatechnology/simpleorm@v0.2.0
```

## 🎯 What's New

### Transaction Interface

```go
type Transaction interface {
    Commit() error
    Rollback() error

    // All CRUD operations available
    InsertOneDBRecord(DBRecord) BasicSQLResult
    ExecOneSQL(string) BasicSQLResult
    SelectOneSQL(string) (DBRecords, error)
    // ... and more
}
```

### Database Implementations

#### PostgreSQL (Stateful)
- Operations execute immediately on server
- Server maintains transaction state
- SELECTs see uncommitted changes
- Traditional RDBMS behavior

#### RQLite (Buffered)
- Operations buffered locally
- All sent atomically on Commit via `/db/request`
- Single HTTP request for all operations
- Network-efficient for distributed deployments

## 📚 Documentation

### New Documentation
- **[RQLITE-TRANSACTIONS.md](https://github.com/medatechnology/simpleorm/blob/main/RQLITE-TRANSACTIONS.md)** - Comprehensive transaction guide
- **[MIGRATION-v0.2.0.md](https://github.com/medatechnology/simpleorm/blob/main/MIGRATION-v0.2.0.md)** - Migration guide from v0.1.0
- **[CHANGELOG.md](https://github.com/medatechnology/simpleorm/blob/main/CHANGELOG.md)** - Complete changelog

### Examples
- `example/transactions.go` - 5 PostgreSQL transaction patterns
- `example/rqlite_transactions_example.go` - 4 RQLite transaction examples

## 💡 Quick Examples

### Deferred Rollback Pattern

```go
func transferMoney(db orm.Database, from, to int, amount float64) error {
    tx, _ := db.BeginTransaction()

    committed := false
    defer func() {
        if !committed {
            tx.Rollback()
        }
    }()

    tx.ExecOneSQLParameterized(orm.ParametereizedSQL{
        Query:  "UPDATE accounts SET balance = balance - $1 WHERE id = $2",
        Values: []interface{}{amount, from},
    })

    tx.ExecOneSQLParameterized(orm.ParametereizedSQL{
        Query:  "UPDATE accounts SET balance = balance + $1 WHERE id = $2",
        Values: []interface{}{amount, to},
    })

    if err := tx.Commit(); err != nil {
        return err
    }

    committed = true
    return nil
}
```

### Complex Transaction

```go
tx, _ := db.BeginTransaction()

// Create order
tx.ExecOneSQL("INSERT INTO orders (user_id, total) VALUES (1, 100)")

// Add order items
tx.InsertManyDBRecordsSameTable(orderItems)

// Update inventory
tx.ExecOneSQL("UPDATE products SET stock = stock - 1 WHERE id = 1")

// All atomic!
tx.Commit()
```

## ✅ Backward Compatibility

**No breaking changes!** All existing code continues to work.

```go
// v0.1.0 code still works
results, err := db.ExecManySQL(sqls)

// v0.2.0 - new transaction API (optional)
tx, _ := db.BeginTransaction()
// ...
tx.Commit()
```

## 🔄 Migration

### Quick Upgrade

```bash
go get github.com/medatechnology/simpleorm@v0.2.0
go mod tidy
go test ./...  # All tests should pass!
```

### No Code Changes Required

Your existing code works as-is. Adopt transactions when beneficial:

**Before (still works):**
```go
db.ExecManySQL([]string{
    "UPDATE accounts SET balance = balance - 100 WHERE id = 1",
    "UPDATE accounts SET balance = balance + 100 WHERE id = 2",
})
```

**After (new option):**
```go
tx, _ := db.BeginTransaction()
tx.ExecOneSQL("UPDATE accounts SET balance = balance - 100 WHERE id = 1")
tx.ExecOneSQL("UPDATE accounts SET balance = balance + 100 WHERE id = 2")
tx.Commit()
```

Both work! Use transactions when you need:
- Fine-grained commit/rollback control
- Deferred rollback patterns
- Reading data within transaction context

## 📊 What Changed

### Added

- `Transaction` interface in `orm.go`
- `BeginTransaction()` method on `Database` interface
- PostgreSQL transaction implementation (`postgres/transaction.go`)
- RQLite transaction implementation (`rqlite/transaction.go`)
- Unified request endpoint for RQLite (`/db/request`)
- Comprehensive documentation and examples

### Changed

- Updated `Database` interface (backward compatible)
- Enhanced example project with transaction demonstrations

### Fixed

- Minor code cleanup in examples

## 📈 Performance

### PostgreSQL
- Standard `database/sql` transaction performance
- N+2 network calls for N operations

### RQLite
- **Improved:** 1 HTTP call for N operations
- Better for high-latency networks
- Ideal for distributed deployments

## 🎯 When to Use Transactions

### ✅ Good Use Cases

- **Related writes:** Order + inventory updates
- **Money transfers:** Deduct + add must both succeed
- **Batch operations:** Multiple inserts that must be atomic
- **Data consistency:** Operations spanning multiple tables

### ❌ Don't Need Transactions For

- Single, independent operations
- Read-only queries
- Operations where partial success is OK

## 🔗 Links

- **Full Changelog:** [CHANGELOG.md](https://github.com/medatechnology/simpleorm/blob/main/CHANGELOG.md)
- **Transaction Guide:** [RQLITE-TRANSACTIONS.md](https://github.com/medatechnology/simpleorm/blob/main/RQLITE-TRANSACTIONS.md)
- **Migration Guide:** [MIGRATION-v0.2.0.md](https://github.com/medatechnology/simpleorm/blob/main/MIGRATION-v0.2.0.md)
- **Examples:** [example/](https://github.com/medatechnology/simpleorm/tree/main/example)

## 🙏 Feedback

Found an issue or have a suggestion?
- **Report Issues:** https://github.com/medatechnology/simpleorm/issues
- **Start Discussion:** https://github.com/medatechnology/simpleorm/discussions

---

**Happy coding with SimpleORM v0.2.0!** 🎉
