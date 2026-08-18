package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	orm "github.com/medatechnology/simpleorm"
)

// sqliteImpl holds the database/sql handle. Exposed via SqliteDirectDB.
type sqliteImpl struct {
	db *sql.DB
}

// open connects to the SQLite file with the configured pragmas.
func open(cfg *SqliteConfig) (*sqliteImpl, error) {
	db, err := sql.Open("sqlite", cfg.dsn())
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: ping: %w", err)
	}
	return &sqliteImpl{db: db}, nil
}

// normalize converts driver values into stable Go types ([]byte → string).
func normalize(v interface{}) interface{} {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}

// scanRows converts sql.Rows into orm.DBRecords keyed by column name.
func scanRows(rows *sql.Rows, tableName string) (orm.DBRecords, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var recs orm.DBRecords
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		rec := orm.DBRecord{TableName: tableName, Data: map[string]interface{}{}}
		for i, c := range cols {
			rec.Data[c] = normalize(vals[i])
		}
		recs = append(recs, rec)
	}
	return recs, rows.Err()
}

// exec runs a non-query statement and returns a BasicSQLResult.
func (s *sqliteImpl) exec(query string, args ...interface{}) orm.BasicSQLResult {
	start := time.Now()
	res, err := s.db.Exec(query, args...)
	r := orm.BasicSQLResult{Error: err, Timing: time.Since(start).Seconds()}
	if err == nil {
		if id, e := res.LastInsertId(); e == nil {
			r.LastInsertID = int(id)
		}
		if n, e := res.RowsAffected(); e == nil {
			r.RowsAffected = int(n)
		}
	}
	return r
}

// query runs a SELECT and returns records (tableName is only metadata).
func (s *sqliteImpl) query(tableName, query string, args ...interface{}) (orm.DBRecords, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows, tableName)
}

// ─── orm.Database: selects ───

func (s *sqliteImpl) SelectOne(tableName string) (orm.DBRecord, error) {
	if err := orm.ValidateTableName(tableName); err != nil {
		return orm.DBRecord{}, err
	}
	recs, err := s.query(tableName, "SELECT * FROM "+tableName+" LIMIT 1")
	if err != nil {
		return orm.DBRecord{}, err
	}
	if len(recs) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}
	return recs[0], nil
}

func (s *sqliteImpl) SelectMany(tableName string) (orm.DBRecords, error) {
	if err := orm.ValidateTableName(tableName); err != nil {
		return nil, err
	}
	return s.query(tableName, "SELECT * FROM "+tableName)
}

func (s *sqliteImpl) SelectOneWithCondition(tableName string, cond *orm.Condition) (orm.DBRecord, error) {
	recs, err := s.selectWithCondition(tableName, cond, true)
	if err != nil {
		return orm.DBRecord{}, err
	}
	if len(recs) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}
	return recs[0], nil
}

func (s *sqliteImpl) SelectManyWithCondition(tableName string, cond *orm.Condition) ([]orm.DBRecord, error) {
	return s.selectWithCondition(tableName, cond, false)
}

func (s *sqliteImpl) selectWithCondition(tableName string, cond *orm.Condition, one bool) ([]orm.DBRecord, error) {
	if err := orm.ValidateTableName(tableName); err != nil {
		return nil, err
	}
	if cond == nil {
		cond = &orm.Condition{}
	}
	c := *cond
	if one && c.Limit == 0 {
		c.Limit = 1
	}
	query, values, err := c.ToSelectString(tableName)
	if err != nil {
		return nil, err
	}
	return s.query(tableName, query, values...)
}

func (s *sqliteImpl) SelectManyComplex(cq *orm.ComplexQuery) ([]orm.DBRecord, error) {
	if cq == nil {
		return nil, fmt.Errorf("sqlite: nil ComplexQuery")
	}
	query, values, err := cq.ToSQL()
	if err != nil {
		return nil, err
	}
	return s.query(cq.From, query, values...)
}

func (s *sqliteImpl) SelectOneComplex(cq *orm.ComplexQuery) (orm.DBRecord, error) {
	recs, err := s.SelectManyComplex(cq)
	if err != nil {
		return orm.DBRecord{}, err
	}
	if len(recs) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}
	return recs[0], nil
}

func (s *sqliteImpl) SelectOneSQL(sql string) (orm.DBRecords, error) {
	return s.query("", sql)
}

func (s *sqliteImpl) SelectManySQL(sqls []string) ([]orm.DBRecords, error) {
	out := make([]orm.DBRecords, 0, len(sqls))
	for _, q := range sqls {
		recs, err := s.query("", q)
		if err != nil {
			return nil, err
		}
		out = append(out, recs)
	}
	return out, nil
}

func (s *sqliteImpl) SelectOnlyOneSQL(sql string) (orm.DBRecord, error) {
	return s.selectOnlyOne("", sql)
}

func (s *sqliteImpl) selectOnlyOne(tableName, sql string, args ...interface{}) (orm.DBRecord, error) {
	recs, err := s.query(tableName, sql, args...)
	if err != nil {
		return orm.DBRecord{}, err
	}
	if len(recs) == 0 {
		return orm.DBRecord{}, orm.ErrSQLNoRows
	}
	if len(recs) > 1 {
		return orm.DBRecord{}, orm.ErrSQLMoreThanOneRow
	}
	return recs[0], nil
}

func (s *sqliteImpl) SelectOneSQLParameterized(p orm.ParametereizedSQL) (orm.DBRecords, error) {
	return s.query("", p.Query, p.Values...)
}

