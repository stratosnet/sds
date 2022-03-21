package types

import (
	"encoding/csv"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/types"
)

const (
	PEER_NOT_CONNECTED = iota
	PEER_CONNECTED
)

type PeerInfo struct {
	NetworkAddress string
	P2pAddress     string
	RestAddress    string
	WalletAddress  string

	DiscoveryTime      int64 // When was this peer discovered for the first time
	LastConnectionTime int64 // When was the last time we connected with this peer
	NetId              int64 // The ID of the current connection with this node, if it exists.
	Status             int
}

func (peerInfo *PeerInfo) String() string {
	return types.NetworkID{P2pAddress: peerInfo.P2pAddress, NetworkAddress: peerInfo.NetworkAddress}.String()
}

type PeerList struct {
	localNetworkAddress   string
	ppListPath            string
	ppMapByNetworkAddress *sync.Map // map[string]*PeerInfo
	ppMapByP2pAddress     *sync.Map // map[string]*PeerInfo

	rwmutex sync.RWMutex
}

func (peerList *PeerList) Init(localNetworkAddress, ppListPath string) {
	peerList.localNetworkAddress = localNetworkAddress
	peerList.ppListPath = ppListPath
	peerList.ppMapByNetworkAddress = &sync.Map{}
	peerList.ppMapByP2pAddress = &sync.Map{}
}

func (peerList *PeerList) loadPPListFromFile() error {
	// TODO: Update this after we switch to JSON
	csvFile, err := os.OpenFile(peerList.ppListPath, os.O_CREATE|os.O_RDWR, 0777)
	defer csvFile.Close()
	if err != nil {
		return errors.Wrap(err, "LoadPPListFromFile cannot open ppList file")
	}

	reader := csv.NewReader(csvFile)
	reader.FieldsPerRecord = -1
	record, err := reader.ReadAll()
	if err != nil {
		return errors.Wrap(err, "LoadPPListFromFile cannot decode csv file")
	}

	for _, item := range record {
		if len(item) < 5 {
			utils.ErrorLogf("LoadPPListFromFile ppList record is incomplete. %v fields (%v expected)", len(item), 5)
			continue
		}
		networkID, err := types.IDFromString(item[0])
		if err != nil {
			utils.ErrorLog("LoadPPListFromFile invalid networkId ["+item[0]+"]", err)
			continue
		}

		discoveryTime, err := strconv.ParseInt(item[3], 10, 64)
		if err != nil {
			utils.ErrorLog("LoadPPListFromFile invalid discoveryTime ["+item[3]+"]", err)
			continue
		}

		lastConnectionTime, err := strconv.ParseInt(item[4], 10, 64)
		if err != nil {
			utils.ErrorLog("LoadPPListFromFile invalid lastConnectionTime ["+item[4]+"]", err)
			continue
		}

		pp := &PeerInfo{
			NetworkAddress:     networkID.NetworkAddress,
			P2pAddress:         networkID.P2pAddress,
			RestAddress:        item[1],
			WalletAddress:      item[2],
			DiscoveryTime:      discoveryTime,
			LastConnectionTime: lastConnectionTime,
			Status:             PEER_NOT_CONNECTED,
		}

		peerList.ppMapByNetworkAddress.Store(pp.NetworkAddress, pp)
		peerList.ppMapByP2pAddress.Store(pp.P2pAddress, pp)
	}
	return nil
}

func (peerList *PeerList) savePPListToFile() error {
	peerList.rwmutex.Lock()
	defer peerList.rwmutex.Unlock()

	// TODO: Switch to JSON or some other format instead of CSV, to make it easier to later provide the PP list to the front-end
	err := os.Truncate(peerList.ppListPath, 0)
	if err != nil {
		return err
	}
	csvFile, err := os.OpenFile(peerList.ppListPath, os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		return err
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)

	linesWritten := 0
	peerList.ppMapByNetworkAddress.Range(func(k, v interface{}) bool {
		pp, ok := v.(*PeerInfo)
		if !ok {
			utils.ErrorLogf("Invalid PP with network address %v in local PP list)", k)
			return true
		}

		line := []string{
			types.NetworkID{P2pAddress: pp.P2pAddress, NetworkAddress: pp.NetworkAddress}.String(),
			pp.RestAddress,
			pp.NetworkAddress,
			strconv.FormatInt(pp.DiscoveryTime, 10),
			strconv.FormatInt(pp.LastConnectionTime, 10),
		}
		err = writer.Write(line)
		if err != nil {
			utils.ErrorLog("error when writing local ppList to csv:", err)
			return true
		}

		linesWritten++
		return true
	})
	writer.Flush()
	utils.DebugLogf("Saved %v PPs in local ppList", linesWritten)
	return nil
}

