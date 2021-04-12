package sql

import (
	"database/sql"
	"errors"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/database/rows"
	"strconv"
	"strings"
	"time"
)

// @version 0.0.1

// Executor Database Driver
type Executor struct {
	db    *sql.DB
	Log   *utils.Logger
	cache cache.Cache
	Debug bool
}

// GetDB
func (exec *Executor) GetDB() *sql.DB {
	return exec.db
}

// Connect db
//
// @params execName, including mysql, sqlite3
// @params dns, format：user:password@tpc(host:port)/databaseName?options
func (exec *Executor) Connect(execName, dns string) (bool, error) {
	db, err := sql.Open(execName, dns)
	if err != nil {
		exec.Log.ErrorLog(err.Error())
		return false, err
	}
	exec.db = db
	return true, err
}

// Execute sql, including insert, delete and update
// @return (int64 number of row affected，int64 the latest record id)
func (exec *Executor) Execute(sql string, args ...interface{}) (int64, int64) {

	if exec.Debug {
		exec.Log.Log(utils.Debug, sql, args)
	}

	stmt, err := exec.db.Prepare(sql)
	if err != nil {
		exec.Log.ErrorLog(err.Error())
		return 0, 0
	}
	defer stmt.Close()

	result, err := stmt.Exec(args...)
	if err != nil {
		exec.Log.ErrorLog(err.Error())
		return 0, 0
	}

	id, err := result.LastInsertId()
	if err != nil {
		exec.Log.ErrorLog(err.Error())
		return 0, 0
	}

	affectRows, err := result.RowsAffected()
	if err != nil {
		exec.Log.ErrorLog(err.Error())
		return 0, 0
	}

	return affectRows, id
}

// Query
// @params sql
// @params arg
func (exec *Executor) Query(sql string, args ...interface{}) *sql.Rows {

	if exec.Debug {
		exec.Log.Log(utils.Debug, sql, args)
	}

	stmt, err := exec.db.Prepare(sql)
	if err != nil {
		exec.Log.ErrorLog(err.Error())
		return nil
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		exec.Log.ErrorLog(err.Error())
		return nil
	}

	return rows
}

// QueryCache
// @params cacheOpt
// @params sql
// @params arg
func (exec *Executor) QueryCache(cacheOpt map[string]interface{}, sql string, args ...interface{}) ([]map[string]interface{}, error) {

	if exec.cache != nil {

		var cacheKey string
		var lifeTime time.Duration

		if key, isCacheKey := cacheOpt["key"]; isCacheKey {
			cacheKey = key.(string)
		} else {
			cacheKey = sql
			for _, val := range args {
				switch val.(type) {
				case int:
					cacheKey = strings.Replace(cacheKey, "?", strconv.Itoa(val.(int)), 1)
				case string:
					cacheKey = strings.Replace(cacheKey, "?", "'"+val.(string)+"'", 1)
				}
			}
			cacheKey = "mysql_cache#" + utils.CalcHash([]byte(cacheKey))
		}

		if t, isCacheLifeTime := cacheOpt["lifeTime"]; isCacheLifeTime {
			lifeTime = t.(time.Duration)
		}

		res := make([]map[string]interface{}, 0)
		if exec.cache.Get(cacheKey, &res) == nil {
			return res, nil
		}

		rs := exec.Query(sql, args...)

		res, err := rows.ToMaps(rs)
		if err != nil {
			return nil, err
		}

		err = exec.cache.Set(cacheKey, res, lifeTime)

		return res, err
	}

	return nil, errors.New("cache engine is nil")
}

// SetCacheEngine
func (exec *Executor) SetCacheEngine(cache cache.Cache) {
	exec.cache = cache
}

// IsCacheEngineOK
func (exec *Executor) IsCacheEngineOK() bool {
	return exec.cache != nil && exec.cache.IsOK() == nil
}

// Close
func (exec *Executor) Close() {
	exec.db.Close()
}
