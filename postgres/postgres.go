// Package postgres provides a PostgreSQL implementation for the orm.Database interface.
package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	orm "github.com/medatechnology/simpleorm"
)

// postgres implements the orm.Database interface for PostgreSQL.
type postgres struct {
	db     *sql.DB        // The underlying database connection pool
	config PostgresConfig // Configuration for connection management
}

// NewDatabase creates a new PostgreSQL database instance.
// It takes a PostgresConfig and returns an implementation of orm.Database.
func NewDatabase(config PostgresConfig) (PostgresDirectDB, error) {
	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPostgresInvalidConfig, err)
	}

	// Build the connection string using the enhanced config
	connStr, err := config.ToSimpleDSN()
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	// Open a new database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		wrappedErr := WrapPostgreSQLError(err, "CONNECT", "", "")
		return nil, fmt.Errorf("%w: %v", ErrPostgresConnectionFailed, wrappedErr)
	}

	// Set connection pool properties from config
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Ping the database to verify the connection
	if err = db.Ping(); err != nil {
		db.Close() // Clean up the connection on failure
		wrappedErr := WrapPostgreSQLError(err, "PING", "", "")
		return nil, fmt.Errorf("%w: %v", ErrPostgresConnectionFailed, wrappedErr)
	}

	// Validate the connection with additional checks
	if err = validatePostgreSQLConnection(db); err != nil {
		db.Close() // Clean up the connection on failure
		return nil, fmt.Errorf("connection validation failed: %w", err)
	}

	return &postgres{
		db:     db,
		config: config,
	}, nil
}

// Close closes the database connection.
func (pdb *postgres) Close() error {
	return pdb.db.Close()
}

// SelectOne retrieves a single record from the specified table.
// It returns a orm.DBRecord or an error if no record is found.
func (pdb *postgres) SelectOne(tableName string) (orm.DBRecord, error) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT 1", tableName)
	rows, err := pdb.db.Query(query)
	if err != nil {
		return orm.DBRecord{}, fmt.Errorf("failed to execute SelectOne query: %w", err)
	}
	defer rows.Close()

	records, err := scanRowsToDBRecords(rows, tableName)
	if err != nil {
		return orm.DBRecord{}, fmt.Errorf("failed to scan rows for SelectOne: %w", err)
	}

	if len(records) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows // No records found
	}

	return records[0], nil
}

// SelectMany retrieves multiple records from the specified table.
// It returns a slice of orm.DBRecord or an error.
func (pdb *postgres) SelectMany(tableName string) (orm.DBRecords, error) {
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	rows, err := pdb.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SelectMany query: %w", err)
	}
	defer rows.Close()

	records, err := scanRowsToDBRecords(rows, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to scan rows for SelectMany: %w", err)
	}

	if len(records) == 0 {
		return nil, orm.ErrSQLNoRows // No records found
	}

	return records, nil
}

