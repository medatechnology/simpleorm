package orm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/medatechnology/goutil/medaerror"
)

const (
	DEFAULT_PAGINATION_LIMIT     = 50
	DEFAULT_MAX_MULTIPLE_INSERTS = 100 // Maximum number of rows to insert in a single SQL statement
)

var (
	// Some global vars are needed so we can change this on the fly later on.
	// For example: we read the MAX_MULTIPLE_INSERTS from db.settings then
	// it will be changed on the fly when function that need this variable got called!
	ErrSQLNoRows         medaerror.MedaError = medaerror.MedaError{Message: "select returns no rows"}
	ErrSQLMoreThanOneRow medaerror.MedaError = medaerror.MedaError{Message: "select returns more than 1 rows"}
	MAX_MULTIPLE_INSERTS int                 = DEFAULT_MAX_MULTIPLE_INSERTS

	// Security: SQL Injection Protection
	ErrInvalidFieldName    medaerror.MedaError = medaerror.MedaError{Message: "invalid field name: must contain only alphanumeric characters and underscores"}
	ErrInvalidOperator     medaerror.MedaError = medaerror.MedaError{Message: "invalid SQL operator: not in allowed list"}
	ErrEmptyTableName      medaerror.MedaError = medaerror.MedaError{Message: "table name cannot be empty"}
	ErrInvalidTableName    medaerror.MedaError = medaerror.MedaError{Message: "invalid table name: must contain only alphanumeric characters and underscores"}
	ErrEmptyConditionField medaerror.MedaError = medaerror.MedaError{Message: "condition field cannot be empty when operator is specified"}
	ErrMissingPrimaryKey   medaerror.MedaError = medaerror.MedaError{Message: "missing primary key in record data"}
	ErrSQLMultipleRows     medaerror.MedaError = medaerror.MedaError{Message: "query returned multiple rows when expecting one"}

	// Whitelist of allowed SQL operators to prevent SQL injection
	allowedOperators = map[string]bool{
		"=":           true,
		"!=":          true,
		"<>":          true,
		">":           true,
		"<":           true,
		">=":          true,
		"<=":          true,
		"LIKE":        true,
		"NOT LIKE":    true,
		"ILIKE":       true, // PostgreSQL case-insensitive LIKE
		"IN":          true,
		"NOT IN":      true,
		"BETWEEN":     true,
		"IS":          true,
		"IS NOT":      true,
		"IS NULL":     true,
		"IS NOT NULL": true,
	}

	// Regular expression for validating SQL identifiers (table/column names)
	// Allows: letters, numbers, underscores; must start with letter or underscore
	sqlIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// Struct to get the schema from sqlite_master table in SQLite
type SchemaStruct struct {
	ObjectType string `json:"type"           db:"type"`
	ObjectName string `json:"name"           db:"name"`
	TableName  string `json:"tbl_name"       db:"tbl_name"`
	RootPage   int    `json:"rootpage"       db:"rootpage"`
	SQLCommand string `json:"sql"            db:"sql"`
	Hidden     bool   `json:"hidden"         db:"hidden"`
}

// Make sure other table struct that you use implement this method
type TableStruct interface {
	TableName() string
}

// mostly used for rawSQL execution, this is the return, empty if it's not applicable
// This is not for query where we return usually DBRecord or DBRecords / []DBRecord
type BasicSQLResult struct {
	Error        error
	Timing       float64
	RowsAffected int
	LastInsertID int
}

type ParametereizedSQL struct {
	Query  string        `json:"query"`
	Values []interface{} `json:"values,omitempty"`
}

// Condition struct for query filtering with JSON and DB tags
// This struct is used to define conditions for filtering data in queries.
// It supports various operations like AND, OR, and nested conditions.
// Sample usage:
//
//	// Simple condition
//	condition := Condition{
//	  Field:    "age",
//	  Operator: ">",
//	  Value:    18,
//	}
//	// Output: WHERE age > 18
//
//	// Nested condition with AND logic
//	condition := Condition{
//	  Logic: "AND",
//	  Nested: []Condition{
//	    Condition{Field: "age", Operator: ">", Value: 18},
//	    Condition{Field: "status", Operator: "=", Value: "active"},
//	  },
//	}
//	// Output: WHERE (age > 18 AND status = 'active')
//
//	// Nested condition with OR logic
//	condition := Condition{
//	  Logic: "OR",
//	  Nested: []Condition{
//	    Condition{Field: "status", Operator: "=", Value: "pending"},
//	    Condition{Field: "status", Operator: "=", Value: "review"},
//	  },
//	}
//	// Output: WHERE (status = 'pending' OR status = 'review')
//
//	// Complex condition with nested AND and OR
//	condition := Condition{
//	  Logic: "OR",
//	  Nested: []Condition{
//	    Condition{
//	      Logic: "AND",
//	      Nested: []Condition{
//	        Condition{Field: "age", Operator: ">", Value: 18},
//	        Condition{Field: "country", Operator: "=", Value: "USA"},
//	      },
//	    },
//	    Condition{
//	      Logic: "AND",
//	      Nested: []Condition{
//	        Condition{Field: "status", Operator: "=", Value: "active"},
//	        Condition{Field: "role", Operator: "=", Value: "admin"},
//	      },
//	    },
//	  },
//	}
//	// Output: WHERE ((age > 18 AND country = 'USA') OR (status = 'active' AND role = 'admin'))
type Condition struct {
	Field    string      `json:"field,omitempty"        db:"field"`
	Operator string      `json:"operator,omitempty"     db:"operator"`
	Value    interface{} `json:"value,omitempty"        db:"value"`
	Logic    string      `json:"logic,omitempty"        db:"logic"`    // "AND" or "OR"
	Nested   []Condition `json:"nested,omitempty"       db:"nested"`   // For nested conditions
	OrderBy  []string    `json:"order_by,omitempty"     db:"order_by"` // Fields to order by
	GroupBy  []string    `json:"group_by,omitempty"     db:"group_by"` // Fields to group by
	Limit    int         `json:"limit,omitempty"        db:"limit"`    // Limit for pagination
	Offset   int         `json:"offset,omitempty"       db:"offset"`   // Offset for pagination
}

// And creates a new Condition with AND logic for the given conditions.
// This method allows chaining multiple conditions together with AND logic.
// Usage:
//
//	condition.And(
//	  Condition{Field: "age", Operator: ">", Value: 18},
//	  Condition{Field: "status", Operator: "=", Value: "active"}
//	)
//
// Returns: A new Condition with nested conditions joined by AND
func (c *Condition) And(conditions ...Condition) *Condition {
	return &Condition{
		Logic:  "AND",
		Nested: conditions,
	}
}

// Or creates a new Condition with OR logic for the given conditions.
// This method allows chaining multiple conditions together with OR logic.
// Usage:
//
//	condition.Or(
//	  Condition{Field: "status", Operator: "=", Value: "pending"},
//	  Condition{Field: "status", Operator: "=", Value: "review"}
//	)
//
// Returns: A new Condition with nested conditions joined by OR
func (c *Condition) Or(conditions ...Condition) *Condition {
	return &Condition{
		Logic:  "OR",
		Nested: conditions,
	}
}

// ToWhereString converts a Condition struct into a WHERE clause string and parameter values.
// It handles nested conditions recursively and supports both AND/OR logic.
// Security: Validates field names and operators to prevent SQL injection attacks.
// Usage:
//
//	whereClause, values, err := condition.ToWhereString()
//	if err != nil {
//	    return "", nil, err
//	}
//
// Returns:
//   - string: SQL WHERE clause with parameterized queries (e.g., "field1 = ? AND (field2 > ?)")
//   - []interface{}: Slice of values corresponding to the parameters
//   - error: Validation error if field name or operator is invalid
func (c *Condition) ToWhereString() (string, []interface{}, error) {
	var clauses []string
	var args []interface{}

	if c.Field != "" { // Base case for simple condition
		// Security: Validate field name to prevent SQL injection
		if err := ValidateFieldName(c.Field); err != nil {
			return "", nil, err
		}

		// Security: Validate operator to prevent SQL injection
		if err := ValidateOperator(c.Operator); err != nil {
			return "", nil, err
		}

		// Ensure both field and operator are present
		if c.Operator == "" {
			return "", nil, ErrInvalidOperator
		}

		clauses = append(clauses, fmt.Sprintf("%s %s ?", c.Field, strings.ToUpper(c.Operator)))
		args = append(args, c.Value)
	} else { // Handle nested conditions
		for _, nested := range c.Nested {
			subClause, subArgs, err := nested.ToWhereString()
			if err != nil {
				return "", nil, err
			}
			if subClause != "" { // Only append non-empty clauses
				clauses = append(clauses, fmt.Sprintf("(%s)", subClause))
				args = append(args, subArgs...)
			}
		}
	}

	logic := strings.ToUpper(strings.TrimSpace(c.Logic))
	if logic == "" {
		logic = "AND" // Default to AND if not specified
	}

	return strings.Join(clauses, fmt.Sprintf(" %s ", logic)), args, nil
}

// ToSelectString generates a complete SELECT SQL query string with WHERE, GROUP BY, ORDER BY,
// and LIMIT/OFFSET clauses based on the Condition struct.
// Security: Validates table name and delegates to ToWhereString for field/operator validation.
// Usage:
//
//	query, values, err := condition.ToSelectString("users")
//	if err != nil {
//	    return "", nil, err
//	}
//
// Returns:
//   - string: Complete SELECT query (e.g., "SELECT * FROM users WHERE age > ? ORDER BY name LIMIT 10")
//   - []interface{}: Slice of values for the parameterized query
//   - error: Validation error if table name, field name, or operator is invalid
func (c *Condition) ToSelectString(tableName string) (string, []interface{}, error) {
	// Security: Validate table name to prevent SQL injection
	if err := ValidateTableName(tableName); err != nil {
		return "", nil, err
	}

	whereClause, values, err := c.ToWhereString()
	if err != nil {
		return "", nil, err
	}

	orderClause := ""
	if len(c.OrderBy) > 0 {
		orderClause = "ORDER BY " + strings.Join(c.OrderBy, ", ")
	}

	groupClause := ""
	if len(c.GroupBy) > 0 {
		groupClause = "GROUP BY " + strings.Join(c.GroupBy, ", ")
	}

	// if offset has value but limit is not, then use default limit
	if c.Offset > 0 && c.Limit < 1 {
		c.Limit = DEFAULT_PAGINATION_LIMIT
	}
	limitClause := ""
	if c.Limit > 0 {
		limitClause = fmt.Sprintf("LIMIT %d", c.Limit)

		if c.Offset > 0 {
			limitClause += fmt.Sprintf(" OFFSET %d", c.Offset)
		}
	}
	// If there is no WHERE statement, just do the order and groupby
	if strings.TrimSpace(whereClause) == "" {
		return fmt.Sprintf("SELECT * FROM %s %s %s %s", tableName, groupClause, orderClause, limitClause), values, nil
	}
	return fmt.Sprintf("SELECT * FROM %s WHERE %s %s %s %s", tableName, whereClause, groupClause, orderClause, limitClause), values, nil
}

// PrintDebug prints debug information about a database schema object.
// If sql parameter is true, it includes the SQL command in the output.
// Usage:
//
//	schema.PrintDebug(true)
//
// Output Example:
//
//	Object [table] : users[users_table] - CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)
func (s SchemaStruct) PrintDebug(sql bool) {
	rawSql := " - " + s.SQLCommand
	if !sql {
		rawSql = ""
	}
	fmt.Printf("Object [%s] : %s[%s] %s\n", s.ObjectType, s.TableName, s.ObjectName, rawSql)
}

// ValidateTableName validates a table name to prevent SQL injection.
// It checks that the name is not empty and contains only alphanumeric characters and underscores.
// The name must start with a letter or underscore.
//
// Usage:
//
//	if err := ValidateTableName("users"); err != nil {
//	    return err
//	}
//
// Returns: error if validation fails, nil otherwise
func ValidateTableName(tableName string) error {
	if tableName == "" {
		return ErrEmptyTableName
	}
	if !sqlIdentifierRegex.MatchString(tableName) {
		return ErrInvalidTableName
	}
	return nil
}

// ValidateFieldName validates a field/column name to prevent SQL injection.
// It checks that the name contains only alphanumeric characters and underscores.
// The name must start with a letter or underscore.
//
// Usage:
//
//	if err := ValidateFieldName("user_id"); err != nil {
//	    return err
//	}
//
// Returns: error if validation fails, nil otherwise
func ValidateFieldName(fieldName string) error {
	if fieldName == "" {
		return nil // Empty field names are allowed in nested conditions
	}
	if !sqlIdentifierRegex.MatchString(fieldName) {
		return ErrInvalidFieldName
	}
	return nil
}

// ValidateOperator validates a SQL operator against a whitelist to prevent SQL injection.
// It checks if the operator is in the allowed list of safe SQL operators.
//
// Usage:
//
//	if err := ValidateOperator("="); err != nil {
//	    return err
//	}
//
// Returns: error if validation fails, nil otherwise
func ValidateOperator(operator string) error {
	if operator == "" {
		return nil // Empty operators are allowed in nested conditions
	}
	upperOp := strings.ToUpper(strings.TrimSpace(operator))
	if !allowedOperators[upperOp] {
		return ErrInvalidOperator
	}
	return nil
}

// JoinType represents the type of SQL JOIN operation
type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
	FullJoin  JoinType = "FULL OUTER JOIN"
	CrossJoin JoinType = "CROSS JOIN"
)

