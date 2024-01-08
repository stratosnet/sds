package utils

import (
	"github.com/cosmos/gogoproto/proto"

	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
)

func GetVolumeReportMsgBytes(msg *potv1.MsgVolumeReport) ([]byte, error) {
	return proto.Marshal(msg)
}

func GetBLSSignBytes(msg *potv1.MsgVolumeReport) ([]byte, error) {
	newMsg := &potv1.MsgVolumeReport{
		WalletVolumes:   msg.GetWalletVolumes(),
		Reporter:        msg.GetReporter(),
		Epoch:           msg.GetEpoch(),
		ReportReference: msg.GetReportReference(),
		ReporterOwner:   msg.GetReporterOwner(),
		BLSSignature:    &potv1.BLSSignatureInfo{},
	}
	return GetVolumeReportMsgBytes(newMsg)
}
