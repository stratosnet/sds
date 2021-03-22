package main

import (
	"github.com/qsnetwork/qsds/pp/peers"
	"github.com/qsnetwork/qsds/pp/setting"
)

func main() {
	setting.LoadConfig("./config/config.yaml")
	peers.Start(true)
}
