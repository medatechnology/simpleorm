# SimpleORM RQLite Implementation

This package provides a direct HTTP-based implementation of the SimpleORM Database interface for RQLite, a lightweight, distributed relational database built on SQLite.

## Features

- **Direct HTTP Access**: Communicates directly with RQLite's HTTP API endpoints
- **Distributed Database Support**: Full support for RQLite cluster operations
- **Consistency Levels**: Configurable consistency levels (none, weak, strong)
- **Connection Management**: Robust HTTP client with retry logic and timeouts
- **Authentication**: Support for basic authentication
- **Cluster Management**: Leader/follower awareness and peer discovery
- **Status Monitoring**: Comprehensive cluster health monitoring
- **Automatic Batching**: Efficient bulk operations with configurable batch sizes

## Installation

```bash
go get github.com/medatechnology/simpleorm
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/medatechnology/simpleorm/rqlite"
    orm "github.com/medatechnology/simpleorm"
)

func main() {
    // Configure RQLite connection
    config := rqlite.RqliteDirectConfig{
        URL:         "http://localhost:4001",
        Consistency: "strong",
        RetryCount:  3,
    }
    
    // Create database instance
    db, err := rqlite.NewDatabase(config)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    
    // Check connection
    if !db.IsConnected() {
        log.Fatal("Database not connected")
    }
    
    fmt.Println("Successfully connected to RQLite!")
    
    // Get cluster status
    status, err := db.Status()
    if err != nil {
        log.Printf("Could not get status: %v", err)
    } else {
        fmt.Printf("Connected to RQLite %s (Node: %s)\n", status.Version, status.NodeID)
        fmt.Printf("Leader: %t, Nodes: %d\n", status.IsLeader, status.Nodes)
    }
}
```

## Configuration

### RqliteDirectConfig Structure

```go
type RqliteDirectConfig struct {
    URL         string        // Base URL for the RQLite node (e.g. "http://localhost:4001")
    Consistency string        // Consistency level: "none", "weak", "strong"
    Username    string        // Optional username for authentication
    Password    string        // Optional password for authentication
    Timeout     time.Duration // HTTP client timeout (default: 60s)
    RetryCount  int           // Number of retries for failed requests (default: 3)
}
```

### Configuration Examples

#### Minimal Configuration

```go
config := rqlite.RqliteDirectConfig{
    URL: "http://localhost:4001",
}

db, err := rqlite.NewDatabase(config)
```

#### Production Configuration

```go
config := rqlite.RqliteDirectConfig{
    URL:         "https://rqlite-cluster.example.com:4001",
    Consistency: "strong",
    Username:    "admin",
    Password:    "secure_password",
    Timeout:     30 * time.Second,
    RetryCount:  5,
}

db, err := rqlite.NewDatabase(config)
```

#### Multi-Node Cluster Setup

```go
// Connect to different nodes in a cluster
nodes := []string{
    "http://node1.cluster.com:4001",
    "http://node2.cluster.com:4001", 
    "http://node3.cluster.com:4001",
}

var db *rqlite.RQLiteDirectDB
var err error

// Try connecting to each node until successful
for _, nodeURL := range nodes {
    config := rqlite.RqliteDirectConfig{
        URL:         nodeURL,
        Consistency: "strong",
        Timeout:     10 * time.Second,
        RetryCount:  2,
    }
    
    db, err = rqlite.NewDatabase(config)
    if err == nil && db.IsConnected() {
        fmt.Printf("Connected to %s\n", nodeURL)
        break
    }
    fmt.Printf("Failed to connect to %s: %v\n", nodeURL, err)
}

if err != nil {
    log.Fatal("Failed to connect to any cluster node")
}
```

## Consistency Levels

RQLite supports three consistency levels that balance performance and data consistency:

### None (Fastest)
- **Performance**: Highest
- **Consistency**: No guarantees
- **Use Case**: High-volume reads where some staleness is acceptable

