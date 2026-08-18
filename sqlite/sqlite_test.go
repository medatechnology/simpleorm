package sqlite

import (
	"path/filepath"
	"testing"

	orm "github.com/medatechnology/simpleorm"
)

func newTestDB(t *testing.T) SqliteDirectDB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := NewDatabase(SqliteConfig{Path: path, WAL: true})
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	r := db.ExecOneSQL(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		age INTEGER,
		active INTEGER DEFAULT 1)`)
	if r.Error != nil {
		t.Fatalf("create table: %v", r.Error)
	}
	return db
}

func TestCRUDAndCondition(t *testing.T) {
	db := newTestDB(t)

	rec := orm.DBRecord{TableName: "users", Data: map[string]interface{}{
		"name": "Alice", "email": "alice@x.com", "age": 30, "active": 1,
	}}
	res := db.InsertOneDBRecord(rec, false)
	if res.Error != nil {
		t.Fatalf("insert: %v", res.Error)
	}
	if res.LastInsertID != 1 {
		t.Fatalf("LastInsertID = %d, want 1", res.LastInsertID)
	}

	rec2 := orm.DBRecord{TableName: "users", Data: map[string]interface{}{
		"name": "Bob", "email": "bob@x.com", "age": 25, "active": 1,
	}}
	if r := db.InsertOneDBRecord(rec2, false); r.Error != nil {
		t.Fatalf("insert2: %v", r.Error)
	}

	// SelectMany
	all, err := db.SelectMany("users")
	if err != nil {
		t.Fatalf("SelectMany: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("SelectMany len = %d, want 2", len(all))
	}

	// SelectOneWithCondition
	one, err := db.SelectOneWithCondition("users", &orm.Condition{
		Field: "email", Operator: "=", Value: "bob@x.com",
	})
	if err != nil {
		t.Fatalf("SelectOneWithCondition: %v", err)
	}
	if one.Data["name"] != "Bob" {
		t.Fatalf("got %v, want Bob", one.Data["name"])
	}

	// Parameterized
	recs, err := db.SelectOneSQLParameterized(orm.ParametereizedSQL{
		Query:  "SELECT * FROM users WHERE age >= ? ORDER BY age",
		Values: []interface{}{26},
	})
	if err != nil {
		t.Fatalf("parameterized: %v", err)
	}
	if len(recs) != 1 || recs[0].Data["name"] != "Alice" {
		t.Fatalf("parameterized result wrong: %+v", recs)
	}

	// Complex query with GROUP BY
	recs, err = db.SelectManyComplex(&orm.ComplexQuery{
		Select:  []string{"active", "COUNT(*) AS n"},
		From:    "users",
		GroupBy: []string{"active"},
	})
	if err != nil {
		t.Fatalf("complex: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("group by len = %d, want 1", len(recs))
	}
	if n, ok := recs[0].Data["n"]; !ok || n.(int64) != 2 {
		t.Fatalf("group by count wrong: %+v", recs[0].Data)
	}

	// Update + delete via Exec
	if r := db.ExecOneSQLParameterized(orm.ParametereizedSQL{
		Query:  "UPDATE users SET active = 0 WHERE name = ?",
		Values: []interface{}{"Bob"},
	}); r.Error != nil || r.RowsAffected != 1 {
		t.Fatalf("update: %+v", r)
	}
	if r := db.ExecOneSQLParameterized(orm.ParametereizedSQL{
		Query:  "DELETE FROM users WHERE name = ?",
		Values: []interface{}{"Bob"},
	}); r.Error != nil || r.RowsAffected != 1 {
		t.Fatalf("delete: %+v", r)
	}
}

func TestOnlyOneSemantics(t *testing.T) {
	db := newTestDB(t)
	insert := func(name, email string) {
		if r := db.InsertOneDBRecord(orm.DBRecord{TableName: "users", Data: map[string]interface{}{
			"name": name, "email": email, "age": 20,
		}}, false); r.Error != nil {
			t.Fatalf("insert: %v", r.Error)
		}
	}
	insert("a", "a@x.com")
	insert("b", "b@x.com")

	if _, err := db.SelectOnlyOneSQL("SELECT * FROM users"); err != orm.ErrSQLMoreThanOneRow {
		t.Fatalf("want ErrSQLMoreThanOneRow, got %v", err)
	}
	if _, err := db.SelectOnlyOneSQL("SELECT * FROM users WHERE name = 'zzz'"); err != orm.ErrSQLNoRows {
		t.Fatalf("want ErrSQLNoRows, got %v", err)
	}
	one, err := db.SelectOnlyOneSQL("SELECT * FROM users WHERE name = 'a'")
	if err != nil {
		t.Fatalf("select one: %v", err)
	}
	if one.Data["email"] != "a@x.com" {
		t.Fatalf("wrong record: %+v", one.Data)
	}
}

func TestTransactionCommitRollback(t *testing.T) {
	db := newTestDB(t)

	// Commit path
	tx, err := db.BeginTransaction()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	r := tx.InsertOneDBRecord(orm.DBRecord{TableName: "users", Data: map[string]interface{}{
		"name": "tx", "email": "tx@x.com", "age": 1,
	}})
	if r.Error != nil {
		tx.Rollback()
		t.Fatalf("tx insert: %v", r.Error)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	recs, _ := db.SelectMany("users")
	if len(recs) != 1 {
		t.Fatalf("after commit len = %d, want 1", len(recs))
	}

	// Rollback path
	tx2, _ := db.BeginTransaction()
	tx2.InsertOneDBRecord(orm.DBRecord{TableName: "users", Data: map[string]interface{}{
		"name": "rb", "email": "rb@x.com", "age": 1,
	}})
	if err := tx2.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	recs, _ = db.SelectMany("users")
	if len(recs) != 1 {
		t.Fatalf("after rollback len = %d, want 1", len(recs))
	}
}

func TestStatusAndSchema(t *testing.T) {
	db := newTestDB(t)
	st, err := db.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if st.DBMS != "sqlite" || st.Mode != "rw" || st.DBSize <= 0 {
		t.Fatalf("bad status: %+v", st)
	}
	schema := db.GetSchema(false, false)
	found := false
	for _, s := range schema {
		if s.ObjectName == "users" && s.ObjectType == "table" {
			found = true
		}
	}
	if !found {
		t.Fatalf("users table not in schema: %+v", schema)
	}
	if !db.IsConnected() {
		t.Fatal("IsConnected = false")
	}
}
