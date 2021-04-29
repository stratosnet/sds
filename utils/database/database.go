package database

import (
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database/config"
	"github.com/stratosnet/sds/utils/database/drivers"
	"reflect"
)

// @version 0.0.2
// @change
// 	0.0.1 add mysql
// 	0.0.2 add sqlite

// New
func New(conf interface{}) drivers.DBDriver {

	connectConf := &config.Connect{}

	switch t := conf.(type) {
	case *config.Connect:
		connectConf = t
	case config.Connect:
		connectConf = &t
	default:
		typeOfConf := reflect.TypeOf(conf)
		if typeOfConf.Kind() == reflect.String {
			connectConf.LoadConfFromYaml(conf.(string))
		} else if typeOfConf.Kind() == reflect.Map {
			connectConf.LoadConfFromMap(conf.(map[interface{}]interface{}))
		} else {
			utils.ErrorLog("do not support conf type")
			return nil
		}
	}

	var driver drivers.DBDriver
	switch connectConf.Driver {

	case "mysql":
		mysql := &drivers.MySQL{}
		if mysql.Init(connectConf) {
			driver = mysql
		}

	case "sqlite":
		sqLite := &drivers.SQLite{}
		if sqLite.Init(connectConf) {
			driver = sqLite
		}

	}

	return driver
}