```go
config := rqlite.RqliteDirectConfig{
    URL:         "http://localhost:4001",
    Consistency: "none",
}
```

### Weak (Balanced)
- **Performance**: Good
- **Consistency**: Eventually consistent
- **Use Case**: Most application scenarios

```go
config := rqlite.RqliteDirectConfig{
    URL:         "http://localhost:4001",
    Consistency: "weak", // Default if not specified
}
```

### Strong (Most Consistent)
- **Performance**: Lower due to consensus
- **Consistency**: Linearizable reads
- **Use Case**: Critical data requiring immediate consistency

```go
config := rqlite.RqliteDirectConfig{
    URL:         "http://localhost:4001",
    Consistency: "strong",
}
```

## Database Operations

### Schema Management

```go
// Get all database schema objects
schemas := db.GetSchema(false, false)
fmt.Printf("Found %d schema objects\n", len(schemas))

// Filter out system tables
schemas = db.GetSchema(true, true) // hideSQL=true, hideSureSQL=true

for _, schema := range schemas {
    fmt.Printf("%-10s %-20s %s\n", schema.ObjectType, schema.TableName, schema.ObjectName)
    
    // Print with SQL details for debugging
    if schema.ObjectType == "table" {
        schema.PrintDebug(true)
    }
}
```

### Table Creation and Management

```go
// Create a users table
createUserTable := `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    age INTEGER CHECK(age >= 0),
    active BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`

result := db.ExecOneSQL(createUserTable)
if result.Error != nil {
    log.Fatal("Failed to create users table:", result.Error)
}
fmt.Printf("Users table created (time: %s ms)\n", orm.SecondToMsString(result.Timing))

// Create an index
createIndex := "CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)"
result = db.ExecOneSQL(createIndex)
if result.Error != nil {
    log.Printf("Warning: Could not create index: %v", result.Error)
}
```

### Insert Operations

#### Single Insert with Parameterized Query

```go
// Safe parameterized insert
insertSQL := orm.ParametereizedSQL{
    Query:  "INSERT INTO users (name, email, age, active) VALUES (?, ?, ?, ?)",
    Values: []interface{}{"John Doe", "john@example.com", 30, true},
}

result := db.ExecOneSQLParameterized(insertSQL)
if result.Error != nil {
    log.Fatal("Insert failed:", result.Error)
}

fmt.Printf("User inserted with ID: %d (affected %d rows)\n", 
    result.LastInsertID, result.RowsAffected)
```

#### Using DBRecord

```go
// Create a DBRecord
user := orm.DBRecord{
    TableName: "users",
    Data: map[string]interface{}{
        "name":   "Jane Smith",
        "email":  "jane@example.com", 
        "age":    28,
        "active": true,
    },
}

result := db.InsertOneDBRecord(user, false)
if result.Error != nil {
    log.Fatal("DBRecord insert failed:", result.Error)
}
fmt.Printf("DBRecord inserted with ID: %d\n", result.LastInsertID)
```

#### Bulk Insert Operations

```go
// Prepare multiple records for bulk insert
users := []orm.DBRecord{
    {
        TableName: "users",
        Data: map[string]interface{}{
            "name":   "Alice Johnson",
            "email":  "alice@example.com",
            "age":    25,
            "active": true,
        },
    },
    {
        TableName: "users", 
        Data: map[string]interface{}{
            "name":   "Bob Wilson",
            "email":  "bob@example.com",
            "age":    32,
            "active": true,
        },
    },
    {
        TableName: "users",
        Data: map[string]interface{}{
            "name":   "Carol Brown",
            "email":  "carol@example.com", 
            "age":    29,
            "active": false,
        },
    },
}

// Efficient bulk insert for same table
results, err := db.InsertManyDBRecordsSameTable(users, false)
if err != nil {
    log.Fatal("Bulk insert failed:", err)
}

// Check results
totalInserted := 0
totalTime := 0.0
for i, result := range results {
    if result.Error != nil {
        fmt.Printf("Record %d failed: %v\n", i, result.Error)
    } else {
        totalInserted += result.RowsAffected
        totalTime += result.Timing
        fmt.Printf("Batch %d: inserted %d records\n", i+1, result.RowsAffected)
    }
}

fmt.Printf("Total: %d records inserted in %s ms\n", 
    totalInserted, orm.SecondToMsString(totalTime))
```

