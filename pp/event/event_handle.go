package event

// client pp event handler
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
)

type VerifierFunc func(context.Context, string, interface{}) error

var (
	veirfierMap map[string]VerifierFunc
)

func registerEvent(cmd string, hf core.HandlerFunc, vf VerifierFunc) {
	core.Register(cmd, hf)
	veirfierMap[cmd] = vf
}
func VerifyMessage(ctx context.Context, cmd string, target interface{}) error {
	verifier, ok := veirfierMap[cmd]
	if !ok || verifier == nil {
		return nil
	}
	return verifier(ctx, cmd, target)
}

// RegisterAllEventHandler
func RegisterAllEventHandlers() {
	veirfierMap = map[string]VerifierFunc{}

	// pp--(req)--sp--(*rsp*)--pp
	registerEvent(header.RspGetPPList, RspGetPPList, SpRspVerifier)
	registerEvent(header.RspGetSPList, RspGetSPList, SpRspVerifier)
	registerEvent(header.RspGetPPStatus, RspGetPPStatus, SpRspVerifier)
	registerEvent(header.RspGetPPDowngradeInfo, RspGetPPDowngradeInfo, SpRspVerifier)
	registerEvent(header.RspGetWalletOz, RspGetWalletOz, SpRspVerifier)
	registerEvent(header.RspReportNodeStatus, RspReportNodeStatus, SpRspVerifier)
	registerEvent(header.RspRegister, RspRegister, SpRspVerifier)
	registerEvent(header.RspActivatePP, RspActivate, SpRspVerifier)
	registerEvent(header.RspUpdateStakePP, RspUpdateStake, SpRspVerifier)
	registerEvent(header.RspStateChangePP, RspStateChange, SpRspVerifier)
	registerEvent(header.RspDeactivatePP, RspDeactivate, SpRspVerifier)
	registerEvent(header.RspUnbondingPP, RspUnbondingPP, SpRspVerifier)
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
	registerEvent(header.RspLatencyCheck, RspLatencyCheck, SpRspVerifier)
	registerEvent(header.RspDeleteFile, RspDeleteFile, SpRspVerifier)

	// not_pp---sp--(*rsp*)--pp
	registerEvent(header.RspActivatedPP, RspActivated, SpAddressVerifier)
	registerEvent(header.RspUpdatedStakePP, RspUpdatedStake, SpAddressVerifier)
	registerEvent(header.RspDeactivatedPP, RspDeactivated, SpAddressVerifier)
	registerEvent(header.RspPrepaid, RspPrepaid, SpAddressVerifier)

	// pp--(*req*)--pp--(*rsp*)--pp
	registerEvent(header.ReqUploadFileSlice, ReqUploadFileSlice, RspUploadFileWithNoReqIdVerifier)
	registerEvent(header.RspUploadFileSlice, RspUploadFileSlice, PpRspVerifier)
	registerEvent(header.ReqBackupFileSlice, ReqBackupFileSlice, RspBackupStatusWithNoReqIdVerifier)
	registerEvent(header.RspBackupFileSlice, RspBackupFileSlice, PpRspVerifier)
	registerEvent(header.ReqDownloadSlice, ReqDownloadSlice, RspFileStorageInfoWithNoReqIdVerifier)
	registerEvent(header.RspDownloadSlice, RspDownloadSlice, PpRspVerifier)
	registerEvent(header.ReqTransferDownload, ReqTransferDownload, ReqFileSliceBackupNoticeVerifier)
	registerEvent(header.RspTransferDownload, RspTransferDownload, nil)
	registerEvent(header.ReqLatencyCheck, ReqLatencyCheckToPp, nil) // shared with pp-sp-pp version

	// pp--(*msg*)--pp
	registerEvent(header.ReqClearDownloadTask, ReqClearDownloadTask, nil)
	registerEvent(header.UploadSpeedOfProgress, UploadSpeedOfProgress, nil)

	// sp--(*msg*)--pp
	registerEvent(header.ReqFileSliceBackupNotice, ReqFileSliceBackupNotice, ReqFileSliceBackupNoticeVerifier)
	registerEvent(header.RspSpUnderMaintenance, RspSpUnderMaintenance, SpAddressVerifier)

	// pp1--(req)--pp2--(rspa)--pp1--(*rspb*)--pp2
	registerEvent(header.RspTransferDownloadResult, RspTransferDownloadResult, nil)

	// framework--(*msg*)--pp
	registerEvent(header.RspBadVersion, RspBadVersion, nil)

	// to be used
	registerEvent(header.ReqGetHDInfo, ReqGetHDInfo, nil)
	registerEvent(header.RspGetHDInfo, RspGetHDInfo, nil)
	registerEvent(header.ReqDeleteSlice, ReqDeleteSlice, nil)
	registerEvent(header.RspDeleteSlice, RspDeleteSlice, nil)

	// re-route
	registerEvent(header.ReqDeleteFile, ReqDeleteFile, nil)
	registerEvent(header.ReqFindMyFileList, ReqFindMyFileList, nil)
	registerEvent(header.ReqFileStorageInfo, ReqFileStorageInfo, nil)
	registerEvent(header.ReqRegister, ReqRegister, nil)
	registerEvent(header.ReqShareLink, ReqShareLink, nil)
	registerEvent(header.RspShareLink, RspShareLink, nil)
	registerEvent(header.ReqShareFile, ReqShareFile, nil)
	registerEvent(header.RspShareFile, RspShareFile, nil)
	registerEvent(header.ReqDeleteShare, ReqDeleteShare, nil)
	registerEvent(header.RspDeleteShare, RspDeleteShare, nil)
	registerEvent(header.ReqGetShareFile, ReqGetShareFile, nil)
	registerEvent(header.RspGetShareFile, RspGetShareFile, nil)
	registerEvent(header.ReqSpLatencyCheck, ReqSpLatencyCheck, nil)
	registerEvent(header.ReqLatencyCheck, ReqLatencyCheckToPp, nil)
	registerEvent(header.RspLatencyCheck, RspLatencyCheck, nil)
	registerEvent(header.ReqDeleteFile, ReqDeleteFile, nil)
	registerEvent(header.RspDeleteFile, RspDeleteFile, nil)
	registerEvent(header.RspBadVersion, RspBadVersion, nil)
	registerEvent(header.RspSpUnderMaintenance, RspSpUnderMaintenance, nil)

	core.RegisterTimeoutHandler(header.ReqDownloadSlice, &DownloadTimeoutHandler{})
}