// InsertOneDBRecord inserts a single DBRecord into the specified table.
// It returns the last insert ID (if applicable) and the number of rows affected.
func (pdb *postgres) InsertOneDBRecord(record orm.DBRecord, queue bool) orm.BasicSQLResult {
	// Validate table name
	if err := orm.ValidateTableName(record.TableName); err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	// PostgreSQL uses $1, $2, etc. for placeholders
	cols := make([]string, 0, len(record.Data))
	placeholders := make([]string, 0, len(record.Data))
	values := make([]interface{}, 0, len(record.Data))
	paramCounter := 1

	for k, v := range record.Data {
		cols = append(cols, k)
		placeholders = append(placeholders, fmt.Sprintf("$%d", paramCounter))
		values = append(values, v)
		paramCounter++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		record.TableName,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	var lastInsertID int64
	err := pdb.db.QueryRow(query, values...).Scan(&lastInsertID)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	// For PostgreSQL, if RETURNING id is successful, we assume 1 row affected.
	return orm.BasicSQLResult{LastInsertID: int(lastInsertID), RowsAffected: 1}
}

// InsertManyDBRecords inserts multiple DBRecords into the database.
func (pdb *postgres) InsertManyDBRecords(records []orm.DBRecord, queue bool) ([]orm.BasicSQLResult, error) {
	if len(records) == 0 {
		return nil, nil
	}

	results := make([]orm.BasicSQLResult, 0, len(records))

	for _, record := range records {
		result := pdb.InsertOneDBRecord(record, queue)
		results = append(results, result)
		if result.Error != nil {
			return results, result.Error
		}
	}

	return results, nil
}

// InsertManyDBRecordsSameTable inserts multiple DBRecords from the same table efficiently.
// Uses PostgreSQL-specific multi-row INSERT for better performance.
func (pdb *postgres) InsertManyDBRecordsSameTable(records []orm.DBRecord, queue bool) ([]orm.BasicSQLResult, error) {
	if len(records) == 0 {
		return nil, nil
	}

	// Queue mode not implemented for PostgreSQL
	if queue {
		return nil, fmt.Errorf("queue mode not supported for PostgreSQL")
	}

	// Validate all records are from the same table
	if len(records) > 0 {
		tableName := records[0].TableName
		for i, record := range records {
			if record.TableName != tableName {
				return nil, fmt.Errorf("all records must be from the same table, record %d has table '%s' but expected '%s'",
					i, record.TableName, tableName)
			}
		}
	}

	// Build optimized batch INSERT using PostgreSQL multi-row syntax
	// This creates a single INSERT statement like:
	// INSERT INTO table (col1, col2) VALUES ($1, $2), ($3, $4), ($5, $6)
	batchSQL, values, err := buildPostgreSQLBatchInsertSQL(records)
	if err != nil {
		return []orm.BasicSQLResult{{Error: err}}, err
	}

	// Execute the batch INSERT
	result, err := pdb.db.Exec(batchSQL, values...)
	if err != nil {
		wrappedErr := WrapPostgreSQLError(err, "INSERT", records[0].TableName, batchSQL)
		return []orm.BasicSQLResult{{Error: wrappedErr}}, wrappedErr
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	// Return a single result representing the batch operation
	return []orm.BasicSQLResult{{
		Error:        nil,
		RowsAffected: int(rowsAffected),
		LastInsertID: int(lastInsertID),
	}}, nil
}

// InsertOneTableStruct inserts a single TableStruct into the database.
func (pdb *postgres) InsertOneTableStruct(obj orm.TableStruct, queue bool) orm.BasicSQLResult {
	record, err := orm.TableStructToDBRecord(obj)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}
	return pdb.InsertOneDBRecord(record, queue)
}

// InsertManyTableStructs inserts multiple TableStructs into the database.
func (pdb *postgres) InsertManyTableStructs(objs []orm.TableStruct, queue bool) ([]orm.BasicSQLResult, error) {
	if len(objs) == 0 {
		return nil, nil
	}

	records := make([]orm.DBRecord, 0, len(objs))
	for _, obj := range objs {
		record, err := orm.TableStructToDBRecord(obj)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return pdb.InsertManyDBRecords(records, queue)
}

// NOTE: Update and Delete methods are not part of the orm.Database interface
// They are commented out as they're not required and the interface deliberately
// excludes them (see comment in orm.go:37-45)

// UpdateOneDBRecord - NOT IN INTERFACE - Commented out
/*
func (pdb *postgres) UpdateOneDBRecord(record orm.DBRecord, ignoreKeyOnUpdate bool) (orm.BasicSQLResult, error) {
	// Assumes 'id' is the primary key and is present in record.Data.
	id, ok := record.Data["id"]
	if !ok {
		return orm.BasicSQLResult{Error: orm.ErrMissingPrimaryKey}, orm.ErrMissingPrimaryKey
	}

	// Pre-allocate slices with known capacity to reduce allocations
	numFields := len(record.Data)
	setClauses := make([]string, 0, numFields)
	values := make([]interface{}, 0, numFields+1) // +1 for WHERE id parameter
	paramCounter := 1

	for k, v := range record.Data {
		if k == "id" && ignoreKeyOnUpdate {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", k, paramCounter))
		values = append(values, v)
		paramCounter++
	}

	values = append(values, id)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d",
		record.TableName,
		strings.Join(setClauses, ", "),
		paramCounter,
	)

	result, err := pdb.db.Exec(query, values...)
	if err != nil {
		return orm.BasicSQLResult{Error: err}, fmt.Errorf("failed to update record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return orm.BasicSQLResult{Error: err}, fmt.Errorf("failed to get rows affected after update: %w", err)
	}

	return orm.BasicSQLResult{RowsAffected: int(rowsAffected)}, nil
}
*/

// ExecOneSQLParameterized executes a single parameterized SQL query that does not return rows.
func (pdb *postgres) ExecOneSQLParameterized(paramSQL orm.ParametereizedSQL) orm.BasicSQLResult {
	result, err := pdb.db.Exec(paramSQL.Query, paramSQL.Values...)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}
	return orm.BasicSQLResult{RowsAffected: int(rowsAffected)}
}

// ExecManySQLParameterized executes multiple parameterized SQL queries in a batch.
func (pdb *postgres) ExecManySQLParameterized(paramSQLs []orm.ParametereizedSQL) ([]orm.BasicSQLResult, error) {
	tx, err := pdb.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	results := make([]orm.BasicSQLResult, 0, len(paramSQLs))
	for _, ps := range paramSQLs {
		result, err := tx.Exec(ps.Query, ps.Values...)
		if err != nil {
			results = append(results, orm.BasicSQLResult{Error: err})
			return results, fmt.Errorf("failed to execute SQL: %w", err)
		}
		rowsAffected, _ := result.RowsAffected()
		results = append(results, orm.BasicSQLResult{RowsAffected: int(rowsAffected)})
	}

	if err := tx.Commit(); err != nil {
		return results, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return results, nil
}

// SelectOneSQLParameterized executes a single parameterized SQL query that returns rows.
func (pdb *postgres) SelectOneSQLParameterized(paramSQL orm.ParametereizedSQL) (orm.DBRecords, error) {
	rows, err := pdb.db.Query(paramSQL.Query, paramSQL.Values...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SelectOneSQLParameterized query: %w", err)
	}
	defer rows.Close()

	records, err := scanRowsToDBRecords(rows, "") // Table name can be empty if not directly from a table
	if err != nil {
		return nil, fmt.Errorf("failed to scan rows for SelectOneSQLParameterized: %w", err)
	}

	return records, nil
}

// SelectManySQLParameterized executes multiple parameterized SQL queries that return multiple rows.
func (pdb *postgres) SelectManySQLParameterized(paramSQLs []orm.ParametereizedSQL) ([]orm.DBRecords, error) {
	allResults := make([]orm.DBRecords, 0, len(paramSQLs))
	for _, ps := range paramSQLs {
		rows, err := pdb.db.Query(ps.Query, ps.Values...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute SelectManySQLParameterized query: %w", err)
		}
		defer rows.Close()

		records, err := scanRowsToDBRecords(rows, "") // Table name can be empty
		if err != nil {
			return nil, fmt.Errorf("failed to scan rows for SelectManySQLParameterized: %w", err)
		}
		allResults = append(allResults, records)
	}
	return allResults, nil
}


// GetSchema retrieves schema information from PostgreSQL.
// hideSQL and hideSureSQL parameters are currently ignored as PostgreSQL's information_schema
// doesn't directly map to these concepts from RQLite.
func (pdb *postgres) GetSchema(hideSQL, hideSureSQL bool) []orm.SchemaStruct {
	// Query PostgreSQL's information_schema to get table and column details.
	query := `
		SELECT
			table_name,
			column_name,
			data_type,
			is_nullable,
			column_default
		FROM
			information_schema.columns
		WHERE
			table_schema = current_schema()
		ORDER BY
			table_name, ordinal_position;
	`
	rows, err := pdb.db.Query(query)
	if err != nil {
		fmt.Printf("Error getting schema: %v\n", err)
		return nil
	}
	defer rows.Close()

	var schemas []orm.SchemaStruct
	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		var columnDefault sql.NullString // Use sql.NullString for nullable default values
		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable, &columnDefault); err != nil {
			fmt.Printf("Error scanning schema row: %v\n", err)
			continue
		}

		// Basic mapping to SchemaStruct based on actual struct fields
		// ObjectType, ObjectName, TableName, RootPage, SQLCommand, Hidden
		schemas = append(schemas, orm.SchemaStruct{
			ObjectType: "table",
			ObjectName: tableName,
			TableName:  tableName,
			SQLCommand: fmt.Sprintf("-- PostgreSQL table: %s, column: %s (%s)", tableName, columnName, dataType),
		})
	}
	return schemas
}

