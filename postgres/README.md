# PostgreSQL Implementation for SimpleORM

A production-ready PostgreSQL database adapter for SimpleORM with advanced configuration, connection pooling, comprehensive error handling, and optimized batch operations.

## Features

- ✅ Full `orm.Database` interface implementation
- ✅ Advanced configuration with connection pooling
- ✅ Comprehensive PostgreSQL-specific error handling
- ✅ Optimized batch INSERT operations
- ✅ Type-safe parameter conversion ($1, $2, $3...)
- ✅ DSN parsing and generation (URL and key=value formats)
- ✅ Connection validation and health checks
- ✅ Real-time database statistics and metrics
- ✅ Fluent configuration API with method chaining
- ✅ Extensive test coverage

## Installation

```bash
go get github.com/medatechnology/simpleorm
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/medatechnology/simpleorm/postgres"
    orm "github.com/medatechnology/simpleorm"
)

func main() {
    // Create configuration
    config := postgres.NewConfig(
        "localhost", // host
        5432,        // port
        "postgres",  // user
        "password",  // password
        "mydb",      // database name
    )

    // Customize configuration
    config.WithSSLMode("disable").
        WithConnectionPool(25, 5, 0, 0).
        WithApplicationName("my-app")

    // Connect to database
    db, err := postgres.NewDatabase(*config)
    if err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer db.Close()

    // Insert a record
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
        log.Fatalf("Insert failed: %v", result.Error)
    }

    // Query records
    users, err := db.SelectMany("users")
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }

    for _, user := range users {
        log.Printf("User: %+v", user.Data)
    }
}
```

## Configuration

### Basic Configuration

```go
// Method 1: Using NewConfig
config := postgres.NewConfig("localhost", 5432, "user", "password", "dbname")

// Method 2: Using NewDefaultConfig with method chaining
config := postgres.NewDefaultConfig().
    WithSSLMode("require").
    WithConnectionPool(50, 10, 10*time.Minute, 5*time.Minute).
    WithTimeouts(20*time.Second, 60*time.Second).
    WithApplicationName("my-service")

// Method 3: Manual configuration
config := &postgres.PostgresConfig{
    Host:            "db.example.com",
    Port:            5432,
    User:            "app_user",
    Password:        "secure_pass",
    DBName:          "production_db",
    SSLMode:         "verify-full",
    MaxOpenConns:    100,
    MaxIdleConns:    20,
    ConnMaxLifetime: 30 * time.Minute,
    ConnMaxIdleTime: 10 * time.Minute,
    ConnectTimeout:  15 * time.Second,
    QueryTimeout:    45 * time.Second,
    ApplicationName: "production-api",
    SearchPath:      "app_schema,public",
    Timezone:        "UTC",
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Host` | string | "localhost" | Database host |
| `Port` | int | 5432 | Database port |
| `User` | string | required | Database user |
| `Password` | string | required | User password |
| `DBName` | string | required | Database name |
| `SSLMode` | string | "disable" | SSL mode: disable, require, verify-ca, verify-full |
| `MaxOpenConns` | int | 25 | Maximum open connections |
| `MaxIdleConns` | int | 5 | Maximum idle connections |
| `ConnMaxLifetime` | duration | 5m | Maximum connection lifetime |
| `ConnMaxIdleTime` | duration | 10m | Maximum connection idle time |
| `ConnectTimeout` | duration | 10s | Connection timeout |
| `QueryTimeout` | duration | 30s | Query execution timeout |
| `ApplicationName` | string | "simpleorm" | Application identifier |
| `SearchPath` | string | "" | Schema search path |
| `Timezone` | string | "" | Timezone setting |
| `ExtraParams` | map[string]string | {} | Additional parameters |

### SSL Modes

- `disable` - No SSL (default)
- `require` - Require SSL, but don't verify server certificate
- `verify-ca` - Require SSL and verify that server certificate is signed by a trusted CA
- `verify-full` - Require SSL and verify that server certificate matches the server hostname

### Connection Pool Tuning

Different workload types benefit from different connection pool settings:

```go
// High-throughput API server
apiConfig := postgres.NewDefaultConfig().
    WithConnectionPool(100, 25, 15*time.Minute, 5*time.Minute)

// Background job processor
jobConfig := postgres.NewDefaultConfig().
    WithConnectionPool(10, 2, 30*time.Minute, 10*time.Minute)

// Analytics/reporting
analyticsConfig := postgres.NewDefaultConfig().
    WithConnectionPool(5, 1, 1*time.Hour, 30*time.Minute)
```

