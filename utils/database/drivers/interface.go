package drivers

import (
	"database/sql"
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/database/config"
)

// DBDriver interface
type DBDriver interface {
	Init(config *config.Connect) bool
	IsConnected() bool
	Insert(tableName string, columns []string, data ...[]interface{}) (int64, int64)
	Delete(tableName string, where map[string]interface{}) int64
	Update(tableName string, data, where map[string]interface{}) int64
	FetchAll(tableName string, params map[string]interface{}) ([]map[string]interface{}, error)
	FetchOne(tableName string, params map[string]interface{}) (map[string]interface{}, error)
	Count(tableName string, params map[string]interface{}) int64
	Sum(tableName string, field string, params map[string]interface{}) int64
	SetCacheEngine(cache cache.Cache)
	IsCacheEngineOK() bool
	GetDB() *sql.DB
}
