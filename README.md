# SimpleORM

A simple, flexible Object-Relational Mapping (ORM) library for Go that provides a unified interface for different database systems. Currently supports RQLite with plans for additional database implementations.

## Features

- **Database Agnostic**: Unified interface that works across different database systems
- **Flexible Querying**: Support for raw SQL, parameterized queries, and condition-based queries
- **Advanced Condition Builder**: Build complex nested queries with AND/OR logic
- **Complex Query Support**: JOINs, aggregations, GROUP BY, HAVING, DISTINCT, and CTEs
- **Type-Safe JOINs**: Multiple JOIN types (INNER, LEFT, RIGHT, FULL, CROSS) with validation
- **Batch Operations**: Efficient bulk insert operations with automatic batching
- **Schema Management**: Built-in schema inspection and management
- **Status Monitoring**: Comprehensive database status and health monitoring
- **Connection Management**: Robust connection handling with retry logic
- **SQL Injection Protection**: Built-in validation and parameterized queries

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Database Operations](#database-operations)
- [Advanced Condition Queries](#advanced-condition-queries)
- [Complex Queries (JOINs and Aggregations)](#complex-queries-joins-and-aggregations)
- [Batch Operations](#batch-operations)
- [Error Handling](#error-handling)
- [Performance Tips](#performance-tips)

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
    
    orm "github.com/medatechnology/simpleorm"
    "github.com/medatechnology/simpleorm/rqlite"
)

// Define your table struct
type User struct {
    ID       int    `json:"id" db:"id"`
    Name     string `json:"name" db:"name"`
    Email    string `json:"email" db:"email"`
    Age      int    `json:"age" db:"age"`
    Active   bool   `json:"active" db:"active"`
    Country  string `json:"country" db:"country"`
    Role     string `json:"role" db:"role"`
}

// Implement TableStruct interface
func (u User) TableName() string {
    return "users"
}

func main() {
    // Initialize database connection
    config := rqlite.RqliteDirectConfig{
        URL:         "http://localhost:4001",
        Consistency: "strong",
        RetryCount:  3,
    }
    
    db, err := rqlite.NewDatabase(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create table
    createTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT UNIQUE NOT NULL,
        age INTEGER,
        active BOOLEAN DEFAULT 1,
        country TEXT,
        role TEXT DEFAULT 'user'
    )`
    
    result := db.ExecOneSQL(createTable)
    if result.Error != nil {
        log.Fatal(result.Error)
    }
    
    // Insert a user using TableStruct
    user := User{
        Name:    "John Doe",
        Email:   "john@example.com",
        Age:     30,
        Active:  true,
        Country: "USA",
        Role:    "admin",
    }
    
    insertResult := db.InsertOneTableStruct(user, false)
    if insertResult.Error != nil {
        log.Fatal(insertResult.Error)
    }
    
    fmt.Printf("User inserted with ID: %d\n", insertResult.LastInsertID)
}
```

## Core Concepts

### Database Interface

The `Database` interface provides a unified API for all database operations:

```go
type Database interface {
    // Schema operations
    GetSchema(hideSQL, hideSureSQL bool) []SchemaStruct
    Status() (NodeStatusStruct, error)
    
    // Simple select operations
    SelectOne(tableName string) (DBRecord, error)
    SelectMany(tableName string) (DBRecords, error)
    
    // Condition-based operations
    SelectOneWithCondition(tableName string, condition *Condition) (DBRecord, error)
    SelectManyWithCondition(tableName string, condition *Condition) ([]DBRecord, error)
    
    // Raw SQL operations
    SelectOneSQL(sql string) (DBRecords, error)
    SelectOnlyOneSQL(sql string) (DBRecord, error)
    SelectOneSQLParameterized(paramSQL ParametereizedSQL) (DBRecords, error)
    SelectManySQLParameterized(paramSQLs []ParametereizedSQL) ([]DBRecords, error)
    
    // Execute operations
    ExecOneSQL(sql string) BasicSQLResult
    ExecOneSQLParameterized(paramSQL ParametereizedSQL) BasicSQLResult
    ExecManySQL(sqls []string) ([]BasicSQLResult, error)
    ExecManySQLParameterized(paramSQLs []ParametereizedSQL) ([]BasicSQLResult, error)
    
    // Insert operations
    InsertOneDBRecord(record DBRecord, queue bool) BasicSQLResult
    InsertManyDBRecords(records []DBRecord, queue bool) ([]BasicSQLResult, error)
    InsertManyDBRecordsSameTable(records []DBRecord, queue bool) ([]BasicSQLResult, error)
    InsertOneTableStruct(obj TableStruct, queue bool) BasicSQLResult
    InsertManyTableStructs(objs []TableStruct, queue bool) ([]BasicSQLResult, error)
    
    // Connection status
    IsConnected() bool
    Leader() (string, error)
    Peers() ([]string, error)
}
```

### TableStruct Interface

Any struct that represents a database table must implement this interface:

```go
type TableStruct interface {
    TableName() string
}

// Example implementation
type Product struct {
    ID          int     `json:"id" db:"id"`
    Name        string  `json:"name" db:"name"`
    Price       float64 `json:"price" db:"price"`
    CategoryID  int     `json:"category_id" db:"category_id"`
    InStock     bool    `json:"in_stock" db:"in_stock"`
}

func (p Product) TableName() string {
    return "products"
}
```

### DBRecord Structure

A flexible record structure that can represent any database row:

```go
type DBRecord struct {
    TableName string
    Data      map[string]interface{}
}

// Example usage
record := orm.DBRecord{
    TableName: "users",
    Data: map[string]interface{}{
        "name":    "Alice Johnson",
        "email":   "alice@example.com",
        "age":     28,
        "active":  true,
        "country": "Canada",
    },
}
```

### Condition Structure

The heart of SimpleORM's querying capabilities:

```go
type Condition struct {
    Field    string      `json:"field,omitempty"`        // Column name
    Operator string      `json:"operator,omitempty"`     // SQL operator (=, >, <, LIKE, etc.)
    Value    interface{} `json:"value,omitempty"`        // Value to compare
    Logic    string      `json:"logic,omitempty"`        // "AND" or "OR" for nested conditions
    Nested   []Condition `json:"nested,omitempty"`       // For complex nested conditions
    OrderBy  []string    `json:"order_by,omitempty"`     // ORDER BY clauses
    GroupBy  []string    `json:"group_by,omitempty"`     // GROUP BY clauses
    Limit    int         `json:"limit,omitempty"`        // LIMIT for pagination
    Offset   int         `json:"offset,omitempty"`       // OFFSET for pagination
}
```

## Database Operations

### Basic CRUD Operations

#### Create and Insert

```go
// Create table
createTableSQL := `
CREATE TABLE IF NOT EXISTS orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER DEFAULT 1,
    total_amount DECIMAL(10,2),
    status TEXT DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`

result := db.ExecOneSQL(createTableSQL)
if result.Error != nil {
    log.Fatal(result.Error)
}

// Insert using DBRecord
order := orm.DBRecord{
    TableName: "orders",
    Data: map[string]interface{}{
        "user_id":      1,
        "product_id":   100,
        "quantity":     2,
        "total_amount": 49.98,
        "status":       "pending",
    },
}

insertResult := db.InsertOneDBRecord(order, false)
if insertResult.Error != nil {
    log.Fatal(insertResult.Error)
}
```

#### Read Operations

```go
// Get all records from a table
allUsers, err := db.SelectMany("users")
if err != nil {
    if err == orm.ErrSQLNoRows {
        fmt.Println("No users found")
    } else {
        log.Fatal(err)
    }
}

// Get single record
firstUser, err := db.SelectOne("users")
if err != nil {
    log.Fatal(err)
}

// Get exactly one record (fails if 0 or >1 found)
specificUser, err := db.SelectOnlyOneSQL("SELECT * FROM users WHERE email = 'john@example.com'")
if err != nil {
    if err == orm.ErrSQLNoRows {
        fmt.Println("User not found")
    } else if err == orm.ErrSQLMoreThanOneRow {
        fmt.Println("Multiple users found - expected only one")
    } else {
        log.Fatal(err)
    }
}
```

### Parameterized Queries

```go
// Safe parameterized query
paramSQL := orm.ParametereizedSQL{
    Query:  "SELECT * FROM users WHERE age > ? AND country = ? ORDER BY name",
    Values: []interface{}{21, "USA"},
}

records, err := db.SelectOneSQLParameterized(paramSQL)
if err != nil {
    log.Fatal(err)
}

// Multiple parameterized queries
paramQueries := []orm.ParametereizedSQL{
    {
        Query:  "SELECT COUNT(*) as user_count FROM users WHERE active = ?",
        Values: []interface{}{true},
    },
    {
        Query:  "SELECT AVG(age) as avg_age FROM users WHERE country = ?",
        Values: []interface{}{"USA"},
    },
}

results, err := db.SelectManySQLParameterized(paramQueries)
if err != nil {
    log.Fatal(err)
}
```

## Advanced Condition Queries

### Simple Conditions

```go
// Single field condition
condition := &orm.Condition{
    Field:    "age",
    Operator: ">",
    Value:    18,
}

adults, err := db.SelectManyWithCondition("users", condition)
// Generates: SELECT * FROM users WHERE age > 18

// With ordering and pagination
condition = &orm.Condition{
    Field:    "active",
    Operator: "=",
    Value:    true,
    OrderBy:  []string{"name ASC", "age DESC"},
    Limit:    10,
    Offset:   20,
}

activeUsers, err := db.SelectManyWithCondition("users", condition)
// Generates: SELECT * FROM users WHERE active = 1 ORDER BY name ASC, age DESC LIMIT 10 OFFSET 20
```

### Complex Nested Conditions

#### AND Logic Example

```go
// Multiple conditions with AND logic
andCondition := &orm.Condition{
    Logic: "AND",
    Nested: []orm.Condition{
        {Field: "age", Operator: ">=", Value: 18},
        {Field: "age", Operator: "<=", Value: 65},
        {Field: "active", Operator: "=", Value: true},
        {Field: "country", Operator: "=", Value: "USA"},
    },
    OrderBy: []string{"name ASC"},
    Limit:   50,
}

workingAgeUsers, err := db.SelectManyWithCondition("users", andCondition)
// Generates: SELECT * FROM users WHERE (age >= 18 AND age <= 65 AND active = 1 AND country = 'USA') ORDER BY name ASC LIMIT 50
```

#### OR Logic Example

```go
// Multiple conditions with OR logic
orCondition := &orm.Condition{
    Logic: "OR",
    Nested: []orm.Condition{
        {Field: "role", Operator: "=", Value: "admin"},
        {Field: "role", Operator: "=", Value: "moderator"},
        {Field: "role", Operator: "=", Value: "editor"},
    },
    OrderBy: []string{"name ASC"},
}

privilegedUsers, err := db.SelectManyWithCondition("users", orCondition)
// Generates: SELECT * FROM users WHERE (role = 'admin' OR role = 'moderator' OR role = 'editor') ORDER BY name ASC
```

#### Complex Nested AND/OR Combinations

```go
// Complex business logic: 
// Find users who are either:
// 1. Adults (18+) from USA who are active, OR
// 2. Premium users from any country, OR  
// 3. Admins regardless of other criteria
complexCondition := &orm.Condition{
    Logic: "OR",
    Nested: []orm.Condition{
        {
            // Group 1: Adult active USA users
            Logic: "AND",
            Nested: []orm.Condition{
                {Field: "age", Operator: ">=", Value: 18},
                {Field: "country", Operator: "=", Value: "USA"},
                {Field: "active", Operator: "=", Value: true},
            },
        },
        {
            // Group 2: Premium users
            Logic: "AND",
            Nested: []orm.Condition{
                {Field: "subscription", Operator: "=", Value: "premium"},
                {Field: "active", Operator: "=", Value: true},
            },
        },
        {
            // Group 3: Admins
            Field:    "role",
            Operator: "=",
            Value:    "admin",
        },
    },
    OrderBy: []string{"role DESC", "subscription DESC", "name ASC"},
    Limit:   100,
}

targetUsers, err := db.SelectManyWithCondition("users", complexCondition)
// Generates: SELECT * FROM users WHERE ((age >= 18 AND country = 'USA' AND active = 1) OR (subscription = 'premium' AND active = 1) OR role = 'admin') ORDER BY role DESC, subscription DESC, name ASC LIMIT 100
```

#### Advanced Query Patterns

```go
// Search pattern with multiple criteria
searchCondition := &orm.Condition{
    Logic: "AND",
    Nested: []orm.Condition{
        {
            // Name or email contains search term
            Logic: "OR",
            Nested: []orm.Condition{
                {Field: "name", Operator: "LIKE", Value: "%john%"},
                {Field: "email", Operator: "LIKE", Value: "%john%"},
            },
        },
        {
            // Active users only
            Field:    "active",
            Operator: "=",
            Value:    true,
        },
        {
            // Age range
            Logic: "AND",
            Nested: []orm.Condition{
                {Field: "age", Operator: ">=", Value: 18},
                {Field: "age", Operator: "<=", Value: 80},
            },
        },
    },
    OrderBy: []string{"name ASC"},
    Limit:   25,
}

searchResults, err := db.SelectManyWithCondition("users", searchCondition)
// Generates: SELECT * FROM users WHERE ((name LIKE '%john%' OR email LIKE '%john%') AND active = 1 AND (age >= 18 AND age <= 80)) ORDER BY name ASC LIMIT 25
```

#### Date Range Queries

```go
import "time"

// Users created in the last 30 days who are active
dateRangeCondition := &orm.Condition{
    Logic: "AND",
    Nested: []orm.Condition{
        {Field: "created_at", Operator: ">=", Value: time.Now().AddDate(0, 0, -30)},
        {Field: "created_at", Operator: "<=", Value: time.Now()},
        {Field: "active", Operator: "=", Value: true},
    },
    OrderBy: []string{"created_at DESC"},
    Limit:   50,
}

recentUsers, err := db.SelectManyWithCondition("users", dateRangeCondition)
```

#### E-commerce Query Examples

```go
// Find products that are either:
// 1. On sale (discount > 0) and in stock, OR
// 2. Featured products regardless of stock, OR
// 3. New arrivals (created in last 7 days)
productCondition := &orm.Condition{
    Logic: "OR",
    Nested: []orm.Condition{
        {
            Logic: "AND",
            Nested: []orm.Condition{
                {Field: "discount_percent", Operator: ">", Value: 0},
                {Field: "in_stock", Operator: "=", Value: true},
            },
        },
        {Field: "featured", Operator: "=", Value: true},
        {Field: "created_at", Operator: ">=", Value: time.Now().AddDate(0, 0, -7)},
    },
    OrderBy: []string{"featured DESC", "discount_percent DESC", "created_at DESC"},
    Limit:   20,
}

promotionalProducts, err := db.SelectManyWithCondition("products", productCondition)

// Customer segmentation: High-value customers
customerCondition := &orm.Condition{
    Logic: "AND",
    Nested: []orm.Condition{
        {
            Logic: "OR",
            Nested: []orm.Condition{
                {Field: "total_spent", Operator: ">=", Value: 1000},
                {Field: "order_count", Operator: ">=", Value: 10},
            },
        },
        {Field: "last_order_date", Operator: ">=", Value: time.Now().AddDate(0, -6, 0)}, // Active in last 6 months
        {Field: "status", Operator: "=", Value: "active"},
    },
    OrderBy: []string{"total_spent DESC", "last_order_date DESC"},
    Limit:   100,
}

vipCustomers, err := db.SelectManyWithCondition("customers", customerCondition)
```

### Using Helper Methods

```go
// You can also use the helper methods for building conditions
condition := &orm.Condition{}

// Build an AND condition
andCond := condition.And(
    orm.Condition{Field: "age", Operator: ">", Value: 18},
    orm.Condition{Field: "status", Operator: "=", Value: "active"},
    orm.Condition{Field: "country", Operator: "=", Value: "USA"},
)

// Build an OR condition  
orCond := condition.Or(
    orm.Condition{Field: "role", Operator: "=", Value: "admin"},
    orm.Condition{Field: "role", Operator: "=", Value: "moderator"},
)

// Use the conditions
users, err := db.SelectManyWithCondition("users", andCond)
privileged, err := db.SelectManyWithCondition("users", orCond)
```

### Debugging Condition Queries

```go
// See what SQL is generated from your conditions
condition := &orm.Condition{
    Logic: "AND",
    Nested: []orm.Condition{
        {Field: "age", Operator: "BETWEEN", Value: []interface{}{18, 65}},
        {Field: "active", Operator: "=", Value: true},
    },
    OrderBy: []string{"name ASC"},
    Limit:   10,
}

// Generate and inspect the SQL
sql, values := condition.ToSelectString("users")
fmt.Printf("Generated SQL: %s\n", sql)
fmt.Printf("Parameters: %v\n", values)

// Output:
// Generated SQL: SELECT * FROM users WHERE (age BETWEEN ? AND ? AND active = ?) ORDER BY name ASC LIMIT 10
// Parameters: [18 65 true]
```

## Complex Queries (JOINs and Aggregations)

SimpleORM now supports complex queries with JOINs, custom SELECT fields, GROUP BY, HAVING, and more through the `ComplexQuery` struct. This provides advanced SQL capabilities while maintaining type safety and SQL injection protection.

### ComplexQuery Structure

```go
type ComplexQuery struct {
    Select    []string   // Custom SELECT fields (default: ["*"])
    Distinct  bool       // Add DISTINCT keyword
    From      string     // Main table name (required)
    FromAlias string     // Alias for main table
    Joins     []Join     // JOIN clauses
    Where     *Condition // WHERE conditions (uses Condition struct)
    GroupBy   []string   // GROUP BY fields
    Having    string     // HAVING clause
    OrderBy   []string   // ORDER BY fields
    Limit     int        // LIMIT value
    Offset    int        // OFFSET value
    CTE       string     // Common Table Expression (WITH clause)
}

type Join struct {
    Type      JoinType // INNER JOIN, LEFT JOIN, RIGHT JOIN, FULL OUTER JOIN, CROSS JOIN
    Table     string   // Table to join
    Alias     string   // Optional table alias
    Condition string   // Join condition (e.g., "users.id = orders.user_id")
}
```

### Basic Complex Query with Custom Fields

```go
// Select specific fields instead of SELECT *
query := &orm.ComplexQuery{
    Select: []string{"id", "name", "email", "created_at"},
    From:   "users",
    Where: &orm.Condition{
        Field:    "status",
        Operator: "=",
        Value:    "active",
    },
    OrderBy: []string{"created_at DESC"},
    Limit:   10,
}

records, err := db.SelectManyComplex(query)
// Generates: SELECT id, name, email, created_at FROM users WHERE status = ? ORDER BY created_at DESC LIMIT 10
```

### Simple JOIN Query

```go
// LEFT JOIN to get users with their profiles
query := &orm.ComplexQuery{
    Select: []string{
        "users.id",
        "users.name",
        "users.email",
        "profiles.bio",
        "profiles.avatar_url",
    },
    From: "users",
    Joins: []orm.Join{
        {
            Type:      orm.LeftJoin,
            Table:     "profiles",
            Condition: "users.id = profiles.user_id",
        },
    },
    Where: &orm.Condition{
        Field:    "users.status",
        Operator: "=",
        Value:    "active",
    },
    OrderBy: []string{"users.created_at DESC"},
    Limit:   20,
}

records, err := db.SelectManyComplex(query)
// Generates: SELECT users.id, users.name, users.email, profiles.bio, profiles.avatar_url
//            FROM users LEFT JOIN profiles ON users.id = profiles.user_id
//            WHERE users.status = ? ORDER BY users.created_at DESC LIMIT 20
```

### Aggregate Functions with GROUP BY

```go
// Get user order statistics with aggregation
query := &orm.ComplexQuery{
    Select: []string{
        "users.id",
        "users.name",
        "COUNT(orders.id) as order_count",
        "SUM(orders.total) as total_spent",
        "AVG(orders.total) as avg_order_value",
        "MAX(orders.created_at) as last_order_date",
    },
    From: "users",
    Joins: []orm.Join{
        {
            Type:      orm.LeftJoin,
            Table:     "orders",
            Condition: "users.id = orders.user_id",
        },
    },
    Where: &orm.Condition{
        Field:    "users.status",
        Operator: "=",
        Value:    "active",
    },
    GroupBy: []string{"users.id", "users.name"},
    Having:  "COUNT(orders.id) > 5",
    OrderBy: []string{"order_count DESC", "total_spent DESC"},
    Limit:   10,
}

records, err := db.SelectManyComplex(query)
// Generates: SELECT users.id, users.name, COUNT(orders.id) as order_count,
//                   SUM(orders.total) as total_spent, AVG(orders.total) as avg_order_value,
//                   MAX(orders.created_at) as last_order_date
//            FROM users LEFT JOIN orders ON users.id = orders.user_id
//            WHERE users.status = ?
//            GROUP BY users.id, users.name
//            HAVING COUNT(orders.id) > 5
//            ORDER BY order_count DESC, total_spent DESC LIMIT 10

// Access aggregated data
for _, record := range records {
    fmt.Printf("User: %s, Orders: %v, Total Spent: %v, Avg: %v\n",
        record.Data["name"],
        record.Data["order_count"],
        record.Data["total_spent"],
        record.Data["avg_order_value"])
}
```

### Multiple JOINs

```go
// Query across multiple related tables
query := &orm.ComplexQuery{
    Select: []string{
        "users.name as customer_name",
        "orders.order_number",
        "orders.total",
        "products.name as product_name",
        "order_items.quantity",
        "order_items.price",
    },
    From: "users",
    Joins: []orm.Join{
        {
            Type:      orm.InnerJoin,
            Table:     "orders",
            Condition: "users.id = orders.user_id",
        },
        {
            Type:      orm.InnerJoin,
            Table:     "order_items",
            Condition: "orders.id = order_items.order_id",
        },
        {
            Type:      orm.InnerJoin,
            Table:     "products",
            Condition: "order_items.product_id = products.id",
        },
    },
    Where: &orm.Condition{
        Logic: "AND",
        Nested: []orm.Condition{
            {Field: "users.status", Operator: "=", Value: "active"},
            {Field: "orders.status", Operator: "=", Value: "completed"},
        },
    },
    OrderBy: []string{"orders.created_at DESC"},
    Limit:   50,
}

records, err := db.SelectManyComplex(query)
// Generates: SELECT users.name as customer_name, orders.order_number, orders.total,
//                   products.name as product_name, order_items.quantity, order_items.price
//            FROM users
//            INNER JOIN orders ON users.id = orders.user_id
//            INNER JOIN order_items ON orders.id = order_items.order_id
//            INNER JOIN products ON order_items.product_id = products.id
//            WHERE (users.status = ? AND orders.status = ?)
//            ORDER BY orders.created_at DESC LIMIT 50
```

### DISTINCT Queries

```go
// Get unique cities where active users are located
query := &orm.ComplexQuery{
    Select:   []string{"city", "country"},
    Distinct: true,
    From:     "users",
    Where: &orm.Condition{
        Field:    "status",
        Operator: "=",
        Value:    "active",
    },
    OrderBy: []string{"country", "city"},
}

records, err := db.SelectManyComplex(query)
// Generates: SELECT DISTINCT city, country FROM users WHERE status = ? ORDER BY country, city
```

### SelectOneComplex - Single Record with JOINs

```go
// Get a specific user with their profile (must return exactly one row)
query := &orm.ComplexQuery{
    Select: []string{
        "users.*",
        "profiles.bio",
        "profiles.avatar_url",
        "profiles.location",
    },
    From: "users",
    Joins: []orm.Join{
        {
            Type:      orm.InnerJoin,
            Table:     "profiles",
            Condition: "users.id = profiles.user_id",
        },
    },
    Where: &orm.Condition{
        Field:    "users.id",
        Operator: "=",
        Value:    123,
    },
}

record, err := db.SelectOneComplex(query)
if err != nil {
    if err == orm.ErrSQLNoRows {
        fmt.Println("User not found")
    } else if err == orm.ErrSQLMoreThanOneRow {
        fmt.Println("Expected one user, found multiple")
    } else {
        log.Fatal(err)
    }
}

fmt.Printf("User: %v\n", record.Data)
```

### Complex Queries with Nested Conditions

```go
// Combine complex JOINs with nested WHERE conditions
query := &orm.ComplexQuery{
    Select: []string{
        "users.id",
        "users.name",
        "COUNT(orders.id) as order_count",
    },
    From: "users",
    Joins: []orm.Join{
        {
            Type:      orm.LeftJoin,
            Table:     "orders",
            Condition: "users.id = orders.user_id",
        },
    },
    Where: &orm.Condition{
        Logic: "OR",
        Nested: []orm.Condition{
            {
                // Active US users over 18
                Logic: "AND",
                Nested: []orm.Condition{
                    {Field: "users.age", Operator: ">", Value: 18},
                    {Field: "users.country", Operator: "=", Value: "USA"},
                    {Field: "users.status", Operator: "=", Value: "active"},
                },
            },
            {
                // Or premium users from any country
                Logic: "AND",
                Nested: []orm.Condition{
                    {Field: "users.subscription", Operator: "=", Value: "premium"},
                    {Field: "users.verified", Operator: "=", Value: true},
                },
            },
        },
    },
    GroupBy: []string{"users.id", "users.name"},
    OrderBy: []string{"order_count DESC"},
    Limit:   25,
}

records, err := db.SelectManyComplex(query)
```

### E-commerce Analytics Example

```go
// Product performance report with multiple metrics
query := &orm.ComplexQuery{
    Select: []string{
        "products.id",
        "products.name",
        "categories.name as category_name",
        "COUNT(DISTINCT orders.id) as total_orders",
        "SUM(order_items.quantity) as units_sold",
        "SUM(order_items.quantity * order_items.price) as revenue",
        "AVG(order_items.price) as avg_price",
    },
    From: "products",
    Joins: []orm.Join{
        {
            Type:      orm.InnerJoin,
            Table:     "categories",
            Condition: "products.category_id = categories.id",
        },
        {
            Type:      orm.LeftJoin,
            Table:     "order_items",
            Condition: "products.id = order_items.product_id",
        },
        {
            Type:      orm.LeftJoin,
            Table:     "orders",
            Condition: "order_items.order_id = orders.id AND orders.status = 'completed'",
        },
    },
    Where: &orm.Condition{
        Field:    "products.active",
        Operator: "=",
        Value:    true,
    },
    GroupBy: []string{"products.id", "products.name", "categories.name"},
    Having:  "SUM(order_items.quantity) > 0",
    OrderBy: []string{"revenue DESC", "units_sold DESC"},
    Limit:   20,
}

topProducts, err := db.SelectManyComplex(query)
```

### Customer Segmentation with JOINs

```go
// Find VIP customers based on order history
query := &orm.ComplexQuery{
    Select: []string{
        "users.id",
        "users.name",
        "users.email",
        "COUNT(orders.id) as lifetime_orders",
        "SUM(orders.total) as lifetime_value",
        "AVG(orders.total) as avg_order_value",
        "MAX(orders.created_at) as last_order_date",
    },
    From: "users",
    Joins: []orm.Join{
        {
            Type:      orm.InnerJoin,
            Table:     "orders",
            Condition: "users.id = orders.user_id",
        },
    },
    Where: &orm.Condition{
        Logic: "AND",
        Nested: []orm.Condition{
            {Field: "users.status", Operator: "=", Value: "active"},
            {Field: "orders.status", Operator: "=", Value: "completed"},
        },
    },
    GroupBy: []string{"users.id", "users.name", "users.email"},
    Having:  "SUM(orders.total) > 1000 AND COUNT(orders.id) > 5",
    OrderBy: []string{"lifetime_value DESC"},
    Limit:   100,
}

vipCustomers, err := db.SelectManyComplex(query)
```

### Available JOIN Types

```go
// All supported JOIN types
orm.InnerJoin      // INNER JOIN
orm.LeftJoin       // LEFT JOIN
orm.RightJoin      // RIGHT JOIN
orm.FullJoin       // FULL OUTER JOIN
orm.CrossJoin      // CROSS JOIN (no ON condition needed)

// Example with different JOIN types
query := &orm.ComplexQuery{
    Select: []string{"users.*", "profiles.bio", "settings.preferences"},
    From:   "users",
    Joins: []orm.Join{
        {
            Type:      orm.LeftJoin,
            Table:     "profiles",
            Condition: "users.id = profiles.user_id",
        },
        {
            Type:      orm.LeftJoin,
            Table:     "settings",
            Condition: "users.id = settings.user_id",
        },
    },
}
```

### Security Features

All complex queries include built-in security:

- **Table name validation**: Prevents SQL injection in table names
- **Parameterized queries**: All WHERE values are parameterized
- **Operator whitelisting**: Only safe SQL operators are allowed
- **Field name validation**: Ensures valid SQL identifiers

```go
// This will fail with validation error
badQuery := &orm.ComplexQuery{
    From: "users; DROP TABLE users; --", // Invalid table name
}

_, err := db.SelectManyComplex(badQuery)
// Returns: ErrInvalidTableName

// This is safe - values are parameterized
safeQuery := &orm.ComplexQuery{
    Select: []string{"*"},
    From:   "users",
    Where: &orm.Condition{
        Field:    "email",
        Operator: "=",
        Value:    userInput, // Safely parameterized, no SQL injection risk
    },
}
```

## Batch Operations

### Bulk Insert Operations

```go
// Prepare multiple records
users := []orm.DBRecord{
    {
        TableName: "users",
        Data: map[string]interface{}{
            "name": "Alice", "email": "alice@example.com", "age": 25, "country": "USA",
        },
    },
    {
        TableName: "users",
        Data: map[string]interface{}{
            "name": "Bob", "email": "bob@example.com", "age": 30, "country": "Canada",
        },
    },
    {
        TableName: "users",
        Data: map[string]interface{}{
            "name": "Carol", "email": "carol@example.com", "age": 28, "country": "UK",
        },
    },
}

// Efficient batch insert for same table (automatically optimized)
results, err := db.InsertManyDBRecordsSameTable(users, false)
if err != nil {
    log.Fatal("Batch insert failed:", err)
}

// Check results
for i, result := range results {
    if result.Error != nil {
        fmt.Printf("Batch %d failed: %v\n", i, result.Error)
    } else {
        fmt.Printf("Batch %d: inserted %d records in %s ms\n", 
            i+1, result.RowsAffected, orm.SecondToMsString(result.Timing))
    }
}

// Get total performance metrics
totalTime := orm.TotalTimeElapsedInSecond(results)
fmt.Printf("Total database time: %s ms\n", orm.SecondToMsString(totalTime))
```

### Batch Insert with TableStruct

```go
// Using structs for type safety
users := []orm.TableStruct{
    User{Name: "David", Email: "david@example.com", Age: 35, Active: true, Country: "Australia"},
    User{Name: "Eve", Email: "eve@example.com", Age: 22, Active: true, Country: "Germany"},
    User{Name: "Frank", Email: "frank@example.com", Age: 40, Active: false, Country: "France"},
}

results, err := db.InsertManyTableStructs(users, false)
if err != nil {
    log.Fatal("TableStruct batch insert failed:", err)
}

fmt.Printf("Inserted %d users in %d batches\n", len(users), len(results))
```

### Configuring Batch Size

```go
// Check current batch size
fmt.Printf("Current max batch size: %d\n", orm.MAX_MULTIPLE_INSERTS)

// Adjust batch size based on your needs
orm.MAX_MULTIPLE_INSERTS = 50  // Smaller batches for memory-constrained environments

// Or increase for high-throughput scenarios  
orm.MAX_MULTIPLE_INSERTS = 200

// Large dataset example
largeDataset := make([]orm.DBRecord, 1000)
for i := range largeDataset {
    largeDataset[i] = orm.DBRecord{
        TableName: "users",
        Data: map[string]interface{}{
            "name":    fmt.Sprintf("User%d", i),
            "email":   fmt.Sprintf("user%d@test.com", i),
            "age":     20 + (i % 50),
            "active":  i%2 == 0,
            "country": []string{"USA", "Canada", "UK", "Germany", "France"}[i%5],
        },
    }
}

results, err := db.InsertManyDBRecordsSameTable(largeDataset, false)
if err != nil {
    log.Fatal("Large dataset insert failed:", err)
}

fmt.Printf("Inserted %d records in %d batches\n", len(largeDataset), len(results))
```

### Multiple Query Batches

```go
// Execute multiple queries in one batch
queries := []string{
    "SELECT COUNT(*) as total_users FROM users",
    "SELECT COUNT(*) as active_users FROM users WHERE active = 1",
    "SELECT AVG(age) as avg_age FROM users",
    "SELECT country, COUNT(*) as count FROM users GROUP BY country",
}

results, err := db.SelectManySQL(queries)
if err != nil {
    log.Fatal("Batch queries failed:", err)
}

// Process each result
for i, records := range results {
    fmt.Printf("Query %d results:\n", i+1)
    for _, record := range records {
        for key, value := range record.Data {
            fmt.Printf("  %s: %v\n", key, value)
        }
    }
    fmt.Println()
}
```

## Error Handling

### Standard Error Types

```go
// Common error patterns
record, err := db.SelectOnlyOneSQL("SELECT * FROM users WHERE id = 999")
if err != nil {
    switch err {
    case orm.ErrSQLNoRows:
        fmt.Println("No user found with that ID")
    case orm.ErrSQLMoreThanOneRow:
        fmt.Println("Multiple users found - expected only one")
    default:
        log.Fatal("Database error:", err)
    }
}

// Batch operation error handling
results, err := db.InsertManyDBRecords(records, false)
if err != nil {
    log.Fatal("Batch operation failed:", err)
}

successCount := 0
for i, result := range results {
    if result.Error != nil {
        fmt.Printf("Record %d failed: %v\n", i, result.Error)
    } else {
        successCount++
    }
}

fmt.Printf("Successfully inserted %d out of %d records\n", successCount, len(records))
```

### Connection Health Monitoring

```go
// Check database connection health
if !db.IsConnected() {
    log.Fatal("Database connection is not healthy")
}

// Get detailed status for monitoring
status, err := db.Status()
if err != nil {
    log.Printf("Could not get database status: %v", err)
} else {
    fmt.Printf("Database: %s %s\n", status.DBMS, status.Version)
    fmt.Printf("Uptime: %v\n", status.Uptime)
    fmt.Printf("Nodes: %d\n", status.Nodes)
    
    // Print full status for debugging
    status.PrintPretty()
}
```

## Performance Tips

### 1. Use Parameterized Queries
```go
// Good - safe and efficient
paramSQL := orm.ParametereizedSQL{
    Query:  "SELECT * FROM users WHERE age > ? AND country = ?",
    Values: []interface{}{25, "USA"},
}

// Avoid - potential SQL injection and less efficient
rawSQL := fmt.Sprintf("SELECT * FROM users WHERE age > %d AND country = '%s'", 25, "USA")
```

### 2. Batch Operations
```go
// Good - efficient batch insert
results, err := db.InsertManyDBRecordsSameTable(manyRecords, false)

// Avoid - multiple individual inserts
for _, record := range manyRecords {
    db.InsertOneDBRecord(record, false) // Less efficient
}
```

### 3. Use Conditions for Complex Queries
```go
// Good - structured and reusable
condition := &orm.Condition{
    Logic: "AND",
    Nested: []orm.Condition{
        {Field: "active", Operator: "=", Value: true},
        {Field: "age", Operator: ">=", Value: 18},
    },
    OrderBy: []string{"name ASC"},
    Limit:   100,
}

users, err := db.SelectManyWithCondition("users", condition)

// Also good - raw SQL when needed for complex operations
complexSQL := `
    SELECT u.*, COUNT(o.id) as order_count 
    FROM users u 
    LEFT JOIN orders o ON u.id = o.user_id 
    WHERE u.active = 1 
    GROUP BY u.id 
    HAVING order_count > 5
    ORDER BY order_count DESC
`
```

### 4. Pagination
```go
// Efficient pagination with conditions
condition := &orm.Condition{
    Field:   "active",
    Operator: "=", 
    Value:   true,
    OrderBy: []string{"created_at DESC"},
    Limit:   20,
    Offset:  page * 20, // page number * page size
}

pageResults, err := db.SelectManyWithCondition("users", condition)
```

### 5. Monitor Performance
```go
// Track query performance
start := time.Now()
results, err := db.SelectManyWithCondition("users", complexCondition)
elapsed := time.Since(start)

fmt.Printf("Query completed in %v\n", elapsed)
```

## Working with Schema

```go
// Get database schema information
schemas := db.GetSchema(true, true) // Hide SQL system tables

for _, schema := range schemas {
    if schema.ObjectType == "table" {
        fmt.Printf("Table: %s\n", schema.TableName)
        
        // Print debug info including SQL
        schema.PrintDebug(true)
    }
}
```

## Converting Between Types

```go
// Convert struct to DBRecord
user := User{Name: "Test", Email: "test@example.com", Age: 25}
record, err := orm.TableStructToDBRecord(user)
if err != nil {
    log.Fatal(err)
}

// Convert DBRecord to parameterized SQL
sql, values := record.ToInsertSQLParameterized()
fmt.Printf("SQL: %s\n", sql)
fmt.Printf("Values: %v\n", values)

// Convert to raw SQL (for debugging)
rawSQL, _ := record.ToInsertSQLRaw()
fmt.Printf("Raw SQL: %s\n", rawSQL)
```

This comprehensive guide covers all the major features of SimpleORM. For implementation-specific details, see the individual database implementation READMEs (e.g., RQLite implementation).