// Status retrieves the status of the PostgreSQL database.
func (pdb *postgres) Status() (orm.NodeStatusStruct, error) {
	var status orm.NodeStatusStruct
	status.DBMS = "postgresql"
	status.DBMSDriver = "lib/pq"

	// Build connection URL (without password for security)
	status.URL = fmt.Sprintf("postgres://%s@%s:%d/%s",
		pdb.config.User,
		pdb.config.Host,
		pdb.config.Port,
		pdb.config.DBName,
	)

	// Get PostgreSQL version
	var version string
	err := pdb.db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		return status, fmt.Errorf("failed to get PostgreSQL version: %w", err)
	}
	status.Version = version

	// Get database statistics
	stats, err := getPostgreSQLStats(pdb.db, pdb.config.DBName)
	if err == nil {
		// Add stats to the status struct
		status.Nodes = 1
		status.IsLeader = true // Single node is always the "leader"

		// Create a peer entry with detailed stats
		peerStatus := orm.StatusStruct{
			NodeID:     fmt.Sprintf("%s:%d", pdb.config.Host, pdb.config.Port),
			URL:        fmt.Sprintf("postgres://%s:%d/%s", pdb.config.Host, pdb.config.Port, pdb.config.DBName),
			Version:    version,
			DBMS:       "postgresql",
			DBMSDriver: "lib/pq",
			IsLeader:   true,
		}

		// Populate database size if available
		if dbSize, ok := stats["db_size"].(int64); ok {
			peerStatus.DBSize = dbSize
		}

		// Store stats in a peer entry for visibility
		status.Peers = make(map[int]orm.StatusStruct)
		status.Peers[0] = peerStatus
	} else {
		// Fallback if stats retrieval fails
		status.Peers = make(map[int]orm.StatusStruct)
		status.Nodes = 1
		status.IsLeader = true
	}

	// Get connection pool stats
	dbStats := pdb.db.Stats()
	status.Leader = fmt.Sprintf("%s:%d (Open:%d Idle:%d InUse:%d)",
		pdb.config.Host,
		pdb.config.Port,
		dbStats.OpenConnections,
		dbStats.Idle,
		dbStats.InUse,
	)

	return status, nil
}

