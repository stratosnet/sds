package types

import (
	"math/big"

	"github.com/cosmos/gogoproto/proto"
	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
)

const (
	SignatureSecp256k1 = 0
	SignatureEd25519   = 1
)

type ResourceNodeState struct {
	IsActive  uint32
	Suspended bool
	Tokens    *big.Int
}

type Traffic struct {
	Volume        uint64
	WalletAddress string
}

func GetVolumeReportMsgBytes(msg *potv1.MsgVolumeReport) []byte {
	bz, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return bz
}

type TxFee struct {
	Fee      Coin
	Gas      uint64
	Simulate bool
}
