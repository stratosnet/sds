package header

/*
  To add a new message type, three steps should be followed:
    * add the message id into the const list. The value is automatically assigned.
    * add the variable for this message type, in order to keep backward compatability. Please note that the variable
      type has changed to a MsgType, instead of a string.
    * register the new message type by calling registerOneMessageType(). This function associates the variable and the
      message id, meanwhile this MsgType variable is assigned to a pointer array. The array provides the fastest search
      from a message id to MsgType.
*/

import (
	"github.com/stratosnet/sds/framework/utils"
)

const (
	MsgHeaderLen   = 21 // in bytes
	CommandTypeLen = 1
)

type MsgType struct {
	Id   uint8
	Name string
}

const (
	MSG_ID_INVALID uint8 = iota

	MSG_ID_REQ_GET_SPLIST
	MSG_ID_RSP_GET_SPLIST
	MSG_ID_REQ_GET_PPSTATUS
	MSG_ID_RSP_GET_PPSTATUS
	MSG_ID_REQ_GET_PPDOWNGRADEINFO
	MSG_ID_RSP_GET_PPDOWNGRADEINFO
	MSG_ID_REQ_GET_WALLETOZ
	MSG_ID_RSP_GET_WALLETOZ
	MSG_ID_REQ_REGISTER
	MSG_ID_RSP_REGISTER
	MSG_ID_REQ_ACTIVATE_PP
	MSG_ID_RSP_ACTIVATE_PP

	MSG_ID_NOTICE_ACTIVATED_SP

	MSG_ID_REQ_UPDATE_STAKE_PP
	MSG_ID_RSP_UPDATE_STAKE_PP
	MSG_ID_NOTICE_STATE_CHANGE_PP
	MSG_ID_REQ_STATE_CHANGE_PP
	MSG_ID_RSP_STATE_CHANGE_PP

	MSG_ID_REQ_UPDATED_STAKE_SP

	MSG_ID_REQ_DEACTIVATE_PP
	MSG_ID_RSP_DEACTIVATE_PP
	MSG_ID_NOTICE_UNBONDING_PP
	MSG_ID_NOTICE_ACTIVATE_PP

	MSG_ID_RSP_PPREGISTERED_TO_SP

	MSG_ID_REQ_PREPAY
	MSG_ID_RSP_PREPAY
	MSG_ID_REQ_PREPAID
	MSG_ID_RSP_PREPAID
	MSG_ID_REQ_MINING
	MSG_ID_RSP_MINING
	MSG_ID_REQ_START_MAINTENANCE
	MSG_ID_RSP_START_MAINTENANCE
	MSG_ID_REQ_STOP_MAINTENANCE
	MSG_ID_RSP_STOP_MAINTENANCE
	MSG_ID_REQ_UPLOAD_FILE
	MSG_ID_RSP_UPLOAD_FILE
	MSG_ID_REQ_UPLOAD_FILESLICE
	MSG_ID_RSP_UPLOAD_FILESLICE
	MSG_ID_REQ_BACKUP_FILESLICE
	MSG_ID_RSP_BACKUP_FILESLICE
	MSG_ID_REQ_UPLOAD_SLICES_WRONG
	MSG_ID_RSP_UPLOAD_SLICES_WRONG
	MSG_ID_REQ_REPORT_UPLOAD_SLICE_RESULT
	MSG_ID_RSP_REPORT_UPLOAD_SLICE_RESULT

	MSG_ID_UPLOAD_SPEED_OF_PROGRESS

	MSG_ID_REQ_FIND_MY_FILELIST
	MSG_ID_RSP_FIND_MY_FILELIST
	MSG_ID_REQ_DELETE_FILE
	MSG_ID_RSP_DELETE_FILE
	MSG_ID_REQ_GET_HDINFO
	MSG_ID_RSP_GET_HDINFO
	MSG_ID_REQ_FILE_STORAGEINFO
	MSG_ID_RSP_FILE_STORAGEINFO
	MSG_ID_REQ_DOWNLOAD_SLICE
	MSG_ID_RSP_DOWNLOAD_SLICE
	MSG_ID_REQ_REPORT_DOWNLOAD_RESULT
	MSG_ID_RSP_REPORT_DOWNLOAD_RESULT
	MSG_ID_REQ_DOWNLOAD_TASKINFO
	MSG_ID_RSP_DOWNLOAD_TASKINFO
	MSG_ID_REQ_DOWNLOAD_FILE_WRONG
	MSG_ID_RSP_DOWNLOAD_FILE_WRONG

	MSG_ID_REQ_CLEAR_DOWNLOAD_TASK

	MSG_ID_REQ_REGISTER_NEWPP
	MSG_ID_RSP_REGISTER_NEWPP

	MSG_ID_NOTICE_FILESLICE_BACKUP

	MSG_ID_REQ_TRANSFER_DOWNLOAD
	MSG_ID_RSP_TRANSFER_DOWNLOAD
	MSG_ID_REQ_TRANSFER_DOWNLOAD_WRONG
	MSG_ID_RSP_TRANSFER_DOWNLOAD_WRONG

	MSG_ID_RSP_TRANSFER_DOWNLOAD_RESULT

	MSG_ID_REQ_REPORT_BACKUP_SLICE_RESULT
	MSG_ID_RSP_REPORT_BACKUP_SLICE_RESULT
	MSG_ID_REQ_FILE_BACKUP_STATUS
	MSG_ID_RSP_FILE_BACKUP_STATUS
	MSG_ID_REQ_FILE_REPLICA_INFO
	MSG_ID_RSP_FILE_REPLICA_INFO
	MSG_ID_REQ_FILE_STATUS
	MSG_ID_RSP_FILE_STATUS
	MSG_ID_REQ_SHARELINK
	MSG_ID_RSP_SHARELINK
	MSG_ID_REQ_SHARE_FILE
	MSG_ID_RSP_SHARE_FILE
	MSG_ID_REQ_DELETE_SHARE
	MSG_ID_RSP_DELETE_SHARE
	MSG_ID_REQ_GET_SHAREFILE
	MSG_ID_RSP_GET_SHAREFILE
	MSG_ID_REQ_SP_LATENCY_CHECK
	MSG_ID_RSP_SP_LATENCY_CHECK

	MSG_ID_REQ_REPORT_NODESTATUS
	MSG_ID_RSP_REPORT_NODESTATUS
	MSG_ID_REQ_SP_STATUS
	MSG_ID_RSP_SP_STATUS
	MSG_ID_REQ_BLS_SIGNATURE
	MSG_ID_RSP_BLS_SIGNATURE

	MSG_ID_RSP_BADVERSION
	MSG_ID_NOTICE_SP_UNDERMAINTENANCE
	MSG_ID_REQ_CLEAR_EXPIRED_SHARE_LINKS
	MSG_ID_RSP_CLEAR_EXPIRED_SHARE_LINKS
	NUMBER_MESSAGE_TYPES
)

