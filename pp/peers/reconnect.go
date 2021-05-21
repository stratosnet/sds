package peers

import (
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
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
				setting.DeletePPList(offline.NetworkAddress)
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
	}
}
