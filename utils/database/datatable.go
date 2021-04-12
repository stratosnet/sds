package database

import (
	"errors"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database/drivers"
	"reflect"
	"sync"
)

const (

	// ConnectFailed
	ConnectFailed = "database connect fail"

	BEFORE_INSERT = 0x01
	BEFORE_DELETE = 0x02
	BEFORE_UPDATE = 0x03
	BEFORE_FETCH  = 0x04

	AFTER_INSERT = 0x05
	AFTER_DELETE = 0x06
	AFTER_UPDATE = 0x07
	AFTER_FETCH  = 0x08
)

// Table
type Table interface {
	TableName() string
	PrimaryKey() []string
	SetData(data map[string]interface{}) (bool, error)
	Event(event int, dt *DataTable)
}

// Table2Map
func Table2Map(table Table) map[string]interface{} {
	data := make(map[string]interface{})
	fields := reflect.TypeOf(table).Elem()
	values := reflect.ValueOf(table)
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		index := utils.Camel2Snake(field.Name)

		data[index] = values.Elem().FieldByName(field.Name).Interface()
	}

	return data
}

// Map2Table
func Map2Table(table Table, data map[string]interface{}) (bool, error) {
	fields := reflect.TypeOf(table).Elem()
	values := reflect.ValueOf(table)
	for i := 0; i < fields.NumField(); i++ {

		v := values.Elem().Field(i)
		if v.Type().Kind() == reflect.Struct {
			for i := 0; i < v.Type().NumField(); i++ {
				f := v.Type().Field(i)
				index := utils.Camel2Snake(f.Name)
				value, ok := data[index]
				if ok && value != nil {
					v.FieldByName(f.Name).Set(reflect.ValueOf(value).Convert(f.Type))
				}
			}
		} else {
			field := fields.Field(i)
			index := utils.Camel2Snake(field.Name)
			value, ok := data[index]
			if ok && value != nil {
				values.Elem().FieldByName(field.Name).Set(reflect.ValueOf(value).Convert(field.Type))
			}
		}
	}
	return true, nil
}

// LoadTable: alias of Map2Table
func LoadTable(table Table, data map[string]interface{}) (bool, error) {
	return Map2Table(table, data)
}

// DataTable
type DataTable struct {
	driver drivers.DBDriver
	sync.Mutex
}

// IsConnected
func (dt *DataTable) IsConnected() bool {
	if !dt.driver.IsConnected() {
		utils.MyLogger.ErrorLog(ConnectFailed)
		return false
	}
	return true
}

// FetchTables
//
// @params tables, eg. tables := []table.PP
// @params params, refer to utils/database/drivers/mysql.go
// @return []Table
func (dt *DataTable) FetchTables(tables interface{}, params map[string]interface{}) (interface{}, error) {

	if !dt.driver.IsConnected() {
		return nil, errors.New(ConnectFailed)
	}

	T := reflect.TypeOf(tables)

	if T.Kind() != reflect.Slice {
		return nil, errors.New("argument #1 must be a slice")
	}

	t := T.Elem()

	v := reflect.New(t)

	tableNameMethod := v.MethodByName("TableName")
	tableName := tableNameMethod.Call(nil)

	if len(tableName) <= 0 {
		return nil, errors.New("missing table name")
	}

	tblName := tableName[0].String()

	rows, _ := dt.driver.FetchAll(tblName, params)

	V := reflect.ValueOf(tables)

	if len(rows) > 0 {
		for _, row := range rows {
			table := reflect.New(t).Interface().(Table)
			LoadTable(table, row)

			V = reflect.Append(V, reflect.ValueOf(table).Elem())
		}
	}

	return V.Interface(), nil
}

// FetchTable
//
// @params table
// @params params, refer to utils/database/drivers/mysql.go
func (dt *DataTable) FetchTable(table Table, params map[string]interface{}) error {

	if !dt.driver.IsConnected() {
		return errors.New(ConnectFailed)
	}

	V := reflect.ValueOf(table)

	table.Event(BEFORE_FETCH, dt)

	tableNameMethod := V.MethodByName("TableName")
	tableName := tableNameMethod.Call(nil)

	if len(tableName) <= 0 {
		return errors.New("miss table name")
	}

	tblName := tableName[0].String()

	row, _ := dt.driver.FetchOne(tblName, params)

	if len(row) <= 0 {
		return errors.New("not found")
	}

	LoadTable(table, row)
	table.Event(AFTER_FETCH, dt)

	return nil

}

