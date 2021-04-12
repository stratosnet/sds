package main

import (
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
)

func main() {
	setting.LoadConfig("./config/config.yaml")
	peers.Start(true)
}