### Query Operations

#### Basic Queries

```go
// Simple SELECT
allUsers, err := db.SelectMany("users")
if err != nil {
    if err == orm.ErrSQLNoRows {
        fmt.Println("No users found")
    } else {
        log.Fatal("Query failed:", err)
    }
} else {
    fmt.Printf("Found %d users\n", len(allUsers))
    for _, user := range allUsers {
        fmt.Printf("- %s (%s)\n", user.Data["name"], user.Data["email"])
    }
}

// Get single user
user, err := db.SelectOne("users")
if err != nil {
    if err == orm.ErrSQLNoRows {
        fmt.Println("No users in database")
    } else {
        log.Fatal("Query failed:", err)
    }
} else {
    fmt.Printf("First user: %s\n", user.Data["name"])
}
```

#### Raw SQL Queries

```go
// Raw SQL with automatic table name detection
sqlQuery := "SELECT name, email, age FROM users WHERE age > 25 ORDER BY age DESC"
records, err := db.SelectOneSQL(sqlQuery)
if err != nil {
    log.Fatal("Raw SQL query failed:", err)
}

fmt.Printf("Users over 25: %d\n", len(records))
for _, record := range records {
    fmt.Printf("- %s: %v years old\n", record.Data["name"], record.Data["age"])
}

// Parameterized raw SQL
paramQuery := orm.ParametereizedSQL{
    Query:  "SELECT * FROM users WHERE age BETWEEN ? AND ? AND active = ? ORDER BY name",
    Values: []interface{}{20, 40, true},
}

records, err = db.SelectOneSQLParameterized(paramQuery)
if err != nil {
    log.Fatal("Parameterized query failed:", err)
}

fmt.Printf("Active users aged 20-40: %d\n", len(records))
```

#### Exact Single Record Queries

```go
// SelectOnlyOneSQL ensures exactly one row is returned
findUserSQL := "SELECT * FROM users WHERE email = 'john@example.com'"
user, err := db.SelectOnlyOneSQL(findUserSQL)
if err != nil {
    if err == orm.ErrSQLNoRows {
        fmt.Println("User not found")
    } else if err == orm.ErrSQLMoreThanOneRow {
        fmt.Println("Multiple users found - email should be unique!")
    } else {
        log.Fatal("Query error:", err)
    }
} else {
    fmt.Printf("Found user: %s (ID: %v)\n", user.Data["name"], user.Data["id"])
}
```

### Condition-Based Queries

```go
// Simple condition
condition := &orm.Condition{
    Field:    "age",
    Operator: ">",
    Value:    18,
    OrderBy:  []string{"name ASC"},
    Limit:    10,
}

adults, err := db.SelectManyWithCondition("users", condition)
if err != nil {
    log.Fatal("Condition query failed:", err)
}
fmt.Printf("Found %d adult users\n", len(adults))

// Complex nested conditions
complexCondition := &orm.Condition{
    Logic: "AND",
    Nested: []orm.Condition{
        {
            Logic: "OR", 
            Nested: []orm.Condition{
                {Field: "age", Operator: ">=", Value: 18},
                {Field: "name", Operator: "LIKE", Value: "%admin%"},
            },
        },
        {Field: "active", Operator: "=", Value: true},
    },
    OrderBy: []string{"created_at DESC", "name ASC"},
    Limit:   50,
    Offset:  0,
}

users, err := db.SelectManyWithCondition("users", complexCondition)
if err != nil {
    log.Fatal("Complex condition query failed:", err)
}
fmt.Printf("Found %d users matching complex criteria\n", len(users))

// Generate the SQL to see what was built
sql, values := complexCondition.ToSelectString("users")
fmt.Printf("Generated SQL: %s\n", sql)
fmt.Printf("Parameters: %v\n", values)
```

