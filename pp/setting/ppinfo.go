package setting

import (
	"encoding/csv"
	"fmt"
	"github.com/stratosnet/sds/utils/types"
	"github.com/tendermint/tendermint/crypto"
	"os"
	"regexp"
	"sync"

	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
)

// IsPP
var IsPP = false

// IsLoginToSP
var IsLoginToSP = false

// IsActive
var IsActive = false

// IsStartMining
var IsStartMining = false

// IsAuto
var IsAuto = false

// WalletAddress
var WalletAddress string

// NetworkAddress
var NetworkAddress string

// PrivateKey
var PrivateKey crypto.PrivKey

// PPList
var PPList []*protos.PPBaseInfo

var rwmutex sync.RWMutex

var TestDownload = false

var TestUpload = false

func PublicKey() crypto.PubKey {
	return PrivateKey.PubKey()
}

func GetNetworkId() *protos.NetworkId {
	return &protos.NetworkId{
		PublicKey:      types.MustBech32ifyStPubKey(PublicKey()),
		NetworkAddress: NetworkAddress,
	}
}

func ToString(networkId *protos.NetworkId) string {
	return fmt.Sprintf("sdm://%s@%s", networkId.PublicKey, networkId.NetworkAddress)
}

func ToNetworkId(networkIdString string) *protos.NetworkId {
	networkIdPattern := regexp.MustCompile(`^sdm://(\w+)@(.+)$`)
	match := networkIdPattern.FindSubmatch([]byte(networkIdString))

	return &protos.NetworkId{
		PublicKey:      string(match[1]),
		NetworkAddress: string(match[2]),
	}
}

// GetLocalPPList
func GetLocalPPList() []*protos.PPBaseInfo {
	if len(PPList) > 0 {
		return PPList
	}
	csvFile, err := os.OpenFile(Config.PPListDir, os.O_CREATE|os.O_RDWR, 0777)
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
			pp := protos.PPBaseInfo{
				NetworkId:     ToNetworkId(item[0]),
				WalletAddress: item[1],
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
		if info.NetworkId != GetNetworkId() {
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
	if err != nil {
		utils.ErrorLog("InitPPList err", err)
		return
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)
	utils.DebugLog("PPList len", len(PPList))
	for _, post := range PPList {
		line := []string{ToString(post.NetworkId), post.WalletAddress}
		err = writer.Write(line)
		if err != nil {
			utils.ErrorLog("csv line ", err)
		}
	}
	writer.Flush()
}

// DeletePPList
func DeletePPList(networkAddress string) {
	utils.DebugLog("delete PP: networkAddress=" + networkAddress)
	for i, pp := range PPList {
		if pp.NetworkId.NetworkAddress == networkAddress {
			PPList = append(PPList[:i], PPList[i+1:]...)
			savePPListLocal()
			return
		}
	}
}
