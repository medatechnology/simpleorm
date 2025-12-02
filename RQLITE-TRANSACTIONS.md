# RQLite Transaction Support

SimpleORM now supports explicit transactions for both PostgreSQL and RQLite! While the API is identical, the underlying implementation differs due to architectural differences between the two databases.

## Transaction API

Both PostgreSQL and RQLite use the same transaction interface:

```go
// Begin a transaction
tx, err := db.BeginTransaction()
if err != nil {
    log.Fatal(err)
}

// Perform operations
tx.InsertOneDBRecord(record)
tx.ExecOneSQL("UPDATE ...")

// Commit or rollback
if err := tx.Commit(); err != nil {
    tx.Rollback()
    log.Fatal(err)
}
```

## PostgreSQL vs RQLite: Architecture Differences

### PostgreSQL (Stateful Transactions)

**Architecture:**
- Persistent TCP connection with `*sql.DB`
- Server-side transaction state
- Operations sent immediately to server
- Server maintains transaction context

**Flow:**
```
BEGIN → INSERT → UPDATE → COMMIT
  ↓       ↓        ↓        ↓
 [Server executes and holds transaction state]
```

**Characteristics:**
- ✅ Operations execute immediately on server
- ✅ SELECT within transaction sees uncommitted changes
- ✅ True isolation from other transactions
- ✅ Can hold locks during transaction

### RQLite (Buffered Transactions)

**Architecture:**
- Stateless HTTP API
- Client-side operation buffering
- All operations sent atomically on Commit
- Uses `/db/request` endpoint with `transaction=true`

**Flow:**
```
BEGIN → INSERT → UPDATE → COMMIT
  ↓       ↓        ↓         ↓
[Buffer] [Buffer] [Buffer] [Send all to /db/request]
```

