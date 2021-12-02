package peers

import (
	"math/rand"
	"time"

	"github.com/alex023/clock"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

func ListenOffline() {
	for {
		select {
		case offline := <-client.OfflineChan:
			if offline.IsSp {
				if setting.IsPP {
					utils.DebugLog("SP is offline")
					reloadConnectSP()
					GetSPList()
				}
			} else {
				utils.Log("PP is offline")
				setting.DeletePPList(offline.NetworkAddress)
				InitPPList()
			}
		}
	}
}

func reloadConnectSP() {
	newConnection, err := ConnectToSP()
	if newConnection {
		RegisterChain(true)
		if setting.IsStartMining {
			StartMining()
		}
	}

	if err != nil {
		utils.Log("couldn't connect to SP node. Retrying in 3 seconds...")
		clock.NewClock().AddJobRepeat(time.Second*3, 1, reloadConnectSP)
	}
}

// ConnectToSP Checks if there is a connection to an SP node. If it doesn't, it attempts to create one with a random SP node.
func ConnectToSP() (newConnection bool, err error) {
	if client.SPConn != nil {
		return false, nil
	}

	if len(setting.Config.SPList) == 0 {
		return false, errors.New("there are no SP nodes in the config file")
	}

	// Select a random SP node to connect to
	spListOrder := rand.Perm(len(setting.Config.SPList))
	for _, index := range spListOrder {
		selectedSP := setting.Config.SPList[index]
		client.SPConn = client.NewClient(selectedSP.NetworkAddress, setting.IsPP)
		if client.SPConn != nil {
			return true, nil
		}
	}

	return false, errors.New("couldn't connect to any SP node")
}

// ConnectToOptSP connect if there is a detected optimal SP node.
func ConnectAndRegisterToOptSP(spNetworkAddr string) error {
	// connect to optimal SP node
	newSpConn := client.NewClient(spNetworkAddr, setting.IsPP)
	if newSpConn != nil {
		// replace ongoing client.SPConn
		client.SPConn = newSpConn
		// register to new client.SPConn
		RegisterChain(true)
	}
	return errors.New("couldn't connect to optimal SP node")
}
