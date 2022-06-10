package relay

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"
)

var Cdc *codec.LegacyAmino

func init() {
	Cdc = codec.NewLegacyAmino()
	sdktypes.RegisterLegacyAminoCodec(Cdc)
	registertypes.RegisterLegacyAminoCodec(Cdc)
	pottypes.RegisterLegacyAminoCodec(Cdc)
	sdstypes.RegisterLegacyAminoCodec(Cdc)
	cryptocodec.RegisterCrypto(Cdc)
	Cdc.Seal()
}
