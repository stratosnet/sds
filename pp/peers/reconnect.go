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
					event.GetSPList()
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
	newConnection, err := setting.ConnectToSP()
	if newConnection {
		event.RegisterChain(true)
		if setting.IsStartMining {
			event.StartMining()
		}
	}

	if err != nil {
		utils.Log("couldn't connect to SP node. Retrying in 3 seconds...")
		clock.NewClock().AddJobRepeat(time.Second*3, 1, reloadConnectSP)
	}
}