// --- Helper Functions ---

// buildSelectQuery constructs a SELECT query string and its arguments from conditions.
func buildSelectQuery(tableName string, conditions []orm.Condition, limit int) (string, []interface{}) {
	var whereClauses []string
	var args []interface{}
	paramCounter := 1

	for _, cond := range conditions {
		clause, vals, count := buildConditionClause(cond, paramCounter)
		whereClauses = append(whereClauses, clause)
		args = append(args, vals...)
		paramCounter += count
	}

	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ") // Simple AND for top-level
	}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	return query, args
}

// buildConditionClause recursively builds a SQL WHERE clause from a Condition struct.
func buildConditionClause(cond orm.Condition, paramCounter int) (string, []interface{}, int) {
	var clause string
	var args []interface{}
	initialParamCounter := paramCounter

	if len(cond.Nested) > 0 {
		// Pre-allocate with known capacity
		nestedClauses := make([]string, 0, len(cond.Nested))
		for _, nestedCond := range cond.Nested {
			nestedClause, nestedArgs, count := buildConditionClause(nestedCond, paramCounter)
			nestedClauses = append(nestedClauses, nestedClause)
			args = append(args, nestedArgs...)
			paramCounter += count
		}
		logic := " AND "
		if cond.Logic != "" {
			logic = " " + strings.ToUpper(cond.Logic) + " "
		}
		clause = fmt.Sprintf("(%s)", strings.Join(nestedClauses, logic))
	} else if cond.Field != "" && cond.Operator != "" {
		// Handle specific operators for PostgreSQL
		operator := cond.Operator
		switch strings.ToUpper(operator) {
		case "LIKE", "ILIKE": // ILIKE for case-insensitive LIKE in PostgreSQL
			clause = fmt.Sprintf("%s %s $%d", cond.Field, operator, paramCounter)
			args = append(args, cond.Value)
			paramCounter++
		case "IN", "NOT IN":
			// For IN/NOT IN, the value should be a slice.
			// PostgreSQL requires unnesting arrays for IN clauses or using ANY/ALL.
			// For simplicity, we'll assume a comma-separated string or a slice of values
			// and build placeholders accordingly.
			if vals, ok := cond.Value.([]interface{}); ok {
				// Pre-allocate with known capacity
				valuePlaceholders := make([]string, 0, len(vals))
				for range vals {
					valuePlaceholders = append(valuePlaceholders, fmt.Sprintf("$%d", paramCounter))
					args = append(args, vals...)
					paramCounter++
				}
				clause = fmt.Sprintf("%s %s (%s)", cond.Field, operator, strings.Join(valuePlaceholders, ", "))
			} else {
				// Fallback or error for unsupported IN value type
				clause = fmt.Sprintf("%s %s ($%d)", cond.Field, operator, paramCounter)
				args = append(args, cond.Value)
				paramCounter++
			}
		default:
			clause = fmt.Sprintf("%s %s $%d", cond.Field, operator, paramCounter)
			args = append(args, cond.Value)
			paramCounter++
		}
	}

	return clause, args, paramCounter - initialParamCounter
}

