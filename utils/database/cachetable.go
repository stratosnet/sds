package database

import (
	"errors"
	"github.com/qsnetwork/sds/utils/cache"
	"github.com/qsnetwork/sds/utils/database/config"
	"time"
)

// CTable Table的缓存扩展版本
type CTable interface {
	Table
	GetTimeOut() time.Duration
	GetCacheKey() string
	Where() map[string]interface{}
}

// CacheTable
type CacheTable struct {
	SmtQueue chan map[string]interface{}
	Cache    cache.Cache
	DataTable
}

// Fetch
// @feature fetch from cache, if not exist, get from db and update cache
func (ct *CacheTable) Fetch(table CTable) error {
	ct.Lock()
	defer ct.Unlock()

	err := ct.Cache.Get(table.GetCacheKey(), table)
	if err == nil {
		return nil
	}

	err = ct.FetchTable(table, table.Where())
	if err == nil {
		return ct.Cache.Set(table.GetCacheKey(), table, table.GetTimeOut())
	}
	return errors.New("not found")
}

// Save
// @feature save to cache, and add request to queue
func (ct *CacheTable) Save(table CTable) error {
	go func() {
		storeSmt := map[string]interface{}{
			"action": "store",
			"table":  table,
		}
		ct.SmtQueue <- storeSmt
	}()
	return nil
}

// Update
// @feature update cache and add request to queue
func (ct *CacheTable) Update(table CTable) error {
	go func() {
		storeSmt := map[string]interface{}{
			"action": "update",
			"table":  table,
		}
		ct.SmtQueue <- storeSmt
	}()
	return nil
}

// Trash
// @feature delete cache, and add request to queue
func (ct *CacheTable) Trash(table CTable) error {

	ct.Lock()
	err := ct.Cache.Delete(table.GetCacheKey())
	go func() {
		storeSmt := map[string]interface{}{
			"action": "remove",
			"table":  table,
		}
		ct.SmtQueue <- storeSmt
	}()
	ct.Unlock()
	return err
}

// handleSmt
// @feature process queue to update to db
func (ct *CacheTable) handleSmt() {
	for {
		smt := <-ct.SmtQueue
		if action, ok := smt["action"]; ok {
			if table, ok := smt["table"]; ok {
				cTable := table.(CTable)
				switch action.(string) {
				case "store":
					ct.Lock()
					ct.StoreTable(cTable)
					ct.Cache.Delete(cTable.GetCacheKey())
					ct.Unlock()
					ct.Fetch(cTable)
				case "update":
					ct.Lock()
					ct.UpdateTable(cTable)
					ct.Cache.Delete(cTable.GetCacheKey())
					ct.Unlock()
					ct.Fetch(cTable)
				case "remove":
					ct.DeleteTable(cTable)
				}
			}
		}
	}
}

// NewCacheTable 实例化事务
func NewCacheTable(cache cache.Cache, dbConf config.Connect) *CacheTable {
	ct := new(CacheTable)
	ct.Cache = cache
	ct.driver = New(dbConf)
	ct.SmtQueue = make(chan map[string]interface{}, 100)

	go ct.handleSmt()

	return ct
}
