package rqlite

import (
	"fmt"
	"strings"

	orm "github.com/medatechnology/simpleorm"
)

// rqliteTransaction implements the orm.Transaction interface for RQLite
// Unlike PostgreSQL which maintains server-side transaction state,
// RQLite uses a buffered approach where all operations are collected
// and sent atomically to the /db/request endpoint on Commit.
type rqliteTransaction struct {
	db              *RQLiteDirectDB
	statements      []string                  // Buffered SQL statements
	paramStatements []orm.ParametereizedSQL   // Buffered parameterized statements
	committed       bool                      // Track if transaction is committed
	rolledBack      bool                      // Track if transaction is rolled back
}

// BeginTransaction starts a new transaction by creating a transaction buffer
func (db *RQLiteDirectDB) BeginTransaction() (orm.Transaction, error) {
	return &rqliteTransaction{
		db:              db,
		statements:      make([]string, 0),
		paramStatements: make([]orm.ParametereizedSQL, 0),
		committed:       false,
		rolledBack:      false,
	}, nil
}

// Commit sends all buffered operations to RQLite atomically via /db/request endpoint
func (tx *rqliteTransaction) Commit() error {
	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction already rolled back")
	}

	// Nothing to commit if no statements
	if len(tx.statements) == 0 && len(tx.paramStatements) == 0 {
		tx.committed = true
		return nil
	}

	// Convert parameterized statements to regular SQL
	// Note: RQLite /db/request supports parameterized queries, but for simplicity
	// we'll convert them for now. Can be optimized later.
	allStatements := make([]string, 0, len(tx.statements)+len(tx.paramStatements))
	allStatements = append(allStatements, tx.statements...)

	for _, paramSQL := range tx.paramStatements {
		// For RQLite, we need to handle parameterized SQL differently
		// RQLite uses ? placeholders like SQLite
		allStatements = append(allStatements, paramSQL.Query)
	}

	// Send all statements atomically via /db/request endpoint
	err := tx.db.execRequestUnified(allStatements, tx.paramStatements)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tx.committed = true
	return nil
}

// Rollback discards all buffered operations without sending them to RQLite
func (tx *rqliteTransaction) Rollback() error {
	if tx.committed {
		return fmt.Errorf("cannot rollback: transaction already committed")
	}
	if tx.rolledBack {
		return nil // Already rolled back, no error
	}

	// Clear all buffered statements
	tx.statements = nil
	tx.paramStatements = nil
	tx.rolledBack = true

	return nil
}

// ExecOneSQL buffers a SQL statement for execution on commit
func (tx *rqliteTransaction) ExecOneSQL(sqlStmt string) orm.BasicSQLResult {
	if tx.committed {
		return orm.BasicSQLResult{Error: fmt.Errorf("transaction already committed")}
	}
	if tx.rolledBack {
		return orm.BasicSQLResult{Error: fmt.Errorf("transaction already rolled back")}
	}

	tx.statements = append(tx.statements, sqlStmt)
	return orm.BasicSQLResult{} // Success will be determined on Commit
}

// ExecOneSQLParameterized buffers a parameterized SQL statement
func (tx *rqliteTransaction) ExecOneSQLParameterized(paramSQL orm.ParametereizedSQL) orm.BasicSQLResult {
	if tx.committed {
		return orm.BasicSQLResult{Error: fmt.Errorf("transaction already committed")}
	}
	if tx.rolledBack {
		return orm.BasicSQLResult{Error: fmt.Errorf("transaction already rolled back")}
	}

	tx.paramStatements = append(tx.paramStatements, paramSQL)
	return orm.BasicSQLResult{} // Success will be determined on Commit
}

// ExecManySQL buffers multiple SQL statements
func (tx *rqliteTransaction) ExecManySQL(sqls []string) ([]orm.BasicSQLResult, error) {
	if tx.committed {
		return nil, fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return nil, fmt.Errorf("transaction already rolled back")
	}

	results := make([]orm.BasicSQLResult, 0, len(sqls))
	for _, sql := range sqls {
		tx.statements = append(tx.statements, sql)
		results = append(results, orm.BasicSQLResult{})
	}

	return results, nil
}

// ExecManySQLParameterized buffers multiple parameterized SQL statements
func (tx *rqliteTransaction) ExecManySQLParameterized(paramSQLs []orm.ParametereizedSQL) ([]orm.BasicSQLResult, error) {
	if tx.committed {
		return nil, fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return nil, fmt.Errorf("transaction already rolled back")
	}

	results := make([]orm.BasicSQLResult, 0, len(paramSQLs))
	for _, paramSQL := range paramSQLs {
		tx.paramStatements = append(tx.paramStatements, paramSQL)
		results = append(results, orm.BasicSQLResult{})
	}

	return results, nil
}

