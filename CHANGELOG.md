# Changelog

All notable changes to SimpleORM will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2025-12-02

### Added - Transaction Support 🎉

#### Core Features
- **Explicit Transaction Interface**: Added `Transaction` interface in `orm.go` with full CRUD operations
- **BeginTransaction() Method**: Start transactions with explicit control over commit/rollback
- **PostgreSQL Transactions**: Full stateful transaction support using `*sql.Tx`
  - Immediate execution on server
  - Server-side transaction state
  - True ACID isolation
  - See uncommitted changes within transaction
- **RQLite Transactions**: Buffered transaction support using `/db/request` endpoint
  - Client-side operation buffering
  - Atomic execution via single HTTP request
  - Network-efficient (single round trip)
  - Automatic transaction parameter handling

#### Implementation Files
- `orm.go`: Added `Transaction` interface and `BeginTransaction()` to `Database` interface
- `postgres/transaction.go`: PostgreSQL transaction implementation (370 lines)
- `rqlite/transaction.go`: RQLite transaction implementation (360 lines)
- `rqlite/helper.go`: Added `execRequestUnified()` method for unified request endpoint

#### Examples
- `example/transactions.go`: 5 comprehensive PostgreSQL transaction examples
  - Explicit transaction with commit
  - Rollback on error with verification
  - Complex multi-step order transaction
  - Deferred rollback pattern
  - Implicit batch operations
- `example/rqlite_transactions_example.go`: 4 RQLite transaction examples
  - Basic transaction with commit
  - Transaction with rollback
  - Complex multi-step transaction
  - Deferred rollback pattern

#### Documentation
- `RQLITE-TRANSACTIONS.md`: Comprehensive 300+ line guide covering:
  - Architecture differences between PostgreSQL and RQLite
  - Transaction API usage
  - Best practices and patterns
  - Performance considerations
  - Detailed comparison table
- `example/README.md`: Updated with transaction documentation
  - Explicit transaction control examples
  - Deferred rollback pattern
  - Code snippets and usage guide

#### Transaction Features
- ✅ Same API for both PostgreSQL and RQLite
- ✅ `Commit()` and `Rollback()` methods
- ✅ All CRUD operations within transactions
- ✅ Deferred rollback pattern support
- ✅ Error handling with automatic rollback
- ✅ Parameterized queries in transactions
- ✅ Batch operations support

### Changed
- Updated `Database` interface to include `BeginTransaction() (Transaction, error)`
- Enhanced `example/main.go` with transaction examples integration

### Fixed
- Fixed unused variable `batchResults` in `example/main.go`

### Documentation
- Added comprehensive transaction documentation
- Updated README with transaction usage patterns
- Created architecture comparison docs
- Added migration guide from v0.1.0

### Compatibility
- ✅ **Fully backward compatible** - no breaking changes
- ✅ All existing code continues to work
- ✅ New transaction API is opt-in
- ✅ Works with Go 1.21+

### Performance
- **PostgreSQL**: Standard database/sql transaction performance
- **RQLite**: Improved network efficiency (1 HTTP call vs N+2 for N operations)

---

## [0.1.0] - 2024-XX-XX

### Added
- Structured CTE (Common Table Expressions) support
- Initial PostgreSQL implementation
- RQLite implementation
- Basic CRUD operations
- Condition-based querying
- Batch insert operations
- Schema management
- Status and health check methods

### Features
- Database-agnostic interface
- Support for DBRecord (map-based) and TableStruct (struct-based) approaches
- Complex query support with JOINs
- Parameterized SQL queries
- Connection pooling for PostgreSQL
- HTTP-based RQLite client

---

## [0.0.4] - 2024-XX-XX

### Added
- Enhanced error handling
- Additional helper methods
- Performance improvements

---

## [0.0.3] - 2024-XX-XX

### Added
- Initial RQLite support
- Basic database operations

---

## [0.0.2] - 2024-XX-XX

### Added
- Core interface improvements
- Additional query methods

---

## [0.0.1] - 2024-XX-XX

### Added
- Initial release
- Basic ORM functionality
- Database interface definition

---

## Upgrade Guides

### Upgrading from v0.1.0 to v0.2.0

No breaking changes! Simply update your dependency:

```bash
go get github.com/medatechnology/simpleorm@v0.2.0
```

**Optional**: Start using transactions for atomic operations:

```go
// Before (v0.1.0) - implicit transaction
results, err := db.ExecManySQL([]string{
    "UPDATE accounts SET balance = balance - 100 WHERE id = 1",
    "UPDATE accounts SET balance = balance + 100 WHERE id = 2",
})

// After (v0.2.0) - explicit transaction control
tx, _ := db.BeginTransaction()
tx.ExecOneSQL("UPDATE accounts SET balance = balance - 100 WHERE id = 1")
tx.ExecOneSQL("UPDATE accounts SET balance = balance + 100 WHERE id = 2")
if err := tx.Commit(); err != nil {
    tx.Rollback()
}
```

Both approaches still work! Use explicit transactions when you need:
- Fine-grained control over commit/rollback
- Deferred rollback patterns
- Reading data within the transaction context

---

## Links

- **Repository**: https://github.com/medatechnology/simpleorm
- **Issues**: https://github.com/medatechnology/simpleorm/issues
- **Examples**: See `example/` directory
- **Documentation**: See README.md and RQLITE-TRANSACTIONS.md

---

## Version History

- [0.2.0] - Transaction Support (Current)
- [0.1.0] - CTE Support and Initial PostgreSQL
- [0.0.4] - Error Handling Improvements
- [0.0.3] - Initial RQLite Support
- [0.0.2] - Core Interface Improvements
- [0.0.1] - Initial Release
