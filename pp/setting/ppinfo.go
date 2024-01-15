package setting

import (
	"net"
	"time"

	externalip "github.com/glendc/go-external-ip"

	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
	"github.com/stratosnet/sds/framework/utils"
	msgtypes "github.com/stratosnet/sds/sds-msg/types"
)

var IsPP = false

var IsPPSyncedWithSP = false

var State uint32 = msgtypes.PP_INACTIVE

var OnlineTime int64 = 0

var WalletAddress string

// WalletPublicKey Public key in compressed format
var WalletPublicKey fwcryptotypes.PubKey

var WalletPrivateKey fwcryptotypes.PrivKey

var NetworkAddress string

var NetworkIP net.IP

var RestAddress string

var MonitorInitialToken string

// SetMyNetworkAddress set the PP's NetworkAddress according to the internal/external config in config file and the network config from OS
func SetMyNetworkAddress() {
	var netAddr string = ""
	if Config.Node.Connectivity.Internal {
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
		netAddr = Config.Node.Connectivity.NetworkAddress
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
		NetworkAddress = netAddr + ":" + Config.Node.Connectivity.NetworkPort
		RestAddress = netAddr + ":" + Config.Streaming.RestPort
		NetworkIP = net.ParseIP(netAddr)
	}
	utils.Log("setting.NetworkAddress", NetworkAddress)
}
