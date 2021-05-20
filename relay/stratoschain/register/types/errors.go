package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ErrInvalid                  = sdkerrors.Register(ModuleName, 1, "custom error message")
	ErrEmptyNetworkAddr         = sdkerrors.Register(ModuleName, 2, "missing network address")
	ErrEmptyOwnerAddr           = sdkerrors.Register(ModuleName, 3, "missing owner address")
	ErrValueNegative            = sdkerrors.Register(ModuleName, 4, "value must be positive")
	ErrEmptyDescription         = sdkerrors.Register(ModuleName, 5, "description must be not empty")
	ErrEmptyResourceNodeAddr    = sdkerrors.Register(ModuleName, 6, "missing resource node address")
	ErrEmptyIndexingNodeAddr    = sdkerrors.Register(ModuleName, 7, "missing indexing node address")
	ErrBadDenom                 = sdkerrors.Register(ModuleName, 8, "invalid coin denomination")
	ErrResourceNodePubKeyExists = sdkerrors.Register(ModuleName, 9, "resource node already exist for this pubkey; must use new resource node pubkey")
	ErrIndexingNodePubKeyExists = sdkerrors.Register(ModuleName, 10, "indexing node already exist for this pubkey; must use new indexing node pubkey")
	ErrNoResourceNodeFound      = sdkerrors.Register(ModuleName, 11, "resource node does not exist")
	ErrNoIndexingNodeFound      = sdkerrors.Register(ModuleName, 12, "indexing node does not exist")
	ErrNoOwnerAccountFound      = sdkerrors.Register(ModuleName, 13, "account of owner does not exist")
	ErrInsufficientBalance      = sdkerrors.Register(ModuleName, 14, "insufficient balance")
)
