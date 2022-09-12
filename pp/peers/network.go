package peers

import (
	"net"
	"time"

	"github.com/glendc/go-external-ip"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// GetNetworkAddress
func GetNetworkAddress() {
	if setting.Config.Internal {
		setting.NetworkAddress = getInternal() + ":" + setting.Config.Port
		setting.RestAddress = getInternal() + ":" + setting.Config.RestPort
	} else {
		externalIP := getExternal()
		setting.NetworkAddress = externalIP + ":" + setting.Config.Port
		setting.RestAddress = externalIP + ":" + setting.Config.RestPort
	}
	utils.Log("setting.NetworkAddress", setting.NetworkAddress)
}

func getExternal() string {
	externalIP := setting.Config.NetworkAddress
	if externalIP == "" {
		consensus := externalip.DefaultConsensus(&externalip.ConsensusConfig{Timeout: 10 * time.Second}, nil)
		ip, err := consensus.ExternalIP()
		if err != nil {
			utils.ErrorLog("Cannot fetch external IP", err.Error())
			return ""
		}
		externalIP = ip.String()
	}
	return externalIP
}

func getInternal() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		utils.ErrorLog(err)
		return ""
	}
	for _, address := range addrs {

		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				//utils.Log(ipnet.IP.String())
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