// --- Additional orm.Database interface methods ---

// SelectOneWithCondition retrieves a single record with conditions.
func (pdb *postgres) SelectOneWithCondition(tableName string, condition *orm.Condition) (orm.DBRecord, error) {
	if condition == nil {
		return pdb.SelectOne(tableName)
	}

	query, params, err := condition.ToSelectString(tableName)
	if err != nil {
		return orm.DBRecord{}, fmt.Errorf("failed to build query: %w", err)
	}

	// Add LIMIT 1 if not present
	if !strings.Contains(strings.ToUpper(query), "LIMIT") {
		query += " LIMIT 1"
	}

	rows, err := pdb.db.Query(query, params...)
	if err != nil {
		return orm.DBRecord{}, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	records, err := scanRowsToDBRecords(rows, tableName)
	if err != nil {
		return orm.DBRecord{}, err
	}

	if len(records) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}

	return records[0], nil
}

// SelectManyWithCondition retrieves multiple records with conditions.
func (pdb *postgres) SelectManyWithCondition(tableName string, condition *orm.Condition) ([]orm.DBRecord, error) {
	if condition == nil {
		return pdb.SelectMany(tableName)
	}

	query, params, err := condition.ToSelectString(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := pdb.db.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	records, err := scanRowsToDBRecords(rows, tableName)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, orm.ErrSQLNoRows
	}

	return records, nil
}

// SelectOneSQL executes a raw SQL query and returns the results.
func (pdb *postgres) SelectOneSQL(sql string) (orm.DBRecords, error) {
	rows, err := pdb.db.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	records, err := scanRowsToDBRecords(rows, "")
	if err != nil {
		return nil, err
	}

	return records, nil
}