### Batch Operations

```go
// Multiple different queries in one request
queries := []string{
    "SELECT COUNT(*) as total_users FROM users",
    "SELECT AVG(age) as avg_age FROM users WHERE active = 1",
    "SELECT name, age FROM users ORDER BY age DESC LIMIT 1",
    "SELECT COUNT(*) as inactive_users FROM users WHERE active = 0",
}

results, err := db.SelectManySQL(queries)
if err != nil {
    log.Fatal("Batch queries failed:", err)
}

// Process each query result
for i, records := range results {
    fmt.Printf("Query %d results:\n", i+1)
    for _, record := range records {
        for key, value := range record.Data {
            fmt.Printf("  %s: %v\n", key, value)
        }
    }
    fmt.Println()
}

// Multiple parameterized queries
paramQueries := []orm.ParametereizedSQL{
    {
        Query:  "SELECT COUNT(*) as count FROM users WHERE age > ?",
        Values: []interface{}{30},
    },
    {
        Query:  "SELECT name FROM users WHERE email LIKE ? LIMIT ?",
        Values: []interface{}{"%@example.com", 5},
    },
}

results, err = db.SelectManySQLParameterized(paramQueries)
if err != nil {
    log.Fatal("Batch parameterized queries failed:", err)
}
```

### Update and Delete Operations

```go
// Update with parameterized query
updateSQL := orm.ParametereizedSQL{
    Query:  "UPDATE users SET age = ?, updated_at = CURRENT_TIMESTAMP WHERE email = ?",
    Values: []interface{}{31, "john@example.com"},
}

result := db.ExecOneSQLParameterized(updateSQL)
if result.Error != nil {
    log.Fatal("Update failed:", result.Error)
}
fmt.Printf("Updated %d rows\n", result.RowsAffected)

// Delete with conditions
deleteSQL := orm.ParametereizedSQL{
    Query:  "DELETE FROM users WHERE active = ? AND created_at < ?",
    Values: []interface{}{false, time.Now().AddDate(0, -6, 0)}, // Inactive users older than 6 months
}

result = db.ExecOneSQLParameterized(deleteSQL)
if result.Error != nil {
    log.Fatal("Delete failed:", result.Error)
}
fmt.Printf("Deleted %d inactive users\n", result.RowsAffected)

// Batch updates
updateQueries := []string{
    "UPDATE users SET active = 0 WHERE last_login < date('now', '-1 year')",
    "UPDATE users SET age = age + 1 WHERE birthday = date('now')",
    "DELETE FROM users WHERE active = 0 AND created_at < date('now', '-2 years')",
}

results, err := db.ExecManySQL(updateQueries)
if err != nil {
    log.Fatal("Batch updates failed:", err)
}

for i, result := range results {
    if result.Error != nil {
        fmt.Printf("Update %d failed: %v\n", i+1, result.Error)
    } else {
        fmt.Printf("Update %d: affected %d rows (time: %s ms)\n", 
            i+1, result.RowsAffected, orm.SecondToMsString(result.Timing))
    }
}
```

## Cluster Management and Monitoring

### Comprehensive Status Information

