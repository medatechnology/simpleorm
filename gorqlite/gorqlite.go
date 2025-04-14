package gorqlite

import (
	"fmt"
	"strings"

	"github.com/medatechnology/goutil/object"
	"github.com/medatechnology/goutil/simplelog"
	orm "github.com/medatechnology/simpleorm"
	"github.com/rqlite/gorqlite"
)

const (
	PREFIX_SURESQL_TABLE = "_"
	PREFIX_SQLITE_TABLE  = "sqlite_"
)

type RqliteConfig struct {
	URL         string
	Consistency string
	Database    string // actually need to be combined into URL
	Username    string // actually need to be combined into URL
	Password    string // actually need to be combined into URL
	Port        string // actually need to be combined into URL
}

type RQLiteDB struct {
	Config RqliteConfig
	conn   *gorqlite.Connection
}

// This is to "connect" to the DB. Basically we use gorqlite to do so.
// If using credential, then it will use that as well and return error if it's not matched
func NewDatabase(config RqliteConfig) (RQLiteDB, error) {
	conn, err := gorqlite.Open(config.URL)
	if err != nil {
		return RQLiteDB{}, err
	}
	// Set consistency Level
	if config.Consistency != "" {
		consLevel, err := gorqlite.ParseConsistencyLevel(config.Consistency)
		if err == nil {
			// conn.SetConsistencyLevel(gorqlite.ConsistencyLevelWeak)
			// conn.SetConsistencyLevel(gorqlite.ConsistencyLevelStrong)
			conn.SetConsistencyLevel(consLevel)
		} // ignores if we cannot parse the consistency level
	}
	return RQLiteDB{Config: config, conn: conn}, nil
}

func (db RQLiteDB) IsConnected() bool {
	return db.conn != nil
}

// Get the leader
func (db RQLiteDB) Leader() (string, error) {
	return db.conn.Leader()
}

func (db RQLiteDB) Peers() ([]string, error) {
	return db.conn.Peers()
}

// Get all the schema for the database, basically returns all the table that exists!
func (db RQLiteDB) GetSchema(hideSQL, hideSureSQL bool) []orm.SchemaStruct {
	// getTableAndView := "SELECT * FROM sqlite_master ORDER BY type,tbl_name, name"
	var schemas []orm.SchemaStruct
	c := orm.Condition{
		OrderBy: []string{"type", "tbl_name", "name"},
	}
	res, err := db.SelectManyWithCondition("sqlite_master", &c)
	if err == nil {
		// if no error, convert to SchemaStruct
		for _, t := range res {
			tableName := t.Data["tbl_name"].(string)
			if !(strings.HasPrefix(tableName, PREFIX_SQLITE_TABLE) && hideSQL) &&
				!(strings.HasPrefix(tableName, PREFIX_SURESQL_TABLE) && hideSureSQL) {
				schemas = append(schemas, object.MapToStruct[orm.SchemaStruct](t.Data))
			}
		}
	}
	return schemas
}

// Returns the status of the database, for Rqlite this will also return the peers and leaders
// Can use this as ping-pong as well
func (db RQLiteDB) Status() ([]string, error) {
	var nodes []string
	leader, err := db.conn.Leader()
	if err != nil {
		simplelog.LogErr(err, "error getting leader")
		return nodes, fmt.Errorf("error getting leader")
	}
	nodes = append(nodes, leader)
	peers, err := db.conn.Peers()
	if err != nil {
		simplelog.LogErr(err, "error getting peers")
		return nodes, fmt.Errorf("error getting leader")
	}
	nodes = append(nodes, peers...)
	// statusStr := fmt.Sprintf("RQLite connected and alive\nleader:%s\nPeers: %v", leader, peers)
	return nodes, nil
}

