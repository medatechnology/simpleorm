package gorqlite

import (
	orm "github.com/medatechnology/simpleorm"
	"github.com/rqlite/gorqlite"
)

func ConditionToParameterized(tableName string, c *orm.Condition) gorqlite.ParameterizedStatement {
	query, values := c.ToSelectString(tableName)
	// fmt.Println("DEBUG: query = ", query)
	// fmt.Println("DEBUG: values = ", values)
	return gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: values,
	}
}

// convert from 1 simpleORM ParameterizedSQL to gorqlite.ParameterizedStatement
func FromOneParameterizedSQL(p orm.ParametereizedSQL) gorqlite.ParameterizedStatement {
	return gorqlite.ParameterizedStatement{
		Query:     p.Query,
		Arguments: p.Values,
	}
}

// convert from many simpleORM ParameterizedSQL to gorqlite.ParameterizedStatement
func FromManyParameterizedSQL(p []orm.ParametereizedSQL) []gorqlite.ParameterizedStatement {
	var ps []gorqlite.ParameterizedStatement
	for _, one := range p {
		ps = append(ps, FromOneParameterizedSQL(one))
	}
	return ps
}

// Convert DBRecord to ParameterizedStatement
func DBRecordToInsertParameterized(d *orm.DBRecord) gorqlite.ParameterizedStatement {
	sql, values := d.ToInsertSQLParameterized()
	return gorqlite.ParameterizedStatement{
		Query:     sql,
		Arguments: values,
	}
}

// Convert DBRecords to []ParameterizedStatement
func DBRecordsToInsertParameterized(d orm.DBRecords) []gorqlite.ParameterizedStatement {
	params := d.ToInsertSQLParameterized()
	return FromManyParameterizedSQL(params)
}

func WriteResultToBasicSQLResult(res gorqlite.WriteResult) orm.BasicSQLResult {
	var ret orm.BasicSQLResult
	ret.Error = res.Err
	ret.LastInsertID = int(res.LastInsertID)
	ret.RowsAffected = int(res.RowsAffected)
	ret.Timing = res.Timing
	return ret
}

func WriteResultsToBasicSQLResults(res []gorqlite.WriteResult) []orm.BasicSQLResult {
	var ret []orm.BasicSQLResult
	for _, one := range res {
		ret = append(ret, WriteResultToBasicSQLResult(one))
	}
	return ret
}