// SelectManySQL executes multiple SQL queries and returns results.
func (pdb *postgres) SelectManySQL(sqls []string) ([]orm.DBRecords, error) {
	results := make([]orm.DBRecords, 0, len(sqls))

	for _, sql := range sqls {
		rows, err := pdb.db.Query(sql)
		if err != nil {
			return results, fmt.Errorf("failed to execute query: %w", err)
		}

		records, err := scanRowsToDBRecords(rows, "")
		rows.Close()
		if err != nil {
			if err == orm.ErrSQLNoRows {
				results = append(results, orm.DBRecords{})
				continue
			}
			return results, err
		}

		results = append(results, records)
	}

	return results, nil
}

// SelectOnlyOneSQL executes a SQL query and ensures exactly one row is returned.
func (pdb *postgres) SelectOnlyOneSQL(sql string) (orm.DBRecord, error) {
	records, err := pdb.SelectOneSQL(sql)
	if err != nil {
		return orm.DBRecord{}, err
	}

	if len(records) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}

	if len(records) > 1 {
		return orm.DBRecord{}, orm.ErrSQLMoreThanOneRow
	}

	return records[0], nil
}

// SelectOnlyOneSQLParameterized executes a parameterized query ensuring exactly one row.
func (pdb *postgres) SelectOnlyOneSQLParameterized(paramSQL orm.ParametereizedSQL) (orm.DBRecord, error) {
	records, err := pdb.SelectOneSQLParameterized(paramSQL)
	if err != nil {
		return orm.DBRecord{}, err
	}

	if len(records) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}

	if len(records) > 1 {
		return orm.DBRecord{}, orm.ErrSQLMoreThanOneRow
	}

	return records[0], nil
}

// ExecOneSQL executes a raw SQL query that does not return rows.
func (pdb *postgres) ExecOneSQL(sql string) orm.BasicSQLResult {
	result, err := pdb.db.Exec(sql)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	return orm.BasicSQLResult{
		RowsAffected: int(rowsAffected),
	}
}

