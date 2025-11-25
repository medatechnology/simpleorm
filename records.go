package orm

import (
	"fmt"
	"strings"

	"github.com/medatechnology/goutil/object"
)

// TODO: replace the DBRecord.Data to be map[string]DataStruct
// .     this will be more robust and flexible
type DataStruct struct {
	TypeDef string // string | int | bool | etc
	Value   interface{}
	Empty   bool // this to replace the nil value if data is not set
}

type DBRecord struct {
	TableName string
	Data      map[string]interface{}
}

type DBRecords []DBRecord

// Append adds a new DBRecord to the DBRecords slice.
// Usage:
//
//	records.Append(newRecord)
func (d *DBRecords) Append(rec DBRecord) {
	*d = append(*d, rec)
}

// Convert DBRecord to SQL Insert string and values but with placeholder (parameterized)
// ToInsertSQLParameterized converts a single DBRecord to a parameterized INSERT SQL statement.
// Usage:
//
//	sql, values := record.ToInsertSQLParameterized()
//
// Returns:
//   - string: Parameterized INSERT query (e.g., "INSERT INTO table (col1, col2) VALUES (?, ?)")
//   - []interface{}: Slice of values for the parameters
func (d *DBRecord) ToInsertSQLParameterized() (string, []interface{}) {
	numFields := len(d.Data)
	columns := make([]string, 0, numFields)
	placeholders := make([]string, 0, numFields)
	values := make([]interface{}, 0, numFields)

	for key, value := range d.Data {
		columns = append(columns, key)
		placeholders = append(placeholders, "?")
		values = append(values, value)
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		d.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	return sql, values
}

// Convert DBRecord to SQL Insert string and values
// ToInsertSQLRaw converts a single DBRecord to a raw INSERT SQL statement with values.
// Usage:
//
//	sql, values := record.ToInsertSQLRaw()
//
// Returns:
//   - string: Complete INSERT query with values (e.g., "INSERT INTO table (col1, col2) VALUES ('value1', 2)")
//   - []interface{}: Slice of original values (for reference)
func (d *DBRecord) ToInsertSQLRaw() (string, []interface{}) {
	numFields := len(d.Data)
	columns := make([]string, 0, numFields)
	// placeholders := make([]string, 0, numFields)
	values := make([]interface{}, 0, numFields)
	valuesStr := make([]string, 0, numFields)

	for key, value := range d.Data {
		columns = append(columns, key)
		// placeholders = append(placeholders, "?")
		values = append(values, value)
		valuesStr = append(valuesStr, InterfaceToSQLString(value))
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		d.TableName,
		strings.Join(columns, ", "),
		strings.Join(valuesStr, ", "),
	)
	return sql, values
}

// NOTE:
// For not same tables that are in DBRecords then use the DBRecord.ToInsertSQLParameterized() function for each
// record in DBRecords.
//
// BULK inserts, one is parameterized and non-parameterized (just plain raw SQL)
// This is always for same table only, not for different tables!
//
// ToInsertSQLParameterized converts multiple DBRecords to a slice of parameterized INSERT statements.
// It automatically batches inserts according to MAX_MULTIPLE_INSERTS limit.
// Usage:
//
//	statements := records.ToInsertSQLParameterized()
//
// Returns: Slice of ParametereizedSQL containing batched INSERT statements and their values
func (records DBRecords) ToInsertSQLParameterized() []ParametereizedSQL {
	if len(records) == 0 {
		return nil
	}

	// Security: Nil check to prevent panic
	if records[0].Data == nil {
		return nil
	}

	// All records should have the same structure, use the first one as template
	tableName := records[0].TableName

	// Get column names from the first record
	numFields := len(records[0].Data)
	if numFields == 0 {
		return nil // No fields to insert
	}

	columns := make([]string, 0, numFields)
	for key := range records[0].Data {
		columns = append(columns, key)
	}

	// Calculate how many statements we'll need
	numStatements := (len(records) + MAX_MULTIPLE_INSERTS - 1) / MAX_MULTIPLE_INSERTS

	paramStatements := make([]ParametereizedSQL, 0, numStatements)

	// Build the column part once - common for all statements
	columnsSQL := fmt.Sprintf("(%s)", strings.Join(columns, ", "))

	// Process in batches
	for i := 0; i < len(records); i += MAX_MULTIPLE_INSERTS {
		end := i + MAX_MULTIPLE_INSERTS
		if end > len(records) {
			end = len(records)
		}

		currentBatch := records[i:end]
		batchSize := len(currentBatch)

		// For each batch, create placeholders and collect values
		placeholderGroups := make([]string, 0, batchSize)
		values := make([]interface{}, 0, batchSize*numFields)

		for _, record := range currentBatch {
			// Create placeholder group for this record (?,?,?)
			placeholders := make([]string, 0, numFields)
			for j := 0; j < numFields; j++ {
				placeholders = append(placeholders, "?")
			}
			placeholderGroups = append(placeholderGroups, fmt.Sprintf("(%s)", strings.Join(placeholders, ", ")))

			// Add values in the correct order (matching column order)
			for _, col := range columns {
				values = append(values, record.Data[col])
			}
		}

		// Create the SQL for this batch
		sql := fmt.Sprintf(
			"INSERT INTO %s %s VALUES %s",
			tableName,
			columnsSQL,
			strings.Join(placeholderGroups, ", "),
		)

		paramStatements = append(paramStatements, ParametereizedSQL{
			Query:  sql,
			Values: values,
		})
	}

	return paramStatements
}

// ToInsertSQLRaw converts multiple DBRecords to a slice of raw INSERT SQL statements.
// It automatically batches inserts according to MAX_MULTIPLE_INSERTS limit.
// Usage:
//
//	statements := records.ToInsertSQLRaw()
//
// Returns: Slice of strings containing complete INSERT statements with values
func (records DBRecords) ToInsertSQLRaw() []string {
	if len(records) == 0 {
		return nil
	}

	// Security: Nil check to prevent panic
	if records[0].Data == nil {
		return nil
	}

	// All records should have the same structure, use the first one as template
	tableName := records[0].TableName

	// Get column names from the first record
	numFields := len(records[0].Data)
	if numFields == 0 {
		return nil // No fields to insert
	}

	columns := make([]string, 0, numFields)
	for key := range records[0].Data {
		columns = append(columns, key)
	}

	// Calculate how many statements we'll need
	numStatements := (len(records) + MAX_MULTIPLE_INSERTS - 1) / MAX_MULTIPLE_INSERTS

	sqlStatements := make([]string, 0, numStatements)

	// Build the column part once - common for all statements
	columnsSQL := fmt.Sprintf("(%s)", strings.Join(columns, ", "))

	// Process in batches
	for i := 0; i < len(records); i += MAX_MULTIPLE_INSERTS {
		end := i + MAX_MULTIPLE_INSERTS
		if end > len(records) {
			end = len(records)
		}

		currentBatch := records[i:end]
		batchSize := len(currentBatch)

		// For each batch, create value strings
		valueGroups := make([]string, 0, batchSize)

		for _, record := range currentBatch {
			// Create values group for this record
			recordValues := make([]string, 0, numFields)
			for _, col := range columns {
				recordValues = append(recordValues, InterfaceToSQLString(record.Data[col]))
			}
			valueGroups = append(valueGroups, fmt.Sprintf("(%s)", strings.Join(recordValues, ", ")))
		}

		// Create the SQL for this batch
		sql := fmt.Sprintf(
			"INSERT INTO %s %s VALUES %s",
			tableName,
			columnsSQL,
			strings.Join(valueGroups, ", "),
		)

		sqlStatements = append(sqlStatements, sql)
	}

	return sqlStatements
}

// Convert DBRecord to ParameterizedStatement
// func (d *DBRecord) ToInsertParameterized() gorqlite.ParameterizedStatement {
// 	sql, values := d.ToInsertSQL()
// 	return gorqlite.ParameterizedStatement{
// 		Query:     sql,
// 		Arguments: values,
// 	}
// }

// FromStruct converts a TableStruct object to a DBRecord.
// It maps the struct fields to the DBRecord's Data map and sets the table name.
// Usage:
//
//	var record DBRecord
//	record.FromStruct(userStruct)
//
// Returns: error if conversion fails
func (d *DBRecord) FromStruct(obj TableStruct) error {
	d.Data = object.StructToMap(obj) // Assume this is implemented and tested
	d.TableName = obj.TableName()
	return nil
}
