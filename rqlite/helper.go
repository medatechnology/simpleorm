package rqlite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	orm "github.com/medatechnology/simpleorm"

	"github.com/medatechnology/goutil/object"
)

// buildURL creates a complete URL with consistency and authentication parameters
func (db *RQLiteDirectDB) buildURL(endpoint string, params url.Values) string {
	if params == nil {
		params = url.Values{}
	}

	// Add consistency level if specified
	if db.Config.Consistency != "" {
		params.Set("level", db.Config.Consistency)
	}

	// Add parameters to URL
	queryString := params.Encode()
	if queryString != "" {
		endpoint = endpoint + "?" + queryString
	}

	return db.Config.URL + endpoint
}

// sendRequest sends a HTTP request to the RQLite server with retries
func (db *RQLiteDirectDB) sendRequest(method, endpoint string, params url.Values, body io.Reader) (*http.Response, error) {
	url := db.buildURL(endpoint, params)
	var lastErr error

	// Retry logic
	for attempt := 0; attempt < db.Config.RetryCount; attempt++ {
		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set content type for POST/PUT requests
		if method == http.MethodPost || method == http.MethodPut {
			req.Header.Set("Content-Type", "application/json")
		}

		// Set basic auth if credentials are provided
		if db.Config.Username != "" || db.Config.Password != "" {
			req.SetBasicAuth(db.Config.Username, db.Config.Password)
		}
		// fmt.Println("sendRequest:", req)
		resp, err := db.HTTPClient.Do(req)
		if err == nil {
			// Check if response indicates success (2xx status code)
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				// resp.Body.Close()
				return resp, nil
			}

			// Read and close response body
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Special handling for authentication issues
			if resp.StatusCode == http.StatusUnauthorized {
				return nil, fmt.Errorf("internal DBMS authentication error: invalid credentials for RQLite server")
			}

			// Create error with status code and response body
			lastErr = fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(respBody))
			// fmt.Printf("Send request error:%s, attemp:%d\n", lastErr, attempt)
		} else {
			fmt.Println("Send request error : ", err)
			lastErr = err
		}

		// Wait before retrying, but only if this isn't the last attempt
		if attempt < db.Config.RetryCount-1 {
			time.Sleep(DEFAULT_RETRY_TIMEOUT)
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", db.Config.RetryCount, lastErr)
}

// execQuery sends a query to the RQLite server
func (db *RQLiteDirectDB) execQuery(queries []string) (*QueryResponse, error) {
	// RQLite expects a simple JSON array of query strings
	requestBody, err := json.Marshal(queries)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}
	// fmt.Println("execQuery RequestBody = ", requestBody)
	resp, err := db.sendRequest(http.MethodPost, ENDPOINT_QUERY, nil, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// fmt.Println("Done execQuery sending success:", resp)

	var queryResp QueryResponse
	err = json.NewDecoder(resp.Body).Decode(&queryResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}

	// Check for errors in any of the results
	for _, result := range queryResp.Results {
		if result.Error != "" {
			return nil, fmt.Errorf("query error: %s", result.Error)
		}
	}

	return &queryResp, nil
}

// execCommand sends a write command to the RQLite server
func (db *RQLiteDirectDB) execCommand(commands []string) (*ExecuteResponse, error) {
	// RQLite expects a simple JSON array of command strings
	requestBody, err := json.Marshal(commands)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal commands: %w", err)
	}

	resp, err := db.sendRequest(http.MethodPost, ENDPOINT_EXECUTE, nil, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var execResp ExecuteResponse
	err = json.NewDecoder(resp.Body).Decode(&execResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode execute response: %w", err)
	}

	// Check for errors in any of the results
	for _, result := range execResp.Results {
		if result.Error != "" {
			return nil, fmt.Errorf("execute error: %s", result.Error)
		}
	}

	return &execResp, nil
}

// execCommandParameterized sends a write command with parameters to the RQLite server
func (db *RQLiteDirectDB) execCommandParameterized(commands []orm.ParametereizedSQL) (*ExecuteResponse, error) {
	// Convert to RQLite's expected format for parameterized queries
	// Each command becomes [statement, param1, param2, ...] or [statement, {param_map}]
	requestCommands := make([]interface{}, len(commands))
	for i, cmd := range commands {
		// Create a slice with the query as the first element
		paramArray := make([]interface{}, 0, len(cmd.Values)+1)
		paramArray = append(paramArray, cmd.Query)

		// Check if this is using named parameters (map) or positional parameters (array)
		if len(cmd.Values) == 1 {
			// Check if the single value is actually a map for named parameters
			if valMap, ok := cmd.Values[0].(map[string]interface{}); ok {
				paramArray = append(paramArray, valMap)
			} else {
				// Regular positional parameter
				paramArray = append(paramArray, cmd.Values[0])
			}
		} else {
			// Multiple positional parameters
			// for _, val := range cmd.Values {
			paramArray = append(paramArray, cmd.Values...)
			// }
		}

		requestCommands[i] = paramArray
	}

	// Encode commands as JSON array
	requestBody, err := json.Marshal(requestCommands)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameterized commands: %w", err)
	}

	resp, err := db.sendRequest(http.MethodPost, ENDPOINT_EXECUTE, nil, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var execResp ExecuteResponse
	err = json.NewDecoder(resp.Body).Decode(&execResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode execute response: %w", err)
	}

	// Check for errors in any of the results
	for _, result := range execResp.Results {
		if result.Error != "" {
			return nil, fmt.Errorf("execute error: %s", result.Error)
		}
	}

	return &execResp, nil
}

