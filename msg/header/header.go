package header

// Author j & cc
import (
	"github.com/stratosnet/sds/utils"
)

const (
	MsgHeaderLen   = 28 // in bytes
	CommandTypeLen = 8
)

// cmd, 8 bytes string, exceeded will be truncate
const (
	ReqGetPPList          = "ReqGPPL" // request to get pp list
	RspGetPPList          = "RspGPPL" // response to get pp list
	ReqGetSPList          = "ReqGSPL" // request to get sp list
	RspGetSPList          = "RspGSPL" // response to get sp list
	ReqGetPPStatus        = "ReqGPPS" // request to get pp status
	RspGetPPStatus        = "RspGPPS" // response to get pp status
	ReqGetPPDowngradeInfo = "ReqGPPD" // request to get pp downgrade information
	RspGetPPDowngradeInfo = "RspGPPD" // response to get pp downgrade information

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

	ReqStartMaintenance = "ReqStMtn"
	RspStartMaintenance = "RspStMtn"
	ReqStopMaintenance  = "ReqSpMtn"
	RspStopMaintenance  = "RspSpMtn"

	// upload
	ReqUploadFile              = "ReqUpl"
	RspUploadFile              = "RspUpl"
	ReqUploadFileSlice         = "ReqUpLFS"
	RspUploadFileSlice         = "RspUpLFS"
	ReqBackupFileSlice         = "ReqBULFS"
	RspBackupFileSlice         = "RspBULFS"
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
	ReqClearDownloadTask    = "ReqCDT"

	// register new pp
	ReqRegisterNewPP = "ReqRgNPP"
	RspRegisterNewPP = "RspRgNPP"

	// backup and transfer
	ReqFileSliceBackupNotice   = "ReqFBNot"
	ReqTransferDownload        = "ReqTdl"
	RspTransferDownload        = "RspTdl"
	ReqTransferDownloadWrong   = "ReqTDW"
	RspTransferDownloadWrong   = "RspTDW"
	RspTransferDownloadResult  = "RspTdlR"
	ReqReportBackupSliceResult = "ReqRBSR"
	RspReportBackupSliceResult = "RspRBSR"
	ReqFileBackupStatus        = "ReqFBSt"
	RspFileBackupStatus        = "RspFBSt"
	ReqFileReplicaInfo         = "ReqFRpIn"
	RspFileReplicaInfo         = "RspFRpIn"

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
	// Check status of SP node
	ReqSpStatus = "ReqSpSta"
	RspSpStatus = "RspSpSta"

	ReqTransferBLSSignature = "ReqTrBLS"
	RspTransferBLSSignature = "RspTrBLS"

	RspBadVersion         = "RspBdVer"
	RspSpUnderMaintenance = "RspMtnc"
)

// MessageHead every field in this struct shall be fixed length. Please change MsgHeaderLen when modifying this struct.
type MessageHead struct {
	Tag     int16
	Len     uint32
	DataLen uint32
	Cmd     []byte //8 byte
	ReqId   int64  //8 byte
	Version uint16
}

func MakeMessageHeader(tag int16, version uint16, length uint32, cmd string) MessageHead {
	var cmdByte = []byte(cmd)[:CommandTypeLen]
	return MessageHead{
		Tag:     tag,
		Len:     length,
		Cmd:     cmdByte,
		Version: version,
	}
}

func CopyMessageHeader(mh MessageHead) MessageHead {
	cmdByte := make([]byte, CommandTypeLen)
	copy(cmdByte, mh.Cmd[:CommandTypeLen])
	return MessageHead{
		Tag:     mh.Tag,
		Len:     mh.Len,
		DataLen: mh.DataLen,
		Cmd:     cmdByte,
		ReqId:   mh.ReqId,
		Version: mh.Version,
	}

}

func (h *MessageHead) Encode(data []byte) int {
	var cmdByte = h.Cmd[:CommandTypeLen]
	var i = 0
	i += copy(data[i:], utils.Int16ToBytes(h.Tag))
	i += copy(data[i:], utils.Uint32ToBytes(h.Len))
	i += copy(data[i:], utils.Uint32ToBytes(h.DataLen))
	i += copy(data[i:], cmdByte)
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
	h.Cmd = packet[i : i+CommandTypeLen]
	i += len(h.Cmd)
	h.ReqId = int64(utils.BytesToUInt64(packet[i : i+utils.SIZE_OF_UINT64]))
	i += utils.SIZE_OF_UINT64
	h.Version = utils.BytesToUint16(packet[i : i+utils.SIZE_OF_UINT16])
}
