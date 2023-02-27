package types

import (
	"math/big"
)

type ResourceNodeState struct {
	IsActive  uint32
	Suspended bool
	Tokens    *big.Int
}
