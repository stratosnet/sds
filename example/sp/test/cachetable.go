package main

import (
	"fmt"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/database"
	"github.com/stratosnet/sds/utils/database/config"
	"time"
)

func main() {

	r := cache.NewRedis(cache.Config{
		Engine: "redis",
		Host:   "localhost",
		Port:   "6789",
		Pass:   "123456",
		DB:     0,
	})

	connectConf := &config.Connect{}
	connectConf.LoadConfFromYaml("examples/sp/test/database.yaml")

	ct := database.NewCacheTable(r, *connectConf)

	file := new(table.File)

	file.Hash = "abcedfghijklmnopqr"

	if ct.Fetch(file) != nil {

		//file.Owner = &table.UserHasFile{
		//	P2PAddress: "12345678",
		//	FileHash:      file.Hash,
		//}
	}

	fmt.Println(file)
	file.WalletAddress = "1234567890"
	file.State = table.STATE_OK
	file.Name = "1.jpg"
	file.Time = time.Now().Unix()
	file.SliceNum = 10
	file.Download = 0
	file.Size = 1000

	ct.Save(file)

	ct.Trash(file)

	//pp := new(table.PP)
	//pp.P2PAddress = "0x999aE343980D80d2A59AffC19d1801eFd489c1Da"
	//
	////ct.Fetch(pp)
	//
	////fmt.Println(pp)
	//pp.PubKey = "654321"
	//
	//ct.Save(pp)
	//
	//fmt.Println(pp)
	//
	//select {}

	//pp.State = 1
	//
	//ct.Save(pp)
	////
	//p2 := new(table.PP)
	////
	//p2.P2PAddress = "0x999aE343980D80d2A59AffC19d1801eFd489c1Da"
	////
	//ct.Load(p2)
	////
	//fmt.Println(p2.RequestNetworkAddress)
	//
	//ct.Store(p2)
	//
	select {}
	//
	//ct.Remove(p2)

}
