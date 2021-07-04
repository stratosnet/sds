package msg

import "github.com/stratosnet/sds/msg/header"

// RelayMsgBuf application layer internal buffer for msg，
type RelayMsgBuf struct {
	// ConnID    int64
	// NetAdress string
	// P2PAddress string
	MSGHead header.MessageHead
	MSGData []byte
	Alloc   *[]byte
}