// NOTE: This is almost not used because very rare cases where we need to select from table without
// -     conditions but only return 1
// Select only 1 row, if multiple rows returned, it only takes the first one
func (db RQLiteDB) SelectOne(tableName string) (orm.DBRecord, error) {
	// el := metrics.StartTimeIt("", 0)
	qr, err := db.conn.QueryOne("SELECT * FROM " + tableName)
	if err != nil {
		return orm.DBRecord{}, err
	}
	// fmt.Printf(" (Q:%s) ", metrics.StopTimeIt(el))
	// Result for O = 1.15s -- SLOW! maybe because this went to internet, not local connection

	// el = metrics.StartTimeIt("", 0)
	// If no row found, count as error! SQL_NO_ROWS usually
	if qr.NumRows() == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}
	qr.Next() // This is go-RQLite quirks, need to do this before scan!
	// fmt.Printf(" (N:%s) ", metrics.StopTimeIt(el))
	// Result for N = 875 nano seconds - FAST

	// put the result in map[string]interface which is key-value where key is the field name on the table
	// result := make(map[string]interface{})
	// err = qr.Scan(&result)
	// fmt.Printf(" (%.5f) ", qr.Timing) // printout the time needed

	// el = metrics.StartTimeIt("", 0)
	result, err := qr.Map()
	if err != nil {
		return orm.DBRecord{}, err
	}
	// fmt.Printf(" (M:%s) ", metrics.StopTimeIt(el))
	// Result of M = 1.834 micro seconds - FAST

	return orm.DBRecord{
		TableName: tableName,
		Data:      result,
	}, nil
}

// Select 1 row from a table passed in tableName with condition (read condition struct for usage)
// Will return only 1 row of type DBRecord and error if any.
func (db RQLiteDB) SelectOneWithCondition(tableName string, condition *orm.Condition) (orm.DBRecord, error) {
	if condition != nil {
		// statement := condition.ToParameterized(tableName)
		statement := ConditionToParameterized(tableName, condition)

		qr, err := db.conn.QueryOneParameterized(statement)
		if err != nil {
			return orm.DBRecord{}, err
		}

		// If no row found, count as error! SQL_NO_ROWS usually
		if qr.NumRows() == 0 {
			return orm.DBRecord{}, orm.ErrSQLNoRows
		}
		// put the result in map[string]interface which is key-value where key is the field name on the table
		qr.Next() // This is go-RQLite quirks, need to do this before scan!
		result, err := qr.Map()
		if err != nil {
			return orm.DBRecord{}, err
		}

		return orm.DBRecord{
			TableName: tableName,
			Data:      result,
		}, nil
	}
	return db.SelectOne(tableName)
}

func (db RQLiteDB) SelectMany(tableName string) (orm.DBRecords, error) {
	var records orm.DBRecords
	qr, err := db.conn.QueryOne("SELECT * FROM " + tableName)
	if err != nil {
		return nil, err
	}
	// If no row found, count as error! SQL_NO_ROWS usually
	if qr.NumRows() == 0 {
		return nil, orm.ErrSQLNoRows
	}
	// NOTE: queryResult has .NumRows and .RowNumber as well as .Next() and .Scan()
	// fmt.Printf("query returned %d rows\n", qr.NumRows)
	for qr.Next() {
		// result := make(map[string]interface{})
		result, err := qr.Map()
		if err != nil {
			return nil, err
		}
		// fmt.Printf("this is row number %d\n", qr.RowNumber)
		records = append(records, orm.DBRecord{
			TableName: tableName,
			Data:      result,
		})
	}
	return records, nil
}

// Select many rows (returned as []DBRecords) with condition.
func (db RQLiteDB) SelectManyWithCondition(tableName string, condition *orm.Condition) ([]orm.DBRecord, error) {
	var records []orm.DBRecord
	// statement := condition.ToParameterized(tableName)
	statement := ConditionToParameterized(tableName, condition)
	// simplelog.LogThis(statement.Query + ";" + fmt.Sprintln(statement.Arguments))
	qr, err := db.conn.QueryOneParameterized(statement)
	if err != nil {
		return nil, err
	}

	// If no row found, count as error! SQL_NO_ROWS usually
	if qr.NumRows() == 0 {
		return nil, orm.ErrSQLNoRows
	}

	// NOTE: queryResult has .NumRows and .RowNumber as well as .Next() and .Scan()
	// fmt.Printf("query returned %d rows\n", qr.NumRows)
	for qr.Next() {
		// result := make(map[string]interface{})
		result, err := qr.Map()
		if err != nil {
			return nil, err
		}
		// fmt.Printf("this is row number %d\n", qr.RowNumber)
		records = append(records, orm.DBRecord{
			TableName: tableName,
			Data:      result,
		})
	}
	return records, nil
}