// execQueryParameterized sends a query with parameters to the RQLite server
func (db *RQLiteDirectDB) execQueryParameterized(queries []orm.ParametereizedSQL) (*QueryResponse, error) {
	// Convert to RQLite's expected format for parameterized queries
	// Each query becomes [statement, param1, param2, ...] or [statement, {param_map}]
	requestQueries := make([]interface{}, len(queries))
	for i, query := range queries {
		// Create a slice with the query as the first element
		paramArray := make([]interface{}, 0, len(query.Values)+1)
		paramArray = append(paramArray, query.Query)

		// Check if this is using named parameters (map) or positional parameters (array)
		if len(query.Values) == 1 {
			// Check if the single value is actually a map for named parameters
			if valMap, ok := query.Values[0].(map[string]interface{}); ok {
				paramArray = append(paramArray, valMap)
			} else {
				// Regular positional parameter
				paramArray = append(paramArray, query.Values[0])
			}
		} else {
			// Multiple positional parameters
			// for _, val := range query.Values {
			paramArray = append(paramArray, query.Values...)
			// }
		}

		requestQueries[i] = paramArray
	}

	// Encode queries as JSON array
	requestBody, err := json.Marshal(requestQueries)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameterized queries: %w", err)
	}

	resp, err := db.sendRequest(http.MethodPost, ENDPOINT_QUERY, nil, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var queryResp QueryResponse
	err = json.NewDecoder(resp.Body).Decode(&queryResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}

	// Check for errors in any of the results
	for _, result := range queryResp.Results {
		if result.Error != "" {
			return nil, fmt.Errorf("query error: %s", result.Error)
		}
	}

	return &queryResp, nil
}

// queryResultToDBRecord converts a RQLite query result to a DBRecord
func queryResultToDBRecord(result QueryResult, tableName string) ([]orm.DBRecord, error) {
	if len(result.Columns) == 0 || len(result.Values) == 0 {
		return nil, orm.ErrSQLNoRows
	}

	records := make([]orm.DBRecord, len(result.Values))

	for i, row := range result.Values {
		record := orm.DBRecord{
			TableName: tableName,
			Data:      make(map[string]interface{}),
		}

		for j, col := range result.Columns {
			if j < len(row) {
				record.Data[col] = row[j]
			}
		}

		records[i] = record
	}

	return records, nil
}

// executeResultToBasicSQLResult converts a RQLite execute result to a BasicSQLResult
func executeResultToBasicSQLResult(result ExecuteResult) orm.BasicSQLResult {
	return orm.BasicSQLResult{
		Error:        nil, // Error is handled separately
		Timing:       result.Time,
		RowsAffected: result.RowsAffected,
		LastInsertID: result.LastInsertID,
	}
}

// executeResultsToBasicSQLResults converts multiple RQLite execute results to BasicSQLResults
func executeResultsToBasicSQLResults(results []ExecuteResult) []orm.BasicSQLResult {
	sqlResults := make([]orm.BasicSQLResult, len(results))
	for i, result := range results {
		sqlResults[i] = executeResultToBasicSQLResult(result)
	}
	return sqlResults
}

// Helper function to attempt to extract table name from SQL
// This is a best-effort function that may not work for complex SQL
func getTableNameFromSQL(sql string) string {
	// Default table name if we can't determine it
	tableName := "unknown"

	// Uppercase for case-insensitive matching
	upperSQL := strings.ToUpper(sql)

	// Try to extract table name from SELECT statement
	if strings.Contains(upperSQL, "FROM") {
		parts := strings.Split(upperSQL, "FROM")
		if len(parts) >= 2 {
			// Get the part after FROM
			tablePart := strings.TrimSpace(parts[1])

			// Extract the table name by taking the part before any space, comma, or where clause
			tableEndMarkers := []string{" ", ",", "WHERE", "JOIN", "INNER", "LEFT", "RIGHT", "OUTER", "CROSS", "NATURAL", "GROUP", "ORDER", "HAVING", "LIMIT", "OFFSET", "UNION", "EXCEPT", "INTERSECT"}

			for _, marker := range tableEndMarkers {
				if strings.Contains(tablePart, marker) {
					tablePart = strings.Split(tablePart, marker)[0]
				}
			}

			// Remove any aliases and clean up
			tableName = strings.TrimSpace(tablePart)

			// Remove quoted identifiers if present
			tableName = strings.Trim(tableName, "\"'`[]")
		}
	}

	return tableName
}

