package setting

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
)

var SPMap = &sync.Map{}

type SPBaseInfo struct {
	P2PAddress     string `json:"p2p_address"`
	P2PPublicKey   string `json:"p2p_public_key"`
	NetworkAddress string `json:"network_address"`
}

type SPList struct {
	SPs []SPBaseInfo `json:"sp_list"`
}

func SaveSPMapToFile() error {
	list := SPList{}
	SPMap.Range(func(k, v interface{}) bool {
		sp := v.(SPBaseInfo)
		list.SPs = append(list.SPs, sp)
		return true
	})

	bytes, err := json.Marshal(&list)
	if err != nil {
		return err
	}

	return os.WriteFile(getSPListPath(), bytes, 0644)
}

func UpdateSpMap(lst []*protos.SPBaseInfo) {
	SPMap = &sync.Map{}
	for _, spInList := range lst {
		spInMap := SPBaseInfo{
			P2PAddress:     spInList.P2PAddress,
			P2PPublicKey:   spInList.P2PPubKey,
			NetworkAddress: spInList.NetworkAddress,
		}
		SPMap.Store(spInList.P2PAddress, spInMap)
	}
}

func GetSPList() []SPBaseInfo {
	var list []SPBaseInfo
	SPMap.Range(func(key, value any) bool {
		sp := value.(SPBaseInfo)
		list = append(list, sp)
		return true
	})
	return list
}

func getSPListPath() string {
	return filepath.Join(Config.Home.PeersPath, "sp-list.json")
}

func initializeSPMap() {
	bytes, err := os.ReadFile(getSPListPath())
	spList := &SPList{}
	if err == nil {
		err = json.Unmarshal(bytes, spList)
	}

	if err != nil {
		SPMap.Store("unknown", SPBaseInfo{NetworkAddress: Config.Node.Connectivity.SeedMetaNode})
		utils.ErrorLogf("Cannot load sp-list, initializing from the seed meta node: %v", err)
		return
	}

	for _, sp := range spList.SPs {
		key := sp.P2PAddress
		if key == "" {
			key = "unknown"
		}
		SPMap.Store(key, sp)
	}
}
