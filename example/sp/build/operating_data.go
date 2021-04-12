package main

import (
	"time"

	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/scripts"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database"
)

func main() {

	od := new(scripts.OperatingData)
	config := new(net.Config)
	utils.LoadYamlConfig(config, "configs/sp.yaml")
	od.DT = database.NewDataTable(config.Database)
	od.Debug = true
	od.SetTime(time.Now())
	od.GetData()
}