```go
// Get detailed cluster status
status, err := db.Status()
if err != nil {
    log.Fatal("Failed to get status:", err)
}

// Print formatted status
fmt.Println("=== RQLite Cluster Status ===")
status.PrintPretty()

// Access individual status fields
fmt.Printf("\nCluster Overview:\n")
fmt.Printf("- DBMS: %s %s\n", status.DBMS, status.Version)
fmt.Printf("- Node ID: %s\n", status.NodeID)
fmt.Printf("- URL: %s\n", status.URL)
fmt.Printf("- Leader: %t\n", status.IsLeader)
fmt.Printf("- Total Nodes: %d\n", status.Nodes)

if status.IsLeader {
    fmt.Printf("- This node is the cluster leader\n")
} else {
    fmt.Printf("- Leader: %s\n", status.Leader)
}

// Database metrics
fmt.Printf("\nDatabase Metrics:\n")
fmt.Printf("- Database Size: %s\n", print.BytesToHumanReadable(status.DBSize, " "))
fmt.Printf("- Directory Size: %s\n", print.BytesToHumanReadable(status.DirSize, " "))
fmt.Printf("- Max Pool Size: %d\n", status.MaxPool)

// Timing information
fmt.Printf("\nTiming Information:\n")
fmt.Printf("- Start Time: %s\n", status.StartTime.Format("2006-01-02 15:04:05"))
fmt.Printf("- Uptime: %s\n", timedate.DurationUptimeShort(status.Uptime))

if !status.LastBackup.IsZero() {
    fmt.Printf("- Last Backup: %s\n", status.LastBackup.Format("2006-01-02 15:04:05"))
}

// Peer information
if len(status.Peers) > 0 {
    fmt.Printf("\nCluster Peers:\n")
    for nodeNum, peer := range status.Peers {
        if peer.NodeNumber != status.NodeNumber {
            fmt.Printf("- Node %d: %s (ID: %s)\n", nodeNum, peer.URL, peer.NodeID)
        }
    }
}
```

### Leader and Peer Discovery

```go
// Check current leader
leader, err := db.Leader()
if err != nil {
    fmt.Printf("Leader information not available: %v\n", err)
} else {
    fmt.Printf("Current cluster leader: %s\n", leader)
}

// Get all cluster peers
peers, err := db.Peers()
if err != nil {
    fmt.Printf("Peer information not available: %v\n", err)
} else {
    fmt.Printf("Cluster has %d peers:\n", len(peers))
    for i, peer := range peers {
        fmt.Printf("  %d. %s\n", i+1, peer)
    }
}

// Health check
if !db.IsConnected() {
    log.Fatal("Database connection is not healthy")
}
fmt.Println("Database connection is healthy")
```

## Error Handling and Resilience

### Connection Error Handling

```go
// Test connection with retry
func testConnection(db *rqlite.RQLiteDirectDB) error {
    maxRetries := 5
    for i := 0; i < maxRetries; i++ {
        if db.IsConnected() {
            _, err := db.Status()
            if err == nil {
                return nil
            }
        }
        
        fmt.Printf("Connection attempt %d/%d failed, retrying...\n", i+1, maxRetries)
        time.Sleep(time.Duration(i+1) * time.Second)
    }
    
    return fmt.Errorf("failed to establish connection after %d attempts", maxRetries)
}

if err := testConnection(db); err != nil {
    log.Fatal(err)
}
```

### Query Error Handling

```go
// Handle different types of query errors
func handleQueryError(err error, operation string) {
    if err == nil {
        return
    }
    
    switch err {
    case orm.ErrSQLNoRows:
        fmt.Printf("%s: No data found\n", operation)
    case orm.ErrSQLMoreThanOneRow:
        fmt.Printf("%s: Expected single row, got multiple\n", operation)
    default:
        fmt.Printf("%s failed: %v\n", operation, err)
    }
}

// Example usage
users, err := db.SelectManyWithCondition("users", &orm.Condition{
    Field: "email", Operator: "=", Value: "nonexistent@example.com",
})
handleQueryError(err, "Finding user by email")

user, err := db.SelectOnlyOneSQL("SELECT * FROM users WHERE age = 25")
handleQueryError(err, "Finding single user by age")
```

### Batch Operation Error Handling

