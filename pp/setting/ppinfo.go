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

// IsPP
var IsPP = false

// IsLoginToSP
var IsLoginToSP = false

// State
var State byte = ppTypes.PP_INACTIVE

// IsStartMining
var IsStartMining = false

// IsAuto
var IsAuto = false

// WalletAddress
var WalletAddress string

// WalletPublicKey Public key in uncompressed format
var WalletPublicKey []byte

// WalletPrivateKey
var WalletPrivateKey []byte

// NetworkAddress
var NetworkAddress string

//RestAddress
var RestAddress string

// P2PAddress
var P2PAddress string

// P2PPublicKey
var P2PPublicKey []byte

// P2PPrivateKey
var P2PPrivateKey []byte

// SPMap
var SPMap = &sync.Map{}

// PPList
var PPList []*protos.PPBaseInfo

var rwmutex sync.RWMutex

// GetLocalPPList
func GetLocalPPList() []*protos.PPBaseInfo {
	if len(PPList) > 0 {
		return PPList
	}

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
	if len(record) > 0 {
		for _, item := range record {
			networkID, err := types.IDFromString(item[0])
			if err == nil {
				pp := protos.PPBaseInfo{
					NetworkAddress: networkID.NetworkAddress,
					P2PAddress:     networkID.P2pAddress,
				}
				PPList = append(PPList, &pp)
			} else {
				utils.ErrorLog("invalid networkID in local PP list: " + item[0])
			}
		}
	} else {
		utils.Log("PPList == nil")
		return nil
	}
	utils.Log("PPList == ", PPList)
	return PPList
}

// SavePPList
func SavePPList(target *protos.RspGetPPList) {
	for _, info := range target.PpList {
		if info.NetworkAddress != NetworkAddress {
			PPList = append(PPList, info)
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
		utils.ErrorLog("InitPPList err", err)
		return
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)
	utils.DebugLog("PPList len", len(PPList))
	for _, post := range PPList {
		line := []string{types.NetworkID{P2pAddress: post.P2PAddress, NetworkAddress: post.NetworkAddress}.String()}
		err = writer.Write(line)
		if err != nil {
			utils.ErrorLog("csv line ", err)
		}
	}
	writer.Flush()
}

// DeletePPList
func DeletePPList(networkAddress string) {
	utils.DebugLog("delete PP: ", networkAddress)
	for i, pp := range PPList {
		if pp.NetworkAddress == networkAddress {
			PPList = append(PPList[:i], PPList[i+1:]...)
			savePPListLocal()
			return
		}
	}
}

func GetNetworkID() types.NetworkID {
	return types.NetworkID{
		P2pAddress:     P2PAddress,
		NetworkAddress: NetworkAddress,
	}
}
