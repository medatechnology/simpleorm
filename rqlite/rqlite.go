package rqlite

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	orm "github.com/medatechnology/simpleorm"
)

// This is our own version of go-rqlite which is a golang package official from RQLite team.
// Because we found the go-rqlite has some limitation for the funcionality, we careate our own
// based on rqlite endpoints directly which are: /query /execute and /request
// but still implementing our own simpleORM.
//
// Original RQLite has these endpoints:
// all: user can perform all operations on a node.
// execute: user may access the execute endpoint at /db/execute.
// query: user may access the query endpoint at /db/query.
// load: user may load an SQLite dump file into a node via the /db/load or /boot endpoints.
// backup: user may retrieve a backup via the endpoint /db/backup.
// snapshot: user may initiate a Raft Snapshot via the endpoint /snapshot.
// status: user can retrieve node status and Go runtime information.
// ready: user can retrieve node readiness via /readyz
// join: user can join a cluster. In practice only a node joins a cluster, so it’s the joining node that must supply the credentials.
// join-read-only: user can join a cluster, but only as a read-only node.
// remove: user can remove a node from a cluster. If a node performs an auto-remove on shutdown, then the -join-as user must have this permission.

// NewDatabase creates a new RQLiteDirectDB instance
func NewDatabase(config RqliteDirectConfig) (*RQLiteDirectDB, error) {
	// Set default timeout if not specified
	timeout := config.Timeout
	if timeout == 0 {
		timeout = DEFAULT_TIMEOUT
	}

	// Ensure URL doesn't end with a slash
	if config.URL != "" && strings.HasSuffix(config.URL, "/") {
		config.URL = config.URL[:len(config.URL)-1]
	}

	// Set default retry count if not specified
	if config.RetryCount == 0 {
		config.RetryCount = DEFAULT_MAX_RETRIES
	}

	return &RQLiteDirectDB{
		Config: config,
		HTTPClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   DEFAULT_TIMEOUT,
					KeepAlive: DEFAULT_KEEP_ALIVE,
				}).Dial,
				TLSHandshakeTimeout:   DEFAULT_TLS_HANDSHAKE_TIMEOUT,
				ResponseHeaderTimeout: DEFAULT_RESPONSE_TIMEOUT,
				ExpectContinueTimeout: DEFAULT_CONTINUE_TIMEOUT,
				MaxIdleConns:          DEFAULT_MAX_IDLE_CONNECTIONS,
				MaxIdleConnsPerHost:   DEFAULT_MAX_IDLE_CONNECTIONS_PER_HOST,
				MaxConnsPerHost:       DEFAULT_MAX_CONNECTIONS_PER_HOST,
				IdleConnTimeout:       DEFAULT_IDLE_CONNECTION_TIMEOUT,
			},
		},
	}, nil
}

// IsConnected checks if the database connection is alive
func (db *RQLiteDirectDB) IsConnected() bool {
	// _, err := db.Status()
	// return err == nil
	return db.HTTPClient != nil
}

// GetSchema returns the database schema
func (db *RQLiteDirectDB) GetSchema(hideSQL, hideSureSQL bool) []orm.SchemaStruct {
	// Query the sqlite_master table to get schema information
	query := "SELECT * FROM " + SCHEMA_TABLE + " ORDER BY type, tbl_name, name"
	resp, err := db.execQuery([]string{query})
	if err != nil {
		return []orm.SchemaStruct{}
	}

	if len(resp.Results) == 0 || len(resp.Results[0].Values) == 0 {
		return []orm.SchemaStruct{}
	}

	var schemas []orm.SchemaStruct
	result := resp.Results[0]

	for _, row := range result.Values {
		// Map columns to struct fields
		var schema orm.SchemaStruct
		for i, col := range result.Columns {
			value := row[i]
			switch col {
			case "type":
				if strVal, ok := value.(string); ok {
					schema.ObjectType = strVal
				}
			case "name":
				if strVal, ok := value.(string); ok {
					schema.ObjectName = strVal
				}
			case "tbl_name":
				if strVal, ok := value.(string); ok {
					schema.TableName = strVal
				}
			case "rootpage":
				if numVal, ok := value.(float64); ok {
					schema.RootPage = int(numVal)
				}
			case "sql":
				if strVal, ok := value.(string); ok {
					schema.SQLCommand = strVal
				}
			}
		}

		// Filter based on hideSQL and hideSureSQL flags
		if (hideSQL && strings.HasPrefix(schema.TableName, PREFIX_SQLITE_TABLE)) ||
			(hideSureSQL && strings.HasPrefix(schema.TableName, PREFIX_SURESQL_TABLE)) {
			continue
		}

		schemas = append(schemas, schema)
	}

	return schemas
}