func (peerList *PeerList) GetPPList() (list []*PeerInfo, total, connected int64) {
	empty := true
	peerList.ppMapByNetworkAddress.Range(func(k, v interface{}) bool {
		empty = false
		return false
	})

	if empty {
		err := peerList.loadPPListFromFile()
		if err != nil {
			utils.ErrorLog("Error when loading the PP list from file", err)
		}
	}

	var ppList []*PeerInfo
	totalCnt := int64(0)
	connectCnt := int64(0)

	peerList.ppMapByNetworkAddress.Range(func(k, v interface{}) bool {
		pp, ok := v.(*PeerInfo)
		if !ok {
			utils.ErrorLogf("Invalid PP with network address %v in local PP map)", k)
			return true
		}

		totalCnt += 1
		if pp.Status == PEER_CONNECTED {
			connectCnt += 1
		}

		ppList = append(ppList, pp)
		return true
	})

	utils.Logf("#pp_in_list:[%d], #pp_connected:[%d]", totalCnt, connectCnt)
	return ppList, totalCnt, connectCnt
}

func (peerList *PeerList) SavePPList(target *protos.RspGetPPList) error {
	addedPeer := false
	for _, info := range target.PpList {
		if info.NetworkAddress == peerList.localNetworkAddress {
			continue
		}
		if info.NetworkAddress == "" && info.P2PAddress == "" {
			continue
		}

		existingPP := peerList.GetPPByNetworkAddress(info.NetworkAddress)
		if existingPP == nil {
			existingPP = peerList.GetPPByP2pAddress(info.P2PAddress)
		}

		if existingPP == nil {
			pp := &PeerInfo{
				NetworkAddress:     info.NetworkAddress,
				P2pAddress:         info.P2PAddress,
				RestAddress:        info.RestAddress,
				WalletAddress:      info.WalletAddress,
				DiscoveryTime:      time.Now().Unix(),
				LastConnectionTime: 0,
				NetId:              0,
				Status:             PEER_NOT_CONNECTED,
			}
			utils.DebugLogf("adding %v to local ppList", pp)
			if info.P2PAddress != "" {
				peerList.ppMapByP2pAddress.Store(info.P2PAddress, pp)
			}
			if info.NetworkAddress != "" {
				peerList.ppMapByNetworkAddress.Store(info.NetworkAddress, pp)
			}
			addedPeer = true
		}
	}

	if addedPeer {
		return peerList.savePPListToFile()
	}
	return nil
}

func (peerList *PeerList) GetPPByNetworkAddress(networkAddress string) *PeerInfo {
	if networkAddress == "" {
		return nil
	}
	value, found := peerList.ppMapByNetworkAddress.Load(networkAddress)
	if !found {
		return nil
	}

	pp, ok := value.(*PeerInfo)
	if !ok {
		utils.ErrorLogf("Invalid PP with network address %v in local PP list)", networkAddress)
		peerList.ppMapByNetworkAddress.Delete(networkAddress)
		return nil
	}
	return pp
}

func (peerList *PeerList) GetPPByP2pAddress(p2pAddress string) *PeerInfo {
	if p2pAddress == "" {
		return nil
	}
	value, found := peerList.ppMapByP2pAddress.Load(p2pAddress)
	if !found {
		return nil
	}

	pp, ok := value.(*PeerInfo)
	if !ok {
		utils.ErrorLogf("Invalid PP with p2p address %v in local PP list)", p2pAddress)
		peerList.ppMapByP2pAddress.Delete(p2pAddress)
		return nil
	}
	return pp
}

