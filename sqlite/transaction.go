package sqlite

import (
	"database/sql"
	"time"

	orm "github.com/medatechnology/simpleorm"
)

// sqliteTransaction implements orm.Transaction over a database/sql transaction.
type sqliteTransaction struct {
	tx *sql.Tx
}

func (t *sqliteTransaction) Commit() error   { return t.tx.Commit() }
func (t *sqliteTransaction) Rollback() error { return t.tx.Rollback() }

func (t *sqliteTransaction) exec(query string, args ...interface{}) orm.BasicSQLResult {
	start := time.Now()
	res, err := t.tx.Exec(query, args...)
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

func (t *sqliteTransaction) query(query string, args ...interface{}) (orm.DBRecords, error) {
	rows, err := t.tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows, "")
}

func (t *sqliteTransaction) selectOnlyOne(query string, args ...interface{}) (orm.DBRecord, error) {
	recs, err := t.query(query, args...)
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

func (t *sqliteTransaction) ExecOneSQL(sql string) orm.BasicSQLResult {
	return t.exec(sql)
}

func (t *sqliteTransaction) ExecOneSQLParameterized(p orm.ParametereizedSQL) orm.BasicSQLResult {
	return t.exec(p.Query, p.Values...)
}

func (t *sqliteTransaction) ExecManySQL(sqls []string) ([]orm.BasicSQLResult, error) {
	out := make([]orm.BasicSQLResult, 0, len(sqls))
	for _, q := range sqls {
		if q == "" || q == ";" {
			continue
		}
		r := t.exec(q)
		if r.Error != nil {
			return out, r.Error
		}
		out = append(out, r)
	}
	return out, nil
}

func (t *sqliteTransaction) ExecManySQLParameterized(ps []orm.ParametereizedSQL) ([]orm.BasicSQLResult, error) {
	out := make([]orm.BasicSQLResult, 0, len(ps))
	for _, p := range ps {
		r := t.exec(p.Query, p.Values...)
		if r.Error != nil {
			return out, r.Error
		}
		out = append(out, r)
	}
	return out, nil
}

func (t *sqliteTransaction) SelectOneSQL(sql string) (orm.DBRecords, error) {
	return t.query(sql)
}

func (t *sqliteTransaction) SelectOnlyOneSQL(sql string) (orm.DBRecord, error) {
	return t.selectOnlyOne(sql)
}

func (t *sqliteTransaction) SelectOneSQLParameterized(p orm.ParametereizedSQL) (orm.DBRecords, error) {
	return t.query(p.Query, p.Values...)
}

func (t *sqliteTransaction) SelectOnlyOneSQLParameterized(p orm.ParametereizedSQL) (orm.DBRecord, error) {
	return t.selectOnlyOne(p.Query, p.Values...)
}

func (t *sqliteTransaction) InsertOneDBRecord(rec orm.DBRecord) orm.BasicSQLResult {
	q, values := rec.ToInsertSQLParameterized()
	return t.exec(q, values...)
}

func (t *sqliteTransaction) InsertManyDBRecords(recs []orm.DBRecord) ([]orm.BasicSQLResult, error) {
	out := make([]orm.BasicSQLResult, 0, len(recs))
	for _, r := range recs {
		res := t.InsertOneDBRecord(r)
		if res.Error != nil {
			return out, res.Error
		}
		out = append(out, res)
	}
	return out, nil
}

func (t *sqliteTransaction) InsertManyDBRecordsSameTable(recs []orm.DBRecord) ([]orm.BasicSQLResult, error) {
	statements := orm.DBRecords(recs).ToInsertSQLParameterized()
	out := make([]orm.BasicSQLResult, 0, len(statements))
	for _, p := range statements {
		r := t.exec(p.Query, p.Values...)
		if r.Error != nil {
			return out, r.Error
		}
		out = append(out, r)
	}
	return out, nil
}

func (t *sqliteTransaction) InsertOneTableStruct(obj orm.TableStruct) orm.BasicSQLResult {
	var rec orm.DBRecord
	if err := rec.FromStruct(obj); err != nil {
		return orm.BasicSQLResult{Error: err}
	}
	q, values := rec.ToInsertSQLParameterized()
	return t.exec(q, values...)
}

func (t *sqliteTransaction) InsertManyTableStructs(objs []orm.TableStruct) ([]orm.BasicSQLResult, error) {
	out := make([]orm.BasicSQLResult, 0, len(objs))
	for _, o := range objs {
		r := t.InsertOneTableStruct(o)
		if r.Error != nil {
			return out, r.Error
		}
		out = append(out, r)
	}
	return out, nil
}
