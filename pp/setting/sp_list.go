package setting

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/stratosnet/framework/utils"
	"github.com/stratosnet/sds-api/protos"
)

var SPMap = &sync.Map{}

type SPBaseInfo struct {
	P2PAddress     string `toml:"p2p_address" json:"p2p_address"`
	P2PPublicKey   string `toml:"p2p_public_key" json:"p2p_public_key"`
	NetworkAddress string `toml:"network_address" json:"network_address"`
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

func InitializeSPMap() error {
	bytes, err := os.ReadFile(getSPListPath())
	spList := &SPList{}
	if err == nil {
		err = json.Unmarshal(bytes, spList)
	}

	if err != nil || len(spList.SPs) == 0 {
		seedNode := Config.Node.Connectivity.SeedMetaNode
		if seedNode.P2PAddress == "" {
			return errors.New("invalid node.connectivity.seed_meta_node.p2p_address config")
		}
		if seedNode.P2PPublicKey == "" {
			return errors.New("invalid node.connectivity.seed_meta_node.p2p_public_key config")
		}
		if seedNode.NetworkAddress == "" {
			return errors.New("invalid node.connectivity.seed_meta_node.network_address config")
		}

		SPMap.Store(seedNode.P2PAddress, seedNode)
		utils.ErrorLogf("Cannot load sp-list or the list is empty. Initializing from the seed meta node: %v", err)
		return nil
	}

	for _, sp := range spList.SPs {
		SPMap.Store(sp.P2PAddress, sp)
	}
	return nil
}
