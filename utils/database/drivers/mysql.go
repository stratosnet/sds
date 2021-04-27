package drivers

// db actions, interact without writing sql
// only include query, delete, update and batch insert

import (
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database/config"
	"github.com/stratosnet/sds/utils/database/rows"
	"github.com/stratosnet/sds/utils/database/sql"
	"os"
	"path/filepath"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

// MySQL
type MySQL struct {
	connected bool
	config    *config.Connect
	sql.Executor
}

// Init
func (my *MySQL) Init(config *config.Connect) bool {

	my.config = config

	path, _ := filepath.Abs(filepath.Dir(my.config.LogFile))
	if _, err := os.Stat(path); err != nil {
		err := os.MkdirAll(path, 0711)
		if err != nil {
			utils.ErrorLog("creating directory failed")
			return false
		}
	}

	my.Log = utils.NewLogger(my.config.LogFile, my.config.Debug, true)
	my.Log.SetLogLevel(utils.Debug)

	if ok, err := my.Connect("mysql", my.config.DNS()); !ok {
		my.Log.ErrorLog(err.Error())
		my.connected = false
		return false
	}

	my.Debug = my.config.Debug
	my.connected = true

	return true
}

// IsConnected
func (my *MySQL) IsConnected() bool {
	return my.connected
}

// Insert
//
// @params tableName
// @params columns.  format: []string{"domain", "url"}, attention: when columns are empty, it means all columns
// @params data... format: []interface{}{ "exsample.com", "/mail" }, same order of columns
func (my *MySQL) Insert(tableName string, columns []string, data ...[]interface{}) (int64, int64) {

	sql, values, err := sql.DefaultBuilder.Insert(tableName, columns, data...)
	if err != nil {
		my.Log.ErrorLog(err)
	}

	if rowAffected, newID := my.Execute(sql, values...); rowAffected > 0 {
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
func (my *MySQL) Delete(tableName string, where map[string]interface{}) int64 {

	sql, values, err := sql.DefaultBuilder.Delete(tableName, where)

	if err != nil {
		my.Log.ErrorLog(err)
	}

	rowsAffected, _ := my.Execute(sql, values...)
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
func (my *MySQL) Update(tableName string, data, where map[string]interface{}) int64 {

	sql, values, err := sql.DefaultBuilder.Update(tableName, data, where)

	if err != nil {
		my.Log.ErrorLog(err)
	}

	rowsAffected, _ := my.Execute(sql, values...)

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
func (my *MySQL) FetchAll(tableName string, params map[string]interface{}) ([]map[string]interface{}, error) {

	sql, values, err := sql.DefaultBuilder.Query(tableName, params)
	if err != nil {
		my.Log.ErrorLog(err)
	}

	// 缓存层
	if cache, isCache := params["cache"]; isCache {
		if my.IsCacheEngineOK() {
			return my.QueryCache(cache.(map[string]interface{}), sql, values...)
		}
		utils.Log("cache engine is nil")
	}

	return rows.ToMaps(my.Query(sql, values...))
}

// FetchOne
//
// refer to FetchAll
//
func (my *MySQL) FetchOne(tableName string, params map[string]interface{}) (map[string]interface{}, error) {

	params["limit"] = 1

	data, err := my.FetchAll(tableName, params)
	if len(data) <= 0 || err != nil {
		return map[string]interface{}{}, err
	}
	return data[0], nil
}

// Count
func (my *MySQL) Count(tableName string, params map[string]interface{}) int64 {

	params["columns"] = "COUNT(*) AS total"

	sql, values, err := sql.DefaultBuilder.Query(tableName, params)

	if err != nil {
		my.Log.ErrorLog(err)
	}

	rows, _ := rows.ToMaps(my.Query(sql, values...))
	if len(rows) == 0 {
		return 0
	}
	data := rows[0]
	if total, ok := data["total"]; ok {
		return total.(int64)
	}

	return 0
}

// Sum
func (my *MySQL) Sum(tableName string, field string, params map[string]interface{}) int64 {

	params["columns"] = "SUM(" + field + ") AS total"

	sql, values, err := sql.DefaultBuilder.Query(tableName, params)

	if err != nil {
		my.Log.ErrorLog(err)
	}

	rows, _ := rows.ToMaps(my.Query(sql, values...))

	data := rows[0]
	if total, ok := data["total"]; ok && total != nil {
		i, _ := strconv.Atoi(total.(string))
		return int64(i)
	}

	return 0
}
