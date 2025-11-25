package postgres

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	orm "github.com/medatechnology/simpleorm"
)

// scanRowToDBRecord converts a single sql.Rows to a DBRecord
func scanRowToDBRecord(rows *sql.Rows, tableName string) (orm.DBRecord, error) {
	columns, err := rows.Columns()
	if err != nil {
		return orm.DBRecord{}, fmt.Errorf("failed to get columns: %w", err)
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
		return orm.DBRecord{}, fmt.Errorf("failed to scan row: %w", err)
	}

	// Convert to map
	data := make(map[string]interface{})
	for i, col := range columns {
		data[col] = convertPostgreSQLValue(values[i])
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	if len(records) == 0 {
		return nil, orm.ErrSQLNoRows
	}

	return records, nil
}

// convertPostgreSQLValue converts PostgreSQL-specific types to Go types
func convertPostgreSQLValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		// PostgreSQL returns many values as []byte, convert to string
		str := string(v)

		// Try to parse as time if it looks like a timestamp
		if len(str) >= 10 && (strings.Contains(str, "-") || strings.Contains(str, ":")) {
			if t, err := parsePostgreSQLTime(str); err == nil {
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

	case sql.NullInt32:
		if v.Valid {
			return v.Int32
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

	case time.Time:
		return v

	case int64, int32, int16, int8, int:
		return v

	case uint64, uint32, uint16, uint8, uint:
		return v

	case float64, float32:
		return v

	case bool:
		return v

	case string:
		return v

	default:
		// For unknown types, return as-is
		return value
	}
}

// parsePostgreSQLTime attempts to parse various PostgreSQL time formats
func parsePostgreSQLTime(str string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05.999999999-07",      // PostgreSQL default with timezone
		"2006-01-02 15:04:05.999999999",         // PostgreSQL without timezone
		"2006-01-02 15:04:05-07",                // With timezone
		"2006-01-02 15:04:05",                   // Without timezone
		"2006-01-02T15:04:05.999999999Z07:00",   // ISO 8601
		"2006-01-02T15:04:05Z07:00",             // ISO 8601 without fractional seconds
		"2006-01-02",                             // Date only
		"15:04:05",                               // Time only
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse PostgreSQL time: %s", str)
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
				keywords := []string{"WHERE", "JOIN", "INNER", "LEFT", "RIGHT", "OUTER",
					"CROSS", "NATURAL", "GROUP", "ORDER", "HAVING", "LIMIT", "OFFSET",
					"UNION", "EXCEPT", "INTERSECT", "AS"}

				for _, keyword := range keywords {
					if strings.HasPrefix(tableName, keyword) {
						break
					}
				}

				// Clean up table name (remove quotes, etc.)
				tableName = strings.Trim(tableName, "\"'`[]")
				return strings.ToLower(tableName)
			}
		}
	} else if strings.HasPrefix(upperSQL, "INSERT INTO") {
		// Extract from INSERT INTO
		if idx := strings.Index(upperSQL, "INSERT INTO"); idx != -1 {
			insertPart := strings.TrimSpace(upperSQL[idx+11:])
			parts := strings.Fields(insertPart)
			if len(parts) > 0 {
				tableName := strings.Trim(parts[0], "\"'`[]")
				return strings.ToLower(tableName)
			}
		}
	} else if strings.HasPrefix(upperSQL, "UPDATE") {
		// Extract from UPDATE
		parts := strings.Fields(upperSQL)
		if len(parts) > 1 {
			tableName := strings.Trim(parts[1], "\"'`[]")
			return strings.ToLower(tableName)
		}
	} else if strings.HasPrefix(upperSQL, "DELETE FROM") {
		// Extract from DELETE FROM
		if idx := strings.Index(upperSQL, "DELETE FROM"); idx != -1 {
			deletePart := strings.TrimSpace(upperSQL[idx+11:])
			parts := strings.Fields(deletePart)
			if len(parts) > 0 {
				tableName := strings.Trim(parts[0], "\"'`[]")
				return strings.ToLower(tableName)
			}
		}
	}

	return "unknown"
}