// Join represents a SQL JOIN clause
type Join struct {
	Type      JoinType `json:"type"`                 // Type of join (INNER, LEFT, RIGHT, FULL, CROSS)
	Table     string   `json:"table"`                // Table to join
	Alias     string   `json:"alias,omitempty"`      // Optional table alias
	Condition string   `json:"condition,omitempty"`  // Join condition (e.g., "users.id = orders.user_id")
}

// CommonTableExpression represents a CTE (WITH clause) in SQL.
// CTEs allow you to define temporary named result sets that can be referenced
// within a SELECT, INSERT, UPDATE, or DELETE statement.
//
// Example:
//
//	cte := CommonTableExpression{
//	    Name: "active_users",
//	    Query: &ComplexQuery{
//	        Select: []string{"id", "name", "email"},
//	        From:   "users",
//	        Where:  &Condition{Field: "status", Operator: "=", Value: "active"},
//	    },
//	}
type CommonTableExpression struct {
	Name      string        `json:"name"`                 // CTE name (required)
	Columns   []string      `json:"columns,omitempty"`    // Optional column list
	Query     *ComplexQuery `json:"query,omitempty"`      // Structured query for CTE
	RawSQL    string        `json:"raw_sql,omitempty"`    // Raw SQL for complex CTEs
	Recursive bool          `json:"recursive,omitempty"`  // Whether this is a recursive CTE
}