var (
	ReqGetSPList          MsgType
	RspGetSPList          MsgType
	ReqGetPPStatus        MsgType
	RspGetPPStatus        MsgType
	ReqGetPPDowngradeInfo MsgType
	RspGetPPDowngradeInfo MsgType
	ReqGetWalletOz        MsgType
	RspGetWalletOz        MsgType
	ReqRegister           MsgType
	RspRegister           MsgType
	ReqActivatePP         MsgType
	RspActivatePP         MsgType

	NoticeActivatedPP MsgType

	ReqUpdateDepositPP MsgType
	RspUpdateDepositPP MsgType

	NoticeUpdatedDepositPP MsgType

	ReqStateChangePP MsgType
	RspStateChangePP MsgType

	ReqUpdatedStakeSP MsgType

	ReqDeactivatePP MsgType
	RspDeactivatePP MsgType

	NoticeUnbondingPP   MsgType
	NoticeDeactivatedPP MsgType

	RspPPRegisteredToSP MsgType

	ReqPrepay  MsgType
	RspPrepay  MsgType
	ReqPrepaid MsgType
	RspPrepaid MsgType
	ReqMining  MsgType
	RspMining  MsgType

	ReqStartMaintenance MsgType
	RspStartMaintenance MsgType
	ReqStopMaintenance  MsgType
	RspStopMaintenance  MsgType

	// upload
	ReqUploadFile              MsgType
	RspUploadFile              MsgType
	ReqUploadFileSlice         MsgType
	RspUploadFileSlice         MsgType
	ReqBackupFileSlice         MsgType
	RspBackupFileSlice         MsgType
	ReqUploadSlicesWrong       MsgType
	RspUploadSlicesWrong       MsgType
	ReqReportUploadSliceResult MsgType
	RspReportUploadSliceResult MsgType

	UploadSpeedOfProgress MsgType

	ReqFindMyFileList MsgType
	RspFindMyFileList MsgType
	ReqDeleteFile     MsgType
	RspDeleteFile     MsgType
	ReqGetHDInfo      MsgType
	RspGetHDInfo      MsgType

	//  download
	ReqFileStorageInfo      MsgType
	RspFileStorageInfo      MsgType
	ReqDownloadSlice        MsgType
	RspDownloadSlice        MsgType
	ReqReportDownloadResult MsgType
	RspReportDownloadResult MsgType
	ReqDownloadTaskInfo     MsgType
	RspDownloadTaskInfo     MsgType
	ReqDownloadFileWrong    MsgType
	RspDownloadFileWrong    MsgType

	ReqClearDownloadTask MsgType

	// register new pp
	ReqRegisterNewPP MsgType
	RspRegisterNewPP MsgType

	// backup and transfer
	NoticeFileSliceBackup MsgType

	ReqTransferDownload      MsgType
	RspTransferDownload      MsgType
	ReqTransferDownloadWrong MsgType
	RspTransferDownloadWrong MsgType

	RspTransferDownloadResult MsgType

	ReqReportBackupSliceResult MsgType
	RspReportBackupSliceResult MsgType
	ReqFileBackupStatus        MsgType
	RspFileBackupStatus        MsgType
	ReqFileReplicaInfo         MsgType
	RspFileReplicaInfo         MsgType
	ReqFileStatus              MsgType
	RspFileStatus              MsgType
	ReqShareLink               MsgType
	RspShareLink               MsgType
	ReqShareFile               MsgType
	RspShareFile               MsgType
	ReqDeleteShare             MsgType
	RspDeleteShare             MsgType
	ReqGetShareFile            MsgType
	RspGetShareFile            MsgType
	ReqSpLatencyCheck          MsgType
	RspSpLatencyCheck          MsgType

	// report node status
	ReqReportNodeStatus MsgType
	RspReportNodeStatus MsgType
	// Check status of SP node
	ReqSpStatus     MsgType
	RspSpStatus     MsgType
	ReqBLSSignature MsgType
	RspBLSSignature MsgType

	RspBadVersion            MsgType
	NoticeSpUnderMaintenance MsgType

	ReqClearExpiredShareLinks MsgType
	RspClearExpiredShareLinks MsgType

	registeredMessages [NUMBER_MESSAGE_TYPES]*MsgType
)

