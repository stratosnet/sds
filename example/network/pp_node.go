package main

import (
	"github.com/stratosnet/sds/pp/api"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/utils"
)

func main() {
	setting.LoadConfig("./configs/config.yaml")

	if setting.Config.Debug {
		utils.MyLogger.SetLogLevel(utils.Debug)
	} else {
		utils.MyLogger.SetLogLevel(utils.Error)
	}

	setting.IsAuto = true
	stratoschain.Url = "http://" + setting.Config.StratosChainAddress + ":" + setting.Config.StratosChainPort

	err := setting.SetupP2PKey()
	if err != nil {
		utils.ErrorLog("Couldn't setup PP node", err)
		return
	}

	if setting.Config.IsWallet {
		go api.StartHTTPServ()
	}

	peers.StartPP()
}