### DSN Generation and Parsing

```go
// Generate URL format DSN
dsn, _ := config.ToDSN()
// postgres://user:pass@localhost:5432/dbname?sslmode=disable

// Generate simple format DSN
dsn, _ := config.ToSimpleDSN()
// host=localhost port=5432 user=user password=pass dbname=dbname

// Parse existing DSN
config, _ := postgres.ParseDSN("postgres://user:pass@host:5432/db")
```

## Basic Operations

### Insert Operations

```go
// Insert single record
record := orm.DBRecord{
    TableName: "users",
    Data: map[string]interface{}{
        "name": "Alice",
        "age":  25,
    },
}
result := db.InsertOneDBRecord(record, false)

// Batch insert (optimized multi-row INSERT)
records := []orm.DBRecord{
    {TableName: "users", Data: map[string]interface{}{"name": "Bob", "age": 30}},
    {TableName: "users", Data: map[string]interface{}{"name": "Carol", "age": 35}},
}
results, err := db.InsertManyDBRecordsSameTable(records, false)

// Insert from struct
type User struct {
    Name  string `db:"name"`
    Email string `db:"email"`
}

user := User{Name: "Dave", Email: "dave@example.com"}
result := db.InsertOneTableStruct(user, false)
```

### Select Operations

```go
// Select one record
record, err := db.SelectOne("users")

// Select all records
records, err := db.SelectMany("users")

// Select with simple condition
condition := &orm.Condition{
    Field:    "age",
    Operator: ">",
    Value:    25,
}
records, err := db.SelectManyWithCondition("users", condition)

// Select with complex nested conditions
condition := &orm.Condition{
    Logic: "OR",
    Nested: []orm.Condition{
        {
            Logic: "AND",
            Nested: []orm.Condition{
                {Field: "age", Operator: ">", Value: 30},
                {Field: "status", Operator: "=", Value: "active"},
            },
        },
        {Field: "role", Operator: "=", Value: "admin"},
    },
}
records, err := db.SelectManyWithCondition("users", condition)
```

### Raw SQL Queries

```go
// Parameterized query
sql := orm.ParametereizedSQL{
    Query:  "SELECT * FROM users WHERE age >= $1 AND status = $2",
    Values: []interface{}{25, "active"},
}
records, err := db.SelectOnlyOneSQLParameterized(sql)

// Multiple queries
sqls := []orm.ParametereizedSQL{
    {Query: "SELECT * FROM users WHERE id = $1", Values: []interface{}{1}},
    {Query: "SELECT * FROM users WHERE id = $1", Values: []interface{}{2}},
}
results, err := db.SelectManySQLParameterized(sqls)
```

## Error Handling

The PostgreSQL adapter provides comprehensive error detection and handling:

### Error Detection Functions

```go
// Constraint violations
if postgres.IsUniqueViolation(err) {
    // Handle duplicate key error
}
if postgres.IsForeignKeyViolation(err) {
    // Handle foreign key constraint error
}
if postgres.IsNotNullViolation(err) {
    // Handle NOT NULL constraint error
}
if postgres.IsConstraintViolation(err) {
    // Handle any constraint violation
}

// Table/column errors
if postgres.IsUndefinedTable(err) {
    // Table doesn't exist
}
if postgres.IsUndefinedColumn(err) {
    // Column doesn't exist
}

// Connection errors
if postgres.IsConnectionError(err) {
    // Connection failed or lost
}

// Transaction errors
if postgres.IsDeadlock(err) {
    // Deadlock detected - retry transaction
}
if postgres.IsSerializationFailure(err) {
    // Serialization failure - retry transaction
}

// General retry check
if postgres.IsRetryable(err) {
    // Error is transient and can be retried
}
```

### Error Details

```go
// Get PostgreSQL error code
code := postgres.GetPostgreSQLErrorCode(err)

// Get detailed error information
code, message, detail, hint := postgres.GetPostgreSQLErrorDetail(err)
fmt.Printf("Error %s: %s\n", code, message)
if detail != "" {
    fmt.Printf("Detail: %s\n", detail)
}
if hint != "" {
    fmt.Printf("Hint: %s\n", hint)
}

// Format error for logging
formatted := postgres.FormatPostgreSQLError(err)
log.Println(formatted)
```

