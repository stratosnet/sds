package utils

import (
	"github.com/cosmos/gogoproto/proto"

	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"

	fwcrypto "github.com/stratosnet/sds/framework/crypto"
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
		MerkleProofData: msg.GetMerkleProofData(),
	}

	msgBytes, err := GetVolumeReportMsgBytes(newMsg)
	if err != nil {
		return nil, err
	}

	return fwcrypto.Keccak256(msgBytes), nil
}
