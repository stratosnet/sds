package drivers

import (
	"errors"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database/config"
	"github.com/stratosnet/sds/utils/database/rows"
	"github.com/stratosnet/sds/utils/database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type SQLite struct {
	connected bool
	config    *config.Connect
	sql.Executor
}

// Init
func (s *SQLite) Init(config *config.Connect) bool {

	s.config = config

	s.Log = utils.NewLogger(s.config.LogFile, false, true)
	s.Log.SetLogLevel(utils.Debug)

	// 连接数据库
	if ok, err := s.Connect("sqlite3", s.config.DNS()); !ok {
		s.Log.ErrorLog(err.Error())
		s.connected = false
		return false
	}

	s.Debug = s.config.Debug
	s.connected = true

	return true
}

// IsConnected
func (s *SQLite) IsConnected() bool {
	return s.connected
}

// Insert
//
// @params tableName
// @params columns.  format: []string{"domain", "url"}, attention: when columns are empty, it means all columns
// @params data... format: []interface{}{ "exsample.com", "/mail" }, same order of columns
func (s *SQLite) Insert(tableName string, columns []string, data ...[]interface{}) (int64, int64) {

	sql, values, err := sql.DefaultBuilder.Insert(tableName, columns, data...)
	if err != nil {
		s.Log.ErrorLog(err)
	}

	if rowAffected, newID := s.Execute(sql, values...); rowAffected > 0 {
		return newID, rowAffected
	}
	return 0, 0
}

// Delete
//
// @params tableName
// @params where, format:
//
// 		map[string]interface{}{"id = ?": 1}
// 		map[string]interface{}{"id = ? AND domain = ?": []interface{}{1, "exsample.com"}}
//
func (s *SQLite) Delete(tableName string, where map[string]interface{}) int64 {

	sql, values, err := sql.DefaultBuilder.Delete(tableName, where)

	if err != nil {
		s.Log.ErrorLog(err)
	}

	rowsAffected, _ := s.Execute(sql, values...)
	return rowsAffected
}

// Update
//
// @params tableName
// @params data, format:
//
// 		map[string]interface{}{"url": "/index", "domain": "exsample.com"}
//
// @params where, format：
//
// 		map[string]interface{}{"id = ?": 1}
// 		map[string]interface{}{"id = ? AND domain = ?": []interface{}{1, "exsample.com"}}
//
func (s *SQLite) Update(tableName string, data, where map[string]interface{}) int64 {

	sql, values, err := sql.DefaultBuilder.Update(tableName, data, where)

	if err != nil {
		s.Log.ErrorLog(err)
	}

	rowsAffected, _ := s.Execute(sql, values...)

	return rowsAffected
}

// FetchAll
//
// @params tableName
// @params params, including：where, group by, having, order by, limit, offset
// eg.
// map[string]interface{}{
//		"columns": "id, domain, url",
//		"where": map[string]interface{}{
//			"domain = ? AND url = ?": []interface{}{"www.exsample.com", "/mail"},
//			which is equally as
//			"domain = ?": "www.exsample.com",
//			"url = ?": "/mail",
//		},
//		"groupBy": "id",
//		"orderBy": "id DESC",
// 		"limit": 1,
//		"offset": 1,
// }
func (s *SQLite) FetchAll(tableName string, params map[string]interface{}) ([]map[string]interface{}, error) {

	sql, values, err := sql.DefaultBuilder.Query(tableName, params)

	if err != nil {
		s.Log.ErrorLog(err)
	}

	return rows.ToMaps(s.Query(sql, values...))
}

// FetchOne
//
// refer to FetchAll
//
func (s *SQLite) FetchOne(tableName string, params map[string]interface{}) (map[string]interface{}, error) {

	params["limit"] = 1

	data, _ := s.FetchAll(tableName, params)
	if len(data) <= 0 {
		return map[string]interface{}{}, errors.New("query no results")
	}
	return data[0], nil
}

// Count
func (s *SQLite) Count(tableName string, params map[string]interface{}) int64 {

	params["columns"] = "COUNT(*) AS total"

	sql, values, err := sql.DefaultBuilder.Query(tableName, params)

	if err != nil {
		s.Log.ErrorLog(err)
	}

	rows, _ := rows.ToMaps(s.Query(sql, values...))

	data := rows[0]
	if total, ok := data["total"]; ok {
		return total.(int64)
	}
	return 0
}

// Sum
func (s *SQLite) Sum(tableName string, field string, params map[string]interface{}) int64 {

	params["columns"] = "SUM(" + field + ") AS total"

	sql, values, err := sql.DefaultBuilder.Query(tableName, params)

	if err != nil {
		s.Log.ErrorLog(err)
	}

	rows, _ := rows.ToMaps(s.Query(sql, values...))

	data := rows[0]
	if total, ok := data["total"]; ok {
		return total.(int64)
	}

	return 0
}