### Retry Pattern

```go
func retryableOperation(db orm.Database, record orm.DBRecord) error {
    maxRetries := 3
    for attempt := 0; attempt < maxRetries; attempt++ {
        result := db.InsertOneDBRecord(record, false)
        if result.Error == nil {
            return nil
        }

        if postgres.IsRetryable(result.Error) {
            time.Sleep(time.Second * time.Duration(attempt+1))
            continue
        }

        return result.Error // Non-retryable
    }
    return fmt.Errorf("max retries exceeded")
}
```

## Database Status and Metrics

```go
status, err := db.Status()
if err != nil {
    log.Fatalf("Failed to get status: %v", err)
}

fmt.Printf("DBMS: %s\n", status.DBMS)                   // "postgresql"
fmt.Printf("Driver: %s\n", status.DBMSDriver)           // "lib/pq"
fmt.Printf("Version: %s\n", status.Version)             // PostgreSQL version string
fmt.Printf("Leader: %s\n", status.Leader)               // Connection info with pool stats
fmt.Printf("Nodes: %d\n", status.Nodes)                 // Always 1 for PostgreSQL
fmt.Printf("Is Leader: %v\n", status.IsLeader)          // Always true for PostgreSQL

// Access peer information (includes DB size and other metrics)
for id, peer := range status.Peers {
    fmt.Printf("Peer %d: %s (Size: %d bytes)\n", id, peer.NodeID, peer.DBSize)
}
```

## Advanced Features

### Batch Operations

The PostgreSQL adapter uses optimized multi-row INSERT syntax:

```go
records := []orm.DBRecord{
    {TableName: "users", Data: map[string]interface{}{"name": "User1"}},
    {TableName: "users", Data: map[string]interface{}{"name": "User2"}},
    {TableName: "users", Data: map[string]interface{}{"name": "User3"}},
}

// Generates: INSERT INTO users (name) VALUES ($1), ($2), ($3)
results, err := db.InsertManyDBRecordsSameTable(records, false)
```

### Custom PostgreSQL Parameters

```go
config := postgres.NewDefaultConfig().
    WithExtraParam("statement_timeout", "60000").                    // 60s query timeout
    WithExtraParam("lock_timeout", "10000").                         // 10s lock timeout
    WithExtraParam("idle_in_transaction_session_timeout", "300000") // 5min idle tx timeout
```

## Testing

Run the test suite:

```bash
cd postgres
go test -v
```

Run tests with coverage:

```bash
go test -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Examples

See the [examples](./examples) directory for complete working examples:

- **[basic_usage.go](./examples/basic_usage.go)** - Basic CRUD operations and queries
- **[advanced_config.go](./examples/advanced_config.go)** - Advanced configuration patterns
- **[error_handling.go](./examples/error_handling.go)** - Comprehensive error handling examples

## Performance Tips

1. **Connection Pooling**: Tune `MaxOpenConns` and `MaxIdleConns` based on your workload
2. **Batch Inserts**: Use `InsertManyDBRecordsSameTable` for bulk operations
3. **Prepared Statements**: Use parameterized queries for repeated operations
4. **Connection Lifetime**: Set appropriate `ConnMaxLifetime` to prevent stale connections
5. **Timeouts**: Configure `QueryTimeout` to prevent long-running queries

## Supported PostgreSQL Versions

- PostgreSQL 9.6+
- PostgreSQL 10.x
- PostgreSQL 11.x
- PostgreSQL 12.x
- PostgreSQL 13.x
- PostgreSQL 14.x
- PostgreSQL 15.x
- PostgreSQL 16.x

## Dependencies

- `github.com/lib/pq` - Pure Go PostgreSQL driver
- `github.com/medatechnology/simpleorm` - SimpleORM core
- `github.com/medatechnology/goutil` - Utility functions

## Thread Safety

All operations are thread-safe. The underlying `*sql.DB` from `database/sql` package handles connection pooling and thread safety automatically.

## Limitations

- Queue mode (`queue bool` parameter) is not supported and will return an error
- `Leader()` and `Peers()` methods return single-node information (PostgreSQL is not clustered like RQLite)
- Some ORM features like migrations are not included (by design - SimpleORM focuses on basic CRUD)

## License

Same as parent SimpleORM project.

## Contributing

See the main [SimpleORM repository](../) for contribution guidelines.

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/medatechnology/simpleorm).