// ExecManySQL executes multiple raw SQL queries in a batch.
func (pdb *postgres) ExecManySQL(sqls []string) ([]orm.BasicSQLResult, error) {
	results := make([]orm.BasicSQLResult, 0, len(sqls))

	tx, err := pdb.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, sql := range sqls {
		result, err := tx.Exec(sql)
		if err != nil {
			results = append(results, orm.BasicSQLResult{Error: err})
			return results, fmt.Errorf("failed to execute SQL: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		results = append(results, orm.BasicSQLResult{
			RowsAffected: int(rowsAffected),
		})
	}

	if err := tx.Commit(); err != nil {
		return results, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return results, nil
}

// IsConnected checks if the database connection is active.
func (pdb *postgres) IsConnected() bool {
	if pdb.db == nil {
		return false
	}
	err := pdb.db.Ping()
	return err == nil
}

// Leader returns the leader node (not applicable for PostgreSQL, returns empty string).
func (pdb *postgres) Leader() (string, error) {
	return "not implemented for PostgreSQL", nil
}

// Peers returns peer nodes (not applicable for PostgreSQL, returns empty slice).
func (pdb *postgres) Peers() ([]string, error) {
	return []string{}, nil
}

// GetTableStruct retrieves the TableStruct for a given table name.
// This would involve querying information_schema for column names and types.
// func (pdb *postgres) GetTableStruct(tableName string) (orm.TableStruct, error) { /* ... */ }

// CreateTable creates a new table based on a TableStruct.
// func (pdb *postgres) CreateTable(table orm.TableStruct) orm.ExecResult { /* ... */ }

// DropTable drops an existing table.
// func (pdb *postgres) DropTable(tableName string) orm.ExecResult { /* ... */ }

// TruncateTable truncates an existing table.
// func (pdb *postgres) TruncateTable(tableName string) orm.ExecResult { /* ... */ }

// RenameTable renames an existing table.
// func (pdb *postgres) RenameTable(oldName, newName string) orm.ExecResult { /* ... */ }

// AddColumn adds a new column to an existing table.
// func (pdb *postgres) AddColumn(tableName string, column orm.ColumnSchema) orm.ExecResult { /* ... */ }

// DropColumn drops a column from an existing table.
// func (pdb *postgres) DropColumn(tableName, columnName string) orm.ExecResult { /* ... */ }

// RenameColumn renames a column in an existing table.
// func (pdb *postgres) RenameColumn(tableName, oldName, newName string) orm.ExecResult { /* ... */ }

// AlterColumn modifies an existing column.
// func (pdb *postgres) AlterColumn(tableName string, column orm.ColumnSchema) orm.ExecResult { /* ... */ }

// CreateIndex creates a new index on a table.
// func (pdb *postgres) CreateIndex(tableName string, index orm.IndexSchema) orm.ExecResult { /* ... */ }

// DropIndex drops an existing index.
// func (pdb *postgres) DropIndex(tableName, indexName string) orm.ExecResult { /* ... */ }

// CreateForeignKey creates a foreign key constraint.
// func (pdb *postgres) CreateForeignKey(tableName string, fk orm.ForeignKeySchema) orm.ExecResult { /* ... */ }

// DropForeignKey drops a foreign key constraint.
// func (pdb *postgres) DropForeignKey(tableName, fkName string) orm.ExecResult { /* ... */ }

// BeginTransaction starts a new database transaction.
// func (pdb *postgres) BeginTransaction() (*sql.Tx, error) { /* ... */ }

// CommitTransaction commits a transaction.
// func (pdb *postgres) CommitTransaction(tx *sql.Tx) error { /* ... */ }

// RollbackTransaction rolls back a transaction.
// func (pdb *postgres) RollbackTransaction(tx *sql.Tx) error { /* ... */ }

// GetLastInsertID retrieves the last insert ID. (Handled by RETURNING in InsertOneDBRecord)
// func (pdb *postgres) GetLastInsertID() (int64, error) { /* ... */ }

// GetRowsAffected retrieves the number of rows affected by the last operation. (Handled by ExecResult)
// func (pdb *postgres) GetRowsAffected() (int64, error) { /* ... */ }

// PrepareStatement prepares a SQL statement.
// func (pdb *postgres) PrepareStatement(query string) (*sql.Stmt, error) { /* ... */ }

// ExecuteStatement executes a prepared statement.
// func (pdb *postgres) ExecuteStatement(stmt *sql.Stmt, args ...interface{}) (orm.ExecResult, error) { /* ... */ }

// QueryStatement queries a prepared statement.
// func (pdb *postgres) QueryStatement(stmt *sql.Stmt, args ...interface{}) ([]orm.DBRecord, error) { /* ... */ }

// CloseStatement closes a prepared statement.
// func (pdb *postgres) CloseStatement(stmt *sql.Stmt) error { /* ... */ }

// IsConnected checks if the database connection is active.
// func (pdb *postgres) IsConnected() bool { /* ... */ }

// GetConnection returns the underlying database connection.
// func (pdb *postgres) GetConnection() *sql.DB { return pdb.db }

// SetLogger sets the logger for the database.
// func (pdb *postgres) SetLogger(logger orm.Logger) { /* ... */ }

// GetLogger returns the logger for the database.
// func (pdb *postgres) GetLogger() orm.Logger { /* ... */ }

// SetDebugMode sets the debug mode for the database.
// func (pdb *postgres) SetDebugMode(debug bool) { /* ... */ }

// IsDebugMode returns true if debug mode is enabled.
// func (pdb *postgres) IsDebugMode() bool { /* ... */ }

// GetDialect returns the database dialect.
// func (pdb *postgres) GetDialect() string { return "postgresql" }
