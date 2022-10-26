package peers

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

type OptimalSp struct {
	networkAddr string
	mtx         sync.Mutex
}

var (
	optSp               = &OptimalSp{}
	minReloadSpInterval = 3
	maxReloadSpInterval = 600
	retry               = 0
)

func ListenOffline() {
	for {
		select {
		case offline := <-client.OfflineChan:
			if offline.IsSp {
				utils.DebugLogf("SP %v is offline", offline.NetworkAddress)
				if setting.IsPP || !setting.IsPPSyncedWithSP {
					// working pp node, or pp status not yet synced with SP after node starts
					setting.IsStartMining = false
					reloadConnectSP()
					GetSPList()
					return
				}
				// not yet registered pp node (IsPPSyncedWithSP = true && IsPP = false)
				ConnectToSP()
				GetSPList()
			} else {
				peerList.PPDisconnected("", offline.NetworkAddress)
				InitPPList()
			}
		}
	}
}

func reloadConnectSP() {
	newConnection, err := ConnectToSP()
	if newConnection {
		RegisterToSP(true)
		retry = 0
		if setting.IsStartMining {
			StartMining()
		}
	}

	if err != nil {
		//calc next reload interval
		reloadSpInterval := minReloadSpInterval * int(math.Ceil(math.Pow(10, float64(retry))))
		reloadSpInterval = int(math.Min(float64(reloadSpInterval), float64(maxReloadSpInterval)))
		utils.Logf("couldn't connect to SP node. Retrying in %v seconds...", reloadSpInterval)
		retry += 1
		ppPeerClock.AddJobWithInterval(time.Duration(reloadSpInterval)*time.Second, reloadConnectSP)
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

	if optSpNetworkAddr, err := GetOptSPAndClear(); err == nil {
		utils.DebugLog("reconnect to detected optimal SP ", optSpNetworkAddr)
		client.SPConn = client.NewClient(optSpNetworkAddr, false)
		if client.SPConn != nil {
			return true, nil
		}
	}
	// Select a random SP node to connect to
	spListOrder := rand.Perm(len(setting.Config.SPList))
	for _, index := range spListOrder {
		selectedSP := setting.Config.SPList[index]
		client.SPConn = client.NewClient(selectedSP.NetworkAddress, false)
		if client.SPConn != nil {
			return true, nil
		}
	}

	return false, errors.New("couldn't connect to any SP node")
}

// ConnectToOptSP connect if there is a detected optimal SP node.
func ConfirmOptSP(spNetworkAddr string) {
	utils.DebugLog("current sp ", client.SPConn.GetName(), " to be altered to new optimal SP ", spNetworkAddr)
	if client.SPConn.GetName() == spNetworkAddr {
		utils.DebugLog("optimal SP already in connection, won't change SP")
		return
	}
	setOptSP(spNetworkAddr)
	client.SPConn.Close()
}

func GetOptSPAndClear() (string, error) {
	if len(optSp.networkAddr) > 0 {
		optSpNetworkAddr := optSp.networkAddr
		optSp = &OptimalSp{}
		return optSpNetworkAddr, nil
	}
	return "", errors.New("optimal SP not detected")
}

func setOptSP(spNetworkAddr string) {
	optSp.networkAddr = spNetworkAddr
}
