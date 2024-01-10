package types

import (
	"math/big"
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

type TxFee struct {
	Fee      Coin
	Gas      uint64
	Simulate bool
}
