package main

import (
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/scripts"
	"github.com/qsnetwork/sds/utils"
	"github.com/qsnetwork/sds/utils/database"
	"time"
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
