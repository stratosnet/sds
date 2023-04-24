package event

// client pp event handler
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
)

type VerifierFunc func(context.Context, interface{}) error

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
	return verifier(ctx, target)
}

// RegisterAllEventHandler
func RegisterAllEventHandlers() {
	veirfierMap = map[string]VerifierFunc{}
	registerEvent(header.RspGetPPList, RspGetPPList, nil)
	registerEvent(header.RspGetSPList, RspGetSPList, nil)
	registerEvent(header.RspGetPPStatus, RspGetPPStatus, nil)
	registerEvent(header.RspGetPPDowngradeInfo, RspGetPPDowngradeInfo, nil)
	registerEvent(header.RspGetWalletOz, RspGetWalletOz, nil)
	registerEvent(header.RspReportNodeStatus, RspReportNodeStatus, nil)
	registerEvent(header.RspRegister, RspRegister, nil)
	registerEvent(header.ReqRegister, ReqRegister, nil)
	registerEvent(header.RspActivatePP, RspActivate, nil)
	registerEvent(header.RspActivatedPP, RspActivated, nil)
	registerEvent(header.RspUpdateStakePP, RspUpdateStake, nil)
	registerEvent(header.RspUpdatedStakePP, RspUpdatedStake, nil)
	registerEvent(header.RspDeactivatePP, RspDeactivate, nil)
	registerEvent(header.RspUnbondingPP, RspUnbondingPP, nil)
	registerEvent(header.RspDeactivatedPP, RspDeactivated, nil)
	registerEvent(header.RspPrepay, RspPrepay, nil)
	registerEvent(header.RspPrepaid, RspPrepaid, nil)
	registerEvent(header.RspMining, RspMining, nil)
	registerEvent(header.RspStartMaintenance, RspStartMaintenance, nil)
	registerEvent(header.RspStopMaintenance, RspStopMaintenance, nil)
	registerEvent(header.RspFindMyFileList, RspFindMyFileList, nil)
	registerEvent(header.ReqFindMyFileList, ReqFindMyFileList, nil)
	registerEvent(header.ReqUploadFileSlice, ReqUploadFileSlice, RspUploadFileVerifier)
	registerEvent(header.RspUploadFile, RspUploadFile, RspUploadFileVerifier)
	registerEvent(header.ReqBackupFileSlice, ReqBackupFileSlice, RspBackupStatusVerifier)
	registerEvent(header.RspBackupFileSlice, RspBackupFileSlice, nil)
	registerEvent(header.RspUploadFileSlice, RspUploadFileSlice, nil)
	registerEvent(header.RspUploadSlicesWrong, RspUploadSlicesWrong, nil)
	registerEvent(header.RspReportUploadSliceResult, RspReportUploadSliceResult, nil)
	registerEvent(header.ReqFileStorageInfo, ReqFileStorageInfo, nil)
	registerEvent(header.ReqDownloadSlice, ReqDownloadSlice, RspFileStorageInfoVerifier)
	registerEvent(header.RspDownloadSlice, RspDownloadSlice, nil)
	registerEvent(header.RspReportDownloadResult, RspReportDownloadResult, nil)
	registerEvent(header.RspRegisterNewPP, RspRegisterNewPP, nil)

	//registerEvent(header.ReqTransferNotice, ReqTransferNotice, nil)
	//registerEvent(header.RspValidateTransferCer, RspValidateTransferCer, nil)
	registerEvent(header.ReqFileSliceBackupNotice, ReqFileSliceBackupNotice, ReqFileSliceBackupNoticeVerifier)
	registerEvent(header.ReqTransferDownload, ReqTransferDownload, ReqFileSliceBackupNoticeVerifier)
	registerEvent(header.RspTransferDownload, RspTransferDownload, nil)
	registerEvent(header.RspTransferDownloadResult, RspTransferDownloadResult, nil)
	registerEvent(header.RspReportBackupSliceResult, RspReportBackupSliceResult, nil)
	registerEvent(header.RspFileBackupStatus, RspBackupStatus, RspBackupStatusVerifier)
	//registerEvent(header.RspReportTransferResult, RspReportTransferResult, nil)

	registerEvent(header.RspFileStorageInfo, RspFileStorageInfo, RspFileStorageInfoVerifier)
	registerEvent(header.RspFileReplicaInfo, RspFileReplicaInfo, nil)
	registerEvent(header.RspDownloadFileWrong, RspDownloadFileWrong, RspFileStorageInfoVerifier)
	registerEvent(header.ReqClearDownloadTask, ReqClearDownloadTask, nil)
	registerEvent(header.ReqGetHDInfo, ReqGetHDInfo, nil)
	registerEvent(header.RspGetHDInfo, RspGetHDInfo, nil)
	registerEvent(header.ReqDeleteSlice, ReqDeleteSlice, nil)
	registerEvent(header.RspDeleteSlice, RspDeleteSlice, nil)
	registerEvent(header.UploadSpeedOfProgress, UploadSpeedOfProgress, nil)
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