// MessageHead every field in this struct shall be fixed length. Please change MsgHeaderLen when modifying this struct.
type MessageHead struct {
	Tag     int16
	Len     uint32
	DataLen uint32
	Cmd     uint8
	ReqId   int64 //8 byte
	Version uint16
}

func registerOneMessageType(msgtype *MsgType, id uint8, name string) {
	msgtype.Id = id
	msgtype.Name = name
	registeredMessages[id] = msgtype
}

func init() {
	registerOneMessageType(&ReqGetSPList, MSG_ID_REQ_GET_SPLIST, "ReqGSPL")                   // request to get sp list
	registerOneMessageType(&RspGetSPList, MSG_ID_RSP_GET_SPLIST, "RspGSPL")                   // response to get sp list
	registerOneMessageType(&ReqGetPPStatus, MSG_ID_REQ_GET_PPSTATUS, "ReqGPPS")               // request to get pp status
	registerOneMessageType(&RspGetPPStatus, MSG_ID_RSP_GET_PPSTATUS, "RspGPPS")               // response to get pp status
	registerOneMessageType(&ReqGetPPDowngradeInfo, MSG_ID_REQ_GET_PPDOWNGRADEINFO, "ReqGPPD") // request to get pp downgrade information
	registerOneMessageType(&RspGetPPDowngradeInfo, MSG_ID_RSP_GET_PPDOWNGRADEINFO, "RspGPPD") // response to get pp downgrade information
	registerOneMessageType(&ReqGetWalletOz, MSG_ID_REQ_GET_WALLETOZ, "ReqGOz")                // request to get wallet ozone
	registerOneMessageType(&RspGetWalletOz, MSG_ID_RSP_GET_WALLETOZ, "RspGOz")                // response to get wallet ozone
	registerOneMessageType(&ReqRegister, MSG_ID_REQ_REGISTER, "ReqReg")                       // request to register
	registerOneMessageType(&RspRegister, MSG_ID_RSP_REGISTER, "RspReg")                       // response to register
	registerOneMessageType(&ReqActivatePP, MSG_ID_REQ_ACTIVATE_PP, "ReqActvp")                // request to activate a PP node
	registerOneMessageType(&RspActivatePP, MSG_ID_RSP_ACTIVATE_PP, "RspActvp")                // response to activate a PP node

	registerOneMessageType(&NoticeActivatedPP, MSG_ID_NOTICE_ACTIVATED_SP, "NotActdp") // notice when a PP node was successfully activated

	registerOneMessageType(&ReqUpdateDepositPP, MSG_ID_REQ_UPDATE_STAKE_PP, "ReqUpp") // request to update stake for a PP node
	registerOneMessageType(&RspUpdateDepositPP, MSG_ID_RSP_UPDATE_STAKE_PP, "RspUpp") // response to update stake for a PP node

	registerOneMessageType(&NoticeUpdatedDepositPP, MSG_ID_NOTICE_STATE_CHANGE_PP, "NotUptdp") // notice when a PP node's stake  was successfully updated

	registerOneMessageType(&ReqStateChangePP, MSG_ID_REQ_STATE_CHANGE_PP, "ReqSCpp")
	registerOneMessageType(&RspStateChangePP, MSG_ID_RSP_STATE_CHANGE_PP, "RspSCpp")

	registerOneMessageType(&ReqUpdatedStakeSP, MSG_ID_REQ_UPDATED_STAKE_SP, "ReqUptds") // request when a SP node's stake was successfully updated

	registerOneMessageType(&ReqDeactivatePP, MSG_ID_REQ_DEACTIVATE_PP, "ReqDctvp")      // request to deactivate a PP node
	registerOneMessageType(&RspDeactivatePP, MSG_ID_RSP_DEACTIVATE_PP, "RspDctvp")      // response to deactivate a PP node
	registerOneMessageType(&NoticeUnbondingPP, MSG_ID_NOTICE_UNBONDING_PP, "NotUbdp")   // notice to unbonding a PP node
	registerOneMessageType(&NoticeDeactivatedPP, MSG_ID_NOTICE_ACTIVATE_PP, "NotDctdp") // notice when a PP node was successfully deactivated

	registerOneMessageType(&RspPPRegisteredToSP, MSG_ID_RSP_PPREGISTERED_TO_SP, "Rspbdsp") // response when a PP node was successfully registered to SP

	registerOneMessageType(&ReqPrepay, MSG_ID_REQ_PREPAY, "ReqPrpay")   // request for a PP node sending a prepay transaction
	registerOneMessageType(&RspPrepay, MSG_ID_RSP_PREPAY, "RspPrpay")   // response for a PP node sending a prepay transaction
	registerOneMessageType(&ReqPrepaid, MSG_ID_REQ_PREPAID, "ReqPrpad") // request when a PP node prepay transaction was successful
	registerOneMessageType(&RspPrepaid, MSG_ID_RSP_PREPAID, "RspPrpad") // response when a PP node prepay transaction was successful
	registerOneMessageType(&ReqMining, MSG_ID_REQ_MINING, "ReqMin")     // request to mining
	registerOneMessageType(&RspMining, MSG_ID_RSP_MINING, "RspMin")     //  response to mining

	registerOneMessageType(&ReqStartMaintenance, MSG_ID_REQ_START_MAINTENANCE, "ReqStMtn")
	registerOneMessageType(&RspStartMaintenance, MSG_ID_RSP_START_MAINTENANCE, "RspStMtn")
	registerOneMessageType(&ReqStopMaintenance, MSG_ID_REQ_STOP_MAINTENANCE, "ReqSpMtn")
	registerOneMessageType(&RspStopMaintenance, MSG_ID_RSP_STOP_MAINTENANCE, "RspSpMtn")

	// upload
	registerOneMessageType(&ReqUploadFile, MSG_ID_REQ_UPLOAD_FILE, "ReqUpl")
	registerOneMessageType(&RspUploadFile, MSG_ID_RSP_UPLOAD_FILE, "RspUpl")
	registerOneMessageType(&ReqUploadFileSlice, MSG_ID_REQ_UPLOAD_FILESLICE, "ReqUpLFS")
	registerOneMessageType(&RspUploadFileSlice, MSG_ID_RSP_UPLOAD_FILESLICE, "RspUpLFS")
	registerOneMessageType(&ReqBackupFileSlice, MSG_ID_REQ_BACKUP_FILESLICE, "ReqBULFS")
	registerOneMessageType(&RspBackupFileSlice, MSG_ID_RSP_BACKUP_FILESLICE, "RspBULFS")
	registerOneMessageType(&ReqUploadSlicesWrong, MSG_ID_REQ_UPLOAD_SLICES_WRONG, "ReqUSW")
	registerOneMessageType(&RspUploadSlicesWrong, MSG_ID_RSP_UPLOAD_SLICES_WRONG, "RspUSW")
	registerOneMessageType(&ReqReportUploadSliceResult, MSG_ID_REQ_REPORT_UPLOAD_SLICE_RESULT, "ReqUFR")
	registerOneMessageType(&RspReportUploadSliceResult, MSG_ID_RSP_REPORT_UPLOAD_SLICE_RESULT, "RspUFR")

	registerOneMessageType(&UploadSpeedOfProgress, MSG_ID_UPLOAD_SPEED_OF_PROGRESS, "USOP")

	registerOneMessageType(&ReqFindMyFileList, MSG_ID_REQ_FIND_MY_FILELIST, "ReqFFL")
	registerOneMessageType(&RspFindMyFileList, MSG_ID_RSP_FIND_MY_FILELIST, "RspFFL")
	registerOneMessageType(&ReqDeleteFile, MSG_ID_REQ_DELETE_FILE, "ReqDF")
	registerOneMessageType(&RspDeleteFile, MSG_ID_RSP_DELETE_FILE, "RspDF")
	registerOneMessageType(&ReqGetHDInfo, MSG_ID_REQ_GET_HDINFO, "ReqHDI")
	registerOneMessageType(&RspGetHDInfo, MSG_ID_RSP_GET_HDINFO, "RspHDI")

	//  download
	registerOneMessageType(&ReqFileStorageInfo, MSG_ID_REQ_FILE_STORAGEINFO, "ReqQDLF")
	registerOneMessageType(&RspFileStorageInfo, MSG_ID_RSP_FILE_STORAGEINFO, "RspQDLF")
	registerOneMessageType(&ReqDownloadSlice, MSG_ID_REQ_DOWNLOAD_SLICE, "ReqDLFS")
	registerOneMessageType(&RspDownloadSlice, MSG_ID_RSP_DOWNLOAD_SLICE, "RspDLFS")
	registerOneMessageType(&ReqReportDownloadResult, MSG_ID_REQ_REPORT_DOWNLOAD_RESULT, "ReqDLRep") // request to download result report
	registerOneMessageType(&RspReportDownloadResult, MSG_ID_RSP_REPORT_DOWNLOAD_RESULT, "RspDLRep") // response to download result report
	registerOneMessageType(&ReqDownloadTaskInfo, MSG_ID_REQ_DOWNLOAD_TASKINFO, "ReqDLTI")
	registerOneMessageType(&RspDownloadTaskInfo, MSG_ID_RSP_DOWNLOAD_TASKINFO, "RspDLTI")
	registerOneMessageType(&ReqDownloadFileWrong, MSG_ID_REQ_DOWNLOAD_FILE_WRONG, "ReqDFW")
	registerOneMessageType(&RspDownloadFileWrong, MSG_ID_RSP_DOWNLOAD_FILE_WRONG, "RspDFW")

	registerOneMessageType(&ReqClearDownloadTask, MSG_ID_REQ_CLEAR_DOWNLOAD_TASK, "ReqCDT")

	// register new pp
	registerOneMessageType(&ReqRegisterNewPP, MSG_ID_REQ_REGISTER_NEWPP, "ReqRgNPP")
	registerOneMessageType(&RspRegisterNewPP, MSG_ID_RSP_REGISTER_NEWPP, "RspRgNPP")

	// backup and transfer
	registerOneMessageType(&NoticeFileSliceBackup, MSG_ID_NOTICE_FILESLICE_BACKUP, "NotFSB")

	registerOneMessageType(&ReqTransferDownload, MSG_ID_REQ_TRANSFER_DOWNLOAD, "ReqTdl")
	registerOneMessageType(&RspTransferDownload, MSG_ID_RSP_TRANSFER_DOWNLOAD, "RspTdl")
	registerOneMessageType(&ReqTransferDownloadWrong, MSG_ID_REQ_TRANSFER_DOWNLOAD_WRONG, "ReqTDW")
	registerOneMessageType(&RspTransferDownloadWrong, MSG_ID_RSP_TRANSFER_DOWNLOAD_WRONG, "RspTDW")

	registerOneMessageType(&RspTransferDownloadResult, MSG_ID_RSP_TRANSFER_DOWNLOAD_RESULT, "RspTdlR")

	registerOneMessageType(&ReqReportBackupSliceResult, MSG_ID_REQ_REPORT_BACKUP_SLICE_RESULT, "ReqRBSR")
	registerOneMessageType(&RspReportBackupSliceResult, MSG_ID_RSP_REPORT_BACKUP_SLICE_RESULT, "RspRBSR")
	registerOneMessageType(&ReqFileBackupStatus, MSG_ID_REQ_FILE_BACKUP_STATUS, "ReqFBSt")
	registerOneMessageType(&RspFileBackupStatus, MSG_ID_RSP_FILE_BACKUP_STATUS, "RspFBSt")
	registerOneMessageType(&ReqFileReplicaInfo, MSG_ID_REQ_FILE_REPLICA_INFO, "ReqFRpIn")
	registerOneMessageType(&RspFileReplicaInfo, MSG_ID_RSP_FILE_REPLICA_INFO, "RspFRpIn")
	registerOneMessageType(&ReqFileStatus, MSG_ID_REQ_FILE_STATUS, "ReqFStat")
	registerOneMessageType(&RspFileStatus, MSG_ID_RSP_FILE_STATUS, "RspFStat")
	registerOneMessageType(&ReqShareLink, MSG_ID_REQ_SHARELINK, "ReqSL")
	registerOneMessageType(&RspShareLink, MSG_ID_RSP_SHARELINK, "RspSL")
	registerOneMessageType(&ReqShareFile, MSG_ID_REQ_SHARE_FILE, "ReqSF")
	registerOneMessageType(&RspShareFile, MSG_ID_RSP_SHARE_FILE, "RspSF")
	registerOneMessageType(&ReqDeleteShare, MSG_ID_REQ_DELETE_SHARE, "ReqDSF")
	registerOneMessageType(&RspDeleteShare, MSG_ID_RSP_DELETE_SHARE, "RspDSF")
	registerOneMessageType(&ReqGetShareFile, MSG_ID_REQ_GET_SHAREFILE, "ReqGSF")
	registerOneMessageType(&RspGetShareFile, MSG_ID_RSP_GET_SHAREFILE, "RspGSF")

	// heartbeat
	registerOneMessageType(&ReqSpLatencyCheck, MSG_ID_REQ_SP_LATENCY_CHECK, "ReqSpLat")
	registerOneMessageType(&RspSpLatencyCheck, MSG_ID_RSP_SP_LATENCY_CHECK, "RspSpLat")

	// report node status
	registerOneMessageType(&ReqReportNodeStatus, MSG_ID_REQ_REPORT_NODESTATUS, "ReqRNS")
	registerOneMessageType(&RspReportNodeStatus, MSG_ID_RSP_REPORT_NODESTATUS, "RspRNS")
	// Check status of SP node
	registerOneMessageType(&ReqSpStatus, MSG_ID_REQ_SP_STATUS, "ReqSpSta")
	registerOneMessageType(&RspSpStatus, MSG_ID_RSP_SP_STATUS, "RspSpSta")
	registerOneMessageType(&ReqBLSSignature, MSG_ID_REQ_BLS_SIGNATURE, "ReqBLS")
	registerOneMessageType(&RspBLSSignature, MSG_ID_RSP_BLS_SIGNATURE, "RspBLS")

	registerOneMessageType(&RspBadVersion, MSG_ID_RSP_BADVERSION, "RspBdVer")
	registerOneMessageType(&NoticeSpUnderMaintenance, MSG_ID_NOTICE_SP_UNDERMAINTENANCE, "NotMtnc")

	registerOneMessageType(&ReqClearExpiredShareLinks, MSG_ID_REQ_CLEAR_EXPIRED_SHARE_LINKS, "ReqCESL")
	registerOneMessageType(&RspClearExpiredShareLinks, MSG_ID_RSP_CLEAR_EXPIRED_SHARE_LINKS, "RspCESL")
}

