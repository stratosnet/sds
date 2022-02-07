package setting

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"sync"

	"github.com/stratosnet/sds/msg/protos"
	ppTypes "github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/types"
)

var IsPP = false

var IsLoginToSP = false

var State byte = ppTypes.PP_INACTIVE

var IsStartMining = false

var IsAuto = false

var WalletAddress string

// WalletPublicKey Public key in uncompressed format
var WalletPublicKey []byte

var WalletPrivateKey []byte

var NetworkAddress string

var RestAddress string

var P2PAddress string

var P2PPublicKey []byte

var P2PPrivateKey []byte

var SPMap = &sync.Map{}

// Map of the PPs that the current node knows about
var ppMapByNetworkAddress = &sync.Map{}
var ppMapByP2pAddress = &sync.Map{}

var rwmutex sync.RWMutex

func GetLocalPPList() []*protos.PPBaseInfo {
	empty := true
	ppMapByNetworkAddress.Range(func(k, v interface{}) bool {
		empty = false
		return false
	})

	if empty {
		csvFile, err := os.OpenFile(filepath.Join(Config.PPListDir, "pp-list"), os.O_CREATE|os.O_RDWR, 0777)
		defer csvFile.Close()
		if err != nil {
			utils.Log("InitPPList err", err)
		}
		reader := csv.NewReader(csvFile)
		reader.FieldsPerRecord = -1
		record, err := reader.ReadAll()
		if err != nil {
			utils.Log("InitPPList err", err)
		}

		for _, item := range record {
			networkID, err := types.IDFromString(item[0])
			if err != nil {
				utils.ErrorLog("invalid networkID in local PP list: " + item[0])
				continue
			}

			pp := &protos.PPBaseInfo{
				NetworkAddress: networkID.NetworkAddress,
				P2PAddress:     networkID.P2pAddress,
			}
			ppMapByNetworkAddress.Store(pp.NetworkAddress, pp)
			ppMapByP2pAddress.Store(pp.P2PAddress, pp)
		}
	}

	var ppList []*protos.PPBaseInfo
	ppMapByNetworkAddress.Range(func(k, v interface{}) bool {
		pp, ok := v.(*protos.PPBaseInfo)
		if !ok {
			utils.ErrorLogf("Invalid PP with network address %v in local PP map)", k)
			return true
		}
		ppList = append(ppList, pp)
		return true
	})

	utils.Log("ppList == ", ppList)
	return ppList
}

func SavePPList(target *protos.RspGetPPList) {
	for _, info := range target.PpList {
		if info.NetworkAddress == NetworkAddress {
			continue
		}

		if _, found := ppMapByP2pAddress.Load(info.P2PAddress); !found {
			utils.DebugLogf("adding %v (%v) to local ppList", info.P2PAddress, info.NetworkAddress)
			ppMapByP2pAddress.Store(info.P2PAddress, info)
			ppMapByNetworkAddress.Store(info.NetworkAddress, info)
		}
	}
	savePPListLocal()
}

func savePPListLocal() {
	rwmutex.Lock()
	defer rwmutex.Unlock()

	ppListPath := filepath.Join(Config.PPListDir, "pp-list")
	os.Truncate(ppListPath, 0)
	csvFile, err := os.OpenFile(ppListPath, os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		utils.ErrorLog("savePPListLocal err", err)
		return
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)

	linesWritten := 0
	ppMapByNetworkAddress.Range(func(k, v interface{}) bool {
		pp, ok := v.(*protos.PPBaseInfo)
		if !ok {
			utils.ErrorLogf("Invalid PP with network address %v in local PP map)", k)
			return true
		}

		line := []string{types.NetworkID{P2pAddress: pp.P2PAddress, NetworkAddress: pp.NetworkAddress}.String()}
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
}

func GetPPByNetworkAddress(networkAddress string) *protos.PPBaseInfo {
	value, found := ppMapByNetworkAddress.Load(networkAddress)
	if !found {
		return nil
	}

	pp, ok := value.(*protos.PPBaseInfo)
	if !ok {
		utils.ErrorLogf("Invalid PP with network address %v in local PP map)", networkAddress)
		ppMapByNetworkAddress.Delete(networkAddress)
		return nil
	}
	return pp
}

func GetPPByP2pAddress(p2pAddress string) *protos.PPBaseInfo {
	value, found := ppMapByP2pAddress.Load(p2pAddress)
	if !found {
		return nil
	}

	pp, ok := value.(*protos.PPBaseInfo)
	if !ok {
		utils.ErrorLogf("Invalid PP with p2p address %v in local PP map)", p2pAddress)
		ppMapByP2pAddress.Delete(p2pAddress)
		return nil
	}
	return pp
}

func DeletePPListByNetworkAddress(networkAddress string) {
	value, found := ppMapByNetworkAddress.Load(networkAddress)
	if !found {
		utils.DebugLogf("Cannot delete PP %v from local ppList: PP doesn't exist")
		return
	}

	pp, ok := value.(*protos.PPBaseInfo)
	if !ok {
		utils.ErrorLogf("Invalid PP with network address %v in local PP map)", networkAddress)
		ppMapByNetworkAddress.Delete(networkAddress)
		return
	}

	utils.DebugLogf("deleting %v (%v) from local ppList", pp.P2PAddress, networkAddress)
	ppMapByNetworkAddress.Delete(networkAddress)
	ppMapByP2pAddress.Delete(pp.P2PAddress)

	savePPListLocal()
}

func GetNetworkID() types.NetworkID {
	return types.NetworkID{
		P2pAddress:     P2PAddress,
		NetworkAddress: NetworkAddress,
	}
}

func GetPPInfo() *protos.PPBaseInfo {
	return &protos.PPBaseInfo{
		P2PAddress:     P2PAddress,
		WalletAddress:  WalletAddress,
		NetworkAddress: NetworkAddress,
		RestAddress:    RestAddress,
	}
}
