package orm

type Database interface {
	GetSchema(bool, bool) []SchemaStruct
	Status() (NodeStatusStruct, error)

	SelectOne(string) (DBRecord, error)   // This is almost unusable, very rare case
	SelectMany(string) (DBRecords, error) // This is almost unusable, very rare case (this is like select ALL rows from the table)
	SelectOneWithCondition(string, *Condition) (DBRecord, error)
	SelectManyWithCondition(string, *Condition) ([]DBRecord, error)

	SelectOneSQL(string) (DBRecords, error)                              // select using one sql statement
	SelectManySQL([]string) ([]DBRecords, error)                         // select using many sql statements
	SelectOnlyOneSQL(string) (DBRecord, error)                           // select only returning 1 row, and also check if actually more than 1 return errors
	SelectOneSQLParameterized(ParametereizedSQL) (DBRecords, error)      // select using one parameterized sql statement
	SelectManySQLParameterized([]ParametereizedSQL) ([]DBRecords, error) // select using many parameterized sql statements
	SelectOnlyOneSQLParameterized(ParametereizedSQL) (DBRecord, error)   // select only returning 1 row, and also check if actually more than 1 return errors

	ExecOneSQL(string) BasicSQLResult
	ExecOneSQLParameterized(ParametereizedSQL) BasicSQLResult
	ExecManySQL([]string) ([]BasicSQLResult, error)
	ExecManySQLParameterized([]ParametereizedSQL) ([]BasicSQLResult, error)

	InsertOneDBRecord(DBRecord, bool) BasicSQLResult
	InsertManyDBRecords([]DBRecord, bool) ([]BasicSQLResult, error)
	InsertManyDBRecordsSameTable([]DBRecord, bool) ([]BasicSQLResult, error)

	// TableStruct is less practical
	InsertOneTableStruct(TableStruct, bool) BasicSQLResult
	InsertManyTableStructs([]TableStruct, bool) ([]BasicSQLResult, error)

	// Status and Health check
	IsConnected() bool
	Leader() (string, error)  // this was originally for RQLite, if not then just return empty string or "not implemented"
	Peers() ([]string, error) // this was originally for RQLite, if not then just return empty string or "not implemented"

	// The reason we don't do UPDATE or DELETE from DBRecord or Tablestruct is because
	// it's hard to tell which is the Where statement and which field is needed to be updated.
	// Like if we have a record/struct with .id=X, .name=somename, .age=10
	// and we pass it to UpdateOneDBRecord or UpdateOneTableStruct
	// we can't tell: update [table] set name=somename, age=10 where id=X
	//           or : update [table] set id=x where name=somename AND age=10
	// and there are still more other possibilities. Same with delete.
	// What we can actually do is DeleteOneWithCondition and DeleteManyWithCondition
	// but might as well do it with ExecRawSQL at this moment.
}
