package header

// Author j & cc
import (
	"github.com/stratosnet/sds/utils"
)

// MessageHead
type MessageHead struct {
	Tag       int16
	Len       uint32
	Cmd       []byte //8 byte
	ReqId     int64  //8 byte
	Version   uint16
	RecvStart int64 //8 byte
}

// MakeMessageHeader
func MakeMessageHeader(tag int16, version uint16, length uint32, cmd string) MessageHead {
	var cmdByte = []byte(cmd)[:8]
	return MessageHead{
		Tag:     tag,
		Len:     length,
		Cmd:     cmdByte,
		Version: version,
	}
}

// GetMessageHeader
func GetMessageHeader(tag int16, varsion uint16, length uint32, cmd string, reqId, recvStart int64, data []byte) {
	var cmdByte = []byte(cmd)[:8]
	copy(data[0:2], utils.Int16ToBytes(tag))
	copy(data[2:6], utils.Uint32ToBytes(length))
	copy(data[6:14], cmdByte)
	copy(data[14:22], utils.Uint64ToBytes(uint64(reqId)))
	copy(data[22:24], utils.Uint16ToBytes(varsion))
	copy(data[24:32], utils.Uint64ToBytes(uint64(recvStart)))
}

//cmd, 8 bytes string, exceeded will be truncate
const (
	ReqGetPPList   = "ReqGPPL" // request to get pp list
	RspGetPPList   = "RspGPPL" // response to get pp list
	ReqGetSPList   = "ReqGSPL" // request to get sp list
	RspGetSPList   = "RspGSPL" // response to get sp list
	ReqGetPPStatus = "ReqGPPS" // request to get pp status
	RspGetPPStatus = "RspGPPS" // response to get pp status

	ReqGetWalletOz = "ReqGOz" // request to get wallet ozone
	RspGetWalletOz = "RspGOz" // response to get wallet ozone

	ReqRegister = "ReqReg" // request to register
	RspRegister = "RspReg" // response to register

	ReqActivatePP  = "ReqActvp" // request to activate a PP node
	RspActivatePP  = "RspActvp" // response to activate a PP node
	ReqActivatedPP = "ReqActdp" // request when a PP node was successfully activated

	ReqActivatedSP = "ReqActds" // request when a SP node was successfully activated
	RspActivatedPP = "RspActdp" // response when a PP node was successfully activated

	ReqUpdateStakePP  = "ReqUpp"   // request to update stake for a PP node
	RspUpdateStakePP  = "RspUpp"   // response to update stake for a PP node
	ReqUpdatedStakePP = "ReqUptdp" // request when a PP node's stake was successfully updated
	ReqUpdatedStakeSP = "ReqUptds" // request when a SP node's stake was successfully updated
	RspUpdatedStakePP = "RspUptdp" // response when a PP node's stake  was successfully updated

	ReqDeactivatePP     = "ReqDctvp" // request to deactivate a PP node
	RspDeactivatePP     = "RspDctvp" // response to deactivate a PP node
	ReqUnbondingPP      = "ReqUbdp"  // request to unbonding a PP node
	RspUnbondingPP      = "RspUbdp"  // response to unbonding a PP node
	ReqDeactivatedPP    = "ReqDctdp" // request when a PP node was successfully deactivated
	RspDeactivatedPP    = "RspDctdp" // response when a PP node was successfully deactivated
	RspPPRegisteredToSP = "Rspbdsp"  // response when a PP node was successfully registered to SP

	ReqPrepay  = "ReqPrpay" // request for a PP node sending a prepay transaction
	RspPrepay  = "RspPrpay" // response for a PP node sending a prepay transaction
	ReqPrepaid = "ReqPrpad" // request when a PP node prepay transaction was successful
	RspPrepaid = "RspPrpad" // response when a PP node prepay transaction was successful

	ReqMining = "ReqMin" // request to mining
	RspMining = "RspMin" //  response to mining

	// upload
	ReqUploadFile              = "ReqUpl"
	RspUploadFile              = "RspUpl"
	ReqUploadFileSlice         = "ReqUpLFS"
	RspUploadFileSlice         = "RspUpLFS"
	ReqUploadSlicesWrong       = "ReqUSW"
	RspUploadSlicesWrong       = "RspUSW"
	ReqReportUploadSliceResult = "ReqUFR"
	RspReportUploadSliceResult = "RspUFR"
	UploadSpeedOfProgress      = "USOP"
	Uploaded                   = "Uploaded"

	ReqFindMyFileList = "ReqFFL"
	RspFindMyFileList = "RspFFL"

	ReqDeleteFile = "ReqDF"
	RspDeleteFile = "RspDF"

	ReqGetHDInfo = "ReqHDI"
	RspGetHDInfo = "RspHDI"

	ReqDeleteSlice = "ReqDLS"
	RspDeleteSlice = "RspDLS"

	//  download
	ReqFileStorageInfo      = "ReqQDLF"
	RspFileStorageInfo      = "RspQDLF"
	ReqDownloadSlice        = "ReqDLFS"
	RspDownloadSlice        = "RspDLFS"
	ReqReportDownloadResult = "ReqDLRep" // request to download result report
	RspReportDownloadResult = "RspDLRep" // response to download result report
	ReqDownloadTaskInfo     = "ReqDLTI"
	RspDownloadTaskInfo     = "RspDLTI"
	ReqDownloadFileWrong    = "ReqDFW"
	RspDownloadFileWrong    = "RspDFW"
	ReqDownloadSliceWrong   = "ReqDSW"
	RspDownloadSliceWrong   = "RspDSW"
	ReqClearDownloadTask    = "ReqCDT"

	// register new pp
	ReqRegisterNewPP = "ReqRgNPP"
	RspRegisterNewPP = "RspRgNPP"

	/* transfer commented out for backup logic redesign QB-897
	ReqTransferNotice         = "ReqTrNot"
	RspTransferNotice         = "RspTrNot"
	ReqValidateTransferCer    = "ReqVTCer" // request to validate transfer certificate PP->SP
	RspValidateTransferCer    = "RspVTCer" // response to validate transfer certificate SP->PP
	ReqReportTransferResult = "ReqTrRep"
	RspReportTransferResult = "RspTrRep"
	*/

	// backup
	ReqFileSliceBackupNotice   = "ReqFBNot"
	ReqTransferDownload        = "ReqTdl"
	RspTransferDownload        = "RspTdl"
	RspTransferDownloadResult  = "RspTdlR"
	ReqReportBackupSliceResult = "ReqRBSR"
	RspReportBackupSliceResult = "RspRBSR"
	ReqFileBackupStatus        = "ReqFBSt"
	RspFileBackupStatus        = "RspFBSt"

	//TODO change to report to SP
	ReqReportTaskBP = "ReqRTBP" // report to BP

	ReqShareLink    = "ReqSL"
	RspShareLink    = "RspSL"
	ReqShareFile    = "ReqSF"
	RspShareFile    = "RspSF"
	ReqDeleteShare  = "ReqDSF"
	RspDeleteShare  = "RspDSF"
	ReqGetShareFile = "ReqGSF"
	RspGetShareFile = "RspGSF"

	// heartbeat
	ReqLatencyCheck = "ReqLaten"
	RspLatencyCheck = "RspLaten"
	// test sp latency
	ReqSpLatencyCheck = "ReqSpLat"
	// report node status
	ReqReportNodeStatus = "ReqRNS"
	RspReportNodeStatus = "RspRNS"

	ReqTransferBLSSignature = "ReqTrBLS"
	RspTransferBLSSignature = "RspTrBLS"

	RspBadVersion = "RspBdVer"
)

func DecodeHeader(packet []byte, msgH *MessageHead) {
	msgH.Tag = utils.BytesToInt16(packet[:2])
	msgH.Len = utils.BytesToUInt32(packet[2:6])
	msgH.Cmd = packet[6:14]
	msgH.ReqId = int64(utils.BytesToUInt64(packet[14:22]))
	msgH.Version = utils.BytesToUint16(packet[22:24])
	msgH.RecvStart = utils.BytesToInt64(packet[24:])
}
