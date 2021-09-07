package peers

import (
	"fmt"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"net"
)

// GetNetworkAddress
func GetNetworkAddress() {

	if setting.Config.Internal {
		setting.NetworkAddress = getInternal() + setting.Config.Port
		setting.StreamingAddress = getInternal() + ":" + setting.Config.StreamingPort
	} else {
		setting.NetworkAddress = getExternal() + setting.Config.Port
		setting.StreamingAddress = getExternal() + ":" + setting.Config.StreamingPort
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
		fmt.Println(err)
		return ""
	}
	for _, address := range addrs {

		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				//fmt.Println(ipnet.IP.String())
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
