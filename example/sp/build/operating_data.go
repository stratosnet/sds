package main

import (
	"time"

	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/scripts"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database"
)

func main() {
	config := &net.Config{}
	utils.LoadYamlConfig(config, "configs/sp.yaml")

	od := &scripts.OperatingData{
		DT:    database.NewDataTable(config.Database),
		Debug: true,
	}
	od.SetTime(time.Now())
	od.GetData()
}
