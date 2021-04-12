package main

import (
	"fmt"
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/database"
)

func main() {

	/* table
	CREATE TABLE `test` (
	  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
	  `domain` varchar(128) NOT NULL DEFAULT '' COMMENT 'domain',
	  `url` varchar(128) NOT NULL DEFAULT '' COMMENT 'url',
	  PRIMARY KEY (`id`)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8

	CREATE TABLE `test2` (
	  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
	  `url_id` int(10) unsigned NOT NULL DEFAULT '0',
	  `name` varchar(16) NOT NULL DEFAULT '',
	  PRIMARY KEY (`id`)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8
	*/

	db := database.New("examples/database/mysql_config.yaml")

	// insert
	columns := []string{"domain", "url"}
	data := [][]interface{}{
		{"exsample.com", "/mail"},
		{"www.163.com", "/index"},
		{"www.baidu.com", "/index"},
		{"www.jd.com", "/list"},
	}

	id, insertRow := db.Insert("test", columns, data...)

	fmt.Printf("added %d row, last record id: %d\n", insertRow, id)

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
			fmt.Println("fetch all ：")
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

	//  count
	fmt.Println(db.Count("test", map[string]interface{}{}))

	// join
	res2, err := db.FetchAll("test", map[string]interface{}{
		"alias":   "e",
		"columns": "e.id, e.domain, e.url, t.name",
		"where": map[string]interface{}{
			"e.id > ?": 1,
		},
		"join": []string{"test2", "e.id = t.url_id", "t"},
	})
	if err == nil {
		if len(res2) > 0 {
			fmt.Println("fetch join：")
			for _, row := range res2 {
				fmt.Printf("id=%v, domain=%v, url=%v, name=%v\n", row["id"], row["domain"], row["url"], row["name"])
			}
		}
	}

	//test cache
	db.SetCacheEngine(cache.NewRedis(cache.Config{
		Engine:   "redis",
		Host:     "localhost",
		Port:     "6789",
		Pass:     "123456",
		LifeTime: 10,
	}))

	r, err := db.FetchOne("test", map[string]interface{}{
		"where": map[string]interface{}{
			"id = ?": 10,
		},
		"cache": map[string]interface{}{
			//"key": "test", //if no key assigned, then key is set by query
			"lifeTime": 10 * 60,
		},
	})
	if err == nil {
		fmt.Println("fetch one：")
		fmt.Printf("id=%v, domain=%v, url=%v\n", r["id"], r["domain"], r["url"])
	}
}