// ToSQL converts a CTE to its SQL representation
func (cte *CommonTableExpression) ToSQL() (string, []interface{}, error) {
	// Validate CTE name
	if err := ValidateTableName(cte.Name); err != nil {
		return "", nil, fmt.Errorf("invalid CTE name: %w", err)
	}

	var values []interface{}
	var querySQL string
	var err error

	// Use structured query if provided, otherwise use raw SQL
	if cte.Query != nil {
		querySQL, values, err = cte.Query.ToSQL()
		if err != nil {
			return "", nil, fmt.Errorf("failed to build CTE query: %w", err)
		}
	} else if cte.RawSQL != "" {
		querySQL = cte.RawSQL
	} else {
		return "", nil, fmt.Errorf("CTE must have either Query or RawSQL defined")
	}

	// Build CTE definition
	cteSQL := cte.Name

	// Add column list if specified
	if len(cte.Columns) > 0 {
		cteSQL += " (" + strings.Join(cte.Columns, ", ") + ")"
	}

	cteSQL += " AS (" + querySQL + ")"

	return cteSQL, values, nil
}

// ComplexQuery represents a complex SQL query structure that supports:
// - Custom SELECT fields (not just SELECT *)
// - Multiple table JOINs
// - Complex WHERE conditions (using Condition struct)
// - GROUP BY with HAVING clauses
// - ORDER BY, LIMIT, OFFSET
// - DISTINCT, CTEs (structured and raw), and subqueries
//
// Example usage:
//
//	query := ComplexQuery{
//	    Select:    []string{"users.id", "users.name", "COUNT(orders.id) as order_count"},
//	    From:      "users",
//	    Joins: []Join{
//	        {Type: LeftJoin, Table: "orders", Condition: "users.id = orders.user_id"},
//	    },
//	    Where: &Condition{
//	        Field:    "users.status",
//	        Operator: "=",
//	        Value:    "active",
//	    },
//	    GroupBy:   []string{"users.id", "users.name"},
//	    Having:    "COUNT(orders.id) > 5",
//	    OrderBy:   []string{"order_count DESC"},
//	    Limit:     10,
//	}
type ComplexQuery struct {
	Select    []string                 `json:"select,omitempty"`     // Fields to select (default: ["*"])
	Distinct  bool                     `json:"distinct,omitempty"`   // Add DISTINCT keyword
	From      string                   `json:"from"`                 // Main table name (required)
	FromAlias string                   `json:"from_alias,omitempty"` // Alias for main table
	Joins     []Join                   `json:"joins,omitempty"`      // JOIN clauses
	Where     *Condition               `json:"where,omitempty"`      // WHERE conditions
	GroupBy   []string                 `json:"group_by,omitempty"`   // GROUP BY fields
	Having    string                   `json:"having,omitempty"`     // HAVING clause (raw SQL)
	OrderBy   []string                 `json:"order_by,omitempty"`   // ORDER BY fields
	Limit     int                      `json:"limit,omitempty"`      // LIMIT value
	Offset    int                      `json:"offset,omitempty"`     // OFFSET value
	CTEs      []CommonTableExpression  `json:"ctes,omitempty"`       // Structured CTEs (recommended)
	CTERaw    string                   `json:"cte_raw,omitempty"`    // Raw CTE string (for backward compatibility)
}

