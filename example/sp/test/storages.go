package main

import (
	"fmt"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database"
)

func main() {

	dataTable := database.NewDataTable("examples/sp/test/database.yaml")

	//find PP
	findPP := &table.PP{}
	err := dataTable.FetchTable(findPP, map[string]interface{}{
		"where": map[string]interface{}{
			"id = ?": 1,
		},
	})
	if err == nil {
		fmt.Println(findPP)
	}

	// find PP all
	result, _ := dataTable.FetchTables([]table.PP{}, map[string]interface{}{
		"where": map[string]interface{}{
			"id >= ?": 1,
		},
		"groupBy": "id",
		"orderBy": "id DESC",
	})

	fmt.Println(result.([]table.PP))

	// save pp
	savePP := new(table.PP)

	data := map[string]interface{}{
		"id":          1,
		"p2p_address": "AABBCCDDeeffgghh",
		"disk_size":   1024 * 1024 * 1024,
		"memory_size": 1024 * 1024,
		"os_and_ver":  "mac os 10",
		"cpu_info":    "A9",
		"mac_address": "f0:18:98:37:15:26",
		"version":     1,
		"pub_key":     "90zsdjfwje020fjsdjjoij2900293",
	}

	savePP.SetData(data)

	dataTable.StoreTable(savePP)

	fmt.Println(dataTable.CountTable(new(table.PP), map[string]interface{}{}))

	//联查扩展结构方式
	type fileSliceEx struct {
		table.FileSlice
		State byte
	}

	res, err := dataTable.FetchTables([]fileSliceEx{}, map[string]interface{}{
		"alias":   "e",
		"columns": "e.*, p.state",
		"where": map[string]interface{}{
			"e.file_hash = ? AND p.state = 1": "116fc8beb51399cab70cfd2564deab774fd5e62e5905b68df0388b1ffd4a51cd",
		},
		"join": [][]string{
			{"file_slice_storage", "fss.slice_hash = e.slice_hash", "fss", "left"},
			{"pp", "fss.p2p_address = p.p2p_address", "p", "left"},
		},
		// 支持多个join或单个join, 最后一个表示join类型，如果没有填写，则表示join
		//"join": []string{"file_slice_storage", "fss.slice_hash = e.slice_hash", "fss"},
		"groupBy": "e.slice_hash",
	})
	if err != nil {
		utils.ErrorLog(err.Error())
	}

	fileSlices := res.([]fileSliceEx)

	fmt.Println(fileSlices)

	dataTable.DeleteTable(&table.FileSlice{Id: 1305})
}
