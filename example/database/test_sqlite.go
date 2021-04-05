package main

import (
	"fmt"
	"github.com/qsnetwork/sds/utils/database"
)

func main() {

	/* table

	CREATE TABLE `test` (
		`id` INTEGER PRIMARY KEY AUTOINCREMENT,
		`domain` VARCHAR(128) NOT NULL DEFAULT '',
		`url` VARCHAR(128) NOT NULL DEFAULT ''
	);

	*/

	db := database.New("examples/database/sqlite_config.yaml")

	// insert
	columns := []string{"domain", "url"}
	data := [][]interface{}{
		{"exsample.com", "/mail"},
		{"www.163.com", "/index"},
		{"www.baidu.com", "/index"},
		{"www.jd.com", "/list"},
	}

	id, insertRow := db.Insert("test", columns, data...)

	fmt.Printf("added %d row，last record id : %d\n", insertRow, id)

	// delete
	where := map[string]interface{}{
		"id = ?": id,
	}
	deleteRows := db.Delete("test", where)

	fmt.Println("deleted ", deleteRows, " row")

	// update
	updateData := map[string]interface{}{"url": "/index", "domain": "www.example.com"}
	updateWhere := map[string]interface{}{"domain = ?": "www.163.com"}
	updateRows := db.Update("test", updateData, updateWhere)

	fmt.Println("updated ", updateRows, " row")

	// fetch all
	result, err := db.FetchAll("test", map[string]interface{}{
		"columns": "id, domain, url",
		"where": map[string]interface{}{
			"url = ? AND id > ?": []interface{}{"/index", 1},
			// equal to the following
			//"url = ?": "/index",
			//"id > ?": 1,
		},
		"groupBy": "id",
		"orderBy": "id DESC",
		//"limit": 1,
		//"offset": 1,
	})
	if err == nil {
		if len(result) > 0 {
			fmt.Println("fetch all：")
			for _, row := range result {
				fmt.Printf("id=%v, domain=%v, url=%v\n", row["id"], row["domain"], row["url"])
			}
		}
	}

	//  fetch one
	oneRow, err := db.FetchOne("test", map[string]interface{}{
		"where": map[string]interface{}{
			"domain = ? AND url = ?": []interface{}{"www.jd.com", "/list"},
		},
	})
	if err == nil {
		fmt.Println("fetch one：")
		fmt.Printf("id=%v, domain=%v, url=%v\n", oneRow["id"], oneRow["domain"], oneRow["url"])
	}
}