// buildPostgreSQLBatchInsertSQL builds PostgreSQL-specific bulk insert SQL
// PostgreSQL uses $1, $2, $3... for parameters instead of ?
func buildPostgreSQLBatchInsertSQL(records []orm.DBRecord) (string, []interface{}, error) {
	if len(records) == 0 {
		return "", nil, fmt.Errorf("no records provided")
	}

	// Validate table name
	tableName := records[0].TableName
	if err := orm.ValidateTableName(tableName); err != nil {
		return "", nil, err
	}

	// Get column names from first record
	if records[0].Data == nil || len(records[0].Data) == 0 {
		return "", nil, fmt.Errorf("first record has no data")
	}

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
		sb.WriteString(col)
	}

	sb.WriteString(") VALUES ")

	// Build VALUES clause with PostgreSQL $N placeholders
	var values []interface{}
	paramIndex := 1

	for i, record := range records {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString("(")
		for j, col := range columns {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("$%d", paramIndex))
			paramIndex++
			values = append(values, record.Data[col])
		}
		sb.WriteString(")")
	}

	return sb.String(), values, nil
}

// validatePostgreSQLConnection validates the PostgreSQL connection and returns detailed info
func validatePostgreSQLConnection(db *sql.DB) error {
	// Test basic connectivity
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Test a simple query
	var version string
	err := db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		return fmt.Errorf("version query failed: %w", err)
	}

	// Check if we can query system catalogs
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pg_catalog.pg_database").Scan(&count)
	if err != nil {
		return fmt.Errorf("system catalog query failed: %w", err)
	}

	return nil
}

// convertToPostgreSQLPlaceholders converts ? placeholders to PostgreSQL $N placeholders
func convertToPostgreSQLPlaceholders(query string) string {
	paramIndex := 1
	result := ""
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(query); i++ {
		ch := query[i]

		// Track if we're inside a quoted string
		if ch == '\'' || ch == '"' {
			if !inQuote {
				inQuote = true
				quoteChar = ch
			} else if ch == quoteChar {
				// Check if it's an escaped quote
				if i+1 < len(query) && query[i+1] == ch {
					result += string(ch)
					i++ // Skip next character
					continue
				}
				inQuote = false
			}
		}

		// Replace ? with $N if not in a quote
		if ch == '?' && !inQuote {
			result += fmt.Sprintf("$%d", paramIndex)
			paramIndex++
		} else {
			result += string(ch)
		}
	}

	return result
}

// getPostgreSQLStats retrieves PostgreSQL database statistics
func getPostgreSQLStats(db *sql.DB, dbName string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get database size
	var dbSize int64
	err := db.QueryRow("SELECT pg_database_size($1)", dbName).Scan(&dbSize)
	if err == nil {
		stats["db_size"] = dbSize
	}

	// Get number of connections
	var numConnections int
	err = db.QueryRow("SELECT COUNT(*) FROM pg_stat_activity WHERE datname = $1", dbName).Scan(&numConnections)
	if err == nil {
		stats["connections"] = numConnections
	}

	// Get transaction stats
	query := `
		SELECT
			xact_commit,
			xact_rollback,
			blks_read,
			blks_hit,
			tup_returned,
			tup_fetched,
			tup_inserted,
			tup_updated,
			tup_deleted
		FROM pg_stat_database
		WHERE datname = $1`

	var xactCommit, xactRollback, blksRead, blksHit, tupReturned, tupFetched, tupInserted, tupUpdated, tupDeleted int64
	err = db.QueryRow(query, dbName).Scan(
		&xactCommit, &xactRollback, &blksRead, &blksHit,
		&tupReturned, &tupFetched, &tupInserted, &tupUpdated, &tupDeleted,
	)
	if err == nil {
		stats["xact_commit"] = xactCommit
		stats["xact_rollback"] = xactRollback
		stats["blks_read"] = blksRead
		stats["blks_hit"] = blksHit
		stats["cache_hit_ratio"] = float64(blksHit) / float64(blksHit+blksRead) * 100.0
		stats["tup_returned"] = tupReturned
		stats["tup_fetched"] = tupFetched
		stats["tup_inserted"] = tupInserted
		stats["tup_updated"] = tupUpdated
		stats["tup_deleted"] = tupDeleted
	}

	return stats, nil
}
