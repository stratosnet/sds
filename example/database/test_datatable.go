package main

import (
	"fmt"
	"github.com/stratosnet/sds/utils/database"
)

/* sqlite

CREATE TABLE `test` (
	`id` INTEGER PRIMARY KEY AUTOINCREMENT,
	`domain` VARCHAR(128) NOT NULL DEFAULT '',
	`url` VARCHAR(128) NOT NULL DEFAULT ''
);

*/

/* mysql

CREATE TABLE `test` (
	`id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
	`domain` varchar(128) NOT NULL DEFAULT '' COMMENT 'domain',
	`url` varchar(128) NOT NULL DEFAULT '' COMMENT 'url',
	PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=25 DEFAULT CHARSET=utf8

*/

//

type Test struct {
	Id     uint32
	Domain string
	Url    string
}

func (t *Test) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(t, data)
}

func (t *Test) Event(event int, dt *database.DataTable) {

}

// TableName
func (t *Test) TableName() string {
	return "test"
}

// PrimaryKey
func (t *Test) PrimaryKey() []string {
	return []string{"id"}
}

// Load
func (t *Test) Load(data map[string]interface{}) (bool, error) {
	return database.LoadTable(t, data)
}

func main() {

	// insert
	dt := database.NewDataTable("examples/database/sqlite_config.yaml")

	test := &Test{
		Id:     1,
		Domain: "www.iqiyi.com",
		Url:    "/index",
	}

	fmt.Println(test)

	if ok, err := dt.StoreTable(test); ok {
		fmt.Println("insert data，ID：", test.Id)
	} else {
		panic(err)
	}

	newTest := &Test{}

	err := dt.FetchTable(newTest, map[string]interface{}{
		"where": map[string]interface{}{
			"id = ?": test.Id,
		},
	})

	if err == nil {
		fmt.Println("get data：", newTest)
	}

}