// Status returns the status of the RQLite cluster
func (db *RQLiteDirectDB) Status() (orm.NodeStatusStruct, error) {
	// Use the /status endpoint to get cluster status
	resp, err := db.sendRequest(http.MethodGet, "/status", nil, nil)
	if err != nil {
		return orm.NodeStatusStruct{}, err
	}
	defer resp.Body.Close()

	var statusResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&statusResp)
	if err != nil {
		return orm.NodeStatusStruct{}, fmt.Errorf("failed to decode status response: %w", err)
	}

	status, err := GetStatusInfoFromResponse(statusResp)
	// // Extract status information
	// var status []string

	// // Try to extract metadata about the cluster
	// if meta, ok := statusResp["metadata"].(map[string]interface{}); ok {
	// 	if nodeID, ok := meta["node_id"].(string); ok {
	// 		status = append(status, fmt.Sprintf("Node ID: %s", nodeID))
	// 	}
	// }

	// // Try to extract build information
	// if build, ok := statusResp["build"].(map[string]interface{}); ok {
	// 	if version, ok := build["version"].(string); ok {
	// 		status = append(status, fmt.Sprintf("Version: %s", version))
	// 	}
	// }

	// // If we couldn't extract specific info, just use the URL as status
	// if len(status) == 0 {
	// 	status = append(status, db.Config.URL)
	// }

	return status, err
}

// Leader returns the leader node of the RQLite cluster
func (db *RQLiteDirectDB) Leader() (string, error) {
	// Use the /status endpoint to get leader information
	resp, err := db.sendRequest(http.MethodGet, "/status", nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var statusResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&statusResp)
	if err != nil {
		return "", fmt.Errorf("failed to decode status response: %w", err)
	}

	// Extract leader information
	if store, ok := statusResp["store"].(map[string]interface{}); ok {
		if raft, ok := store["raft"].(map[string]interface{}); ok {
			if leader, ok := raft["leader"].(string); ok {
				return leader, nil
			}
		}
	}

	return "", fmt.Errorf("leader information not available")
}

// Peers returns the peer nodes of the RQLite cluster
func (db *RQLiteDirectDB) Peers() ([]string, error) {
	// Use the /status endpoint to get peer information
	resp, err := db.sendRequest(http.MethodGet, "/status", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var statusResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&statusResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	var peers []string

	// Extract peer information
	if store, ok := statusResp["store"].(map[string]interface{}); ok {
		if raft, ok := store["raft"].(map[string]interface{}); ok {
			if peerList, ok := raft["peers"].([]interface{}); ok {
				for _, peer := range peerList {
					if peerStr, ok := peer.(string); ok {
						peers = append(peers, peerStr)
					}
				}
			}
		}
	}

	if len(peers) == 0 {
		return nil, fmt.Errorf("peer information not available")
	}

	return peers, nil
}

// SelectOne selects a single record from the table
func (db *RQLiteDirectDB) SelectOne(tableName string) (orm.DBRecord, error) {
	// Security: Validate table name to prevent SQL injection
	if err := orm.ValidateTableName(tableName); err != nil {
		return orm.DBRecord{}, orm.WrapSelectError(err, tableName)
	}

	query := fmt.Sprintf("SELECT * FROM %s LIMIT 1", tableName)
	resp, err := db.execQuery([]string{query})
	if err != nil {
		return orm.DBRecord{}, orm.WrapErrorWithQuery(err, "SELECT", tableName, query)
	}

	if len(resp.Results) == 0 || len(resp.Results[0].Values) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}

	records, err := queryResultToDBRecord(resp.Results[0], tableName)
	if err != nil {
		return orm.DBRecord{}, orm.WrapSelectError(err, tableName)
	}

	return records[0], nil
}

// SelectMany selects multiple records from the table
func (db *RQLiteDirectDB) SelectMany(tableName string) (orm.DBRecords, error) {
	// Security: Validate table name to prevent SQL injection
	if err := orm.ValidateTableName(tableName); err != nil {
		return nil, orm.WrapSelectError(err, tableName)
	}

	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	resp, err := db.execQuery([]string{query})
	if err != nil {
		return nil, orm.WrapErrorWithQuery(err, "SELECT", tableName, query)
	}

	if len(resp.Results) == 0 || len(resp.Results[0].Values) == 0 {
		return nil, orm.ErrSQLNoRows
	}

	records, err := queryResultToDBRecord(resp.Results[0], tableName)
	if err != nil {
		return nil, orm.WrapSelectError(err, tableName)
	}

	return records, nil
}