// UpdateTable
//
// @params table, must include data to update
// @return bool success/failure, error
func (dt *DataTable) UpdateTable(table Table) (bool, error) {

	if !dt.driver.IsConnected() {
		return false, errors.New(ConnectFailed)
	}

	V := reflect.ValueOf(table)

	tableNameMethod := V.MethodByName("TableName")
	tableName := tableNameMethod.Call(nil)
	if len(tableName) <= 0 {
		return false, errors.New("miss table name")
	}
	tblName := tableName[0].String()

	pkMethod := V.MethodByName("PrimaryKey")
	pks := pkMethod.Call(nil)

	pkNames := pks[0].Interface().([]string)

	if len(pkNames) <= 0 {
		return false, errors.New("primary key is null")
	}

	data := Table2Map(table)

	for col, val := range data {
		if utils.StrInSlices(pkNames, col) {
			delete(data, col)
			continue
		}
		if reflect.TypeOf(val).Kind() == reflect.Struct {
			delete(data, col)
			continue
		}
	}

	where := make(map[string]interface{})
	for _, pkName := range pkNames {
		pkValue := V.Elem().FieldByName(utils.Snake2Camel(pkName))
		if pkValue.Interface() == reflect.Zero(pkValue.Type()).Interface() {
			return false, errors.New("primary key value is not enough, " + pkName + "is null")
		}
		where[pkName+" = ?"] = pkValue.Interface()
	}
	table.Event(BEFORE_UPDATE, dt)

	if len(data) <= 0 {
		return false, errors.New(tblName + " no column to update")
	}

	affectRows := dt.driver.Update(tblName, data, where)
	if affectRows <= 0 {
		return false, errors.New(tblName + " save fail")
	}

	table.Event(AFTER_UPDATE, dt)
	return true, nil
}

// InsertTable
//
// @params table, must including data to insert
// @return bool success/failure，error
func (dt *DataTable) InsertTable(table Table) (bool, error) {

	if !dt.driver.IsConnected() {
		return false, errors.New(ConnectFailed)
	}

	V := reflect.ValueOf(table)

	tableNameMethod := V.MethodByName("TableName")
	tableName := tableNameMethod.Call(nil)
	if len(tableName) <= 0 {
		return false, errors.New("miss table name")
	}
	tblName := tableName[0].String()

	pkMethod := V.MethodByName("PrimaryKey")
	pks := pkMethod.Call(nil)

	pkNames := pks[0].Interface().([]string)

	if len(pkNames) <= 0 {
		return false, errors.New("primary key is null")
	}

	data := Table2Map(table)

	columns := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	for col, val := range data {
		if reflect.TypeOf(val).Kind() == reflect.Struct {
			delete(data, col)
			continue
		}
		columns = append(columns, col)
		values = append(values, val)
	}

	if len(columns) <= 0 || len(values) <= 0 {
		return false, errors.New(tblName + " no data given")
	}

	table.Event(BEFORE_INSERT, dt)
	newID, insertRow := dt.driver.Insert(tblName, columns, values)
	if insertRow > 0 {
		if newID > 0 {
			data[pkNames[0]] = newID
		}
		Map2Table(table, data)
		table.Event(AFTER_INSERT, dt)
		return true, nil
	}

	return false, errors.New(tblName + " insert fail")
}

// StoreTable
//
// @notice if table has auto-incremental id primary key, action is update, otherwise action is insert
// @params table, must have data to store
// @return bool success/failure，error
func (dt *DataTable) StoreTable(table Table) (bool, error) {

	if !dt.driver.IsConnected() {
		return false, errors.New(ConnectFailed)
	}

	V := reflect.ValueOf(table)

	tableNameMethod := V.MethodByName("TableName")
	tableName := tableNameMethod.Call(nil)
	if len(tableName) <= 0 {
		return false, errors.New("miss table name")
	}
	tblName := tableName[0].String()

	pkMethod := V.MethodByName("PrimaryKey")
	pks := pkMethod.Call(nil)

	pkNames := pks[0].Interface().([]string)

	if len(pkNames) <= 0 {
		return false, errors.New("primary key is null")
	}

	data := Table2Map(table)

	columns := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	for col, val := range data {
		if col == "id" {
			continue
		}
		if reflect.TypeOf(val).Kind() == reflect.Struct {
			delete(data, col)
			continue
		}
		columns = append(columns, col)
		values = append(values, val)
	}

	table.Event(BEFORE_UPDATE, dt)
	if utils.StrInSlices(pkNames, "id") {
		id := V.Elem().FieldByName("Id")
		if id.Interface() != reflect.Zero(id.Type()).Interface() {

			if len(data) <= 0 {
				return false, errors.New(tblName + " no column to update")
			}

			affectRows := dt.driver.Update(tblName, data, map[string]interface{}{"id = ?": id.Interface()})
			table.Event(AFTER_UPDATE, dt)
			return affectRows > 0, nil
		}
	}

	if len(columns) <= 0 || len(values) <= 0 {
		return false, errors.New(tblName + " no data given")
	}

	table.Event(BEFORE_INSERT, dt)
	newID, insertRow := dt.driver.Insert(tblName, columns, values)
	if insertRow > 0 {
		if utils.StrInSlices(columns, "id") {
			data["id"] = newID
		}
		Map2Table(table, data)
		table.Event(AFTER_INSERT, dt)
		return true, nil
	}

	return false, errors.New(tblName + " save fail")
}

