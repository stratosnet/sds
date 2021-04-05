package main

import (
	"github.com/qsnetwork/sds/pp/peers"
	"github.com/qsnetwork/sds/pp/setting"
)

func main() {
	setting.LoadConfig("./config/config.yaml")
	peers.Start(true)
}
