package peers

import (
	"net"

	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// GetNetworkAddress
func GetNetworkAddress() {

	if setting.Config.Internal {
		setting.NetworkAddress = getInternal() + ":" + setting.Config.Port
		setting.RestAddress = getInternal() + ":" + setting.Config.RestPort
	} else {
		setting.NetworkAddress = getExternal() + ":" + setting.Config.Port
		setting.RestAddress = getExternal() + ":" + setting.Config.RestPort
	}
	// utils.Log("setting.NetworkAddress", setting.NetworkAddress)

}

func getExternal() string {
	utils.Log("setting.NetworkAddress", setting.Config.NetworkAddress)
	return setting.Config.NetworkAddress
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