func GetMsgTypeFromId(id uint8) *MsgType {
	if id >= NUMBER_MESSAGE_TYPES {
		return nil
	}
	return registeredMessages[id]
}

func GetReqIdFromRspId(reqId uint8) uint8 {
	switch reqId {
	case MSG_ID_RSP_GET_SPLIST:
		return MSG_ID_REQ_GET_SPLIST
	case MSG_ID_RSP_GET_PPSTATUS:
		return MSG_ID_REQ_GET_PPSTATUS
	case MSG_ID_RSP_GET_PPDOWNGRADEINFO:
		return MSG_ID_REQ_GET_PPDOWNGRADEINFO
	case MSG_ID_RSP_GET_WALLETOZ:
		return MSG_ID_REQ_GET_WALLETOZ
	case MSG_ID_RSP_REGISTER:
		return MSG_ID_REQ_REGISTER
	case MSG_ID_RSP_ACTIVATE_PP:
		return MSG_ID_REQ_ACTIVATE_PP
	case MSG_ID_RSP_UPDATE_STAKE_PP:
		return MSG_ID_REQ_UPDATE_STAKE_PP
	case MSG_ID_RSP_STATE_CHANGE_PP:
		return MSG_ID_REQ_STATE_CHANGE_PP
	case MSG_ID_RSP_DEACTIVATE_PP:
		return MSG_ID_REQ_DEACTIVATE_PP
	case MSG_ID_RSP_PREPAY:
		return MSG_ID_REQ_PREPAY
	case MSG_ID_RSP_PREPAID:
		return MSG_ID_REQ_PREPAID
	case MSG_ID_RSP_MINING:
		return MSG_ID_REQ_MINING
	case MSG_ID_RSP_START_MAINTENANCE:
		return MSG_ID_REQ_START_MAINTENANCE
	case MSG_ID_RSP_STOP_MAINTENANCE:
		return MSG_ID_REQ_STOP_MAINTENANCE
	case MSG_ID_RSP_UPLOAD_FILE:
		return MSG_ID_REQ_UPLOAD_FILE
	case MSG_ID_RSP_UPLOAD_FILESLICE:
		return MSG_ID_REQ_UPLOAD_FILESLICE
	case MSG_ID_RSP_BACKUP_FILESLICE:
		return MSG_ID_REQ_BACKUP_FILESLICE
	case MSG_ID_RSP_UPLOAD_SLICES_WRONG:
		return MSG_ID_REQ_UPLOAD_SLICES_WRONG
	case MSG_ID_RSP_REPORT_UPLOAD_SLICE_RESULT:
		return MSG_ID_REQ_REPORT_UPLOAD_SLICE_RESULT
	case MSG_ID_RSP_FIND_MY_FILELIST:
		return MSG_ID_REQ_FIND_MY_FILELIST
	case MSG_ID_RSP_DELETE_FILE:
		return MSG_ID_REQ_DELETE_FILE
	case MSG_ID_RSP_GET_HDINFO:
		return MSG_ID_REQ_GET_HDINFO
	case MSG_ID_RSP_FILE_STORAGEINFO:
		return MSG_ID_REQ_FILE_STORAGEINFO
	case MSG_ID_RSP_DOWNLOAD_SLICE:
		return MSG_ID_REQ_DOWNLOAD_SLICE
	case MSG_ID_RSP_REPORT_DOWNLOAD_RESULT:
		return MSG_ID_REQ_REPORT_DOWNLOAD_RESULT
	case MSG_ID_RSP_DOWNLOAD_TASKINFO:
		return MSG_ID_REQ_DOWNLOAD_TASKINFO
	case MSG_ID_RSP_DOWNLOAD_FILE_WRONG:
		return MSG_ID_REQ_DOWNLOAD_FILE_WRONG
	case MSG_ID_RSP_REGISTER_NEWPP:
		return MSG_ID_REQ_REGISTER_NEWPP
	case MSG_ID_RSP_TRANSFER_DOWNLOAD:
		return MSG_ID_REQ_TRANSFER_DOWNLOAD
	case MSG_ID_RSP_TRANSFER_DOWNLOAD_WRONG:
		return MSG_ID_REQ_TRANSFER_DOWNLOAD_WRONG
	case MSG_ID_RSP_REPORT_BACKUP_SLICE_RESULT:
		return MSG_ID_REQ_REPORT_BACKUP_SLICE_RESULT
	case MSG_ID_RSP_FILE_BACKUP_STATUS:
		return MSG_ID_REQ_FILE_BACKUP_STATUS
	case MSG_ID_RSP_FILE_REPLICA_INFO:
		return MSG_ID_REQ_FILE_REPLICA_INFO
	case MSG_ID_RSP_FILE_STATUS:
		return MSG_ID_REQ_FILE_STATUS
	case MSG_ID_RSP_SHARELINK:
		return MSG_ID_REQ_SHARELINK
	case MSG_ID_RSP_SHARE_FILE:
		return MSG_ID_REQ_SHARE_FILE
	case MSG_ID_RSP_DELETE_SHARE:
		return MSG_ID_REQ_DELETE_SHARE
	case MSG_ID_RSP_GET_SHAREFILE:
		return MSG_ID_REQ_GET_SHAREFILE
	case MSG_ID_RSP_SP_LATENCY_CHECK:
		return MSG_ID_REQ_SP_LATENCY_CHECK
	case MSG_ID_RSP_REPORT_NODESTATUS:
		return MSG_ID_REQ_REPORT_NODESTATUS
	case MSG_ID_RSP_SP_STATUS:
		return MSG_ID_REQ_SP_STATUS
	case MSG_ID_RSP_BLS_SIGNATURE:
		return MSG_ID_REQ_BLS_SIGNATURE
	case MSG_ID_RSP_CLEAR_EXPIRED_SHARE_LINKS:
		return MSG_ID_REQ_CLEAR_EXPIRED_SHARE_LINKS
	default:
		return MSG_ID_INVALID
	}
}