func (s *sqliteImpl) SelectManySQLParameterized(ps []orm.ParametereizedSQL) ([]orm.DBRecords, error) {
	out := make([]orm.DBRecords, 0, len(ps))
	for _, p := range ps {
		recs, err := s.query("", p.Query, p.Values...)
		if err != nil {
			return nil, err
		}
		out = append(out, recs)
	}
	return out, nil
}

func (s *sqliteImpl) SelectOnlyOneSQLParameterized(p orm.ParametereizedSQL) (orm.DBRecord, error) {
	return s.selectOnlyOne("", p.Query, p.Values...)
}

// ─── orm.Database: exec ───

func (s *sqliteImpl) ExecOneSQL(sql string) orm.BasicSQLResult {
	return s.exec(sql)
}

func (s *sqliteImpl) ExecOneSQLParameterized(p orm.ParametereizedSQL) orm.BasicSQLResult {
	return s.exec(p.Query, p.Values...)
}

func (s *sqliteImpl) ExecManySQL(sqls []string) ([]orm.BasicSQLResult, error) {
	out := make([]orm.BasicSQLResult, 0, len(sqls))
	for _, q := range sqls {
		if q == "" || q == ";" {
			continue
		}
		r := s.exec(q)
		if r.Error != nil {
			return out, r.Error
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *sqliteImpl) ExecManySQLParameterized(ps []orm.ParametereizedSQL) ([]orm.BasicSQLResult, error) {
	out := make([]orm.BasicSQLResult, 0, len(ps))
	for _, p := range ps {
		r := s.exec(p.Query, p.Values...)
		if r.Error != nil {
			return out, r.Error
		}
		out = append(out, r)
	}
	return out, nil
}

// ─── orm.Database: inserts ───

func (s *sqliteImpl) InsertOneDBRecord(rec orm.DBRecord, queue bool) orm.BasicSQLResult {
	q, values := rec.ToInsertSQLParameterized()
	return s.exec(q, values...)
}

func (s *sqliteImpl) InsertManyDBRecords(recs []orm.DBRecord, queue bool) ([]orm.BasicSQLResult, error) {
	out := make([]orm.BasicSQLResult, 0, len(recs))
	for _, r := range recs {
		res := s.InsertOneDBRecord(r, queue)
		if res.Error != nil {
			return out, res.Error
		}
		out = append(out, res)
	}
	return out, nil
}

func (s *sqliteImpl) InsertManyDBRecordsSameTable(recs []orm.DBRecord, queue bool) ([]orm.BasicSQLResult, error) {
	statements := orm.DBRecords(recs).ToInsertSQLParameterized()
	out := make([]orm.BasicSQLResult, 0, len(statements))
	for _, p := range statements {
		r := s.exec(p.Query, p.Values...)
		if r.Error != nil {
			return out, r.Error
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *sqliteImpl) InsertOneTableStruct(obj orm.TableStruct, queue bool) orm.BasicSQLResult {
	var rec orm.DBRecord
	if err := rec.FromStruct(obj); err != nil {
		return orm.BasicSQLResult{Error: err}
	}
	q, values := rec.ToInsertSQLParameterized()
	return s.exec(q, values...)
}

func (s *sqliteImpl) InsertManyTableStructs(objs []orm.TableStruct, queue bool) ([]orm.BasicSQLResult, error) {
	out := make([]orm.BasicSQLResult, 0, len(objs))
	for _, o := range objs {
		r := s.InsertOneTableStruct(o, queue)
		if r.Error != nil {
			return out, r.Error
		}
		out = append(out, r)
	}
	return out, nil
}

// ─── orm.Database: schema / status ───

func (s *sqliteImpl) GetSchema(hideSQL, hideSureSQL bool) []orm.SchemaStruct {
	recs, err := s.query("", "SELECT type, name, tbl_name, rootpage, sql FROM sqlite_master ORDER BY type, tbl_name, name")
	if err != nil {
		return nil
	}
	var out []orm.SchemaStruct
	for _, r := range recs {
		ss := orm.SchemaStruct{}
		if v, ok := r.Data["type"].(string); ok {
			ss.ObjectType = v
		}
		if v, ok := r.Data["name"].(string); ok {
			ss.ObjectName = v
		}
		if v, ok := r.Data["tbl_name"].(string); ok {
			ss.TableName = v
		}
		if v, ok := r.Data["rootpage"].(int64); ok {
			ss.RootPage = int(v)
		}
		if v, ok := r.Data["sql"].(string); ok {
			ss.SQLCommand = v
		}
		out = append(out, ss)
	}
	return out
}

func (s *sqliteImpl) Status() (orm.NodeStatusStruct, error) {
	st := orm.NodeStatusStruct{
		StatusStruct: orm.StatusStruct{
			DBMS:     "sqlite",
			Mode:     "rw",
			Nodes:    1,
			IsLeader: true,
			Leader:   "local",
		},
		Peers: map[int]orm.StatusStruct{},
	}
	var version string
	if err := s.db.QueryRow("SELECT sqlite_version()").Scan(&version); err == nil {
		st.Version = version
	}
	var pages, pageSize int64
	if err := s.db.QueryRow("PRAGMA page_count").Scan(&pages); err == nil {
		if err := s.db.QueryRow("PRAGMA page_size").Scan(&pageSize); err == nil {
			st.DBSize = pages * pageSize
		}
	}
	return st, nil
}

// ─── orm.Database: connection ───

func (s *sqliteImpl) IsConnected() bool {
	return s.db.Ping() == nil
}

func (s *sqliteImpl) Leader() (string, error) {
	return "not implemented for sqlite", nil
}

func (s *sqliteImpl) Peers() ([]string, error) {
	return []string{}, nil
}

func (s *sqliteImpl) BeginTransaction() (orm.Transaction, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	return &sqliteTransaction{tx: tx}, nil
}
