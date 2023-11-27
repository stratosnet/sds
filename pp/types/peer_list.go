package types

import (
	"context"
	"encoding/csv"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stratosnet/framework/utils/types"
	"github.com/stratosnet/sds-api/protos"
	"github.com/stratosnet/sds/pp"
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

	DiscoveryTime      int64  // When was this peer discovered for the first time
	LastConnectionTime int64  // When was the last time we connected with this peer
	Latency            int64  // The latency in ms
	ConnetionType      string // the network for pp server listen on, 'tcp' or 'tcp4' or 'tcp6'
	NetId              int64  // The ID of the current connection with this node, if it exists.
	Status             int
}

func (peerInfo *PeerInfo) String() string {
	return types.NetworkID{P2pAddress: peerInfo.P2pAddress, NetworkAddress: peerInfo.NetworkAddress}.String()
}

type PeerList struct {
	localNetworkAddress   string
	ppListPath            string
	PpMapByNetworkAddress *sync.Map // map[string]*PeerInfo
	ppMapByP2pAddress     *sync.Map // map[string]*PeerInfo

	rwmutex sync.RWMutex
}

func (peerList *PeerList) Init(localNetworkAddress, ppListPath string) {
	peerList.localNetworkAddress = localNetworkAddress
	peerList.ppListPath = ppListPath
	peerList.PpMapByNetworkAddress = &sync.Map{}
	peerList.ppMapByP2pAddress = &sync.Map{}
}

func (peerList *PeerList) loadPPListFromFile(ctx context.Context) error {
	// TODO: Update this after we switch to JSON
	csvFile, err := os.OpenFile(peerList.ppListPath, os.O_CREATE|os.O_RDWR, 0600)
	defer func() {
		_ = csvFile.Close()
	}()
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
			pp.ErrorLogf(ctx, "LoadPPListFromFile ppList record is incomplete. %v fields (%v expected)", len(item), 5)
			continue
		}
		networkID, err := types.IDFromString(item[0])
		if err != nil {
			pp.ErrorLog(ctx, "LoadPPListFromFile invalid networkId ["+item[0]+"]", err)
			continue
		}

		discoveryTime, err := strconv.ParseInt(item[3], 10, 64)
		if err != nil {
			pp.ErrorLog(ctx, "LoadPPListFromFile invalid discoveryTime ["+item[3]+"]", err)
			continue
		}

		lastConnectionTime, err := strconv.ParseInt(item[4], 10, 64)
		if err != nil {
			pp.ErrorLog(ctx, "LoadPPListFromFile invalid lastConnectionTime ["+item[4]+"]", err)
			continue
		}

		ppNode := &PeerInfo{
			NetworkAddress:     networkID.NetworkAddress,
			P2pAddress:         networkID.P2pAddress,
			RestAddress:        item[1],
			WalletAddress:      item[2],
			DiscoveryTime:      discoveryTime,
			LastConnectionTime: lastConnectionTime,
			Status:             PEER_NOT_CONNECTED,
		}

		peerList.PpMapByNetworkAddress.Store(ppNode.NetworkAddress, ppNode)
		peerList.ppMapByP2pAddress.Store(ppNode.P2pAddress, ppNode)
	}
	return nil
}

func (peerList *PeerList) SavePPListToFile(ctx context.Context) error {
	peerList.rwmutex.Lock()
	defer peerList.rwmutex.Unlock()

	// TODO: Switch to JSON or some other format instead of CSV, to make it easier to later provide the PP list to the front-end
	err := os.Truncate(peerList.ppListPath, 0)
	if err != nil {
		return err
	}
	csvFile, err := os.OpenFile(peerList.ppListPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)

	linesWritten := 0
	peerList.PpMapByNetworkAddress.Range(func(k, v interface{}) bool {
		ppNode, ok := v.(*PeerInfo)
		if !ok {
			pp.ErrorLogf(ctx, "Invalid PP with network address %v in local PP list)", k)
			return true
		}

		line := []string{
			types.NetworkID{P2pAddress: ppNode.P2pAddress, NetworkAddress: ppNode.NetworkAddress}.String(),
			ppNode.RestAddress,
			ppNode.NetworkAddress,
			strconv.FormatInt(ppNode.DiscoveryTime, 10),
			strconv.FormatInt(ppNode.LastConnectionTime, 10),
			strconv.FormatInt(ppNode.Latency, 10),
		}

		err = writer.Write(line)
		if err != nil {
			pp.ErrorLog(ctx, "error when writing local ppList to csv:", err)
			return true
		}

		linesWritten++
		return true
	})
	writer.Flush()
	pp.DebugLogf(ctx, "Saved %v PPs in local ppList", linesWritten)
	return nil
}

func (peerList *PeerList) GetPPList(ctx context.Context) (list []*PeerInfo, total, connected int64) {
	empty := true
	peerList.PpMapByNetworkAddress.Range(func(k, v interface{}) bool {
		empty = false
		return false
	})

	if empty {
		err := peerList.loadPPListFromFile(ctx)
		if err != nil {
			pp.ErrorLog(ctx, "Error when loading the PP list from file", err)
		}
	}

	var ppList []*PeerInfo
	totalCnt := int64(0)
	connectCnt := int64(0)

	peerList.PpMapByNetworkAddress.Range(func(k, v interface{}) bool {
		ppNode, ok := v.(*PeerInfo)
		if !ok {
			pp.ErrorLogf(ctx, "Invalid PP with network address %v in local PP map)", k)
			return true
		}

		totalCnt += 1
		if ppNode.Status == PEER_CONNECTED {
			connectCnt += 1
		}

		ppList = append(ppList, ppNode)
		return true
	})

	return ppList, totalCnt, connectCnt
}

