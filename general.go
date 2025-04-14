package orm

import (
	"fmt"
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
// Usage:
//
//	whereClause, values := condition.ToWhereString()
//
// Returns:
//   - string: SQL WHERE clause with parameterized queries (e.g., "field1 = ? AND (field2 > ?)")
//   - []interface{}: Slice of values corresponding to the parameters
func (c *Condition) ToWhereString() (string, []interface{}) {
	var clauses []string
	var args []interface{}

	if c.Field != "" { // Base case for simple condition
		clauses = append(clauses, fmt.Sprintf("%s %s ?", c.Field, c.Operator))
		args = append(args, c.Value)
	} else { // Handle nested conditions
		for _, nested := range c.Nested {
			subClause, subArgs := nested.ToWhereString()
			clauses = append(clauses, fmt.Sprintf("(%s)", subClause))
			args = append(args, subArgs...)
		}
	}

	return strings.Join(clauses, fmt.Sprintf(" %s ", strings.ToUpper(c.Logic))), args
}

// ToSelectString generates a complete SELECT SQL query string with WHERE, GROUP BY, ORDER BY,
// and LIMIT/OFFSET clauses based on the Condition struct.
// Usage:
//
//	query, values := condition.ToSelectString("users")
//
// Returns:
//   - string: Complete SELECT query (e.g., "SELECT * FROM users WHERE age > ? ORDER BY name LIMIT 10")
//   - []interface{}: Slice of values for the parameterized query
func (c *Condition) ToSelectString(tableName string) (string, []interface{}) {
	whereClause, values := c.ToWhereString()

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
			limitClause += fmt.Sprintf("OFFSET %d", c.Offset)
		}
	}
	// If there is no WHERE statement, just do the order and groupby
	if strings.TrimSpace(whereClause) == "" {
		return fmt.Sprintf("SELECT * FROM %s %s %s %s", tableName, groupClause, orderClause, limitClause), values
	}
	return fmt.Sprintf("SELECT * FROM %s WHERE %s %s %s %s", tableName, whereClause, groupClause, orderClause, limitClause), values
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
