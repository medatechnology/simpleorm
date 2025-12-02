package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	orm "github.com/medatechnology/simpleorm"
)

// postgresTransaction implements the orm.Transaction interface
type postgresTransaction struct {
	tx *sql.Tx // The underlying database/sql transaction
}

// BeginTransaction starts a new database transaction
func (pdb *postgres) BeginTransaction() (orm.Transaction, error) {
	tx, err := pdb.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &postgresTransaction{
		tx: tx,
	}, nil
}

// Commit commits the transaction
func (ptx *postgresTransaction) Commit() error {
	if ptx.tx == nil {
		return fmt.Errorf("transaction is nil or already closed")
	}
	err := ptx.tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback rolls back the transaction
func (ptx *postgresTransaction) Rollback() error {
	if ptx.tx == nil {
		return fmt.Errorf("transaction is nil or already closed")
	}
	err := ptx.tx.Rollback()
	if err != nil && err != sql.ErrTxDone {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

// ExecOneSQL executes a single SQL statement within the transaction
func (ptx *postgresTransaction) ExecOneSQL(sqlStmt string) orm.BasicSQLResult {
	if ptx.tx == nil {
		return orm.BasicSQLResult{Error: fmt.Errorf("transaction is nil or already closed")}
	}

	result, err := ptx.tx.Exec(sqlStmt)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	lastInsertID, _ := result.LastInsertId()

	return orm.BasicSQLResult{
		RowsAffected: int(rowsAffected),
		LastInsertID: int(lastInsertID),
	}
}

// ExecOneSQLParameterized executes a parameterized SQL statement within the transaction
func (ptx *postgresTransaction) ExecOneSQLParameterized(paramSQL orm.ParametereizedSQL) orm.BasicSQLResult {
	if ptx.tx == nil {
		return orm.BasicSQLResult{Error: fmt.Errorf("transaction is nil or already closed")}
	}

	result, err := ptx.tx.Exec(paramSQL.Query, paramSQL.Values...)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	lastInsertID, _ := result.LastInsertId()

	return orm.BasicSQLResult{
		RowsAffected: int(rowsAffected),
		LastInsertID: int(lastInsertID),
	}
}

// ExecManySQL executes multiple SQL statements within the transaction
func (ptx *postgresTransaction) ExecManySQL(sqls []string) ([]orm.BasicSQLResult, error) {
	if ptx.tx == nil {
		return nil, fmt.Errorf("transaction is nil or already closed")
	}

	results := make([]orm.BasicSQLResult, 0, len(sqls))

	for _, sqlStmt := range sqls {
		result, err := ptx.tx.Exec(sqlStmt)
		if err != nil {
			results = append(results, orm.BasicSQLResult{Error: err})
			return results, fmt.Errorf("failed to execute SQL: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		lastInsertID, _ := result.LastInsertId()

		results = append(results, orm.BasicSQLResult{
			RowsAffected: int(rowsAffected),
			LastInsertID: int(lastInsertID),
		})
	}

	return results, nil
}

// ExecManySQLParameterized executes multiple parameterized SQL statements within the transaction
func (ptx *postgresTransaction) ExecManySQLParameterized(paramSQLs []orm.ParametereizedSQL) ([]orm.BasicSQLResult, error) {
	if ptx.tx == nil {
		return nil, fmt.Errorf("transaction is nil or already closed")
	}

	results := make([]orm.BasicSQLResult, 0, len(paramSQLs))

	for _, paramSQL := range paramSQLs {
		result, err := ptx.tx.Exec(paramSQL.Query, paramSQL.Values...)
		if err != nil {
			results = append(results, orm.BasicSQLResult{Error: err})
			return results, fmt.Errorf("failed to execute parameterized SQL: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		lastInsertID, _ := result.LastInsertId()

		results = append(results, orm.BasicSQLResult{
			RowsAffected: int(rowsAffected),
			LastInsertID: int(lastInsertID),
		})
	}

	return results, nil
}

// SelectOneSQL executes a SELECT query within the transaction
func (ptx *postgresTransaction) SelectOneSQL(sqlStmt string) (orm.DBRecords, error) {
	if ptx.tx == nil {
		return nil, fmt.Errorf("transaction is nil or already closed")
	}

	rows, err := ptx.tx.Query(sqlStmt)
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

// SelectOnlyOneSQL executes a SELECT query that must return exactly one row
func (ptx *postgresTransaction) SelectOnlyOneSQL(sqlStmt string) (orm.DBRecord, error) {
	records, err := ptx.SelectOneSQL(sqlStmt)
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

// SelectOneSQLParameterized executes a parameterized SELECT query within the transaction
func (ptx *postgresTransaction) SelectOneSQLParameterized(paramSQL orm.ParametereizedSQL) (orm.DBRecords, error) {
	if ptx.tx == nil {
		return nil, fmt.Errorf("transaction is nil or already closed")
	}

	rows, err := ptx.tx.Query(paramSQL.Query, paramSQL.Values...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute parameterized query: %w", err)
	}
	defer rows.Close()

	records, err := scanRowsToDBRecords(rows, "")
	if err != nil {
		return nil, err
	}

	return records, nil
}

// SelectOnlyOneSQLParameterized executes a parameterized SELECT query that must return exactly one row
func (ptx *postgresTransaction) SelectOnlyOneSQLParameterized(paramSQL orm.ParametereizedSQL) (orm.DBRecord, error) {
	records, err := ptx.SelectOneSQLParameterized(paramSQL)
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

// InsertOneDBRecord inserts a single DBRecord within the transaction
func (ptx *postgresTransaction) InsertOneDBRecord(record orm.DBRecord) orm.BasicSQLResult {
	if ptx.tx == nil {
		return orm.BasicSQLResult{Error: fmt.Errorf("transaction is nil or already closed")}
	}

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

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		record.TableName,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := ptx.tx.Exec(query, values...)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return orm.BasicSQLResult{
		RowsAffected: int(rowsAffected),
		LastInsertID: int(lastInsertID),
	}
}

// InsertManyDBRecords inserts multiple DBRecords within the transaction
func (ptx *postgresTransaction) InsertManyDBRecords(records []orm.DBRecord) ([]orm.BasicSQLResult, error) {
	if ptx.tx == nil {
		return nil, fmt.Errorf("transaction is nil or already closed")
	}

	if len(records) == 0 {
		return nil, nil
	}

	results := make([]orm.BasicSQLResult, 0, len(records))

	for _, record := range records {
		result := ptx.InsertOneDBRecord(record)
		results = append(results, result)
		if result.Error != nil {
			return results, result.Error
		}
	}

	return results, nil
}

// InsertManyDBRecordsSameTable inserts multiple DBRecords from the same table efficiently
func (ptx *postgresTransaction) InsertManyDBRecordsSameTable(records []orm.DBRecord) ([]orm.BasicSQLResult, error) {
	if ptx.tx == nil {
		return nil, fmt.Errorf("transaction is nil or already closed")
	}

	if len(records) == 0 {
		return nil, nil
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

	// Build optimized batch INSERT
	batchSQL, values, err := buildPostgreSQLBatchInsertSQL(records)
	if err != nil {
		return []orm.BasicSQLResult{{Error: err}}, err
	}

	// Execute the batch INSERT
	result, err := ptx.tx.Exec(batchSQL, values...)
	if err != nil {
		wrappedErr := WrapPostgreSQLError(err, "INSERT", records[0].TableName, batchSQL)
		return []orm.BasicSQLResult{{Error: wrappedErr}}, wrappedErr
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return []orm.BasicSQLResult{{
		Error:        nil,
		RowsAffected: int(rowsAffected),
		LastInsertID: int(lastInsertID),
	}}, nil
}

// InsertOneTableStruct inserts a single TableStruct within the transaction
func (ptx *postgresTransaction) InsertOneTableStruct(obj orm.TableStruct) orm.BasicSQLResult {
	record, err := orm.TableStructToDBRecord(obj)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}
	return ptx.InsertOneDBRecord(record)
}

// InsertManyTableStructs inserts multiple TableStructs within the transaction
func (ptx *postgresTransaction) InsertManyTableStructs(objs []orm.TableStruct) ([]orm.BasicSQLResult, error) {
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

	return ptx.InsertManyDBRecords(records)
}