// When calling rqlite/status it returns long JSON format we only
// take what is needed
func GetStatusInfoFromResponse(raw map[string]interface{}) (orm.NodeStatusStruct, error) {
	// NOTE: this is the rqlite status return time format
	layout := time.RFC3339Nano // This is the format for timestamps like the one in the example.
	info := orm.NodeStatusStruct{}
	info.Peers = make(map[int]orm.StatusStruct)

	// Predefined value based on this package
	info.DBMS = "rqlite"
	info.DBMSDriver = "direct-rqlite"

	// Get version if available
	if build, ok := raw["build"].(map[string]interface{}); ok {
		if version, ok := build["version"].(string); ok {
			info.Version = version
		}
	}

	// Get directory size if available
	if store, ok := raw["store"].(map[string]interface{}); ok {
		// Node ID
		if nodeID, ok := store["node_id"].(string); ok {
			info.NodeID = nodeID
		}

		if url, ok := store["addr"].(string); ok {
			info.URL = url
		}

		// Directory size (convert from bytes to human-readable)
		if dirSize, ok := store["dir_size"].(float64); ok {
			info.DirSize = int64(dirSize)
		}

		// Database size from sqlite3 section
		if sqlite3, ok := store["sqlite3"].(map[string]interface{}); ok {
			if dbSize, ok := sqlite3["db_size"].(float64); ok {
				info.DBSize = int64(dbSize)
			}

			// Connection information on sqlite object
			if connPool, ok := sqlite3["conn_pool_stats"].(map[string]interface{}); ok {
				var minPool, rwPool, roPool int
				if roConn, ok := connPool["ro"].(map[string]interface{}); ok {
					if roPool, ok = roConn["max_open_connections"].(int); !ok {
						roPool = 1 // always assume 1 max_pool if no info, just to be safe
					}
					// 0 means no limit in rqlite
					if roPool == 0 {
						roPool = DEFAULT_MAX_POOL
					}
				}
				if rwConn, ok := connPool["rw"].(map[string]interface{}); ok {
					if rwPool, ok = rwConn["max_open_connections"].(int); !ok {
						rwPool = 1 // always assume 1 max_pool if no info, just to be safe
					}
					// 0 means no limit in rqlite
					if rwPool == 0 {
						rwPool = DEFAULT_MAX_POOL
					}
				}
				minPool = roPool
				// maxPool = rwPool
				if roPool > rwPool {
					minPool = rwPool
					// maxPool = roPool
				}
				info.MaxPool = minPool // take the minimum to be safe
			}
		}

		// Get nodes if available
		if nodes, ok := store["nodes"].([]interface{}); ok {
			for _, n := range nodes {
				if node, ok := n.(map[string]interface{}); ok {
					newNode := orm.StatusStruct{}
					if id, ok := node["id"].(string); ok {
						newNode.NodeID = id
						newNode.NodeNumber = object.Int(id, false)
					}
					if addr, ok := node["addr"].(string); ok {
						newNode.URL = addr
					}
					if newNode.NodeID != "" || newNode.URL != "" {
						info.Peers[newNode.NodeNumber] = newNode
						// info.Peers = append(info.Peers, newNode)
					}
				}
			}
			info.Nodes = len(nodes)
		} else {
			info.Nodes = 1
		}

		// Get leader if available
		if leader, ok := store["leader"].(map[string]interface{}); ok {
			if addr, ok := leader["addr"].(string); ok {
				info.Leader = addr
			}
			if leaderID, ok := leader["node_id"].(string); ok {
				if leaderID == info.NodeID {
					info.IsLeader = true
				} else {
					info.IsLeader = false
				}
			}
		}
	}

	// Get node times if available
	if node, ok := raw["node"].(map[string]interface{}); ok {
		if start, ok := node["start_time"].(string); ok {
			// Parse the string into time.Time
			parsedTime, err := time.Parse(layout, start)
			if err != nil {
				// If parsing fails, return the zero time value
				parsedTime = time.Time{}
			}
			info.StartTime = parsedTime
		}
		if uptime, ok := node["uptime"].(string); ok {
			// Parse the string into time.Duration
			parsedDuration, err := time.ParseDuration(uptime)
			if err != nil {
				// If parsing fails, return a zero duration
				fmt.Println("Error parsing duration:", err)
				parsedDuration = 0
			}
			info.Uptime = parsedDuration
		}
	}

	// Get last backup if available
	if lastBackup, ok := raw["last_backup_time"].(string); ok {
		// Parse the string into time.Time
		parsedTime, err := time.Parse(layout, lastBackup)
		if err != nil {
			// If parsing fails, return the zero time value
			parsedTime = time.Time{}
		}
		info.LastBackup = parsedTime
	}

	return info, nil
}
