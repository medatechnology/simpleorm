# Creating a New Database Implementation for SimpleORM

This guide walks you through creating a new database implementation for SimpleORM. We'll use MySQL as an example, but the same principles apply to PostgreSQL, SQL Server, or any other database.

## Table of Contents

- [Overview](#overview)
- [Project Structure](#project-structure)
- [Step-by-Step Implementation](#step-by-step-implementation)
- [Required Interface Methods](#required-interface-methods)
- [Implementation Examples](#implementation-examples)
- [Testing Your Implementation](#testing-your-implementation)
- [Best Practices](#best-practices)
- [Common Patterns](#common-patterns)

## Overview

To add a new database to SimpleORM, you need to:

1. Create a new package for your database (e.g., `mysql`, `postgresql`)
2. Implement the `orm.Database` interface
3. Create configuration structures
4. Add helper functions for data conversion
5. Handle database-specific features and error handling

## Project Structure

Create your implementation in the following structure:

```
/mysql
├── mysql.go          // Main implementation
├── config.go         // Configuration structures  
├── models.go         // Database-specific models
├── helpers.go        // Helper functions
├── errors.go         // Error handling
├── README.md         // MySQL-specific documentation
└── examples/
    ├── basic_usage.go
    └── advanced_features.go
```

## Step-by-Step Implementation

### Step 1: Create Configuration Structure

Create `config.go`:

```go
package mysql

import (
    "time"
)

// MySQLConfig holds configuration for MySQL connections
type MySQLConfig struct {
    Host         string        `json:"host"`
    Port         int           `json:"port"`
    Database     string        `json:"database"`
    Username     string        `json:"username"`
    Password     string        `json:"password"`
    
    // Connection pool settings
    MaxOpenConns    int           `json:"max_open_conns"`
    MaxIdleConns    int           `json:"max_idle_conns"`
    ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
    ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
    
    // Timeouts
    ConnectTimeout time.Duration `json:"connect_timeout"`
    ReadTimeout    time.Duration `json:"read_timeout"`
    WriteTimeout   time.Duration `json:"write_timeout"`
    
    // SSL settings
    SSLMode    string `json:"ssl_mode"`    // "disabled", "required", "verify-ca", "verify-full"
    SSLCert    string `json:"ssl_cert"`
    SSLKey     string `json:"ssl_key"`
    SSLRootCA  string `json:"ssl_root_ca"`
    
    // Additional parameters
    Charset   string `json:"charset"`
    Collation string `json:"collation"`
    Location  string `json:"location"`
    
    // Performance settings
    ParseTime       bool `json:"parse_time"`
    AllowNativePasswords bool `json:"allow_native_passwords"`
}

// DefaultMySQLConfig returns a configuration with sensible defaults
func DefaultMySQLConfig() MySQLConfig {
    return MySQLConfig{
        Host:         "localhost",
        Port:         3306,
        Database:     "",
        Username:     "",
        Password:     "",
        
        MaxOpenConns:    25,
        MaxIdleConns:    5,
        ConnMaxLifetime: 5 * time.Minute,
        ConnMaxIdleTime: 2 * time.Minute,
        
        ConnectTimeout: 10 * time.Second,
        ReadTimeout:    30 * time.Second,
        WriteTimeout:   30 * time.Second,
        
        SSLMode:   "disabled",
        Charset:   "utf8mb4",
        Collation: "utf8mb4_unicode_ci",
        Location:  "UTC",
        
        ParseTime:            true,
        AllowNativePasswords: true,
    }
}

// DSN generates a MySQL Data Source Name from the configuration
func (c MySQLConfig) DSN() string {
    // Build DSN string
    // Format: [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
    
    dsn := ""
    
    // Add credentials
    if c.Username != "" {
        dsn += c.Username
        if c.Password != "" {
            dsn += ":" + c.Password
        }
        dsn += "@"
    }
    
    // Add host and port
    dsn += fmt.Sprintf("tcp(%s:%d)", c.Host, c.Port)
    
    // Add database
    dsn += "/" + c.Database
    
    // Add parameters
    params := make([]string, 0)
    
    if c.Charset != "" {
        params = append(params, "charset="+c.Charset)
    }
    if c.Collation != "" {
        params = append(params, "collation="+c.Collation)
    }
    if c.Location != "" {
        params = append(params, "loc="+url.QueryEscape(c.Location))
    }
    if c.ParseTime {
        params = append(params, "parseTime=true")
    }
    if c.AllowNativePasswords {
        params = append(params, "allowNativePasswords=true")
    }
    
    // Add timeouts
    if c.ConnectTimeout > 0 {
        params = append(params, fmt.Sprintf("timeout=%s", c.ConnectTimeout))
    }
    if c.ReadTimeout > 0 {
        params = append(params, fmt.Sprintf("readTimeout=%s", c.ReadTimeout))
    }
    if c.WriteTimeout > 0 {
        params = append(params, fmt.Sprintf("writeTimeout=%s", c.WriteTimeout))
    }
    
    // Add SSL settings
    if c.SSLMode != "" {
        params = append(params, "tls="+c.SSLMode)
    }
    
    if len(params) > 0 {
        dsn += "?" + strings.Join(params, "&")
    }
    
    return dsn
}
```

### Step 2: Create Models and Constants

Create `models.go`:

```go
package mysql

import (
    "database/sql"
    "time"
    
    orm "github.com/medatechnology/simpleorm"
)

const (
    DEFAULT_MAX_OPEN_CONNECTIONS = 25
    DEFAULT_MAX_IDLE_CONNECTIONS = 5
    DEFAULT_CONNECTION_TIMEOUT   = 10 * time.Second
    DEFAULT_QUERY_TIMEOUT        = 30 * time.Second
    
    // MySQL specific constants
    MYSQL_DRIVER_NAME = "mysql"
    MYSQL_PORT        = 3306
    
    // Schema queries
    SCHEMA_QUERY = `
        SELECT 
            TABLE_NAME as table_name,
            TABLE_TYPE as table_type,
            TABLE_COMMENT as table_comment,
            ENGINE as engine,
            TABLE_COLLATION as collation
        FROM INFORMATION_SCHEMA.TABLES 
        WHERE TABLE_SCHEMA = ? 
        ORDER BY TABLE_NAME`
    
    COLUMNS_QUERY = `
        SELECT 
            COLUMN_NAME as column_name,
            DATA_TYPE as data_type,
            IS_NULLABLE as is_nullable,
            COLUMN_DEFAULT as column_default,
            COLUMN_KEY as column_key,
            EXTRA as extra,
            COLUMN_COMMENT as column_comment
        FROM INFORMATION_SCHEMA.COLUMNS 
        WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
        ORDER BY ORDINAL_POSITION`
)

// MySQLDB implements the orm.Database interface for MySQL
type MySQLDB struct {
    Config MySQLConfig
    DB     *sql.DB
    
    // Internal state
    connected bool
    dbName    string
}

// MySQLTableInfo represents MySQL table information
type MySQLTableInfo struct {
    TableName    string `db:"table_name"`
    TableType    string `db:"table_type"`
    TableComment string `db:"table_comment"`
    Engine       string `db:"engine"`
    Collation    string `db:"collation"`
}

// MySQLColumnInfo represents MySQL column information
type MySQLColumnInfo struct {
    ColumnName    string         `db:"column_name"`
    DataType      string         `db:"data_type"`
    IsNullable    string         `db:"is_nullable"`
    ColumnDefault sql.NullString `db:"column_default"`
    ColumnKey     string         `db:"column_key"`
    Extra         string         `db:"extra"`
    ColumnComment string         `db:"column_comment"`
}

// MySQLStatus represents MySQL server status
type MySQLStatus struct {
    Version       string
    Uptime        time.Duration
    Connections   int
    QueriesPerSec float64
    SlowQueries   int64
}
```

### Step 3: Create Main Implementation

Create `mysql.go`:

```go
package mysql

import (
    "database/sql"
    "fmt"
    "strings"
    "time"
    
    _ "github.com/go-sql-driver/mysql" // MySQL driver
    orm "github.com/medatechnology/simpleorm"
)

// NewDatabase creates a new MySQL database instance
func NewDatabase(config MySQLConfig) (*MySQLDB, error) {
    // Apply defaults if needed
    if config.Host == "" {
        defaultConfig := DefaultMySQLConfig()
        config.Host = defaultConfig.Host
        config.Port = defaultConfig.Port
    }
    
    // Create DSN
    dsn := config.DSN()
    
    // Open database connection
    db, err := sql.Open(MYSQL_DRIVER_NAME, dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(config.MaxOpenConns)
    db.SetMaxIdleConns(config.MaxIdleConns)
    db.SetConnMaxLifetime(config.ConnMaxLifetime)
    db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
    
    mysqlDB := &MySQLDB{
        Config: config,
        DB:     db,
        dbName: config.Database,
    }
    
    // Test connection
    if err := mysqlDB.ping(); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to ping MySQL: %w", err)
    }
    
    mysqlDB.connected = true
    return mysqlDB, nil
}

// ping tests the database connection
func (db *MySQLDB) ping() error {
    ctx, cancel := context.WithTimeout(context.Background(), db.Config.ConnectTimeout)
    defer cancel()
    
    return db.DB.PingContext(ctx)
}

// IsConnected checks if the database connection is alive
func (db *MySQLDB) IsConnected() bool {
    if !db.connected {
        return false
    }
    
    if err := db.ping(); err != nil {
        db.connected = false
        return false
    }
    
    return true
}

// Close closes the database connection
func (db *MySQLDB) Close() error {
    db.connected = false
    return db.DB.Close()
}

// GetSchema returns the database schema
func (db *MySQLDB) GetSchema(hideSQL, hideSureSQL bool) []orm.SchemaStruct {
    rows, err := db.DB.Query(SCHEMA_QUERY, db.dbName)
    if err != nil {
        return []orm.SchemaStruct{}
    }
    defer rows.Close()
    
    var schemas []orm.SchemaStruct
    
    for rows.Next() {
        var table MySQLTableInfo
        err := rows.Scan(
            &table.TableName,
            &table.TableType,
            &table.TableComment,
            &table.Engine,
            &table.Collation,
        )
        if err != nil {
            continue
        }
        
        // Apply filters
        if hideSQL && strings.HasPrefix(table.TableName, "mysql_") {
            continue
        }
        if hideSureSQL && strings.HasPrefix(table.TableName, "_") {
            continue
        }
        
        schema := orm.SchemaStruct{
            ObjectType: strings.ToLower(table.TableType),
            ObjectName: table.TableName,
            TableName:  table.TableName,
            SQLCommand: fmt.Sprintf("-- %s (%s)", table.TableComment, table.Engine),
        }
        
        schemas = append(schemas, schema)
    }
    
    return schemas
}

// Status returns the status of the MySQL server
func (db *MySQLDB) Status() (orm.NodeStatusStruct, error) {
    status := orm.NodeStatusStruct{}
    status.DBMS = "mysql"
    status.DBMSDriver = "mysql-driver"
    status.URL = fmt.Sprintf("%s:%d", db.Config.Host, db.Config.Port)
    
    // Get MySQL version
    var version string
    err := db.DB.QueryRow("SELECT VERSION()").Scan(&version)
    if err == nil {
        status.Version = version
    }
    
    // Get uptime
    var uptime int64
    err = db.DB.QueryRow("SHOW STATUS LIKE 'Uptime'").Scan(&uptime)
    if err == nil {
        status.Uptime = time.Duration(uptime) * time.Second
    }
    
    // Connection pool info
    stats := db.DB.Stats()
    status.MaxPool = stats.MaxOpenConnections
    
    status.Nodes = 1 // Single MySQL instance
    status.IsLeader = true
    
    return status, nil
}

// Leader returns the leader (not applicable for MySQL)
func (db *MySQLDB) Leader() (string, error) {
    return fmt.Sprintf("%s:%d", db.Config.Host, db.Config.Port), nil
}

// Peers returns empty slice (MySQL is not distributed)
func (db *MySQLDB) Peers() ([]string, error) {
    return []string{}, nil
}

// SelectOne selects a single record from the table
func (db *MySQLDB) SelectOne(tableName string) (orm.DBRecord, error) {
    query := fmt.Sprintf("SELECT * FROM %s LIMIT 1", tableName)
    
    rows, err := db.DB.Query(query)
    if err != nil {
        return orm.DBRecord{}, err
    }
    defer rows.Close()
    
    if !rows.Next() {
        return orm.DBRecord{}, orm.ErrSQLNoRows
    }
    
    return scanRowToDBRecord(rows, tableName)
}

// SelectMany selects multiple records from the table
func (db *MySQLDB) SelectMany(tableName string) (orm.DBRecords, error) {
    query := fmt.Sprintf("SELECT * FROM %s", tableName)
    
    rows, err := db.DB.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    return scanRowsToDBRecords(rows, tableName)
}

// SelectOneWithCondition selects a single record with a condition
func (db *MySQLDB) SelectOneWithCondition(tableName string, condition *orm.Condition) (orm.DBRecord, error) {
    if condition == nil {
        return db.SelectOne(tableName)
    }
    
    query, params := condition.ToSelectString(tableName)
    // Add LIMIT 1 if not already specified
    if !strings.Contains(strings.ToUpper(query), "LIMIT") {
        query += " LIMIT 1"
    }
    
    rows, err := db.DB.Query(query, params...)
    if err != nil {
        return orm.DBRecord{}, err
    }
    defer rows.Close()
    
    if !rows.Next() {
        return orm.DBRecord{}, orm.ErrSQLNoRows
    }
    
    return scanRowToDBRecord(rows, tableName)
}

// SelectManyWithCondition selects multiple records with a condition
func (db *MySQLDB) SelectManyWithCondition(tableName string, condition *orm.Condition) ([]orm.DBRecord, error) {
    if condition == nil {
        return db.SelectMany(tableName)
    }
    
    query, params := condition.ToSelectString(tableName)
    
    rows, err := db.DB.Query(query, params...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    return scanRowsToDBRecords(rows, tableName)
}

// SelectOneSQL executes a single SQL query and returns the results
func (db *MySQLDB) SelectOneSQL(sql string) (orm.DBRecords, error) {
    rows, err := db.DB.Query(sql)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    tableName := extractTableNameFromSQL(sql)
    return scanRowsToDBRecords(rows, tableName)
}

// SelectOnlyOneSQL executes a SQL query and ensures exactly one row is returned
func (db *MySQLDB) SelectOnlyOneSQL(sql string) (orm.DBRecord, error) {
    rows, err := db.DB.Query(sql)
    if err != nil {
        return orm.DBRecord{}, err
    }
    defer rows.Close()
    
    if !rows.Next() {
        return orm.DBRecord{}, orm.ErrSQLNoRows
    }
    
    tableName := extractTableNameFromSQL(sql)
    record, err := scanRowToDBRecord(rows, tableName)
    if err != nil {
        return orm.DBRecord{}, err
    }
    
    // Check if there's another row
    if rows.Next() {
        return orm.DBRecord{}, orm.ErrSQLMoreThanOneRow
    }
    
    return record, nil
}

// SelectOneSQLParameterized executes a single parameterized SQL query
func (db *MySQLDB) SelectOneSQLParameterized(paramSQL orm.ParametereizedSQL) (orm.DBRecords, error) {
    rows, err := db.DB.Query(paramSQL.Query, paramSQL.Values...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    tableName := extractTableNameFromSQL(paramSQL.Query)
    return scanRowsToDBRecords(rows, tableName)
}

// SelectManySQLParameterized executes multiple parameterized SQL queries
func (db *MySQLDB) SelectManySQLParameterized(paramSQLs []orm.ParametereizedSQL) ([]orm.DBRecords, error) {
    results := make([]orm.DBRecords, 0, len(paramSQLs))
    
    for _, paramSQL := range paramSQLs {
        records, err := db.SelectOneSQLParameterized(paramSQL)
        if err != nil {
            return results, err
        }
        results = append(results, records)
    }
    
    return results, nil
}

// SelectOnlyOneSQLParameterized executes a parameterized SQL query and ensures exactly one row
func (db *MySQLDB) SelectOnlyOneSQLParameterized(paramSQL orm.ParametereizedSQL) (orm.DBRecord, error) {
    rows, err := db.DB.Query(paramSQL.Query, paramSQL.Values...)
    if err != nil {
        return orm.DBRecord{}, err
    }
    defer rows.Close()
    
    if !rows.Next() {
        return orm.DBRecord{}, orm.ErrSQLNoRows
    }
    
    tableName := extractTableNameFromSQL(paramSQL.Query)
    record, err := scanRowToDBRecord(rows, tableName)
    if err != nil {
        return orm.DBRecord{}, err
    }
    
    if rows.Next() {
        return orm.DBRecord{}, orm.ErrSQLMoreThanOneRow
    }
    
    return record, nil
}

// ExecOneSQL executes a single SQL statement
func (db *MySQLDB) ExecOneSQL(sql string) orm.BasicSQLResult {
    start := time.Now()
    
    result, err := db.DB.Exec(sql)
    
    timing := time.Since(start).Seconds()
    
    if err != nil {
        return orm.BasicSQLResult{
            Error:  err,
            Timing: timing,
        }
    }
    
    lastInsertID, _ := result.LastInsertId()
    rowsAffected, _ := result.RowsAffected()
    
    return orm.BasicSQLResult{
        Error:        nil,
        Timing:       timing,
        RowsAffected: int(rowsAffected),
        LastInsertID: int(lastInsertID),
    }
}

// ExecOneSQLParameterized executes a single parameterized SQL statement
func (db *MySQLDB) ExecOneSQLParameterized(paramSQL orm.ParametereizedSQL) orm.BasicSQLResult {
    start := time.Now()
    
    result, err := db.DB.Exec(paramSQL.Query, paramSQL.Values...)
    
    timing := time.Since(start).Seconds()
    
    if err != nil {
        return orm.BasicSQLResult{
            Error:  err,
            Timing: timing,
        }
    }
    
    lastInsertID, _ := result.LastInsertId()
    rowsAffected, _ := result.RowsAffected()
    
    return orm.BasicSQLResult{
        Error:        nil,
        Timing:       timing,
        RowsAffected: int(rowsAffected),
        LastInsertID: int(lastInsertID),
    }
}

// ExecManySQL executes multiple SQL statements
func (db *MySQLDB) ExecManySQL(sqls []string) ([]orm.BasicSQLResult, error) {
    results := make([]orm.BasicSQLResult, 0, len(sqls))
    
    for _, sql := range sqls {
        result := db.ExecOneSQL(sql)
        results = append(results, result)
        
        if result.Error != nil {
            return results, result.Error
        }
    }
    
    return results, nil
}

// ExecManySQLParameterized executes multiple parameterized SQL statements
func (db *MySQLDB) ExecManySQLParameterized(paramSQLs []orm.ParametereizedSQL) ([]orm.BasicSQLResult, error) {
    results := make([]orm.BasicSQLResult, 0, len(paramSQLs))
    
    for _, paramSQL := range paramSQLs {
        result := db.ExecOneSQLParameterized(paramSQL)
        results = append(results, result)
        
        if result.Error != nil {
            return results, result.Error
        }
    }
    
    return results, nil
}

// InsertOneDBRecord inserts a single record
func (db *MySQLDB) InsertOneDBRecord(record orm.DBRecord, queue bool) orm.BasicSQLResult {
    sql, values := record.ToInsertSQLParameterized()
    
    // MySQL doesn't have a queue mechanism like RQLite, so we ignore the queue parameter
    return db.ExecOneSQLParameterized(orm.ParametereizedSQL{
        Query:  sql,
        Values: values,
    })
}

// InsertManyDBRecords inserts multiple records
func (db *MySQLDB) InsertManyDBRecords(records []orm.DBRecord, queue bool) ([]orm.BasicSQLResult, error) {
    var results []orm.BasicSQLResult
    
    for _, record := range records {
        result := db.InsertOneDBRecord(record, queue)
        results = append(results, result)
        
        if result.Error != nil {
            return results, result.Error
        }
    }
    
    return results, nil
}

// InsertManyDBRecordsSameTable inserts multiple records into the same table efficiently
func (db *MySQLDB) InsertManyDBRecordsSameTable(records []orm.DBRecord, queue bool) ([]orm.BasicSQLResult, error) {
    if len(records) == 0 {
        return nil, fmt.Errorf("no records to insert")
    }
    
    // Use the DBRecords batch insert functionality
    paramSQLs := orm.DBRecords(records).ToInsertSQLParameterized()
    
    return db.ExecManySQLParameterized(paramSQLs)
}

// InsertOneTableStruct inserts a single table struct
func (db *MySQLDB) InsertOneTableStruct(obj orm.TableStruct, queue bool) orm.BasicSQLResult {
    record, err := orm.TableStructToDBRecord(obj)
    if err != nil {
        return orm.BasicSQLResult{Error: err}
    }
    
    return db.InsertOneDBRecord(record, queue)
}

// InsertManyTableStructs inserts multiple table structs
func (db *MySQLDB) InsertManyTableStructs(objs []orm.TableStruct, queue bool) ([]orm.BasicSQLResult, error) {
    if len(objs) == 0 {
        return nil, fmt.Errorf("no objects to insert")
    }
    
    records := make([]orm.DBRecord, len(objs))
    for i, obj := range objs {
        record, err := orm.TableStructToDBRecord(obj)
        if err != nil {
            return nil, err
        }
        records[i] = record
    }
    
    // Check if all records are for the same table
    sameTables := true
    tableName := records[0].TableName
    for i := 1; i < len(records); i++ {
        if records[i].TableName != tableName {
            sameTables = false
            break
        }
    }
    
    if sameTables {
        return db.InsertManyDBRecordsSameTable(records, queue)
    }
    
    return db.InsertManyDBRecords(records, queue)
}

// SelectManySQL executes multiple SQL queries and returns the results of each
func (db *MySQLDB) SelectManySQL(sqls []string) ([]orm.DBRecords, error) {
    results := make([]orm.DBRecords, 0, len(sqls))
    
    for _, sql := range sqls {
        records, err := db.SelectOneSQL(sql)
        if err != nil {
            if err == orm.ErrSQLNoRows {
                // Append empty slice for no results instead of failing
                results = append(results, orm.DBRecords{})
                continue
            }
            return results, err
        }
        results = append(results, records)
    }
    
    return results, nil
}
```

### Step 4: Create Helper Functions

Create `helpers.go`:

```go
package mysql

import (
    "database/sql"
    "fmt"
    "reflect"
    "strings"
    "time"
    
    orm "github.com/medatechnology/simpleorm"
)

// scanRowToDBRecord converts a single sql.Rows to a DBRecord
func scanRowToDBRecord(rows *sql.Rows, tableName string) (orm.DBRecord, error) {
    columns, err := rows.Columns()
    if err != nil {
        return orm.DBRecord{}, err
    }
    
    // Create slice to hold column values
    values := make([]interface{}, len(columns))
    valuePtrs := make([]interface{}, len(columns))
    
    for i := range columns {
        valuePtrs[i] = &values[i]
    }
    
    // Scan the row
    err = rows.Scan(valuePtrs...)
    if err != nil {
        return orm.DBRecord{}, err
    }
    
    // Convert to map
    data := make(map[string]interface{})
    for i, col := range columns {
        data[col] = convertMySQLValue(values[i])
    }
    
    return orm.DBRecord{
        TableName: tableName,
        Data:      data,
    }, nil
}

// scanRowsToDBRecords converts sql.Rows to DBRecords
func scanRowsToDBRecords(rows *sql.Rows, tableName string) (orm.DBRecords, error) {
    var records orm.DBRecords
    
    for rows.Next() {
        record, err := scanRowToDBRecord(rows, tableName)
        if err != nil {
            return nil, err
        }
        records = append(records, record)
    }
    
    if len(records) == 0 {
        return nil, orm.ErrSQLNoRows
    }
    
    return records, nil
}

// convertMySQLValue converts MySQL-specific types to Go types
func convertMySQLValue(value interface{}) interface{} {
    if value == nil {
        return nil
    }
    
    switch v := value.(type) {
    case []byte:
        // MySQL returns many values as []byte, convert to string if it's text
        str := string(v)
        
        // Try to parse as time if it looks like a timestamp
        if len(str) >= 10 && (strings.Contains(str, "-") || strings.Contains(str, ":")) {
            if t, err := parseTime(str); err == nil {
                return t
            }
        }
        
        return str
        
    case sql.NullString:
        if v.Valid {
            return v.String
        }
        return nil
        
    case sql.NullInt64:
        if v.Valid {
            return v.Int64
        }
        return nil
        
    case sql.NullFloat64:
        if v.Valid {
            return v.Float64
        }
        return nil
        
    case sql.NullTime:
        if v.Valid {
            return v.Time
        }
        return nil
        
    case sql.NullBool:
        if v.Valid {
            return v.Bool
        }
        return nil
        
    default:
        return value
    }
}

// parseTime attempts to parse various MySQL time formats
func parseTime(str string) (time.Time, error) {
    formats := []string{
        "2006-01-02 15:04:05",
        "2006-01-02T15:04:05Z",
        "2006-01-02T15:04:05.000Z",
        "2006-01-02",
        "15:04:05",
        time.RFC3339,
        time.RFC3339Nano,
    }
    
    for _, format := range formats {
        if t, err := time.Parse(format, str); err == nil {
            return t, nil
        }
    }
    
    return time.Time{}, fmt.Errorf("unable to parse time: %s", str)
}

// extractTableNameFromSQL attempts to extract table name from SQL query
func extractTableNameFromSQL(sql string) string {
    // Simple extraction - this could be made more sophisticated
    upperSQL := strings.ToUpper(strings.TrimSpace(sql))
    
    if strings.HasPrefix(upperSQL, "SELECT") {
        // Look for FROM clause
        if idx := strings.Index(upperSQL, "FROM"); idx != -1 {
            fromPart := strings.TrimSpace(upperSQL[idx+4:])
            
            // Get the first word after FROM
            parts := strings.Fields(fromPart)
            if len(parts) > 0 {
                tableName := parts[0]
                
                // Remove common SQL keywords that might follow table name
                keywords := []string{"WHERE", "JOIN", "INNER", "LEFT", "RIGHT", "GROUP", "ORDER", "LIMIT"}
                for _, keyword := range keywords {
                    if strings.HasPrefix(tableName, keyword) {
                        break
                    }
                }
                
                // Clean up table name (remove quotes, etc.)
                tableName = strings.Trim(tableName, "`\"'[]")
                return strings.ToLower(tableName)
            }
        }
    }
    
    return "unknown"
}

// buildMySQLInsertSQL builds MySQL-specific bulk insert SQL
func buildMySQLInsertSQL(records []orm.DBRecord) (string, []interface{}, error) {
    if len(records) == 0 {
        return "", nil, fmt.Errorf("no records provided")
    }
    
    tableName := records[0].TableName
    
    // Get column names from first record
    var columns []string
    for col := range records[0].Data {
        columns = append(columns, col)
    }
    
    // Build INSERT statement
    var sb strings.Builder
    sb.WriteString("INSERT INTO ")
    sb.WriteString(tableName)
    sb.WriteString(" (")
    
    for i, col := range columns {
        if i > 0 {
            sb.WriteString(", ")
        }
        sb.WriteString("`")
        sb.WriteString(col)
        sb.WriteString("`")
    }
    
    sb.WriteString(") VALUES ")
    
    // Build VALUES clause
    var values []interface{}
    for i, record := range records {
        if i > 0 {
            sb.WriteString(", ")
        }
        
        sb.WriteString("(")
        for j, col := range columns {
            if j > 0 {
                sb.WriteString(", ")
            }
            sb.WriteString("?")
            values = append(values, record.Data[col])
        }
        sb.WriteString(")")
    }
    
    return sb.String(), values, nil
}

// validateMySQLConnection validates the MySQL connection and returns detailed info
func validateMySQLConnection(db *sql.DB) error {
    // Test basic connectivity
    if err := db.Ping(); err != nil {
        return fmt.Errorf("ping failed: %w", err)
    }
    
    // Test a simple query
    var version string
    err := db.QueryRow("SELECT VERSION()").Scan(&version)
    if err != nil {
        return fmt.Errorf("version query failed: %w", err)
    }
    
    return nil
}
```

### Step 5: Create Error Handling

Create `errors.go`:

```go
package mysql

import (
    "database/sql/driver"
    "errors"
    "fmt"
    "strings"
    
    "github.com/go-sql-driver/mysql"
    orm "github.com/medatechnology/simpleorm"
)

// MySQL-specific error codes
const (
    // Common MySQL error codes
    ER_DUP_ENTRY                = 1062
    ER_NO_SUCH_TABLE           = 1146
    ER_TABLE_EXISTS_ERROR      = 1050
    ER_BAD_FIELD_ERROR         = 1054
    ER_PARSE_ERROR             = 1064
    ER_ACCESS_DENIED_ERROR     = 1045
    ER_DBACCESS_DENIED_ERROR   = 1044
    ER_TABLEACCESS_DENIED_ERROR = 1142
    ER_TOO_MANY_CONNECTIONS    = 1040
    ER_LOCK_WAIT_TIMEOUT       = 1205
    ER_LOCK_DEADLOCK           = 1213
)

// MySQLError represents a MySQL-specific error
type MySQLError struct {
    Code    uint16
    Message string
    State   string
    Err     error
}

func (e MySQLError) Error() string {
    return fmt.Sprintf("MySQL Error %d (%s): %s", e.Code, e.State, e.Message)
}

func (e MySQLError) Unwrap() error {
    return e.Err
}

// WrapMySQLError wraps a MySQL driver error into our custom error type
func WrapMySQLError(err error) error {
    if err == nil {
        return nil
    }
    
    // Check if it's already a MySQL error
    var mysqlErr *mysql.MySQLError
    if errors.As(err, &mysqlErr) {
        return MySQLError{
            Code:    mysqlErr.Number,
            Message: mysqlErr.Message,
            State:   mysqlErr.SQLState[0:5],
            Err:     err,
        }
    }
    
    // Handle driver.ErrBadConn
    if errors.Is(err, driver.ErrBadConn) {
        return fmt.Errorf("bad MySQL connection: %w", err)
    }
    
    // Handle other common errors
    errStr := err.Error()
    switch {
    case strings.Contains(errStr, "connection refused"):
        return fmt.Errorf("MySQL connection refused - check if server is running: %w", err)
    case strings.Contains(errStr, "timeout"):
        return fmt.Errorf("MySQL operation timeout: %w", err)
    case strings.Contains(errStr, "no such host"):
        return fmt.Errorf("MySQL host not found: %w", err)
    default:
        return err
    }
}

// IsDuplicateKeyError checks if the error is a duplicate key error
func IsDuplicateKeyError(err error) bool {
    var mysqlErr MySQLError
    if errors.As(err, &mysqlErr) {
        return mysqlErr.Code == ER_DUP_ENTRY
    }
    return false
}

// IsTableNotExistError checks if the error is a table not exist error
func IsTableNotExistError(err error) bool {
    var mysqlErr MySQLError
    if errors.As(err, &mysqlErr) {
        return mysqlErr.Code == ER_NO_SUCH_TABLE
    }
    return false
}

// IsConnectionError checks if the error is related to connection issues
func IsConnectionError(err error) bool {
    if err == nil {
        return false
    }
    
    var mysqlErr MySQLError
    if errors.As(err, &mysqlErr) {
        return mysqlErr.Code == ER_TOO_MANY_CONNECTIONS ||
               mysqlErr.Code == ER_ACCESS_DENIED_ERROR ||
               mysqlErr.Code == ER_DBACCESS_DENIED_ERROR
    }
    
    errStr := err.Error()
    return strings.Contains(errStr, "connection") ||
           strings.Contains(errStr, "timeout") ||
           strings.Contains(errStr, "refused")
}
```

## Required Interface Methods

Your implementation must provide all methods from the `orm.Database` interface:

### Schema and Status Methods
- `GetSchema(hideSQL, hideSureSQL bool) []SchemaStruct`
- `Status() (NodeStatusStruct, error)`
- `IsConnected() bool`
- `Leader() (string, error)`
- `Peers() ([]string, error)`

### Select Methods  
- `SelectOne(tableName string) (DBRecord, error)`
- `SelectMany(tableName string) (DBRecords, error)`
- `SelectOneWithCondition(tableName string, condition *Condition) (DBRecord, error)`
- `SelectManyWithCondition(tableName string, condition *Condition) ([]DBRecord, error)`

### Raw SQL Methods
- `SelectOneSQL(sql string) (DBRecords, error)`
- `SelectManySQL(sqls []string) ([]DBRecords, error)`
- `SelectOnlyOneSQL(sql string) (DBRecord, error)`
- `SelectOneSQLParameterized(paramSQL ParametereizedSQL) (DBRecords, error)`
- `SelectManySQLParameterized(paramSQLs []ParametereizedSQL) ([]DBRecords, error)`
- `SelectOnlyOneSQLParameterized(paramSQL ParametereizedSQL) (DBRecord, error)`

### Execute Methods
- `ExecOneSQL(sql string) BasicSQLResult`
- `ExecOneSQLParameterized(paramSQL ParametereizedSQL) BasicSQLResult`
- `ExecManySQL(sqls []string) ([]BasicSQLResult, error)`
- `ExecManySQLParameterized(paramSQLs []ParametereizedSQL) ([]BasicSQLResult, error)`

### Insert Methods
- `InsertOneDBRecord(record DBRecord, queue bool) BasicSQLResult`
- `InsertManyDBRecords(records []DBRecord, queue bool) ([]BasicSQLResult, error)`
- `InsertManyDBRecordsSameTable(records []DBRecord, queue bool) ([]BasicSQLResult, error)`
- `InsertOneTableStruct(obj TableStruct, queue bool) BasicSQLResult`
- `InsertManyTableStructs(objs []TableStruct, queue bool) ([]BasicSQLResult, error)`

## Testing Your Implementation

### Step 6: Create Basic Tests

Create `mysql_test.go`:

```go
package mysql

import (
    "testing"
    "time"
    
    orm "github.com/medatechnology/simpleorm"
)

// Test user struct
type TestUser struct {
    ID    int    `json:"id" db:"id"`
    Name  string `json:"name" db:"name"`
    Email string `json:"email" db:"email"`
    Age   int    `json:"age" db:"age"`
}

func (u TestUser) TableName() string {
    return "test_users"
}

func TestMySQLConnection(t *testing.T) {
    config := DefaultMySQLConfig()
    config.Database = "test_db"
    config.Username = "test_user"
    config.Password = "test_pass"
    
    db, err := NewDatabase(config)
    if err != nil {
        t.Skipf("MySQL not available: %v", err)
    }
    defer db.Close()
    
    if !db.IsConnected() {
        t.Error("Database should be connected")
    }
}

func TestMySQLSchema(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    schemas := db.GetSchema(false, false)
    if len(schemas) == 0 {
        t.Error("Expected at least one schema object")
    }
}

func TestMySQLInsertAndSelect(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    // Create test table
    createSQL := `
    CREATE TABLE IF NOT EXISTS test_users (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        age INT
    )`
    
    result := db.ExecOneSQL(createSQL)
    if result.Error != nil {
        t.Fatal("Failed to create test table:", result.Error)
    }
    
    // Test insert
    user := TestUser{
        Name:  "John Doe",
        Email: "john@test.com",
        Age:   30,
    }
    
    insertResult := db.InsertOneTableStruct(user, false)
    if insertResult.Error != nil {
        t.Fatal("Failed to insert user:", insertResult.Error)
    }
    
    if insertResult.LastInsertID <= 0 {
        t.Error("Expected positive last insert ID")
    }
    
    // Test select
    users, err := db.SelectMany("test_users")
    if err != nil {
        t.Fatal("Failed to select users:", err)
    }
    
    if len(users) == 0 {
        t.Error("Expected at least one user")
    }
    
    // Clean up
    db.ExecOneSQL("DROP TABLE test_users")
}

func TestMySQLConditions(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    // Setup test data
    setupTestData(t, db)
    
    // Test simple condition
    condition := &orm.Condition{
        Field:    "age",
        Operator: ">",
        Value:    25,
    }
    
    users, err := db.SelectManyWithCondition("test_users", condition)
    if err != nil {
        t.Fatal("Failed to select with condition:", err)
    }
    
    for _, user := range users {
        age := user.Data["age"].(int)
        if age <= 25 {
            t.Errorf("Expected age > 25, got %d", age)
        }
    }
    
    // Clean up
    db.ExecOneSQL("DROP TABLE test_users")
}

func setupTestDB(t *testing.T) *MySQLDB {
    config := DefaultMySQLConfig()
    config.Database = "test_db"
    config.Username = "test_user"
    config.Password = "test_pass"
    
    db, err := NewDatabase(config)
    if err != nil {
        t.Skipf("MySQL test database not available: %v", err)
    }
    
    return db
}

func setupTestData(t *testing.T, db *MySQLDB) {
    // Create table
    createSQL := `
    CREATE TABLE IF NOT EXISTS test_users (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        age INT
    )`
    
    result := db.ExecOneSQL(createSQL)
    if result.Error != nil {
        t.Fatal("Failed to create test table:", result.Error)
    }
    
    // Insert test data
    users := []orm.TableStruct{
        TestUser{Name: "Alice", Email: "alice@test.com", Age: 25},
        TestUser{Name: "Bob", Email: "bob@test.com", Age: 30},
        TestUser{Name: "Carol", Email: "carol@test.com", Age: 35},
    }
    
    _, err := db.InsertManyTableStructs(users, false)
    if err != nil {
        t.Fatal("Failed to insert test users:", err)
    }
}
```

### Step 7: Create Example Usage

Create `examples/basic_usage.go`:

```go
package main

import (
    "fmt"
    "log"
    
    "your-module/mysql"
    orm "github.com/medatechnology/simpleorm"
)

type User struct {
    ID       int    `json:"id" db:"id"`
    Name     string `json:"name" db:"name"`
    Email    string `json:"email" db:"email"`
    Age      int    `json:"age" db:"age"`
    Active   bool   `json:"active" db:"active"`
}

func (u User) TableName() string {
    return "users"
}

func main() {
    // Configure MySQL
    config := mysql.DefaultMySQLConfig()
    config.Host = "localhost"
    config.Port = 3306
    config.Database = "myapp"
    config.Username = "user"
    config.Password = "password"
    
    // Connect to database
    db, err := mysql.NewDatabase(config)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    defer db.Close()
    
    // Test connection
    if !db.IsConnected() {
        log.Fatal("Database not connected")
    }
    
    fmt.Println("✅ Connected to MySQL successfully!")
    
    // Get status
    status, err := db.Status()
    if err != nil {
        log.Printf("Could not get status: %v", err)
    } else {
        fmt.Printf("MySQL Version: %s\n", status.Version)
        fmt.Printf("Uptime: %v\n", status.Uptime)
    }
    
    // Create table
    createTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        age INT,
        active BOOLEAN DEFAULT TRUE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`
    
    result := db.ExecOneSQL(createTable)
    if result.Error != nil {
        log.Fatal("Failed to create table:", result.Error)
    }
    
    fmt.Println("✅ Table created successfully!")
    
    // Insert a user
    user := User{
        Name:   "John Doe",
        Email:  "john@example.com",
        Age:    30,
        Active: true,
    }
    
    insertResult := db.InsertOneTableStruct(user, false)
    if insertResult.Error != nil {
        log.Fatal("Failed to insert user:", insertResult.Error)
    }
    
    fmt.Printf("✅ User inserted with ID: %d\n", insertResult.LastInsertID)
    
    // Query users with conditions
    condition := &orm.Condition{
        Field:    "active",
        Operator: "=",
        Value:    true,
        OrderBy:  []string{"name ASC"},
        Limit:    10,
    }
    
    users, err := db.SelectManyWithCondition("users", condition)
    if err != nil {
        log.Fatal("Failed to query users:", err)
    }
    
    fmt.Printf("✅ Found %d active users:\n", len(users))
    for _, u := range users {
        fmt.Printf("  - %s (%s)\n", u.Data["name"], u.Data["email"])
    }
}
```

## Best Practices

### 1. Connection Management
```go
// Always configure connection pooling
func configureConnectionPool(db *sql.DB, config MySQLConfig) {
    db.SetMaxOpenConns(config.MaxOpenConns)
    db.SetMaxIdleConns(config.MaxIdleConns)
    db.SetConnMaxLifetime(config.ConnMaxLifetime)
    db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}

// Always test connections
func (db *MySQLDB) ping() error {
    ctx, cancel := context.WithTimeout(context.Background(), db.Config.ConnectTimeout)
    defer cancel()
    return db.DB.PingContext(ctx)
}
```

### 2. Error Handling
```go
// Wrap database-specific errors
func (db *MySQLDB) execWithErrorHandling(query string, args ...interface{}) (sql.Result, error) {
    result, err := db.DB.Exec(query, args...)
    if err != nil {
        return nil, WrapMySQLError(err)
    }
    return result, nil
}
```

### 3. Type Conversion
```go
// Handle database-specific type conversions
func convertToGoType(value interface{}, targetType reflect.Type) interface{} {
    // Implementation specific to your database
    switch targetType.Kind() {
    case reflect.String:
        if b, ok := value.([]byte); ok {
            return string(b)
        }
    case reflect.Int64:
        if s, ok := value.(string); ok {
            if i, err := strconv.ParseInt(s, 10, 64); err == nil {
                return i
            }
        }
    }
    return value
}
```

### 4. Performance Optimization
```go
// Use prepared statements for repeated queries
type MySQLDB struct {
    // ... other fields
    preparedStatements map[string]*sql.Stmt
    stmtMutex         sync.RWMutex
}

func (db *MySQLDB) getOrPrepareStatement(query string) (*sql.Stmt, error) {
    db.stmtMutex.RLock()
    stmt, exists := db.preparedStatements[query]
    db.stmtMutex.RUnlock()
    
    if exists {
        return stmt, nil
    }
    
    db.stmtMutex.Lock()
    defer db.stmtMutex.Unlock()
    
    // Double-check pattern
    if stmt, exists := db.preparedStatements[query]; exists {
        return stmt, nil
    }
    
    stmt, err := db.DB.Prepare(query)
    if err != nil {
        return nil, err
    }
    
    if db.preparedStatements == nil {
        db.preparedStatements = make(map[string]*sql.Stmt)
    }
    db.preparedStatements[query] = stmt
    
    return stmt, nil
}
```

## Common Patterns

### Database-Specific SQL Generation
```go
// Override SQL generation for database-specific syntax
func (db *MySQLDB) buildLimitClause(limit, offset int) string {
    if limit <= 0 {
        return ""
    }
    
    if offset > 0 {
        return fmt.Sprintf("LIMIT %d, %d", offset, limit)  // MySQL syntax
    }
    
    return fmt.Sprintf("LIMIT %d", limit)
}
```

### Schema Introspection
```go
func (db *MySQLDB) getTableColumns(tableName string) ([]string, error) {
    query := `
    SELECT COLUMN_NAME 
    FROM INFORMATION_SCHEMA.COLUMNS 
    WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
    ORDER BY ORDINAL_POSITION`
    
    rows, err := db.DB.Query(query, db.dbName, tableName)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var columns []string
    for rows.Next() {
        var column string
        if err := rows.Scan(&column); err != nil {
            return nil, err
        }
        columns = append(columns, column)
    }
    
    return columns, nil
}
```

### Transaction Support (Optional Enhancement)
```go
type MySQLTx struct {
    tx *sql.Tx
    db *MySQLDB
}

func (db *MySQLDB) BeginTx() (*MySQLTx, error) {
    tx, err := db.DB.Begin()
    if err != nil {
        return nil, WrapMySQLError(err)
    }
    
    return &MySQLTx{tx: tx, db: db}, nil
}

func (tx *MySQLTx) Commit() error {
    return WrapMySQLError(tx.tx.Commit())
}

func (tx *MySQLTx) Rollback() error {
    return WrapMySQLError(tx.tx.Rollback())
}
```

This guide provides a complete foundation for implementing any database backend for SimpleORM. The key is maintaining consistency with the interface while leveraging the specific features and optimizations of your target database system.