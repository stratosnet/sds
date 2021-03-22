package peers

import (
	"github.com/qsnetwork/qsds/pp/client"
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils"
	"time"

	"github.com/alex023/clock"
)

func listenOffline() {
	for {
		select {
		case offline := <-client.OfflineChan:
			if offline.IsSp {
				if setting.IsPP {
					utils.DebugLog("SP is offline")
					reloadConnectSP()
				}
			} else {
				utils.Log("PP is offline")
				setting.DeletePPList(offline.NetWorkAddress)
				initPPList()
			}
		}
	}
}

func reloadConnectSP() {
	if client.SPConn == nil {
		utils.Log("reconnect SP")
		clock := clock.NewClock()
		clock.AddJobRepeat(time.Second*3, 1, reloadConnectSP)
		client.SPConn = client.NewClient(setting.Config.SPNetAddress, setting.IsPP)
		event.RegisterChain(true)
		if setting.IsSatrtMining {
			event.StartMining()
		}
	}
}
