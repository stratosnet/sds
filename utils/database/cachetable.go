package database

import (
	"errors"
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/database/config"
	"time"
)

// CTable Table's cache version
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

	// TODO readjust the ttl for each table
	//err := ct.Cache.Get(table.GetCacheKey(), table)
	//if err == nil {
	//	return nil
	//}

	err := ct.FetchTable(table, table.Where())
	if err == nil {
		// TODO readjust the ttl for each table
		//return ct.Cache.Set(table.GetCacheKey(), table, table.GetTimeOut())
		return nil
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
	for smt := range ct.SmtQueue {
		action, found := smt["action"]
		if !found {
			continue
		}

		table, found := smt["table"]
		if !found {
			continue
		}

		cTable := table.(CTable)
		switch action.(string) {
		// TODO readjust the ttl for each table
		case "store":
			ct.Lock()
			ct.StoreTable(cTable)
			//ct.Cache.Delete(cTable.GetCacheKey())
			ct.Unlock()
			//ct.Fetch(cTable)
		case "update":
			ct.Lock()
			ct.UpdateTable(cTable)
			//ct.Cache.Delete(cTable.GetCacheKey())
			ct.Unlock()
			//ct.Fetch(cTable)
		case "remove":
			ct.DeleteTable(cTable)
		}
	}
}

// NewCacheTable instantiate
func NewCacheTable(cache cache.Cache, dbConf config.Connect) *CacheTable {
	ct := &CacheTable{
		Cache: cache,
		DataTable: DataTable{
			driver: New(dbConf),
		},
		SmtQueue: make(chan map[string]interface{}, 100),
	}

	go ct.handleSmt()

	return ct
}
