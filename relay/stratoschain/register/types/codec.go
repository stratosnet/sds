package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// RegisterCodec registers concrete types on codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgCreateResourceNode{}, "register/MsgCreateResourceNode", nil)
	cdc.RegisterConcrete(MsgRemoveResourceNode{}, "register/MsgRemoveResourceNode", nil)
	cdc.RegisterConcrete(MsgCreateIndexingNode{}, "register/MsgCreateIndexingNode", nil)
	cdc.RegisterConcrete(MsgRemoveIndexingNode{}, "register/MsgRemoveIndexingNode", nil)
}

// ModuleCdc defines the module codec
var ModuleCdc *codec.Codec

func init() {
	ModuleCdc = codec.New()
	RegisterCodec(ModuleCdc)
	codec.RegisterCrypto(ModuleCdc)
	ModuleCdc.Seal()
}
