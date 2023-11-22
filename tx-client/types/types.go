package types

import (
	"math/big"

	"github.com/cosmos/gogoproto/proto"
	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
)

type NodeType uint32

const (
	STORAGE     NodeType = 4
	DATABASE    NodeType = 2
	COMPUTATION NodeType = 1

	PP_INACTIVE  uint32 = iota
	PP_ACTIVE           = 1
	PP_UNBONDING        = 2

	SignatureSecp256k1 = iota
	SignatureEd25519
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