func MakeMessageHeader(tag int16, version uint16, length uint32, cmd MsgType) MessageHead {
	return MessageHead{
		Tag:     tag,
		Len:     length,
		Cmd:     cmd.Id,
		Version: version,
	}
}

func CopyMessageHeader(mh MessageHead) MessageHead {
	return MessageHead{
		Tag:     mh.Tag,
		Len:     mh.Len,
		DataLen: mh.DataLen,
		Cmd:     mh.Cmd,
		ReqId:   mh.ReqId,
		Version: mh.Version,
	}

}

func (h *MessageHead) Encode(data []byte) int {
	var i = 0
	i += copy(data[i:], utils.Int16ToBytes(h.Tag))
	i += copy(data[i:], utils.Uint32ToBytes(h.Len))
	i += copy(data[i:], utils.Uint32ToBytes(h.DataLen))
	i += copy(data[i:], utils.Uint8ToBytes(h.Cmd))
	i += copy(data[i:], utils.Uint64ToBytes(uint64(h.ReqId)))
	i += copy(data[i:], utils.Uint16ToBytes(h.Version))
	return i
}

func (h *MessageHead) Decode(packet []byte) {
	var i = 0
	h.Tag = utils.BytesToInt16(packet[i : i+utils.SIZE_OF_INT16])
	i += utils.SIZE_OF_INT16
	h.Len = utils.BytesToUInt32(packet[i : i+utils.SIZE_OF_UINT32])
	i += utils.SIZE_OF_UINT32
	h.DataLen = utils.BytesToUInt32(packet[i : i+utils.SIZE_OF_UINT32])
	i += utils.SIZE_OF_UINT32
	h.Cmd = packet[i]
	i += utils.SIZE_OF_UINT8
	h.ReqId = int64(utils.BytesToUInt64(packet[i : i+utils.SIZE_OF_UINT64]))
	i += utils.SIZE_OF_UINT64
	h.Version = utils.BytesToUint16(packet[i : i+utils.SIZE_OF_UINT16])
}
