package utils

import (
	"google.golang.org/protobuf/proto"

	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
)

func GetBLSSignBytes(msg *potv1.MsgVolumeReport) ([]byte, error) {
	newMsg := potv1.MsgVolumeReport{
		WalletVolumes:   msg.GetWalletVolumes(),
		Reporter:        msg.GetReporter(),
		Epoch:           msg.GetEpoch(),
		ReportReference: msg.GetReportReference(),
		ReporterOwner:   msg.GetReporterOwner(),
		BLSSignature:    &potv1.BLSSignatureInfo{},
	}
	return proto.Marshal(&newMsg)
}