// SelectOneWithCondition selects a single record with a condition
func (db *RQLiteDirectDB) SelectOneWithCondition(tableName string, condition *orm.Condition) (orm.DBRecord, error) {
	if condition == nil {
		return db.SelectOne(tableName)
	}

	query, params, err := condition.ToSelectString(tableName)
	if err != nil {
		return orm.DBRecord{}, orm.WrapSelectError(fmt.Errorf("failed to build query: %w", err), tableName)
	}

	// Add LIMIT 1 to ensure we only get one record if not already specified
	if !strings.Contains(strings.ToUpper(query), "LIMIT") {
		query += " LIMIT 1"
	}

	paramSQL := orm.ParametereizedSQL{
		Query:  query,
		Values: params,
	}

	resp, err := db.execQueryParameterized([]orm.ParametereizedSQL{paramSQL})
	if err != nil {
		return orm.DBRecord{}, orm.WrapErrorWithQuery(err, "SELECT", tableName, query)
	}

	if len(resp.Results) == 0 || len(resp.Results[0].Values) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}

	records, err := queryResultToDBRecord(resp.Results[0], tableName)
	if err != nil {
		return orm.DBRecord{}, orm.WrapSelectError(err, tableName)
	}

	return records[0], nil
}

// SelectManyWithCondition selects multiple records with a condition
func (db *RQLiteDirectDB) SelectManyWithCondition(tableName string, condition *orm.Condition) ([]orm.DBRecord, error) {
	if condition == nil {
		return db.SelectMany(tableName)
	}

	query, params, err := condition.ToSelectString(tableName)
	if err != nil {
		return nil, orm.WrapSelectError(fmt.Errorf("failed to build query: %w", err), tableName)
	}

	paramSQL := orm.ParametereizedSQL{
		Query:  query,
		Values: params,
	}

	resp, err := db.execQueryParameterized([]orm.ParametereizedSQL{paramSQL})
	if err != nil {
		return nil, orm.WrapErrorWithQuery(err, "SELECT", tableName, query)
	}

	if len(resp.Results) == 0 || len(resp.Results[0].Values) == 0 {
		return nil, orm.ErrSQLNoRows
	}

	records, err := queryResultToDBRecord(resp.Results[0], tableName)
	if err != nil {
		return nil, orm.WrapSelectError(err, tableName)
	}

	return records, nil
}

// SelectOneSQL executes a single SQL query and returns the results
func (db *RQLiteDirectDB) SelectOneSQL(sql string) (orm.DBRecords, error) {
	resp, err := db.execQuery([]string{sql})
	if err != nil {
		tableName := getTableNameFromSQL(sql)
		return nil, orm.WrapErrorWithQuery(err, "SELECT", tableName, sql)
	}

	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no results returned")
	}

	// Convert the first result to DBRecords
	// We're using a placeholder table name since the actual table name is unknown
	// from the raw SQL statement
	tableName := getTableNameFromSQL(sql)
	records, err := queryResultToDBRecord(resp.Results[0], tableName)
	if err != nil {
		return nil, orm.WrapSelectError(err, tableName)
	}

	return records, nil
}

// SelectManySQL executes multiple SQL queries and returns the results of each
func (db *RQLiteDirectDB) SelectManySQL(sqls []string) ([]orm.DBRecords, error) {
	resp, err := db.execQuery(sqls)
	if err != nil {
		return nil, orm.WrapError(err, "SELECT", "")
	}

	results := make([]orm.DBRecords, 0, len(resp.Results))

	for i, result := range resp.Results {
		// Use table name from SQL if possible
		tableName := "unknown"
		if i < len(sqls) {
			tableName = getTableNameFromSQL(sqls[i])
		}

		if result.Error != "" {
			queryErr := fmt.Errorf("query error: %s", result.Error)
			return results, orm.WrapErrorWithQuery(queryErr, "SELECT", tableName, sqls[i])
		}

		records, err := queryResultToDBRecord(result, tableName)
		if err != nil {
			// If this specific query returns no rows, append an empty slice instead of failing the entire batch
			if err == orm.ErrSQLNoRows {
				results = append(results, orm.DBRecords{})
				continue
			}
			return results, orm.WrapSelectError(err, tableName)
		}

		results = append(results, records)
	}

	return results, nil
}

