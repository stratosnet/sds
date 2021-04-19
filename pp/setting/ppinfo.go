package setting

import (
	"crypto/ecdsa"
	"encoding/csv"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
	"os"
	"sync"
)

// IsPP
var IsPP = false

// IsLoginToSP
var IsLoginToSP = false

// IsSatrtMining
var IsSatrtMining = false

// IsAuto
var IsAuto = false

// WalletAddress
var WalletAddress string

// NetworkAddress
var NetworkAddress string

// PublicKey
var PublicKey []byte

// PrivateKey
var PrivateKey *ecdsa.PrivateKey

// PPList
var PPList []*protos.PPBaseInfo

var rwmutex sync.RWMutex

var TestDownload = false

var TestUpload = false

// GetLocalPPList
func GetLocalPPList() []*protos.PPBaseInfo {
	if len(PPList) > 0 {
		return PPList
	}
	csvFile, err := os.OpenFile(Config.PPListDir, os.O_CREATE|os.O_RDWR, 0777)
	defer csvFile.Close()
	if utils.CheckError(err) {
		utils.Log("InitPPList err", err)
	}
	reader := csv.NewReader(csvFile)
	reader.FieldsPerRecord = -1
	record, err := reader.ReadAll()
	if utils.CheckError(err) {
		utils.Log("InitPPList err", err)
	}
	if len(record) > 0 {
		for _, item := range record {
			pp := protos.PPBaseInfo{
				NetworkAddress: item[0],
				WalletAddress:  item[1],
			}
			PPList = append(PPList, &pp)
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

	os.Truncate(Config.PPListDir, 0)
	csvFile, err := os.OpenFile(Config.PPListDir, os.O_CREATE|os.O_RDWR, 0777)
	if utils.CheckError(err) {
		utils.ErrorLog("InitPPList err", err)
		return
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)
	utils.DebugLog("PPList len", len(PPList))
	for _, post := range PPList {
		line := []string{post.NetworkAddress, post.WalletAddress}
		err := writer.Write(line)
		if utils.CheckError(err) {
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