// SelectOneSQL buffers a SELECT query (will be executed on Commit, results deferred)
// Note: This is a limitation of RQLite's buffered transaction model
// For immediate results, don't use transactions for SELECT queries
func (tx *rqliteTransaction) SelectOneSQL(sqlStmt string) (orm.DBRecords, error) {
	if tx.committed {
		return nil, fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return nil, fmt.Errorf("transaction already rolled back")
	}

	// For SELECT within transactions in RQLite, we need to execute immediately
	// because the buffered model doesn't support deferred reads
	// This is a fundamental difference from PostgreSQL
	return tx.db.SelectOneSQL(sqlStmt)
}

// SelectOnlyOneSQL executes a SELECT query that must return exactly one row
func (tx *rqliteTransaction) SelectOnlyOneSQL(sqlStmt string) (orm.DBRecord, error) {
	records, err := tx.SelectOneSQL(sqlStmt)
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

// SelectOneSQLParameterized executes a parameterized SELECT query
func (tx *rqliteTransaction) SelectOneSQLParameterized(paramSQL orm.ParametereizedSQL) (orm.DBRecords, error) {
	if tx.committed {
		return nil, fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return nil, fmt.Errorf("transaction already rolled back")
	}

	// SELECT queries must be executed immediately in RQLite transactions
	return tx.db.SelectOneSQLParameterized(paramSQL)
}

// SelectOnlyOneSQLParameterized executes a parameterized SELECT that returns exactly one row
func (tx *rqliteTransaction) SelectOnlyOneSQLParameterized(paramSQL orm.ParametereizedSQL) (orm.DBRecord, error) {
	records, err := tx.SelectOneSQLParameterized(paramSQL)
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

// InsertOneDBRecord buffers an insert operation
func (tx *rqliteTransaction) InsertOneDBRecord(record orm.DBRecord) orm.BasicSQLResult {
	if tx.committed {
		return orm.BasicSQLResult{Error: fmt.Errorf("transaction already committed")}
	}
	if tx.rolledBack {
		return orm.BasicSQLResult{Error: fmt.Errorf("transaction already rolled back")}
	}

	// Validate table name
	if err := orm.ValidateTableName(record.TableName); err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	// RQLite uses ? placeholders like SQLite
	cols := make([]string, 0, len(record.Data))
	placeholders := make([]string, 0, len(record.Data))
	values := make([]interface{}, 0, len(record.Data))

	for k, v := range record.Data {
		cols = append(cols, k)
		placeholders = append(placeholders, "?")
		values = append(values, v)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		record.TableName,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	// Buffer the parameterized statement
	tx.paramStatements = append(tx.paramStatements, orm.ParametereizedSQL{
		Query:  query,
		Values: values,
	})

	return orm.BasicSQLResult{} // Success determined on Commit
}

// InsertManyDBRecords buffers multiple insert operations
func (tx *rqliteTransaction) InsertManyDBRecords(records []orm.DBRecord) ([]orm.BasicSQLResult, error) {
	if tx.committed {
		return nil, fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return nil, fmt.Errorf("transaction already rolled back")
	}

	if len(records) == 0 {
		return nil, nil
	}

	results := make([]orm.BasicSQLResult, 0, len(records))
	for _, record := range records {
		result := tx.InsertOneDBRecord(record)
		results = append(results, result)
		if result.Error != nil {
			return results, result.Error
		}
	}

	return results, nil
}

// InsertManyDBRecordsSameTable buffers batch insert for same table
func (tx *rqliteTransaction) InsertManyDBRecordsSameTable(records []orm.DBRecord) ([]orm.BasicSQLResult, error) {
	if tx.committed {
		return nil, fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return nil, fmt.Errorf("transaction already rolled back")
	}

	if len(records) == 0 {
		return nil, nil
	}

	// Validate all records are from the same table
	tableName := records[0].TableName
	for i, record := range records {
		if record.TableName != tableName {
			return nil, fmt.Errorf("all records must be from the same table, record %d has table '%s' but expected '%s'",
				i, record.TableName, tableName)
		}
	}

	// For RQLite, we'll just buffer individual inserts
	// Could be optimized with multi-row INSERT syntax
	return tx.InsertManyDBRecords(records)
}

// InsertOneTableStruct buffers insert for a TableStruct
func (tx *rqliteTransaction) InsertOneTableStruct(obj orm.TableStruct) orm.BasicSQLResult {
	record, err := orm.TableStructToDBRecord(obj)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}
	return tx.InsertOneDBRecord(record)
}

// InsertManyTableStructs buffers inserts for multiple TableStructs
func (tx *rqliteTransaction) InsertManyTableStructs(objs []orm.TableStruct) ([]orm.BasicSQLResult, error) {
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

	return tx.InsertManyDBRecords(records)
}
