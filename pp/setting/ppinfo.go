package setting

import (
	"sync"

	"github.com/stratosnet/sds/msg/protos"
	ppTypes "github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils/types"
)

var IsPP = false

var IsLoginToSP = false

var State byte = ppTypes.PP_INACTIVE

var IsStartMining = false

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

// Peers is a list of the know PP node peers
var Peers ppTypes.PeerList

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
