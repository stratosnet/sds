package types

import (
	"math/big"
)

const (
	PP_INACTIVE  uint32 = iota
	PP_ACTIVE           = 1
	PP_UNBONDING        = 2
)

type ResourceNodeState struct {
	IsActive  uint32
	Suspended bool
	Tokens    *big.Int
}
