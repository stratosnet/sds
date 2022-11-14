package setting

import (
	"net"
	"sync"
	"time"

	externalip "github.com/glendc/go-external-ip"
	"github.com/stratosnet/sds/msg/protos"
	ppTypes "github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/types"
)

var IsPP = false

var IsPPSyncedWithSP = false

var IsLoginToSP = false

var State uint32 = ppTypes.PP_INACTIVE

var OnlineTime int64 = 0

var IsStartMining = false // Is the node currently mining

var IsAuto = false

var WalletAddress string

// WalletPublicKey Public key in compressed format
var WalletPublicKey []byte

var WalletPrivateKey []byte

var NetworkAddress string

var RestAddress string

var P2PAddress string

var P2PPublicKey []byte

var P2PPrivateKey []byte

var SPMap = &sync.Map{}

var MonitorInitialToken string

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

// SetMyNetworkAddress set the PP's NetworkAddress according to the internal/external config in config file and the network config from OS
func SetMyNetworkAddress() {
	var netAddr string = ""
	if Config.Internal {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			utils.ErrorLog(err)
		}
		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					netAddr = ipnet.IP.String()
				}
			}
		}
	} else {
		netAddr = Config.NetworkAddress
		if netAddr == "" {
			consensus := externalip.DefaultConsensus(&externalip.ConsensusConfig{Timeout: 10 * time.Second}, nil)
			ip, err := consensus.ExternalIP()
			if err != nil {
				utils.ErrorLog("Cannot fetch external IP", err.Error())
			}
			netAddr = ip.String()
		}
	}

	if netAddr != "" {
		NetworkAddress = netAddr + ":" + Config.Port
		RestAddress = netAddr + ":" + Config.RestPort
	}
	utils.Log("setting.NetworkAddress", NetworkAddress)
}
