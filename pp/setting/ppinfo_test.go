package setting

import (
	"github.com/stratosnet/sds/msg/protos"
	"reflect"
	"testing"
)

func TestToString(t *testing.T) {
	networkIdStr := "sdm://374c345c7fafe263f8c6fbfa27953d2cf06028494c35bfaadbd4d6177209f5dd@192.168.1.100:5678"
	networkId := ToNetworkId(networkIdStr)
	if networkId.PublicKey != "374c345c7fafe263f8c6fbfa27953d2cf06028494c35bfaadbd4d6177209f5dd" {
		t.Error("parsing publicKey error.")
	}
	if networkId.NetworkAddress != "192.168.1.100:5678" {
		t.Error("parsing networkAddress error.")
	}
}

func TestToNetworkId(t *testing.T) {
	networkId := protos.NetworkId{
		PublicKey: "374c345c7fafe263f8c6fbfa27953d2cf06028494c35bfaadbd4d6177209f5dd",
		NetworkAddress: "192.168.1.100:5678",
	}
    networkIdStr := ToString(&networkId)
    if networkIdStr != "sdm://374c345c7fafe263f8c6fbfa27953d2cf06028494c35bfaadbd4d6177209f5dd@192.168.1.100:5678" {
    	t.Error("build networkId error.")
	}
}

func TestNetworkIdEquals(t *testing.T) {
	networkId1 := &protos.NetworkId{
		PublicKey: "374c345c7fafe263f8c6fbfa27953d2cf06028494c35bfaadbd4d6177209f5dd",
		NetworkAddress: "192.168.1.100:5678",
	}
	networkId2 := ToNetworkId("sdm://374c345c7fafe263f8c6fbfa27953d2cf06028494c35bfaadbd4d6177209f5dd@192.168.1.100:5678")
	if !reflect.DeepEqual(networkId1, networkId2) {
		t.Error("deep equal networkId error.")
	}
}