func (peerList *PeerList) SavePPList(ctx context.Context, target *protos.RspGetPPList) error {
	addedPeer := false
	for _, info := range target.PpList {
		if info.NetworkAddress == peerList.localNetworkAddress {
			continue
		}
		if info.NetworkAddress == "" && info.P2PAddress == "" {
			continue
		}

		existingPP := peerList.GetPPByNetworkAddress(ctx, info.NetworkAddress)
		if existingPP == nil {
			existingPP = peerList.GetPPByP2pAddress(ctx, info.P2PAddress)
		}

		if existingPP == nil {
			ppNode := &PeerInfo{
				NetworkAddress:     info.NetworkAddress,
				P2pAddress:         info.P2PAddress,
				RestAddress:        info.RestAddress,
				WalletAddress:      info.WalletAddress,
				DiscoveryTime:      time.Now().Unix(),
				LastConnectionTime: 0,
				Latency:            0,
				NetId:              0,
				Status:             PEER_NOT_CONNECTED,
			}
			pp.DebugLogf(ctx, "adding %v to local ppList", ppNode)
			if info.P2PAddress != "" {
				peerList.ppMapByP2pAddress.Store(info.P2PAddress, ppNode)
			}
			if info.NetworkAddress != "" {
				peerList.PpMapByNetworkAddress.Store(info.NetworkAddress, ppNode)
			}
			addedPeer = true
		}
	}

	if addedPeer {
		return peerList.SavePPListToFile(ctx)
	}
	return nil
}

func (peerList *PeerList) GetPPByNetworkAddress(ctx context.Context, networkAddress string) *PeerInfo {
	if networkAddress == "" {
		return nil
	}
	value, found := peerList.PpMapByNetworkAddress.Load(networkAddress)
	if !found {
		return nil
	}

	ppNode, ok := value.(*PeerInfo)
	if !ok {
		pp.ErrorLogf(ctx, "Invalid PP with network address %v in local PP list)", networkAddress)
		peerList.PpMapByNetworkAddress.Delete(networkAddress)
		return nil
	}
	return ppNode
}

func (peerList *PeerList) GetPPByP2pAddress(ctx context.Context, p2pAddress string) *PeerInfo {
	if p2pAddress == "" {
		return nil
	}
	value, found := peerList.ppMapByP2pAddress.Load(p2pAddress)
	if !found {
		return nil
	}

	ppNode, ok := value.(*PeerInfo)
	if !ok {
		pp.ErrorLogf(ctx, "Invalid PP with p2p address %v in local PP list)", p2pAddress)
		peerList.ppMapByP2pAddress.Delete(p2pAddress)
		return nil
	}
	return ppNode
}

func (peerList *PeerList) DeletePPByNetworkAddress(ctx context.Context, networkAddress string) {
	if networkAddress == "" {
		return
	}
	ppNode := peerList.GetPPByNetworkAddress(ctx, networkAddress)
	if ppNode == nil {
		pp.DebugLogf(ctx, "Cannot delete PP %v from local PP list: PP doesn't exist", networkAddress)
		return
	}

	pp.DebugLogf(ctx, "deleting %v from local ppList", ppNode)
	peerList.PpMapByNetworkAddress.Delete(networkAddress)
	peerList.ppMapByP2pAddress.Delete(ppNode.P2pAddress)

	err := peerList.SavePPListToFile(ctx)
	if err != nil {
		pp.ErrorLog(ctx, "Error when saving PP list to file", err)
	}
}

func (peerList *PeerList) UpdatePP(ctx context.Context, ppNode *PeerInfo) {
	existingPP := peerList.GetPPByNetworkAddress(ctx, ppNode.NetworkAddress)
	if existingPP == nil {
		existingPP = peerList.GetPPByP2pAddress(ctx, ppNode.P2pAddress)
	}

	now := time.Now().Unix()
	if existingPP == nil {
		// Add new peer
		if ppNode.DiscoveryTime == 0 {
			ppNode.DiscoveryTime = now
		}
		if ppNode.Status == PEER_CONNECTED && ppNode.LastConnectionTime == 0 {
			ppNode.LastConnectionTime = now
		}

		if ppNode.P2pAddress != "" {
			peerList.ppMapByP2pAddress.Store(ppNode.P2pAddress, ppNode)
		}
		if ppNode.NetworkAddress != "" {
			peerList.PpMapByNetworkAddress.Store(ppNode.NetworkAddress, ppNode)
		}
	} else {
		// Update existing peer info
		if ppNode.P2pAddress != "" && existingPP.P2pAddress == "" {
			existingPP.P2pAddress = ppNode.P2pAddress
			peerList.ppMapByP2pAddress.Store(ppNode.P2pAddress, existingPP)
		}
		if ppNode.NetworkAddress != "" && existingPP.NetworkAddress == "" {
			existingPP.NetworkAddress = ppNode.NetworkAddress
			peerList.PpMapByNetworkAddress.Store(ppNode.NetworkAddress, existingPP)
		}

		if ppNode.RestAddress != "" {
			existingPP.RestAddress = ppNode.RestAddress
		}
		if ppNode.WalletAddress != "" {
			existingPP.WalletAddress = ppNode.WalletAddress
		}
		if ppNode.LastConnectionTime != 0 {
			existingPP.LastConnectionTime = ppNode.LastConnectionTime
		}

		existingPP.Status = ppNode.Status
		if ppNode.Status != PEER_NOT_CONNECTED {
			existingPP.NetId = ppNode.NetId
		} else {
			if ppNode.LastConnectionTime == 0 {
				existingPP.LastConnectionTime = now
			}
		}
	}

	err := peerList.SavePPListToFile(ctx)
	if err != nil {
		pp.ErrorLog(ctx, "Error when saving PP list to file", err)
	}
}