// Execute 1 raw sql statement, can be anything. Query, Update, Insert, etc
// Combine the error from write function to result.Err
func (db RQLiteDB) ExecOneSQL(sql string) orm.BasicSQLResult {
	res, err := db.conn.WriteOne(sql)
	ret := WriteResultToBasicSQLResult(res)
	if err != nil {
		ret.Error = fmt.Errorf("failed to execute sql: %w", err)
	}
	return ret
}

// Execute 1 raw sql statement, can be anything. Query, Update, Insert, etc
func (db RQLiteDB) ExecOneSQLParameterized(p orm.ParametereizedSQL) orm.BasicSQLResult {
	res, err := db.conn.WriteOneParameterized(FromOneParameterizedSQL(p))
	ret := WriteResultToBasicSQLResult(res)
	if err != nil {
		ret.Error = fmt.Errorf("failed to execute parameterized sql: %w", err)
	}
	return ret
}

// Execute many raw sql statement, can be anything. Query, Update, Insert, etc
func (db RQLiteDB) ExecManySQL(sql []string) ([]orm.BasicSQLResult, error) {
	res, err := db.conn.Write(sql)
	if err != nil {
		return nil, err
	}
	return WriteResultsToBasicSQLResults(res), nil
}

// Execute many raw sql statement, can be anything. Query, Update, Insert, etc
func (db RQLiteDB) ExecManySQLParameterized(p []orm.ParametereizedSQL) ([]orm.BasicSQLResult, error) {
	res, err := db.conn.WriteParameterized(FromManyParameterizedSQL(p))
	if err != nil {
		return nil, fmt.Errorf("failed to execute parameterized sql: %w", err)
	}
	return WriteResultsToBasicSQLResults(res), nil
}

// Insert DBRecord into the DB. The table name is stored in DBRecord
// if queue == true, this means we are using the queueFunction and there is no error (most of the time)
// because queueFunction is returning the API status from rqlite, if it succeed it's not
// the sql command but the API is succeed.
// if false then we can get the error right away from the command
func (db RQLiteDB) InsertOneDBRecord(record orm.DBRecord, queue bool) orm.BasicSQLResult {
	statement := DBRecordToInsertParameterized(&record)

	var res orm.BasicSQLResult
	var err error
	// if this is queued then return rowAffected=1 and LastInsertID as the sequence number
	if queue {
		var seq int64
		seq, err = db.conn.QueueOneParameterized(statement)
		res.LastInsertID = int(seq)
		res.RowsAffected = 1
	} else {
		var r gorqlite.WriteResult // need to declare this so err can use parent's scope
		r, err = db.conn.WriteOneParameterized(statement)
		// ret := WriteResultToBasicSQLResult(res)
		// record.Data["id"] = r.LastInsertID
		res = WriteResultToBasicSQLResult(r)
	}
	if err != nil {
		res.Error = fmt.Errorf("failed to insert record for table %s: %w", record.TableName, err)
	}
	return res
}

// Insert many records, that can be of different tables, that means we have
// to execute per statement
// When queue=true the query result is not used, it will return only 1 result
// with LastInsertID = sequence number and RowsAffected as len(records)
func (db RQLiteDB) InsertManyDBRecords(records []orm.DBRecord, queue bool) ([]orm.BasicSQLResult, error) {
	var statements []gorqlite.ParameterizedStatement
	// This is assuming that function is NEVER called with 0 array or nul
	tableName := records[0].TableName
	for _, record := range records {
		statement := DBRecordToInsertParameterized(&record)
		statements = append(statements, statement)
	}
	var err error
	var reses []orm.BasicSQLResult
	// var ress []gorqlite.WriteResult
	if queue {
		var seq int64
		seq, err = db.conn.QueueParameterized(statements)
		reses = append(reses, orm.BasicSQLResult{LastInsertID: int(seq), RowsAffected: len(records)})
	} else {
		var res []gorqlite.WriteResult // need to declare this so err can use parent's scope
		// res, err = db.conn.WriteParameterized(statements)
		res, err = db.conn.WriteParameterized(statements)
		// NOTE: cannot put the last inserted ID back to records parameter because
		//       golang slice/array is not in order!!
		// for i, r := range res {
		// 	records[i].Data["id"] = r.LastInsertID
		// }
		reses = WriteResultsToBasicSQLResults(res)
	}
	if err != nil {
		return reses, fmt.Errorf("failed to queue record for table %s: %w", tableName, err)
	}
	return reses, nil
}