```go
// Handle errors in batch operations
queries := []string{
    "SELECT COUNT(*) FROM users",
    "SELECT COUNT(*) FROM nonexistent_table", // This will fail
    "SELECT AVG(age) FROM users",
}

results, err := db.SelectManySQL(queries)
if err != nil {
    fmt.Printf("Batch operation failed: %v\n", err)
} else {
    for i, records := range results {
        if len(records) == 0 {
            fmt.Printf("Query %d returned no results\n", i+1)
        } else {
            fmt.Printf("Query %d succeeded with %d records\n", i+1, len(records))
        }
    }
}

// Check individual operation results
inserts := []orm.DBRecord{
    {TableName: "users", Data: map[string]interface{}{"name": "Test1", "email": "test1@test.com"}},
    {TableName: "invalid_table", Data: map[string]interface{}{"name": "Test2"}}, // This will fail
    {TableName: "users", Data: map[string]interface{}{"name": "Test3", "email": "test3@test.com"}},
}

insertResults, err := db.InsertManyDBRecords(inserts, false)
if err != nil {
    fmt.Printf("Batch insert failed: %v\n", err)
} else {
    successCount := 0
    for i, result := range insertResults {
        if result.Error != nil {
            fmt.Printf("Insert %d failed: %v\n", i+1, result.Error)
        } else {
            successCount++
            fmt.Printf("Insert %d succeeded: ID %d\n", i+1, result.LastInsertID)
        }
    }
    fmt.Printf("Batch complete: %d/%d operations succeeded\n", successCount, len(insertResults))
}
```

## Performance Optimization

### Batch Size Configuration

```go
// Check current batch size setting
fmt.Printf("Current max batch size: %d\n", orm.MAX_MULTIPLE_INSERTS)

// Adjust batch size for your workload
// Smaller batches = lower memory usage, more network calls
// Larger batches = higher memory usage, fewer network calls
orm.MAX_MULTIPLE_INSERTS = 50 // Reduce for memory-constrained environments

// Or increase for high-throughput scenarios
orm.MAX_MULTIPLE_INSERTS = 200

// Monitor batch performance
largeDataset := make([]orm.DBRecord, 500)
for i := range largeDataset {
    largeDataset[i] = orm.DBRecord{
        TableName: "users",
        Data: map[string]interface{}{
            "name":  fmt.Sprintf("User%d", i),
            "email": fmt.Sprintf("user%d@test.com", i),
            "age":   20 + (i % 50),
        },
    }
}

startTime := time.Now()
results, err := db.InsertManyDBRecordsSameTable(largeDataset, false)
if err != nil {
    log.Fatal("Large batch insert failed:", err)
}

totalTime := orm.TotalTimeElapsedInSecond(results)
elapsed := time.Since(startTime)

fmt.Printf("Inserted %d records in %d batches\n", len(largeDataset), len(results))
fmt.Printf("Database time: %s ms\n", orm.SecondToMsString(totalTime))
fmt.Printf("Total time: %v\n", elapsed)
fmt.Printf("Throughput: %.2f records/second\n", float64(len(largeDataset))/elapsed.Seconds())
```

### Connection Pool Optimization

```go
// The RQLite implementation automatically configures HTTP connection pooling
// with these optimized settings:

// DEFAULT_MAX_IDLE_CONNECTIONS          = 100
// DEFAULT_MAX_IDLE_CONNECTIONS_PER_HOST = 100  
// DEFAULT_MAX_CONNECTIONS_PER_HOST      = 1000
// DEFAULT_IDLE_CONNECTION_TIMEOUT       = 90 * time.Second
// DEFAULT_TIMEOUT                       = 60 * time.Second
// DEFAULT_KEEP_ALIVE                    = 30 * time.Second

// These settings are automatically applied when creating a new database connection
// and are optimized for most production workloads
```

### Query Performance Monitoring

