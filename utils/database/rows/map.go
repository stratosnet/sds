package rows

import (
	"database/sql"
	"errors"
)

// convert sql response to map
func ToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {

	if rows == nil {
		return []map[string]interface{}{}, errors.New("query no results")
	}

	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return []map[string]interface{}{}, err
	}
	count := len(columns)
	tableData := make([]map[string]interface{}, 0)
	values := make([]interface{}, count)
	valuePointer := make([]interface{}, count)
	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePointer[i] = &values[i]
		}
		rows.Scan(valuePointer...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		tableData = append(tableData, entry)
	}
	return tableData, nil
}
