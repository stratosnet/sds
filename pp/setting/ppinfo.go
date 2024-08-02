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

var BeneficiaryAddress string

var WalletAddress string

var WalletPublicKey fwcryptotypes.PubKey

var WalletPrivateKey fwcryptotypes.PrivKey

var NetworkAddress string

var NetworkIP net.IP

var RestAddress string

var MonitorInitialToken string

// SetMyNetworkAddress set the PP's NetworkAddress according to the internal/external config in config file and the network config from OS
func SetMyNetworkAddress() {
	defer func() {
		if NetworkAddress == "" {
			utils.ErrorLog("NetworkAddress is empty")
		} else {
			utils.Log("setting.NetworkAddress", NetworkAddress)
		}
	}()

	var netAddr string = ""
	if Config.Node.Connectivity.Internal {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			utils.ErrorLog(utils.FormatError(err))
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
	if netAddr == "" {
		return
	}

	NetworkIP = net.ParseIP(netAddr)
	if NetworkIP == nil {
		ipList, err := net.LookupIP(netAddr)
		if err != nil {
			utils.ErrorLog(utils.FormatError(err))
		}
		if len(ipList) == 0 {
			return
		}
		NetworkIP = ipList[0]
	}
	NetworkAddress = NetworkIP.String() + ":" + Config.Node.Connectivity.NetworkPort
	RestAddress = NetworkIP.String() + ":" + Config.Streaming.RestPort
}

func GetP2pServerPort() string {
	if Config.Node.Connectivity.LocalPort == "" {
		return Config.Node.Connectivity.NetworkPort
	}
	return Config.Node.Connectivity.LocalPort
}