```go
// Monitor query performance
func monitorQuery(db *rqlite.RQLiteDirectDB, description string, queryFunc func() error) {
    start := time.Now()
    err := queryFunc()
    elapsed := time.Since(start)
    
    if err != nil {
        fmt.Printf("❌ %s failed in %v: %v\n", description, elapsed, err)
    } else {
        fmt.Printf("✅ %s completed in %v\n", description, elapsed)
    }
}

// Example usage
monitorQuery(db, "Complex user query", func() error {
    condition := &orm.Condition{
        Logic: "AND",
        Nested: []orm.Condition{
            {Field: "age", Operator: "BETWEEN", Value: []interface{}{20, 40}},
            {Field: "active", Operator: "=", Value: true},
        },
        OrderBy: []string{"created_at DESC"},
        Limit:   100,
    }
    
    _, err := db.SelectManyWithCondition("users", condition)
    return err
})

monitorQuery(db, "Batch status queries", func() error {
    queries := []string{
        "SELECT COUNT(*) FROM users",
        "SELECT COUNT(*) FROM users WHERE active = 1",
        "SELECT AVG(age) FROM users",
    }
    
    _, err := db.SelectManySQL(queries)
    return err
})
```

## RQLite-Specific Features

### HTTP Endpoints Used

The implementation leverages these RQLite HTTP API endpoints:

```go
// Read operations
ENDPOINT_QUERY    = "/db/query"    // SELECT statements

// Write operations  
ENDPOINT_EXECUTE  = "/db/execute"  // INSERT, UPDATE, DELETE statements

// Cluster management
ENDPOINT_STATUS   = "/status"      // Cluster status and health
ENDPOINT_READY    = "/readyz"      // Readiness check (future use)

// Backup/Restore (planned features)
ENDPOINT_BACKUP   = "/db/backup"   // Database backup
ENDPOINT_LOAD     = "/db/load"     // Database restore
ENDPOINT_BOOT     = "/boot"        // Database bootstrap
```

### SQLite Data Type Handling

RQLite uses SQLite as its storage engine, so data types are handled according to SQLite's type system:

```go
// SQLite type affinity examples
data := map[string]interface{}{
    // INTEGER affinity
    "id":          1,
    "count":       int64(1000),
    "big_number":  uint64(9223372036854775807),
    
    // TEXT affinity  
    "name":        "John Doe",
    "description": "A long text description",
    "json_data":   `{"key": "value"}`,
    
    // REAL affinity
    "price":       99.99,
    "percentage":  float32(85.5),
    
    // BOOLEAN (stored as INTEGER 0/1)
    "is_active":   true,
    "is_deleted":  false,
    
    // DATETIME (stored as TEXT in ISO8601 format)
    "created_at":  time.Now(),
    "updated_at":  time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC),
    
    // BLOB affinity
    "binary_data": []byte("binary content"),
    "image":       []byte{0x89, 0x50, 0x4E, 0x47}, // PNG header
}

record := orm.DBRecord{
    TableName: "mixed_types",
    Data:      data,
}

result := db.InsertOneDBRecord(record, false)
if result.Error != nil {
    log.Fatal("Failed to insert mixed types:", result.Error)
}
```

### Advanced SQL Features

```go
// Common Table Expressions (CTEs)
cteQuery := `
WITH RECURSIVE user_hierarchy AS (
    SELECT id, name, manager_id, 1 as level
    FROM users 
    WHERE manager_id IS NULL
    
    UNION ALL
    
    SELECT u.id, u.name, u.manager_id, uh.level + 1
    FROM users u
    JOIN user_hierarchy uh ON u.manager_id = uh.id
)
SELECT * FROM user_hierarchy ORDER BY level, name`

records, err := db.SelectOneSQL(cteQuery)
if err != nil {
    log.Fatal("CTE query failed:", err)
}

// Window functions
windowQuery := `
SELECT 
    name,
    age,
    department,
    salary,
    AVG(salary) OVER (PARTITION BY department) as dept_avg_salary,
    ROW_NUMBER() OVER (PARTITION BY department ORDER BY salary DESC) as dept_rank
FROM employees
ORDER BY department, dept_rank`

records, err = db.SelectOneSQL(windowQuery)

// JSON operations (SQLite 3.38+)
jsonQuery := orm.ParametereizedSQL{
    Query: `
    SELECT 
        id,
        name,
        JSON_EXTRACT(metadata, '$.department') as department,
        JSON_EXTRACT(metadata, '$.skills') as skills
    FROM users 