package sql

import (
	"errors"
	"strconv"
	"strings"
)

// @version 0.0.2
// @change
// added combine query

// todo 0.0.3: support where、join、groupBy、orderBy

// Builder sql query builder
type Builder struct{}

// Insert
func (b *Builder) Insert(tableName string, columns []string, data ...[]interface{}) (string, []interface{}, error) {

	if tableName == "" {
		return "", nil, errors.New("no table select")
	}
	if len(data) <= 0 {
		return "", nil, errors.New("no data given")
	}

	sql := "INSERT INTO $table "
	if len(columns) > 0 {
		sql = sql + "(" + strings.Join(columns, ", ") + ") "
	}

	sql = sql + "VALUES "
	values := make([]interface{}, 0, len(data[0])*len(data))
	for idx, row := range data {

		ds := make([]string, 0, len(row))
		for _, val := range row {
			ds = append(ds, "?")
			values = append(values, val)
		}

		dsInString := strings.Join(ds, ",")
		if idx == 0 {
			sql = sql + "(" + dsInString + ")"
		} else {
			sql = sql + ", (" + dsInString + ")"
		}
	}

	sql = strings.Replace(sql, "$table", tableName, -1)

	return sql, values, nil
}

// Delete
func (b *Builder) Delete(tableName string, where map[string]interface{}) (string, []interface{}, error) {

	if tableName == "" {
		return "", nil, errors.New("no table select")
	}
	if len(where) <= 0 {
		return "", nil, errors.New("no where given")
	}

	sql := strings.Replace("DELETE FROM $table WHERE $where", "$table", tableName, -1)

	conditionsInString, values := b.Where(where)

	if conditionsInString != "" {
		sql = strings.Replace(sql, "$where", conditionsInString, -1)
	}

	return sql, values, nil
}

// Update
func (b *Builder) Update(tableName string, data, where map[string]interface{}) (string, []interface{}, error) {

	if tableName == "" {
		return "", nil, errors.New("no table select")
	}
	if len(data) <= 0 {
		return "", nil, errors.New("no data given")
	}

	sql := strings.Replace("UPDATE $table SET $set", "$table", tableName, -1)

	values := make([]interface{}, 0, len(data))
	sets := make([]string, 0, len(data))
	for col, val := range data {
		sets = append(sets, col+" = ?")
		values = append(values, val)
	}
	setsInString := strings.Join(sets, ", ")
	sql = strings.Replace(sql, "$set", setsInString, -1)

	if len(where) > 0 {
		conditionsInString, valuesInWhere := b.Where(where)
		if conditionsInString != "" {
			sql = sql + " WHERE " + conditionsInString
		}
		if len(valuesInWhere) > 0 {
			values = append(values, valuesInWhere...)
		}
	}

	return sql, values, nil
}

// Query
func (b *Builder) Query(tableName string, params map[string]interface{}) (string, []interface{}, error) {

	if tableName == "" {
		return "", nil, errors.New("no table select")
	}

	sql := strings.Replace("SELECT $columns FROM $table", "$table", tableName, -1)

	if alias, isAlias := params["alias"].(string); isAlias {
		sql = strings.Replace(sql, tableName, tableName+" AS "+alias, -1)
	}

	var joinArr [][]string
	if join, isJoin := params["join"]; isJoin {
		switch join.(type) {
		case [][]string:
			for _, arr := range join.([][]string) {
				joinArr = append(joinArr, arr)
			}
		case []string:
			joinArr = append(joinArr, join.([]string))
		}
		if len(joinArr) > 0 {
			for _, j := range joinArr {

				joinType := " JOIN "
				if len(j) == 4 {
					if strings.ToUpper(j[3]) == "LEFT" {
						joinType = " LEFT JOIN "
					} else {
						joinType = " RIGHT JOIN "
					}
				}
				sql = sql + joinType + j[0] + " AS " + j[2] + " ON " + j[1]
			}
		}
	}

	where, isSetWhere := params["where"].(map[string]interface{})
	values := make([]interface{}, 0, 0)
	conditionsInString := ""
	if isSetWhere {
		conditionsInString, values = b.Where(where)
		if conditionsInString != "" {
			sql = sql + " WHERE " + conditionsInString
		}
	}

	groupBy, isSetGroupBy := params["groupBy"].(string)
	if isSetGroupBy {
		sql = sql + " GROUP BY " + groupBy
	}

	having, isSetHaving := params["having"].(string)
	if isSetHaving {
		sql = sql + " HAVING " + having
	}

	orderBy, isSetOrderBy := params["orderBy"].(string)
	if isSetOrderBy {
		sql = sql + " ORDER BY " + orderBy
	}

	limit, isSetLimit := params["limit"]
	if isSetLimit {
		ob, ok := limit.(int)
		if ok {
			sql = sql + " LIMIT " + strconv.Itoa(ob)
		} else {
			sql = sql + " LIMIT " + limit.(string)
		}

		offset, isSetOffset := params["offset"]
		if isSetOffset {
			ob, ok := offset.(int)
			if ok {
				sql = sql + " OFFSET " + strconv.Itoa(ob)
			} else {
				sql = sql + " OFFSET " + offset.(string)
			}
		}
	}

	columns, isSetColumns := params["columns"].(string)
	if !isSetColumns {
		columns = "*"
	}
	sql = strings.Replace(sql, "$columns", columns, -1)

	return sql, values, nil
}

// Where
func (b *Builder) Where(where map[string]interface{}) (string, []interface{}) {

	valuesTotal := 0
	for _, val := range where {
		_, ok := val.([]interface{})
		if ok {
			valuesTotal += len(val.([]interface{}))
		} else {
			valuesTotal++
		}
	}

	values := make([]interface{}, 0, valuesTotal)
	conditionsInString := ""
	conditions := make([]string, 0, len(where))
	for whereInSQL, val := range where {
		conditions = append(conditions, whereInSQL)
		_, ok := val.([]interface{})
		if ok {
			values = append(values, val.([]interface{})...)
		} else {
			values = append(values, val.(interface{}))
		}
	}
	conditionsInString = strings.Join(conditions, " AND ")

	return conditionsInString, values
}

// DefaultBuilder
var DefaultBuilder Builder
