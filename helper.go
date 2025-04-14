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

// Convert the .sql file into each individual sql commands
// Input is []string which are the content of the .sql file
// Output is []string of each sql commands.
func ConvertSQLCommands(lines []string) []string {
	var commands []string
	var currentCommand strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		commentIndex := strings.Index(line, "--")
		if commentIndex != -1 {
			line = line[:commentIndex] // Remove comment part
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		currentCommand.WriteString(line)
		currentCommand.WriteString(" ")

		if strings.Contains(line, ";") {
			parts := strings.Split(currentCommand.String(), ";")
			for _, part := range parts[:len(parts)-1] { // Process parts before the last one
				command := strings.TrimSpace(part)
				if command != "" {
					commands = append(commands, command)
				}
			}
			currentCommand.Reset()
			lastPart := parts[len(parts)-1]
			currentCommand.WriteString(lastPart)
		}
	}

	if currentCommand.Len() > 0 {
		command := strings.TrimSpace(currentCommand.String())
		if command != "" {
			commands = append(commands, command)
		}
	}

	return commands
}

// ===== THis is for debugging purposes
func PrintDebug(msg string) {
	fmt.Println(msg)
}