// SelectOnlyOneSQL executes a SQL query and ensures exactly one row is returned
func (db *RQLiteDirectDB) SelectOnlyOneSQL(sql string) (orm.DBRecord, error) {
	records, err := db.SelectOneSQL(sql)
	if err != nil {
		return orm.DBRecord{}, err // Already wrapped in SelectOneSQL
	}

	// Check if we got exactly one row
	if len(records) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}

	// because OnlyOne, if there are more than 1, that counts as error
	if len(records) > 1 {
		tableName := getTableNameFromSQL(sql)
		return orm.DBRecord{}, orm.WrapSelectError(orm.ErrSQLMoreThanOneRow, tableName)
	}

	return records[0], nil
}

// SelectOneSQLParameterized executes a single parameterized SQL query
func (db *RQLiteDirectDB) SelectOneSQLParameterized(paramSQL orm.ParametereizedSQL) (orm.DBRecords, error) {
	resp, err := db.execQueryParameterized([]orm.ParametereizedSQL{paramSQL})
	if err != nil {
		tableName := getTableNameFromSQL(paramSQL.Query)
		return nil, orm.WrapErrorWithQuery(err, "SELECT", tableName, paramSQL.Query)
	}

	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no results returned")
	}

	// Convert the first result to DBRecords
	tableName := getTableNameFromSQL(paramSQL.Query)
	records, err := queryResultToDBRecord(resp.Results[0], tableName)
	if err != nil {
		return nil, orm.WrapSelectError(err, tableName)
	}

	return records, nil
}

// SelectManySQLParameterized executes multiple parameterized SQL queries
func (db *RQLiteDirectDB) SelectManySQLParameterized(paramSQLs []orm.ParametereizedSQL) ([]orm.DBRecords, error) {
	resp, err := db.execQueryParameterized(paramSQLs)
	if err != nil {
		return nil, orm.WrapError(err, "SELECT", "")
	}

	results := make([]orm.DBRecords, 0, len(resp.Results))

	for i, result := range resp.Results {
		// Use table name from SQL if possible
		tableName := "unknown"
		if i < len(paramSQLs) {
			tableName = getTableNameFromSQL(paramSQLs[i].Query)
		}

		if result.Error != "" {
			queryErr := fmt.Errorf("query error: %s", result.Error)
			query := ""
			if i < len(paramSQLs) {
				query = paramSQLs[i].Query
			}
			return results, orm.WrapErrorWithQuery(queryErr, "SELECT", tableName, query)
		}

		records, err := queryResultToDBRecord(result, tableName)
		if err != nil {
			// If this specific query returns no rows, append an empty slice instead of failing the entire batch
			if err == orm.ErrSQLNoRows {
				results = append(results, orm.DBRecords{})
				continue
			}
			return results, orm.WrapSelectError(err, tableName)
		}

		results = append(results, records)
	}

	return results, nil
}

// SelectOnlyOneSQLParameterized executes a parameterized SQL query and ensures exactly one row is returned
func (db *RQLiteDirectDB) SelectOnlyOneSQLParameterized(paramSQL orm.ParametereizedSQL) (orm.DBRecord, error) {
	records, err := db.SelectOneSQLParameterized(paramSQL)
	if err != nil {
		return orm.DBRecord{}, err // Already wrapped in SelectOneSQLParameterized
	}

	// Check if we got exactly one row
	if len(records) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}
	// because OnlyOne, if there are more than 1, that counts as error
	if len(records) > 1 {
		tableName := getTableNameFromSQL(paramSQL.Query)
		return orm.DBRecord{}, orm.WrapSelectError(orm.ErrSQLMoreThanOneRow, tableName)
	}

	return records[0], nil
}

// ExecOneSQL executes a single SQL statement
func (db *RQLiteDirectDB) ExecOneSQL(sql string) orm.BasicSQLResult {
	resp, err := db.execCommand([]string{sql})
	if err != nil {
		tableName := getTableNameFromSQL(sql)
		operation := getOperationFromSQL(sql)
		wrappedErr := orm.WrapErrorWithQuery(err, operation, tableName, sql)
		return orm.BasicSQLResult{Error: wrappedErr}
	}

	if len(resp.Results) == 0 {
		return orm.BasicSQLResult{Error: fmt.Errorf("no results returned")}
	}

	result := executeResultToBasicSQLResult(resp.Results[0])
	return result
}

