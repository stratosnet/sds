package msg

import "github.com/qsnetwork/qsds/msg/header"

// RelayMsgBuf application layer internal buffer for msg，
type RelayMsgBuf struct {
	// ConnID    int64
	// NetAdress string
	// WalletAddress string
	MSGHead header.MessageHead
	MSGData []byte
	Alloc   *[]byte
}
