package orm

import (
	"fmt"
	"strings"

	"github.com/medatechnology/goutil/object"
)

func SQLAndValuesToParameterized(q string, p []interface{}) ParametereizedSQL {
	return ParametereizedSQL{
		Query:  q,
		Values: p,
	}
}

// For a slice of DBRecord:
// records := []DBRecord{...}
// paramSQLs := ToInsertSQLParameterizedFromSlice(records)
// results, err := db.ExecManySQLParameterized(paramSQLs)

// Or for a DBRecords directly:
// records := DBRecords{...}
// paramSQLs := records.ToInsertSQLParameterized()
// results, err := db.ExecManySQLParameterized(paramSQLs)

// DBRecordsFromSlice converts a slice of DBRecord to DBRecords type
// and uses the DBRecords methods
func DBRecordsFromSlice(records []DBRecord) DBRecords {
	return DBRecords(records)
}

// ToInsertSQLParameterizedFromSlice converts a slice of DBRecord to a slice of ParametereizedSQL
// by converting to DBRecords first
func ToInsertSQLParameterizedFromSlice(records []DBRecord) []ParametereizedSQL {
	return DBRecordsFromSlice(records).ToInsertSQLParameterized()
}

// ToInsertSQLRawFromSlice converts a slice of DBRecord to a slice of raw SQL statements
// by converting to DBRecords first
func ToInsertSQLRawFromSlice(records []DBRecord) []string {
	return DBRecordsFromSlice(records).ToInsertSQLRaw()
}

func TableStructToDBRecord(obj TableStruct) (DBRecord, error) {
	// Type assertion to get the underlying value
	// valueObj, ok := obj.(TableStruct)
	// if !ok {
	// 	// Handle error: couldn't convert to TableStruct
	// 	return DBRecord{}, fmt.Errorf("tablestruct to DB record, cannot assert object %v", obj)
	// }
	// fmt.Println("valueObj ==> ", valueObj)
	data := object.StructToMap(obj) // Assume this is implemented and tested
	// fmt.Println("Struct : ", obj)
	// fmt.Println("Map : ", data)
	return DBRecord{
		TableName: obj.TableName(),
		Data:      data,
	}, nil
}

// Decided to just use MapToStruct manually
// func DBRecordToStruct(rec DBRecord) TableStruct {
// 	return object.MapToStruct[TableStruct](rec.Data)
// }

// Get all sum timing
func TotalTimeElapsedInSecond(reses []BasicSQLResult) float64 {
	sum := 0.0
	for i := range reses {
		sum += reses[i].Timing // Direct indexing is slightly faster
	}
	return sum
}

func SecondToMs(s float64) float64 {
	return s * 1000
}

func SecondToMsString(s float64) string {
	return fmt.Sprintf("%.5f", SecondToMs(s))
}

func InterfaceToSQLString(interfaceVal interface{}) string {
	sqlStr := ""
	switch v := interfaceVal.(type) {
	case int, int16, int32, int64, uint, uint16, uint32, uint64:
		// Without single quote
		sqlStr = fmt.Sprintf("%d", v)
	case float64, float32:
		// Without single quote
		sqlStr = fmt.Sprintf("%f", v)
	case string:
		// This is the only important key, we add single quote '%s'
		sqlStr = fmt.Sprintf("'%s'", v)
	default:
		// Default is without single quote
		sqlStr = fmt.Sprintf("%v", v)
	}
	return sqlStr
}

// Convert the .sql file into each individual sql commands.
//
// Input is []string which are the lines of the .sql file (or of any SQL block),
// output is []string of individual SQL statements, split on ';' boundaries.
//
// The splitter is aware of:
//   - single/double/backtick quoted strings, including doubled-quote escapes
//     ('it”s', "a""b") — a ';' inside a string never splits a statement
//   - '--' line comments and '/* ... */' block comments (which may span lines) —
//     a ';' inside a comment never splits a statement
//
// Comments are stripped from the returned statements and empty statements are
// skipped. A statement left unterminated at the end of the input is returned
// as-is (so multi-line statements are supported).
func ConvertSQLCommands(lines []string) []string {
	var commands []string
	var cur strings.Builder
	var quote byte // 0 = outside a string; otherwise ' " or `
	inBlockComment := false

	for _, line := range lines {
		i := 0
		for i < len(line) {
			ch := line[i]
			next := byte(0)
			if i+1 < len(line) {
				next = line[i+1]
			}

			switch {
			case inBlockComment:
				if ch == '*' && next == '/' {
					inBlockComment = false
					i++ // consume the '/'
				}
			case quote != 0:
				cur.WriteByte(ch)
				if ch == quote {
					if next == quote { // doubled-quote escape, e.g. 'it''s'
						cur.WriteByte(next)
						i++
					} else {
						quote = 0
					}
				}
			default:
				switch {
				case ch == '\'' || ch == '"' || ch == '`':
					quote = ch
					cur.WriteByte(ch)
				case ch == '-' && next == '-':
					// line comment: skip the rest of this line
					i = len(line)
					continue
				case ch == '/' && next == '*':
					inBlockComment = true
					i++ // skip the '*'
				case ch == ';':
					if s := strings.TrimSpace(cur.String()); s != "" {
						commands = append(commands, s)
					}
					cur.Reset()
				default:
					cur.WriteByte(ch)
				}
			}
			i++
		}
	}

	if s := strings.TrimSpace(cur.String()); s != "" {
		commands = append(commands, s)
	}
	return commands
}

// ===== THis is for debugging purposes
func PrintDebug(msg string) {
	fmt.Println(msg)
}
