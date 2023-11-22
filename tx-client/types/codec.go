package types

import (
	"github.com/tendermint/go-amino"

	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
)

var AminoCodec *amino.Codec

func init() {
	AminoCodec = amino.NewCodec()
	AminoCodec.RegisterConcrete(potv1.MsgVolumeReport{}, "pot/VolumeReportTx", nil)
}
