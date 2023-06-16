package event

// client pp event handler
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
)

type VerifierFunc func(context.Context, header.MsgType, interface{}) error

var (
	verifierMap [header.NUMBER_MESSAGE_TYPES]VerifierFunc
)

func registerEvent(msgType header.MsgType, hf core.HandlerFunc, vf VerifierFunc) {
	core.Register(msgType, hf)
	verifierMap[msgType.Id] = vf
}
func VerifyMessage(ctx context.Context, msgType header.MsgType, target interface{}) error {
	verifier := verifierMap[msgType.Id]
	if verifier == nil {
		return nil
	}
	return verifier(ctx, msgType, target)
}

// RegisterAllEventHandler
func RegisterAllEventHandlers() {

	// pp--(req)--sp--(*rsp*)--pp
	registerEvent(header.RspGetPPList, RspGetPPList, SpRspVerifier)
	registerEvent(header.RspGetSPList, RspGetSPList, SpRspVerifier)
	registerEvent(header.RspGetPPStatus, RspGetPPStatus, SpRspVerifier)
	registerEvent(header.RspGetPPDowngradeInfo, RspGetPPDowngradeInfo, SpRspVerifier)
	registerEvent(header.RspGetWalletOz, RspGetWalletOz, SpRspVerifier)
	registerEvent(header.RspReportNodeStatus, RspReportNodeStatus, SpRspVerifier)
	registerEvent(header.RspRegister, RspRegister, SpRspVerifier)
	registerEvent(header.RspActivatePP, RspActivate, SpRspVerifier)
	registerEvent(header.RspUpdateDepositPP, RspUpdateDeposit, SpRspVerifier)
	registerEvent(header.RspStateChangePP, RspStateChange, SpRspVerifier)
	registerEvent(header.RspDeactivatePP, RspDeactivate, SpRspVerifier)
	registerEvent(header.NoticeUnbondingPP, NoticeUnbondingPP, SpRspVerifier)
	registerEvent(header.RspPrepay, RspPrepay, SpRspVerifier)
	registerEvent(header.RspMining, RspMining, SpRspVerifier)
	registerEvent(header.RspStartMaintenance, RspStartMaintenance, SpRspVerifier)
	registerEvent(header.RspStopMaintenance, RspStopMaintenance, SpRspVerifier)
	registerEvent(header.RspFindMyFileList, RspFindMyFileList, SpRspVerifier)
	registerEvent(header.RspUploadFile, RspUploadFile, RspUploadFileVerifier)
	registerEvent(header.RspReportUploadSliceResult, RspReportUploadSliceResult, SpRspVerifier)
	registerEvent(header.RspRegisterNewPP, RspRegisterNewPP, SpRspVerifier)
	registerEvent(header.RspReportDownloadResult, RspReportDownloadResult, SpRspVerifier)
	registerEvent(header.RspUploadSlicesWrong, RspUploadSlicesWrong, SpRspVerifier)
	registerEvent(header.RspReportBackupSliceResult, RspReportBackupSliceResult, SpRspVerifier)
	registerEvent(header.RspFileBackupStatus, RspBackupStatus, RspBackupStatusVerifier)
	registerEvent(header.RspFileStorageInfo, RspFileStorageInfo, RspFileStorageInfoVerifier)
	registerEvent(header.RspFileReplicaInfo, RspFileReplicaInfo, SpRspVerifier)
	registerEvent(header.RspDownloadFileWrong, RspDownloadFileWrong, RspFileStorageInfoVerifier)
	registerEvent(header.RspShareLink, RspShareLink, SpRspVerifier)
	registerEvent(header.RspShareFile, RspShareFile, SpRspVerifier)
	registerEvent(header.RspGetShareFile, RspGetShareFile, SpRspVerifier)
	registerEvent(header.RspDeleteShare, RspDeleteShare, SpRspVerifier)
	registerEvent(header.RspSpLatencyCheck, RspSpLatencyCheck, SpRspVerifier)
	registerEvent(header.RspDeleteFile, RspDeleteFile, SpRspVerifier)

	// not_pp---sp--(*rsp*)--pp
	registerEvent(header.NoticeActivatedPP, NoticeActivatedPP, SpAddressVerifier)
	registerEvent(header.NoticeUpdatedDepositPP, NoticeUpdatedDeposit, SpAddressVerifier)
	registerEvent(header.NoticeDeactivatedPP, NoticeDeactivatedPP, SpAddressVerifier)

	// pp--(*req*)--pp--(*rsp*)--pp
	registerEvent(header.ReqUploadFileSlice, ReqUploadFileSlice, RspUploadFileWithNoReqIdVerifier)
	registerEvent(header.RspUploadFileSlice, RspUploadFileSlice, PpRspVerifier)
	registerEvent(header.ReqBackupFileSlice, ReqBackupFileSlice, RspBackupStatusWithNoReqIdVerifier)
	registerEvent(header.RspBackupFileSlice, RspBackupFileSlice, PpRspVerifier)
	registerEvent(header.ReqDownloadSlice, ReqDownloadSlice, RspFileStorageInfoWithNoReqIdVerifier)
	registerEvent(header.RspDownloadSlice, RspDownloadSlice, PpRspVerifier)
	registerEvent(header.ReqTransferDownload, ReqTransferDownload, NoticeFileSliceBackupVerifier)
	registerEvent(header.RspTransferDownload, RspTransferDownload, nil)
	registerEvent(header.ReqPpLatencyCheck, ReqPpLatencyCheck, nil)
	registerEvent(header.RspPpLatencyCheck, RspPpLatencyCheck, PpRspVerifier)

	// pp--(*msg*)--pp
	registerEvent(header.ReqClearDownloadTask, ReqClearDownloadTask, nil)
	registerEvent(header.UploadSpeedOfProgress, UploadSpeedOfProgress, nil)

	// sp--(*msg*)--pp
	registerEvent(header.NoticeFileSliceBackup, NoticeFileSliceBackup, NoticeFileSliceBackupVerifier)
	registerEvent(header.NoticeSpUnderMaintenance, NoticeSpUnderMaintenance, SpAddressVerifier)

	// pp1--(req)--pp2--(rspa)--pp1--(*rspb*)--pp2
	registerEvent(header.RspTransferDownloadResult, RspTransferDownloadResult, nil)

	// framework--(*msg*)--pp
	registerEvent(header.RspBadVersion, RspBadVersion, nil)

	// to be used
	registerEvent(header.ReqGetHDInfo, ReqGetHDInfo, nil)
	registerEvent(header.RspGetHDInfo, RspGetHDInfo, nil)

	core.RegisterTimeoutHandler(header.ReqDownloadSlice, &DownloadTimeoutHandler{})
}
