package rqlite

import (
	"net/url"
	"testing"

	orm "github.com/medatechnology/simpleorm"
)

// TestBuildURL tests the URL building helper
func TestBuildURL(t *testing.T) {
	db := &RQLiteDirectDB{
		Config: RqliteDirectConfig{
			URL: "http://localhost:4001",
		},
	}

	tests := []struct {
		name     string
		endpoint string
		params   url.Values
		expected string
	}{
		{
			name:     "Query endpoint without params",
			endpoint: "/db/query",
			params:   nil,
			expected: "http://localhost:4001/db/query",
		},
		{
			name:     "Execute endpoint without params",
			endpoint: "/db/execute",
			params:   nil,
			expected: "http://localhost:4001/db/execute",
		},
		{
			name:     "Query endpoint with consistency param",
			endpoint: "/db/query",
			params: url.Values{
				"consistency": []string{"strong"},
			},
			expected: "http://localhost:4001/db/query?consistency=strong",
		},
		{
			name:     "Query endpoint with multiple params",
			endpoint: "/db/query",
			params: url.Values{
				"consistency": []string{"weak"},
				"pretty":      []string{"true"},
			},
			expected: "http://localhost:4001/db/query?consistency=weak&pretty=true",
		},
		{
			name:     "Status endpoint",
			endpoint: "/status",
			params:   nil,
			expected: "http://localhost:4001/status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := db.buildURL(tt.endpoint, tt.params)
			// For tests with params, we can't guarantee order, so just check if it starts correctly
			if tt.params == nil {
				if result != tt.expected {
					t.Errorf("buildURL() = %q; want %q", result, tt.expected)
				}
			} else {
				// Check that URL starts with base and endpoint
				baseWithEndpoint := db.Config.URL + tt.endpoint
				if len(result) < len(baseWithEndpoint) || result[:len(baseWithEndpoint)] != baseWithEndpoint {
					t.Errorf("buildURL() = %q; should start with %q", result, baseWithEndpoint)
				}
				// Check that params are in the URL
				for key, values := range tt.params {
					for _, value := range values {
						if !contains(result, key) || !contains(result, value) {
							t.Errorf("buildURL() = %q; should contain param %s=%s", result, key, value)
						}
					}
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestConvertToRQLiteParameterizedFormat tests the parameter conversion helper
func TestConvertToRQLiteParameterizedFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    []orm.ParametereizedSQL
		expected int // Length of result
	}{
		{
			name:     "Empty input",
			input:    []orm.ParametereizedSQL{},
			expected: 0,
		},
		{
			name: "Single param with values",
			input: []orm.ParametereizedSQL{
				{
					Query:  "INSERT INTO users (name, age) VALUES (?, ?)",
					Values: []interface{}{"Alice", 30},
				},
			},
			expected: 1,
		},
		{
			name: "Multiple params",
			input: []orm.ParametereizedSQL{
				{
					Query:  "INSERT INTO users (name) VALUES (?)",
					Values: []interface{}{"Bob"},
				},
				{
					Query:  "UPDATE users SET age = ? WHERE id = ?",
					Values: []interface{}{25, 1},
				},
			},
			expected: 2,
		},
		{
			name: "Param with map value",
			input: []orm.ParametereizedSQL{
				{
					Query: "INSERT INTO users (name, age) VALUES (:name, :age)",
					Values: []interface{}{
						map[string]interface{}{
							"name": "Charlie",
							"age":  35,
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "Param with no values",
			input: []orm.ParametereizedSQL{
				{
					Query:  "SELECT * FROM users",
					Values: []interface{}{},
				},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToRQLiteParameterizedFormat(tt.input)

			if len(result) != tt.expected {
				t.Errorf("convertToRQLiteParameterizedFormat() returned %d items; want %d", len(result), tt.expected)
			}

			// Verify each result is a slice with query as first element
			for i, item := range result {
				slice, ok := item.([]interface{})
				if !ok {
					t.Errorf("Result[%d] is not a slice", i)
					continue
				}

				if len(slice) == 0 {
					t.Errorf("Result[%d] is an empty slice", i)
					continue
				}

				// First element should be the query string
				if i < len(tt.input) {
					query, ok := slice[0].(string)
					if !ok {
						t.Errorf("Result[%d][0] is not a string", i)
					} else if query != tt.input[i].Query {
						t.Errorf("Result[%d][0] = %q; want %q", i, query, tt.input[i].Query)
					}
				}
			}
		})
	}
}

// TestCheckRQLiteErrors tests the error checking helper
func TestCheckRQLiteErrors(t *testing.T) {
	tests := []struct {
		name        string
		results     []interface{}
		expectError bool
	}{
		{
			name:        "Empty results",
			results:     []interface{}{},
			expectError: false,
		},
		{
			name: "Results without errors",
			results: []interface{}{
				map[string]interface{}{
					"error": "",
				},
				map[string]interface{}{
					"error": "",
				},
			},
			expectError: false,
		},
		{
			name: "Results with error",
			results: []interface{}{
				map[string]interface{}{
					"error": "",
				},
				map[string]interface{}{
					"error": "constraint violation",
				},
			},
			expectError: true,
		},
		{
			name: "Results with non-map item",
			results: []interface{}{
				"not a map",
			},
			expectError: false, // Non-map items are skipped
		},
		{
			name: "Results without error field",
			results: []interface{}{
				map[string]interface{}{
					"other_field": "value",
				},
			},
			expectError: false,
		},
	}

	errorField := func(item interface{}) string {
		if m, ok := item.(map[string]interface{}); ok {
			if err, ok := m["error"].(string); ok {
				return err
			}
		}
		return ""
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkRQLiteErrors(tt.results, errorField)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestGetStatusInfoFromResponse tests status info extraction
func TestGetStatusInfoFromResponse(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]interface{}
		expectError bool
		checkDBMS   bool
	}{
		{
			name:        "Empty response",
			input:       map[string]interface{}{},
			expectError: false,
			checkDBMS:   true,
		},
		{
			name: "Response with build info",
			input: map[string]interface{}{
				"build": map[string]interface{}{
					"version": "7.0.0",
					"commit":  "abc123",
				},
			},
			expectError: false,
			checkDBMS:   true,
		},
		{
			name: "Response with store info",
			input: map[string]interface{}{
				"store": map[string]interface{}{
					"raft": map[string]interface{}{
						"state": "Leader",
					},
				},
			},
			expectError: false,
			checkDBMS:   true,
		},
		{
			name: "Response with runtime info",
			input: map[string]interface{}{
				"runtime": map[string]interface{}{
					"GOOS":   "linux",
					"GOARCH": "amd64",
				},
			},
			expectError: false,
			checkDBMS:   true,
		},
		{
			name: "Complex response",
			input: map[string]interface{}{
				"build": map[string]interface{}{
					"version": "7.0.0",
				},
				"store": map[string]interface{}{
					"raft": map[string]interface{}{
						"state":  "Leader",
						"leader": "node1:4002",
					},
				},
				"runtime": map[string]interface{}{
					"GOOS": "linux",
				},
			},
			expectError: false,
			checkDBMS:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetStatusInfoFromResponse(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.checkDBMS && result.DBMS != "rqlite" {
				t.Errorf("Expected DBMS to be 'rqlite', got '%s'", result.DBMS)
			}

			// Verify Peers map is initialized
			if result.Peers == nil {
				t.Error("Expected Peers map to be initialized")
			}
		})
	}
}

// TestConvertToRQLiteParameterizedFormatWithMapValues tests map value handling
func TestConvertToRQLiteParameterizedFormatWithMapValues(t *testing.T) {
	input := []orm.ParametereizedSQL{
		{
			Query: "INSERT INTO users (name, age) VALUES (:name, :age)",
			Values: []interface{}{
				map[string]interface{}{
					"name": "Alice",
					"age":  30,
				},
			},
		},
	}

	result := convertToRQLiteParameterizedFormat(input)

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	slice, ok := result[0].([]interface{})
	if !ok {
		t.Fatal("Result is not a slice")
	}

	if len(slice) < 2 {
		t.Fatalf("Expected at least 2 elements in slice, got %d", len(slice))
	}

	// First element should be the query
	query, ok := slice[0].(string)
	if !ok {
		t.Fatal("First element is not a string")
	}
	if query != input[0].Query {
		t.Errorf("Query = %q; want %q", query, input[0].Query)
	}

	// Second element should be the map
	paramMap, ok := slice[1].(map[string]interface{})
	if !ok {
		t.Fatal("Second element is not a map")
	}

	if paramMap["name"] != "Alice" {
		t.Errorf("paramMap['name'] = %v; want 'Alice'", paramMap["name"])
	}
	if paramMap["age"] != 30 {
		t.Errorf("paramMap['age'] = %v; want 30", paramMap["age"])
	}
}

// TestConvertToRQLiteParameterizedFormatWithMultipleValues tests multiple value handling
func TestConvertToRQLiteParameterizedFormatWithMultipleValues(t *testing.T) {
	input := []orm.ParametereizedSQL{
		{
			Query:  "INSERT INTO users (name, age, email) VALUES (?, ?, ?)",
			Values: []interface{}{"Bob", 25, "bob@example.com"},
		},
	}

	result := convertToRQLiteParameterizedFormat(input)

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	slice, ok := result[0].([]interface{})
	if !ok {
		t.Fatal("Result is not a slice")
	}

	if len(slice) != 4 { // query + 3 values
		t.Fatalf("Expected 4 elements in slice (query + 3 values), got %d", len(slice))
	}

	// Check query
	if slice[0] != input[0].Query {
		t.Errorf("Query = %q; want %q", slice[0], input[0].Query)
	}

	// Check values
	if slice[1] != "Bob" {
		t.Errorf("Value[0] = %v; want 'Bob'", slice[1])
	}
	if slice[2] != 25 {
		t.Errorf("Value[1] = %v; want 25", slice[2])
	}
	if slice[3] != "bob@example.com" {
		t.Errorf("Value[2] = %v; want 'bob@example.com'", slice[3])
	}
}
