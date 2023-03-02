package relay

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	stratoscodec "github.com/stratosnet/stratos-chain/crypto/codec"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"
)

var ProtoCdc *codec.ProtoCodec
var Ir codectypes.InterfaceRegistry

func init() {
	Ir = codectypes.NewInterfaceRegistry()
	ProtoCdc = codec.NewProtoCodec(Ir)
	registertypes.RegisterInterfaces(Ir)
	pottypes.RegisterInterfaces(Ir)
	sdstypes.RegisterInterfaces(Ir)
	authtypes.RegisterInterfaces(Ir)
	cryptocodec.RegisterInterfaces(Ir)
	stratoscodec.RegisterInterfaces(Ir)
}
