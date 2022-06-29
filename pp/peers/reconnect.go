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

type OptimalIndexNode struct {
	networkAddr string
	mtx         sync.Mutex
}

var (
	optIndexNode               = &OptimalIndexNode{}
	minReloadIndexNodeInterval = 3
	maxReloadIndexNodeInterval = 600
	retry                      = 0
)

func ListenOffline() {
	for {
		select {
		case offline := <-client.OfflineChan:
			if offline.IsIndexNode {
				if setting.IsPP {
					utils.DebugLogf("IndexNode %v is offline", offline.NetworkAddress)
					setting.IsStartMining = false
					reloadConnectIndexNode()
					GetIndexNodeList()
				}
			} else {
				peerList.PPDisconnected("", offline.NetworkAddress)
				InitPPList()
			}
		}
	}
}

func reloadConnectIndexNode() {
	newConnection, err := ConnectToIndexNode()
	if newConnection {
		RegisterToIndexNode(true)
		retry = 0
		if setting.IsStartMining {
			StartMining()
		}
	}

	if err != nil {
		//calc next reload interval
		reloadIndexNodeInterval := minReloadIndexNodeInterval * int(math.Ceil(math.Pow(10, float64(retry))))
		reloadIndexNodeInterval = int(math.Min(float64(reloadIndexNodeInterval), float64(maxReloadIndexNodeInterval)))
		utils.Logf("couldn't connect to IndexNode node. Retrying in %v seconds...", reloadIndexNodeInterval)
		retry += 1
		ppPeerClock.AddJobWithInterval(time.Duration(reloadIndexNodeInterval)*time.Second, reloadConnectIndexNode)
	}
}

// ConnectToIndexNode Checks if there is a connection to an IndexNode node. If it doesn't, it attempts to create one with a random IndexNode node.
func ConnectToIndexNode() (newConnection bool, err error) {
	if client.IndexNodeConn != nil {
		return false, nil
	}
	if len(setting.Config.IndexNodeList) == 0 {
		return false, errors.New("there are no Index Node nodes in the config file")
	}

	if optIndexNodeNetworkAddr, err := GetOptIndexNodeAndClear(); err == nil {
		utils.DebugLog("reconnect to detected optimal Index Node", optIndexNodeNetworkAddr)
		client.IndexNodeConn = client.NewClient(optIndexNodeNetworkAddr, false)
		if client.IndexNodeConn != nil {
			return true, nil
		}
	}
	// Select a random Index node to connect to
	indexNodeListOrder := rand.Perm(len(setting.Config.IndexNodeList))
	for _, index := range indexNodeListOrder {
		selectedIndexNode := setting.Config.IndexNodeList[index]
		client.IndexNodeConn = client.NewClient(selectedIndexNode.NetworkAddress, false)
		if client.IndexNodeConn != nil {
			return true, nil
		}
	}

	return false, errors.New("couldn't connect to any Index Node node")
}

// ConnectToOptIndexNode connect if there is a detected optimal Index Node node.
func ConfirmOptIndexNode(IndexNodeNetworkAddr string) {
	utils.DebugLog("current Index Node ", client.IndexNodeConn.GetName(), " to be altered to new optimal Index Node ", IndexNodeNetworkAddr)
	if client.IndexNodeConn.GetName() == IndexNodeNetworkAddr {
		utils.DebugLog("optimal Index Node already in connection, won't change Index Node")
		return
	}
	setOptIndexNode(IndexNodeNetworkAddr)
	client.IndexNodeConn.Close()
}

func GetOptIndexNodeAndClear() (string, error) {
	if len(optIndexNode.networkAddr) > 0 {
		optIndexNodeNetworkAddr := optIndexNode.networkAddr
		optIndexNode = &OptimalIndexNode{}
		return optIndexNodeNetworkAddr, nil
	}
	return "", errors.New("optimal Index Node not detected")
}

func setOptIndexNode(indexNodeNetworkAddr string) {
	optIndexNode.networkAddr = indexNodeNetworkAddr
}
