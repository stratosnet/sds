package relay

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"
)

var Cdc *codec.Codec

func init() {
	Cdc = codec.New()
	codec.RegisterCrypto(Cdc)
	sdktypes.RegisterCodec(Cdc)
	registertypes.RegisterCodec(Cdc)
	sdstypes.RegisterCodec(Cdc)
	pottypes.RegisterCodec(Cdc)
	Cdc.Seal()
}