// DeleteTable delete based on primary key
func (dt *DataTable) DeleteTable(table Table) (bool, error) {

	if !dt.driver.IsConnected() {
		return false, errors.New(ConnectFailed)
	}

	V := reflect.ValueOf(table)

	table.Event(BEFORE_DELETE, dt)

	tableNameMethod := V.MethodByName("TableName")
	tableName := tableNameMethod.Call(nil)
	if len(tableName) <= 0 {
		return false, errors.New("miss table name")
	}
	tblName := tableName[0].String()

	pkMethod := V.MethodByName("PrimaryKey")
	pks := pkMethod.Call(nil)
	pkNames := pks[0].Interface().([]string)
	if len(pkNames) <= 0 {
		return false, errors.New("primary key is null")
	}

	data := Table2Map(table)

	columns := make([]string, 0, len(data))
	for col, val := range data {
		if utils.StrInSlices(pkNames, col) {
			delete(data, col)
			continue
		}
		if reflect.TypeOf(val).Kind() == reflect.Struct {
			delete(data, col)
			continue
		}
		columns = append(columns, col)
	}

	where := make(map[string]interface{})
	for _, pkName := range pkNames {
		pkValue := V.Elem().FieldByName(utils.Snake2Camel(pkName))
		if pkValue.Interface() == reflect.Zero(pkValue.Type()).Interface() {
			return false, errors.New("primary key value is not enough, " + pkName + "is null")
		}
		where[pkName+" = ?"] = pkValue.Interface()
	}
	table.Event(BEFORE_DELETE, dt)

	deleteRows := dt.driver.Delete(tblName, where)
	if deleteRows > 0 {
		table.Event(AFTER_DELETE, dt)
		return true, nil
	}
	return false, errors.New("delete fail")
}

// CountTable
func (dt *DataTable) CountTable(table Table, params map[string]interface{}) (int64, error) {

	if !dt.driver.IsConnected() {
		return 0, errors.New(ConnectFailed)
	}

	V := reflect.ValueOf(table)

	pkMethod := V.MethodByName("PrimaryKey")
	pks := pkMethod.Call(nil)

	tableNameMethod := V.MethodByName("TableName")
	tableName := tableNameMethod.Call(nil)

	if len(tableName) <= 0 {
		return 0, errors.New("miss table name")
	}
	if len(pks) <= 0 {
		return 0, errors.New("miss table primary key")
	}

	tblName := tableName[0].String()

	total := dt.driver.Count(tblName, params)

	return total, nil
}

// SumTable based on column
func (dt *DataTable) SumTable(table Table, field string, params map[string]interface{}) (int64, error) {

	if !dt.driver.IsConnected() {
		return 0, errors.New(ConnectFailed)
	}

	V := reflect.ValueOf(table)

	pkMethod := V.MethodByName("PrimaryKey")
	pks := pkMethod.Call(nil)

	tableNameMethod := V.MethodByName("TableName")
	tableName := tableNameMethod.Call(nil)

	if len(tableName) <= 0 {
		return 0, errors.New("miss table name")
	}
	if len(pks) <= 0 {
		return 0, errors.New("miss table primary key")
	}

	if field == "" {
		return 0, errors.New("miss sum field")
	}

	tblName := tableName[0].String()

	total := dt.driver.Sum(tblName, field, params)

	return total, nil
}

// GetDriver
func (dt *DataTable) GetDriver() drivers.DBDriver {
	return dt.driver
}

// NewDataTable
// @conf two possible format：1. yaml config file path，2. map including all configuration
func NewDataTable(conf interface{}) *DataTable {
	return &DataTable{
		driver: New(conf),
	}
}