// ExecOneSQLParameterized executes a single parameterized SQL statement
func (db *RQLiteDirectDB) ExecOneSQLParameterized(paramSQL orm.ParametereizedSQL) orm.BasicSQLResult {
	resp, err := db.execCommandParameterized([]orm.ParametereizedSQL{paramSQL})
	if err != nil {
		tableName := getTableNameFromSQL(paramSQL.Query)
		operation := getOperationFromSQL(paramSQL.Query)
		wrappedErr := orm.WrapErrorWithQuery(err, operation, tableName, paramSQL.Query)
		return orm.BasicSQLResult{Error: wrappedErr}
	}

	if len(resp.Results) == 0 {
		return orm.BasicSQLResult{Error: fmt.Errorf("no results returned")}
	}

	result := executeResultToBasicSQLResult(resp.Results[0])
	return result
}

// ExecManySQL executes multiple SQL statements
func (db *RQLiteDirectDB) ExecManySQL(sqls []string) ([]orm.BasicSQLResult, error) {
	resp, err := db.execCommand(sqls)
	if err != nil {
		return nil, orm.WrapError(err, "EXEC", "")
	}

	return executeResultsToBasicSQLResults(resp.Results), nil
}

// ExecManySQLParameterized executes multiple parameterized SQL statements
func (db *RQLiteDirectDB) ExecManySQLParameterized(paramSQLs []orm.ParametereizedSQL) ([]orm.BasicSQLResult, error) {
	resp, err := db.execCommandParameterized(paramSQLs)
	if err != nil {
		return nil, orm.WrapError(err, "EXEC", "")
	}

	return executeResultsToBasicSQLResults(resp.Results), nil
}

// InsertOneDBRecord inserts a single record
func (db *RQLiteDirectDB) InsertOneDBRecord(record orm.DBRecord, queue bool) orm.BasicSQLResult {
	sql, values := record.ToInsertSQLParameterized()
	paramSQL := orm.ParametereizedSQL{
		Query:  sql,
		Values: values,
	}

	// RQLite doesn't have a queue mechanism like gorqlite, so we ignore the queue parameter
	return db.ExecOneSQLParameterized(paramSQL)
}

// InsertManyDBRecords inserts multiple records
func (db *RQLiteDirectDB) InsertManyDBRecords(records []orm.DBRecord, queue bool) ([]orm.BasicSQLResult, error) {
	paramSQLs := make([]orm.ParametereizedSQL, 0, len(records))

	for _, record := range records {
		sql, values := record.ToInsertSQLParameterized()
		paramSQL := orm.ParametereizedSQL{
			Query:  sql,
			Values: values,
		}
		paramSQLs = append(paramSQLs, paramSQL)
	}

	// RQLite doesn't have a queue mechanism like gorqlite, so we ignore the queue parameter
	return db.ExecManySQLParameterized(paramSQLs)
}

// InsertManyDBRecordsSameTable inserts multiple records into the same table
func (db *RQLiteDirectDB) InsertManyDBRecordsSameTable(records []orm.DBRecord, queue bool) ([]orm.BasicSQLResult, error) {
	if len(records) == 0 {
		return nil, orm.WrapInsertError(fmt.Errorf("no records to insert"), "")
	}

	// For records of the same table, we can optimize by using the batch insert functionality
	tableName := records[0].TableName
	paramSQLs := orm.DBRecords(records).ToInsertSQLParameterized()

	// RQLite doesn't have a queue mechanism like gorqlite, so we ignore the queue parameter
	results, err := db.ExecManySQLParameterized(paramSQLs)
	if err != nil {
		return results, orm.WrapInsertError(err, tableName)
	}
	return results, nil
}

// InsertOneTableStruct inserts a single table struct
func (db *RQLiteDirectDB) InsertOneTableStruct(obj orm.TableStruct, queue bool) orm.BasicSQLResult {
	record, err := orm.TableStructToDBRecord(obj)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}

	return db.InsertOneDBRecord(record, queue)
}

// InsertManyTableStructs inserts multiple table structs
func (db *RQLiteDirectDB) InsertManyTableStructs(objs []orm.TableStruct, queue bool) ([]orm.BasicSQLResult, error) {
	if len(objs) == 0 {
		return nil, orm.WrapInsertError(fmt.Errorf("no objects to insert"), "")
	}

	records := make([]orm.DBRecord, 0, len(objs))
	for _, obj := range objs {
		record, err := orm.TableStructToDBRecord(obj)
		if err != nil {
			return nil, orm.WrapInsertError(err, obj.TableName())
		}
		records = append(records, record)
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