func (peerList *PeerList) DeletePPByNetworkAddress(networkAddress string) {
	if networkAddress == "" {
		return
	}
	pp := peerList.GetPPByNetworkAddress(networkAddress)
	if pp == nil {
		utils.DebugLogf("Cannot delete PP %v from local PP list: PP doesn't exist")
		return
	}

	utils.DebugLogf("deleting %v from local ppList", pp)
	peerList.ppMapByNetworkAddress.Delete(networkAddress)
	peerList.ppMapByP2pAddress.Delete(pp.P2pAddress)

	err := peerList.savePPListToFile()
	if err != nil {
		utils.ErrorLog("Error when saving PP list to file", err)
	}
}

func (peerList *PeerList) UpdatePP(pp *PeerInfo) {
	existingPP := peerList.GetPPByNetworkAddress(pp.NetworkAddress)
	if existingPP == nil {
		existingPP = peerList.GetPPByP2pAddress(pp.P2pAddress)
	}

	now := time.Now().Unix()
	if existingPP == nil {
		// Add new peer
		if pp.DiscoveryTime == 0 {
			pp.DiscoveryTime = now
		}
		if pp.Status == PEER_CONNECTED && pp.LastConnectionTime == 0 {
			pp.LastConnectionTime = now
		}

		if pp.P2pAddress != "" {
			peerList.ppMapByP2pAddress.Store(pp.P2pAddress, pp)
		}
		if pp.NetworkAddress != "" {
			peerList.ppMapByNetworkAddress.Store(pp.NetworkAddress, pp)
		}
	} else {
		// Update existing peer info
		if pp.P2pAddress != "" && existingPP.P2pAddress == "" {
			existingPP.P2pAddress = pp.P2pAddress
			peerList.ppMapByP2pAddress.Store(pp.P2pAddress, existingPP)
		}
		if pp.NetworkAddress != "" && existingPP.NetworkAddress == "" {
			existingPP.NetworkAddress = pp.NetworkAddress
			peerList.ppMapByNetworkAddress.Store(pp.NetworkAddress, existingPP)
		}

		if pp.RestAddress != "" {
			existingPP.RestAddress = pp.RestAddress
		}
		if pp.WalletAddress != "" {
			existingPP.WalletAddress = pp.WalletAddress
		}
		if pp.LastConnectionTime != 0 {
			existingPP.LastConnectionTime = pp.LastConnectionTime
		}

		existingPP.Status = pp.Status
		if pp.Status != PEER_NOT_CONNECTED {
			existingPP.NetId = pp.NetId
		} else {
			if pp.LastConnectionTime == 0 {
				existingPP.LastConnectionTime = now
			}
		}
	}

	err := peerList.savePPListToFile()
	if err != nil {
		utils.ErrorLog("Error when saving PP list to file", err)
	}
}

func (peerList *PeerList) PPDisconnected(p2pAddress, networkAddress string) {
	pp := peerList.GetPPByP2pAddress(p2pAddress)
	if pp == nil {
		pp = peerList.GetPPByNetworkAddress(networkAddress)
	}

	if pp == nil {
		utils.DebugLogf("PP %v (%v) is offline. It was not in the local PP list", p2pAddress, networkAddress)
	} else {
		pp.Status = PEER_NOT_CONNECTED
		pp.LastConnectionTime = time.Now().Unix()
		utils.DebugLogf("PP %v is offline", pp)

		err := peerList.savePPListToFile()
		if err != nil {
			utils.ErrorLog("Error when saving PP list to file", err)
		}
	}
}

func (peerList *PeerList) PPDisconnectedNetId(netId int64) {
	found := false
	peerList.ppMapByNetworkAddress.Range(func(k, v interface{}) bool {
		pp, ok := v.(*PeerInfo)
		if !ok {
			utils.ErrorLogf("Invalid PP with network address %v in local PP list)", k)
			return true
		}
		if pp.Status == PEER_CONNECTED && pp.NetId == netId {
			peerList.PPDisconnected(pp.P2pAddress, pp.NetworkAddress)
			found = true
			return false
		}
		return true
	})

	if !found {
		utils.DebugLogf("PP with netId %v is offline, but it was not found in the local PP list", netId)
	}
}
