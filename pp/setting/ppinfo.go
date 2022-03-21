package setting

import (
	"sync"

	"github.com/stratosnet/sds/msg/protos"
	ppTypes "github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils/types"
)

var IsPP = false

var IsLoginToSP = false

var State uint32 = ppTypes.PP_INACTIVE

var IsStartMining = false // Is the node currently mining

var IsAuto = false

var WalletAddress string

// WalletPublicKey Public key in uncompressed format
var WalletPublicKey []byte

var WalletPrivateKey []byte

var NetworkAddress string

var RestAddress string

var P2PAddress string

var P2PPublicKey []byte

var P2PPrivateKey []byte

var SPMap = &sync.Map{}

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