**Characteristics:**
- ✅ All operations atomic (all-or-nothing)
- ✅ No network overhead until Commit
- ❌ SELECT executes immediately (can't see buffered changes)
- ✅ Rollback is instant (just clears buffer)
- ✅ Better for high-latency networks (single HTTP request)

## Key Differences

### 1. SELECT Behavior

**PostgreSQL:**
```go
tx, _ := db.BeginTransaction()
tx.InsertOneDBRecord(user)

// Sees the uncommitted insert
users, _ := tx.SelectOneSQL("SELECT * FROM users")
// ✅ Includes the newly inserted user
```

**RQLite:**
```go
tx, _ := db.BeginTransaction()
tx.InsertOneDBRecord(user) // Buffered locally

// Executes immediately against current database state
users, _ := tx.SelectOneSQL("SELECT * FROM users")
// ❌ Does NOT include buffered insert (not yet sent to server)
```

**Workaround for RQLite:** Perform SELECTs before or after transactions, not within.

### 2. Rollback Timing

**PostgreSQL:**
```go
tx.ExecOneSQL("UPDATE ...") // Sent to server immediately
tx.Rollback()               // Tells server to rollback
```

**RQLite:**
```go
tx.ExecOneSQL("UPDATE ...") // Buffered locally, not sent yet
tx.Rollback()               // Clears local buffer (instant)
```

### 3. Error Handling

**PostgreSQL:**
```go
result := tx.ExecOneSQL("UPDATE ...")
if result.Error != nil {
    // Error from database server
    tx.Rollback()
}
```

**RQLite:**
```go
result := tx.ExecOneSQL("UPDATE ...")
// result.Error is nil (buffered, not executed yet)

err := tx.Commit()
if err != nil {
    // Error from executing all buffered operations
}
```

### 4. Network Efficiency

**PostgreSQL:**
- N+2 network calls for N operations (BEGIN + N operations + COMMIT)

**RQLite:**
- 1 network call for N operations (all sent in Commit)
- Better for high-latency or unreliable networks

## Best Practices

### When to Use Transactions

**Good Use Cases:**
```go
// ✅ Related writes that must succeed together
tx.ExecOneSQL("UPDATE accounts SET balance = balance - 100 WHERE id = 1")
tx.ExecOneSQL("UPDATE accounts SET balance = balance + 100 WHERE id = 2")
tx.Commit()

// ✅ Insert with related records
tx.InsertOneDBRecord(order)
tx.InsertManyDBRecords(orderItems)
tx.Commit()
```

**Avoid:**
```go
// ❌ SELECT-heavy operations in RQLite
tx.SelectOneSQL("SELECT ...")  // Executes immediately
tx.InsertOneDBRecord(record)   // Buffered
result := tx.SelectOneSQL("SELECT ...") // Won't see buffered insert!
```

### Deferred Rollback Pattern

Use this pattern for safe transaction cleanup:

```go
func transferMoney(db orm.Database, from, to int, amount float64) error {
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

    // Perform operations
    result := tx.ExecOneSQLParameterized(orm.ParametereizedSQL{
        Query:  "UPDATE accounts SET balance = balance - ? WHERE id = ?",
        Values: []interface{}{amount, from},
    })
    if result.Error != nil {
        return result.Error // defer will rollback
    }

    result = tx.ExecOneSQLParameterized(orm.ParametereizedSQL{
        Query:  "UPDATE accounts SET balance = balance + ? WHERE id = ?",
        Values: []interface{}{amount, to},
    })
    if result.Error != nil {
        return result.Error // defer will rollback
    }

    // Commit
    if err := tx.Commit(); err != nil {
        return err
    }

    committed = true
    return nil
}
```

## RQLite-Specific Considerations

### 1. Consistency Levels

RQLite supports different consistency levels. Transactions respect the configured level:

```go
config := rqlite.RqliteDirectConfig{
    URL:         "http://localhost:4001",
    Consistency: "strong", // strong, weak, or none
}
```

- **strong**: Guarantees linearizable reads (slower, most consistent)
- **weak**: Eventually consistent reads (faster)
- **none**: No consistency guarantees (fastest)

For transactions, **use "strong"** to ensure you see the latest data.

### 2. Transaction Endpoint

RQLite transactions use the `/db/request` endpoint with `transaction=true`:

```http
POST /db/request?transaction=true
Content-Type: application/json

[
  "INSERT INTO users (name, email) VALUES (?, ?)",
  ["Alice", "alice@example.com"],
  "UPDATE products SET stock = stock - 1 WHERE id = 1"
]
```

All statements execute atomically on the leader node.

### 3. Cluster Behavior

- Transactions are always executed on the **leader node**
- Writes are replicated to followers via Raft
- Ensure your client connects to the leader or use automatic leader discovery

### 4. Performance Tips

**Batch Operations:**
```go
// ✅ Good: Single transaction for related operations
tx, _ := db.BeginTransaction()
for _, record := range records {
    tx.InsertOneDBRecord(record)
}
tx.Commit() // Single HTTP request with all inserts
```

```go
// ❌ Avoid: Individual commits
for _, record := range records {
    db.InsertOneDBRecord(record, false) // N HTTP requests
}
```

## Examples

### Basic Transaction

```go
tx, _ := db.BeginTransaction()

// Buffer operations
tx.ExecOneSQL("INSERT INTO users (name) VALUES ('Alice')")
tx.ExecOneSQL("UPDATE products SET stock = stock - 1 WHERE id = 1")

// Send all atomically
if err := tx.Commit(); err != nil {
    log.Fatal(err)
}
```

### With Error Handling

```go
tx, err := db.BeginTransaction()
if err != nil {
    return err
}

result := tx.InsertOneDBRecord(record)
if result.Error != nil {
    tx.Rollback()
    return result.Error
}

if err := tx.Commit(); err != nil {
    return err
}
```

### Complex Multi-Step

```go
tx, _ := db.BeginTransaction()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
        panic(r)
    }
}()

// Get data (executes immediately)
user, _ := tx.SelectOnlyOneSQL("SELECT * FROM users WHERE id = 1")
product, _ := tx.SelectOnlyOneSQL("SELECT * FROM products WHERE id = 1")

// Buffer writes
tx.InsertOneDBRecord(order)
tx.ExecOneSQL("UPDATE products SET stock = stock - 1")
tx.ExecOneSQL("UPDATE users SET balance = balance - 100")

// Commit all writes atomically
tx.Commit()
```

## Running the Example

```bash
# Start RQLite
rqlited ~/node.1

# Run the example
go run example/rqlite_transactions_example.go
```

## Comparison Table

| Feature | PostgreSQL | RQLite |
|---------|-----------|--------|
| API | `BeginTransaction()` | `BeginTransaction()` |
| Commit/Rollback | ✅ Same API | ✅ Same API |
| Write buffering | ❌ Sent immediately | ✅ Buffered until Commit |
| SELECT behavior | Sees uncommitted changes | Executes immediately |
| Network calls | N+2 for N operations | 1 for N operations |
| Isolation | Full ACID isolation | Atomic execution |
| Error timing | Immediate | On Commit |
| Best for | Low-latency networks | High-latency networks |
| Server state | Stateful | Stateless |

## Summary

Both PostgreSQL and RQLite support the same transaction API in SimpleORM:

```go
tx, _ := db.BeginTransaction()
// ... operations ...
tx.Commit() // or tx.Rollback()
```

**Choose PostgreSQL transactions when:**
- You need to SELECT within transactions and see uncommitted changes
- You need true database-level isolation
- You're on a low-latency network

**Choose RQLite transactions when:**
- You're primarily doing writes
- Network latency is a concern (fewer round trips)
- You want distributed consensus with Raft

Both provide **ACID guarantees** and **atomic execution** - just with different implementation strategies!