// Insert array of records which we know for sure it's the same table. This should be
// faster than using InsertManyDBRecords if bulk inserting 1 table with many records
// When queue=true the query result is not used, instead the first array of
// Result{LastInsertID : is the sequence number }
func (db RQLiteDB) InsertManyDBRecordsSameTable(records []orm.DBRecord, queue bool) ([]orm.BasicSQLResult, error) {
	var reses []orm.BasicSQLResult
	numRecs := len(records)
	if numRecs == 0 {
		return reses, fmt.Errorf("records empty, nothing to insert")
	}
	tableName := records[0].TableName
	statements := FromManyParameterizedSQL(orm.ToInsertSQLParameterizedFromSlice(records))
	var err error
	if queue {
		var seq int64
		seq, err = db.conn.QueueParameterized(statements)
		// Use the first array of Result, using LastInsertID (int)
		reses = append(reses, orm.BasicSQLResult{LastInsertID: int(seq), RowsAffected: numRecs})
	} else {
		var res []gorqlite.WriteResult
		res, err = db.conn.WriteParameterized(statements)
		reses = WriteResultsToBasicSQLResults(res)
	}
	if err != nil {
		return reses, fmt.Errorf("failed to queue record for table %s: %w", tableName, err)
	}
	return reses, nil
}

// NOTE: with inserting struct, be careful on the ID field. Usually we have
//
//	ID in a table with autoincrement, this struct when converted to map
//	if the struct.ID is not set, it will be converted to the map[ID]=0
//	So the insert will fail! Use this accordingly!! Saver to use
//	InsertOneDBRecord() function because we can NOT set the map[ID] to any
//	value, then the DB will use the default or autoincrement value!
//
// Insert TableStruct to DB
func (db RQLiteDB) InsertOneTableStruct(obj orm.TableStruct, queue bool) orm.BasicSQLResult {
	// fmt.Println("struct : ", obj)
	record, err := orm.TableStructToDBRecord(obj)
	// fmt.Println("record : ", record)
	if err != nil {
		return orm.BasicSQLResult{Error: err}
	}
	return db.InsertOneDBRecord(record, queue)
	// statement := DBRecordToInsertParameterized(&record)

	// _, err = db.conn.QueueOneParameterized(statement)
	// if err != nil {
	// 	return fmt.Errorf("failed to queue record for table %s: %w", record.TableName, err)
	// }
	// return nil
}

// Insert many table structs (in array) but use the DB records.
func (db RQLiteDB) InsertManyTableStructs(objs []orm.TableStruct, queue bool) ([]orm.BasicSQLResult, error) {
	var records orm.DBRecords
	for _, obj := range objs {
		record, err := orm.TableStructToDBRecord(obj)
		if err != nil {
			return []orm.BasicSQLResult{}, err
		}
		records = append(records, record)
	}
	return db.InsertManyDBRecords(records, queue)
	// var statements []gorqlite.ParameterizedStatement
	// // This is assuming that function is NEVER called with 0 array or nul
	// tableName := objs[0].TableName()
	// for _, obj := range objs {
	// 	record, err := orm.TableStructToDBRecord(obj)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	statement := DBRecordToInsertParameterized(&record)
	// 	statements = append(statements, statement)
	// }
	// _, err := db.conn.QueueParameterized(statements)
	// if err != nil {
	// 	return fmt.Errorf("failed to queue record for table %s: %w", tableName, err)
	// }
	// return nil
}