// ToSQL converts a ComplexQuery to a SQL query string with parameterized values.
// Security: Validates table names, field names, and uses parameterized queries.
//
// Returns:
//   - string: Complete SQL query with placeholders
//   - []interface{}: Values for the parameterized query
//   - error: Validation error if any field is invalid
func (cq *ComplexQuery) ToSQL() (string, []interface{}, error) {
	var queryParts []string
	var values []interface{}

	// Security: Validate main table name
	if err := ValidateTableName(cq.From); err != nil {
		return "", nil, err
	}

	// CTE (WITH clause) - added first if present
	// Support both structured CTEs and raw CTE string
	if len(cq.CTEs) > 0 {
		cteStrings := make([]string, 0, len(cq.CTEs))
		recursive := false

		for _, cte := range cq.CTEs {
			cteSQL, cteValues, err := cte.ToSQL()
			if err != nil {
				return "", nil, fmt.Errorf("failed to build CTE: %w", err)
			}
			cteStrings = append(cteStrings, cteSQL)
			values = append(values, cteValues...)

			if cte.Recursive {
				recursive = true
			}
		}

		withClause := "WITH "
		if recursive {
			withClause = "WITH RECURSIVE "
		}
		withClause += strings.Join(cteStrings, ", ")
		queryParts = append(queryParts, withClause)
	} else if cq.CTERaw != "" {
		// Fallback to raw CTE string for backward compatibility
		queryParts = append(queryParts, cq.CTERaw)
	}

	// SELECT clause
	selectClause := "SELECT"
	if cq.Distinct {
		selectClause += " DISTINCT"
	}
	if len(cq.Select) == 0 {
		selectClause += " *"
	} else {
		selectClause += " " + strings.Join(cq.Select, ", ")
	}
	queryParts = append(queryParts, selectClause)

	// FROM clause
	fromClause := "FROM " + cq.From
	if cq.FromAlias != "" {
		fromClause += " AS " + cq.FromAlias
	}
	queryParts = append(queryParts, fromClause)

	// JOIN clauses
	for _, join := range cq.Joins {
		// Security: Validate join table name
		if err := ValidateTableName(join.Table); err != nil {
			return "", nil, fmt.Errorf("invalid join table: %w", err)
		}

		joinClause := string(join.Type) + " " + join.Table
		if join.Alias != "" {
			joinClause += " AS " + join.Alias
		}
		if join.Condition != "" && join.Type != CrossJoin {
			joinClause += " ON " + join.Condition
		}
		queryParts = append(queryParts, joinClause)
	}

	// WHERE clause
	if cq.Where != nil {
		whereClause, whereValues, err := cq.Where.ToWhereString()
		if err != nil {
			return "", nil, fmt.Errorf("failed to build WHERE clause: %w", err)
		}
		if whereClause != "" {
			queryParts = append(queryParts, "WHERE "+whereClause)
			values = append(values, whereValues...)
		}
	}

	// GROUP BY clause
	if len(cq.GroupBy) > 0 {
		queryParts = append(queryParts, "GROUP BY "+strings.Join(cq.GroupBy, ", "))
	}

	// HAVING clause
	if cq.Having != "" {
		queryParts = append(queryParts, "HAVING "+cq.Having)
	}

	// ORDER BY clause
	if len(cq.OrderBy) > 0 {
		queryParts = append(queryParts, "ORDER BY "+strings.Join(cq.OrderBy, ", "))
	}

	// LIMIT and OFFSET
	if cq.Offset > 0 && cq.Limit < 1 {
		cq.Limit = DEFAULT_PAGINATION_LIMIT
	}
	if cq.Limit > 0 {
		queryParts = append(queryParts, fmt.Sprintf("LIMIT %d", cq.Limit))
	}
	if cq.Offset > 0 {
		queryParts = append(queryParts, fmt.Sprintf("OFFSET %d", cq.Offset))
	}

	return strings.Join(queryParts, " "), values, nil
}
