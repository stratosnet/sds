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
	registerEvent(header.RspGetPPList, RspGetPPList, SpP2pAddressVerifier)
	registerEvent(header.RspGetSPList, RspGetSPList, SpP2pAddressVerifier)
	registerEvent(header.RspGetPPStatus, RspGetPPStatus, SpP2pAddressVerifier)
	registerEvent(header.RspGetPPDowngradeInfo, RspGetPPDowngradeInfo, SpP2pAddressVerifier)
	registerEvent(header.RspGetWalletOz, RspGetWalletOz, SpP2pAddressVerifier)
	registerEvent(header.RspReportNodeStatus, RspReportNodeStatus, SpP2pAddressVerifier)
	registerEvent(header.RspRegister, RspRegister, SpP2pAddressVerifier)
	registerEvent(header.ReqRegister, ReqRegister, nil)
	registerEvent(header.RspActivatePP, RspActivate, SpP2pAddressVerifier)
	registerEvent(header.RspActivatedPP, RspActivated, SpP2pAddressVerifier)
	registerEvent(header.RspUpdateStakePP, RspUpdateStake, SpP2pAddressVerifier)
	registerEvent(header.RspUpdatedStakePP, RspUpdatedStake, SpP2pAddressVerifier)
	registerEvent(header.RspDeactivatePP, RspDeactivate, SpP2pAddressVerifier)
	registerEvent(header.RspDeactivatedPP, RspDeactivated, SpP2pAddressVerifier)
	registerEvent(header.RspPrepay, RspPrepay, SpP2pAddressVerifier)
	registerEvent(header.RspPrepaid, RspPrepaid, nil)
	registerEvent(header.RspMining, RspMining, SpP2pAddressVerifier)
	registerEvent(header.RspStartMaintenance, RspStartMaintenance, SpP2pAddressVerifier)
	registerEvent(header.RspStopMaintenance, RspStopMaintenance, SpP2pAddressVerifier)
	registerEvent(header.RspFindMyFileList, RspFindMyFileList, SpP2pAddressVerifier)
	registerEvent(header.ReqFindMyFileList, ReqFindMyFileList, nil)
	registerEvent(header.ReqUploadFileSlice, ReqUploadFileSlice, RspUploadFileVerifier)
	registerEvent(header.RspUploadFile, RspUploadFile, RspUploadFileVerifier)
	registerEvent(header.ReqBackupFileSlice, ReqBackupFileSlice, RspBackupStatusVerifier)
	registerEvent(header.RspBackupFileSlice, RspBackupFileSlice, nil)
	registerEvent(header.RspUploadFileSlice, RspUploadFileSlice, nil)
	registerEvent(header.RspUploadSlicesWrong, RspUploadSlicesWrong, SpP2pAddressVerifier)
	registerEvent(header.RspReportUploadSliceResult, RspReportUploadSliceResult, SpP2pAddressVerifier)
	registerEvent(header.ReqFileStorageInfo, ReqFileStorageInfo, nil)
	registerEvent(header.ReqDownloadSlice, ReqDownloadSlice, RspFileStorageInfoVerifier)
	registerEvent(header.RspDownloadSlice, RspDownloadSlice, nil)
	registerEvent(header.RspReportDownloadResult, RspReportDownloadResult, SpP2pAddressVerifier)
	registerEvent(header.RspRegisterNewPP, RspRegisterNewPP, SpP2pAddressVerifier)

	//registerEvent(header.ReqTransferNotice, ReqTransferNotice, nil)
	//registerEvent(header.RspValidateTransferCer, RspValidateTransferCer, nil)
	registerEvent(header.ReqFileSliceBackupNotice, ReqFileSliceBackupNotice, ReqFileSliceBackupNoticeVerifier)
	registerEvent(header.ReqTransferDownload, ReqTransferDownload, ReqFileSliceBackupNoticeVerifier)
	registerEvent(header.RspTransferDownload, RspTransferDownload, nil)
	registerEvent(header.RspTransferDownloadResult, RspTransferDownloadResult, nil)
	registerEvent(header.RspReportBackupSliceResult, RspReportBackupSliceResult, SpP2pAddressVerifier)
	registerEvent(header.RspFileBackupStatus, RspBackupStatus, RspBackupStatusVerifier)
	//registerEvent(header.RspReportTransferResult, RspReportTransferResult, nil)

	registerEvent(header.RspFileStorageInfo, RspFileStorageInfo, RspFileStorageInfoVerifier)
	registerEvent(header.RspFileReplicaInfo, RspFileReplicaInfo, SpP2pAddressVerifier)
	registerEvent(header.RspDownloadFileWrong, RspDownloadFileWrong, RspFileStorageInfoVerifier)
	registerEvent(header.ReqClearDownloadTask, ReqClearDownloadTask, nil)
	registerEvent(header.ReqGetHDInfo, ReqGetHDInfo, nil)
	registerEvent(header.RspGetHDInfo, RspGetHDInfo, nil)
	registerEvent(header.ReqDeleteSlice, ReqDeleteSlice, nil)
	registerEvent(header.RspDeleteSlice, RspDeleteSlice, nil)
	registerEvent(header.UploadSpeedOfProgress, UploadSpeedOfProgress, nil)
	registerEvent(header.ReqShareLink, ReqShareLink, nil)
	registerEvent(header.RspShareLink, RspShareLink, SpP2pAddressVerifier)
	registerEvent(header.ReqShareFile, ReqShareFile, nil)
	registerEvent(header.RspShareFile, RspShareFile, SpP2pAddressVerifier)
	registerEvent(header.ReqDeleteShare, ReqDeleteShare, nil)
	registerEvent(header.RspDeleteShare, RspDeleteShare, SpP2pAddressVerifier)
	registerEvent(header.ReqGetShareFile, ReqGetShareFile, nil)
	registerEvent(header.RspGetShareFile, RspGetShareFile, SpP2pAddressVerifier)
	registerEvent(header.ReqSpLatencyCheck, ReqSpLatencyCheck, nil)
	registerEvent(header.ReqLatencyCheck, ReqLatencyCheckToPp, nil)
	registerEvent(header.RspLatencyCheck, RspLatencyCheck, nil)
	registerEvent(header.ReqDeleteFile, ReqDeleteFile, nil)
	registerEvent(header.RspDeleteFile, RspDeleteFile, SpP2pAddressVerifier)
	registerEvent(header.RspBadVersion, RspBadVersion, nil)
	registerEvent(header.RspSpUnderMaintenance, RspSpUnderMaintenance, nil)

	core.RegisterTimeoutHandler(header.ReqDownloadSlice, &DownloadTimeoutHandler{})
}
