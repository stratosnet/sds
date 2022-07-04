package requests

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"math"
	"path"
	"reflect"
	"time"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/relay"
	"github.com/stratosnet/sds/utils"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
)

func ReqRegisterData() *protos.ReqRegister {
	return &protos.ReqRegister{
		Address: &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
			RestAddress:    setting.RestAddress,
		},
		MyAddress: &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
			RestAddress:    setting.RestAddress,
		},
		PublicKey: setting.P2PPublicKey,
	}
}

func ReqRegisterDataTR(target *protos.ReqRegister) *msg.RelayMsgBuf {
	req := target
	req.MyAddress = &protos.PPBaseInfo{
		P2PAddress:     setting.P2PAddress,
		WalletAddress:  setting.WalletAddress,
		NetworkAddress: setting.NetworkAddress,
		RestAddress:    setting.RestAddress,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeaderWithoutReqId(data, header.ReqRegister),
		MSGData: data,
	}
}

func ReqMiningData() *protos.ReqMining {
	return &protos.ReqMining{
		Address: &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
			RestAddress:    setting.RestAddress,
		},
		PublicKey: setting.P2PPublicKey,
		Sign:      setting.GetSign(setting.P2PAddress),
	}
}

func ReqGetPPlistData() *protos.ReqGetPPList {
	return &protos.ReqGetPPList{
		MyAddress: &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
			RestAddress:    setting.RestAddress,
		},
	}
}

func ReqGetSPlistData() *protos.ReqGetSPList {
	return &protos.ReqGetSPList{
		MyAddress: &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
			RestAddress:    setting.RestAddress,
		},
	}
}

func ReqGetPPStatusData(initPPList bool) *protos.ReqGetPPStatus {
	return &protos.ReqGetPPStatus{
		MyAddress: &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
			RestAddress:    setting.RestAddress,
		},
		InitPpList: initPPList,
	}
}

func ReqGetWalletOzData(walletAddr string) *protos.ReqGetWalletOz {
	return &protos.ReqGetWalletOz{
		WalletAddress: walletAddr,
	}
}

// RequestUploadFileData RequestUploadFileData, ownerWalletAddress can be either pp node's walletAddr or file owner's walletAddr
func RequestUploadFileData(paths, storagePath, reqID, ownerWalletAddress string, isCover, isVideoStream, isEncrypted bool) *protos.ReqUploadFile {
	info := file.GetFileInfo(paths)
	if info == nil {
		utils.ErrorLog("wrong filePath")
		return nil
	}
	fileName := info.Name()
	utils.Log("fileName~~~~~~~~~~~~~~~~~~~~~~~~", fileName)
	encryptionTag := ""
	if isEncrypted {
		encryptionTag = utils.GetRandomString(8)
	}
	fileHash := file.GetFileHash(paths, encryptionTag)
	utils.Log("fileHash~~~~~~~~~~~~~~~~~~~~~~", fileHash)

	p2pFileString := setting.WalletAddress + setting.P2PAddress + ownerWalletAddress + fileHash + header.ReqUploadFile

	req := &protos.ReqUploadFile{
		FileInfo: &protos.FileInfo{
			FileSize:           uint64(info.Size()),
			FileName:           fileName,
			FileHash:           fileHash,
			StoragePath:        storagePath,
			EncryptionTag:      encryptionTag,
			OwnerWalletAddress: ownerWalletAddress,
		},
		MyAddress: &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
			RestAddress:    setting.RestAddress,
		},
		Sign:          setting.GetSign(p2pFileString),
		IsCover:       isCover,
		ReqId:         reqID,
		IsVideoStream: isVideoStream,
	}
	if isCover {
		fileSuffix := path.Ext(paths)
		req.FileInfo.FileName = fileHash + fileSuffix
	}
	if isVideoStream {
		duration, err := file.GetVideoDuration(paths)
		if err != nil {
			utils.ErrorLog("Failed to get the length of the video: ", err)
			return nil
		}
		req.FileInfo.Duration = duration
	}
	p2pFileHash := []byte(p2pFileString)
	utils.DebugLogf("setting.WalletAddress + fileHash : %v", hex.EncodeToString(p2pFileHash))

	if !ed25519.Verify(setting.P2PPublicKey, p2pFileHash, req.Sign) {
		utils.ErrorLog("ed25519 verification failed")
		return nil
	}

	// info
	p := &task.UpProgress{
		Total:     info.Size(),
		HasUpload: 0,
	}
	task.UploadProgressMap.Store(fileHash, p)
	// if isCover {
	// 	os.Remove(path)
	// }
	return req
}

func RspDownloadSliceData(target *protos.ReqDownloadSlice) *protos.RspDownloadSlice {
	slice := task.GetDownloadSlice(target)
	return &protos.RspDownloadSlice{
		P2PAddress:     target.P2PAddress,
		WalletAddress:  target.WalletAddress,
		SliceInfo:      target.SliceInfo,
		FileCrc:        slice.FileCrc,
		FileHash:       target.FileHash,
		TaskId:         target.TaskId,
		Data:           slice.Data,
		SliceSize:      uint64(len(slice.Data)),
		SavePath:       target.SavePath,
		Result:         &protos.Result{State: protos.ResultState_RES_SUCCESS, Msg: ""},
		ReqId:          target.ReqId,
		IsEncrypted:    target.IsEncrypted,
		SpP2PAddress:   target.SpP2PAddress,
		IsVideoCaching: target.IsVideoCaching,
	}
}

func RspDownloadSliceDataSplit(rsp *protos.RspDownloadSlice, dataStart, dataEnd, offsetStart, offsetEnd, sliceOffsetStart, sliceOffsetEnd uint64, last bool) *protos.RspDownloadSlice {
	rspDownloadSlice := &protos.RspDownloadSlice{
		SliceInfo: &protos.SliceOffsetInfo{
			SliceHash: rsp.SliceInfo.SliceHash,
			SliceOffset: &protos.SliceOffset{
				SliceOffsetStart: offsetStart,
				SliceOffsetEnd:   offsetEnd,
			},
			EncryptedSliceOffset: &protos.SliceOffset{
				SliceOffsetStart: dataStart,
				SliceOffsetEnd:   dataEnd,
			},
		},
		FileCrc:        rsp.FileCrc,
		FileHash:       rsp.FileHash,
		Data:           rsp.Data[dataStart:],
		P2PAddress:     rsp.P2PAddress,
		WalletAddress:  rsp.WalletAddress,
		TaskId:         rsp.TaskId,
		SliceSize:      rsp.SliceSize,
		Result:         rsp.Result,
		NeedReport:     last,
		SavePath:       rsp.SavePath,
		ReqId:          rsp.ReqId,
		SpP2PAddress:   rsp.SpP2PAddress,
		IsEncrypted:    rsp.IsEncrypted,
		IsVideoCaching: rsp.IsVideoCaching,
	}

	if last {
		rspDownloadSlice.SliceInfo.SliceOffset.SliceOffsetEnd = sliceOffsetEnd
		rspDownloadSlice.SliceInfo.EncryptedSliceOffset.SliceOffsetEnd = rsp.SliceSize
	} else {
		rspDownloadSlice.Data = rsp.Data[dataStart:dataEnd]
	}

	if rsp.IsEncrypted {
		rspDownloadSlice.SliceInfo.SliceOffset = &protos.SliceOffset{
			SliceOffsetStart: sliceOffsetStart,
			SliceOffsetEnd:   sliceOffsetEnd,
		}
	}

	return rspDownloadSlice
}

func ReqUploadFileSliceData(task *task.UploadSliceTask, sign []byte) *protos.ReqUploadFileSlice {
	return &protos.ReqUploadFileSlice{
		TaskId:        task.TaskID,
		FileCrc:       task.FileCRC,
		SliceNumAddr:  task.SliceNumAddr,
		SliceInfo:     task.SliceOffsetInfo,
		Data:          task.Data,
		FileHash:      task.FileHash,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
		SliceSize:     task.SliceTotalSize,
		SpP2PAddress:  task.SpP2pAddress,
		Sign:          sign,
	}
}

func ReqReportUploadSliceResultData(target *protos.RspUploadFileSlice) *protos.ReportUploadSliceResult {

	utils.DebugLog("reqReportUploadSliceResultData____________________", target.SliceSize)
	return &protos.ReportUploadSliceResult{
		TaskId:        target.TaskId,
		SliceNumAddr:  target.SliceNumAddr,
		SliceHash:     target.SliceHash,
		IsPP:          false,
		UploadSuccess: true,
		FileHash:      target.FileHash,
		SliceSize:     target.SliceSize,
		Sign:          setting.GetSign(setting.P2PAddress + target.FileHash),
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
		SpP2PAddress:  target.SpP2PAddress,
	}
}
func ReqReportUploadSliceResultDataPP(target *protos.ReqUploadFileSlice) *protos.ReportUploadSliceResult {
	utils.DebugLog("____________________", target.SliceSize)
	return &protos.ReportUploadSliceResult{
		TaskId:        target.TaskId,
		SliceNumAddr:  target.SliceNumAddr,
		SliceHash:     target.SliceInfo.SliceHash,
		IsPP:          true,
		UploadSuccess: true,
		FileHash:      target.FileHash,
		SliceSize:     target.SliceSize,
		Sign:          setting.GetSign(setting.P2PAddress + target.FileHash),
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
		SpP2PAddress:  target.SpP2PAddress,
	}
}

func RspUploadFileSliceData(target *protos.ReqUploadFileSlice) *protos.RspUploadFileSlice {
	return &protos.RspUploadFileSlice{
		TaskId:        target.TaskId,
		FileHash:      target.FileHash,
		SliceHash:     target.SliceInfo.SliceHash,
		P2PAddress:    target.P2PAddress,
		WalletAddress: target.WalletAddress,
		SliceNumAddr:  target.SliceNumAddr,
		SliceSize:     target.SliceSize,
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		SpP2PAddress: target.SpP2PAddress,
	}
}

func ReqReportDownloadResultData(target *protos.RspDownloadSlice, isPP bool) *protos.ReqReportDownloadResult {

	utils.DebugLog("#################################################################", target.SliceInfo.SliceHash)
	repReq := &protos.ReqReportDownloadResult{
		IsPP:                    isPP,
		DownloaderP2PAddress:    target.P2PAddress,
		DownloaderWalletAddress: target.WalletAddress,
		MyP2PAddress:            setting.P2PAddress,
		MyWalletAddress:         setting.WalletAddress,
		FileHash:                target.FileHash,
		Sign:                    setting.GetSign(setting.P2PAddress + target.FileHash),
		TaskId:                  target.TaskId,
		SpP2PAddress:            target.SpP2PAddress,
	}
	if isPP {
		utils.Log("PP ReportDownloadResult ")
		if dlTask, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); ok {
			downloadTask := dlTask.(*task.DownloadTask)
			utils.DebugLog("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^downloadTask", downloadTask)
			if sInfo, ok := downloadTask.SliceInfo[target.SliceInfo.SliceHash]; ok {
				repReq.SliceInfo = sInfo
				repReq.SliceInfo.VisitResult = true
			} else {
				utils.DebugLog("ReportDownloadResult failed~~~~~~~~~~~~~~~~~~~~~~~~~~")
			}

		} else {
			repReq.SliceInfo = &protos.DownloadSliceInfo{
				SliceStorageInfo: &protos.SliceStorageInfo{
					SliceHash: target.SliceInfo.SliceHash,
					SliceSize: target.SliceSize,
				},
			}
		}
	} else {
		repReq.SliceInfo = &protos.DownloadSliceInfo{
			SliceStorageInfo: &protos.SliceStorageInfo{
				SliceHash: target.SliceInfo.SliceHash,
				SliceSize: target.SliceSize,
			},
		}
	}
	return repReq
}

func ReqDownloadSliceData(target *protos.RspFileStorageInfo, rsp *protos.DownloadSliceInfo) *protos.ReqDownloadSlice {
	return &protos.ReqDownloadSlice{
		P2PAddress:    setting.P2PAddress,
		WalletAddress: target.WalletAddress,
		FileHash:      target.FileHash,
		TaskId:        rsp.TaskId,
		SliceInfo: &protos.SliceOffsetInfo{
			SliceHash:   rsp.SliceStorageInfo.SliceHash,
			SliceOffset: rsp.SliceOffset,
		},
		SavePath:     target.SavePath,
		ReqId:        uuid.New().String(),
		IsEncrypted:  target.EncryptionTag != "",
		SliceNumber:  rsp.SliceNumber,
		Sign:         target.Sign,
		SpP2PAddress: target.SpP2PAddress,
	}
}

func ReqRegisterNewPPData() *protos.ReqRegisterNewPP {
	sysInfo := utils.GetSysInfo(setting.Config.StorehousePath)
	return &protos.ReqRegisterNewPP{
		P2PAddress:     setting.P2PAddress,
		WalletAddress:  setting.WalletAddress,
		DiskSize:       sysInfo.DiskSize,
		FreeDisk:       sysInfo.FreeDisk,
		MemorySize:     sysInfo.MemorySize,
		OsAndVer:       sysInfo.OSInfo,
		CpuInfo:        sysInfo.CPUInfo,
		MacAddress:     sysInfo.MacAddress,
		Version:        uint32(setting.Config.Version.AppVer),
		PubKey:         setting.P2PPublicKey,
		Sign:           setting.GetSign(setting.P2PAddress),
		NetworkAddress: setting.NetworkAddress,
	}
}

func ReqTransferDownloadData(notice *protos.ReqFileSliceBackupNotice, newPpP2pAddress string) *msg.RelayMsgBuf {
	protoMsg := &protos.ReqTransferDownload{
		TaskId:           notice.TaskId,
		NewPp:            &protos.PPBaseInfo{P2PAddress: newPpP2pAddress},
		OriginalPp:       notice.PpInfo,
		SliceStorageInfo: notice.SliceStorageInfo,
		SpP2PAddress:     notice.SpP2PAddress,
		FileHash:         notice.FileHash,
		SliceNum:         notice.SliceNumber,
		DeleteOrigin:     notice.DeleteOrigin,
	}
	data, err := proto.Marshal(protoMsg)
	if err != nil {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeaderWithoutReqId(data, header.ReqTransferDownload),
		MSGData: data,
	}
}

//TODO: Change to BP to SP
func ReqReportTaskBPData(taskID string, traffic uint64) *msg.RelayMsgBuf {
	utils.DebugLog("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~reqReportTaskBPDatareqReportTaskBPData  taskID ==", taskID, "traffic == ", traffic)
	sendTager := &protos.ReqReportTaskBP{
		TaskId:  taskID,
		Traffic: traffic,
		Reporter: &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
			RestAddress:    setting.RestAddress,
		},
	}
	data, err := proto.Marshal(sendTager)
	if err != nil {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeaderWithoutReqId(data, header.ReqReportTaskBP),
		MSGData: data,
	}
}

func ReqFileStorageInfoData(path, savePath, reqID, walletAddr, saveAs string, isVideoStream bool, shareRequest *protos.ReqGetShareFile) *protos.ReqFileStorageInfo {
	return &protos.ReqFileStorageInfo{
		FileIndexes: &protos.FileIndexes{
			P2PAddress:    setting.P2PAddress,
			WalletAddress: walletAddr,
			FilePath:      path,
			SavePath:      savePath,
			SaveAs:        saveAs,
		},
		Sign:          setting.GetSign(walletAddr + setting.P2PAddress + path + header.ReqFileStorageInfo),
		ReqId:         reqID,
		IsVideoStream: isVideoStream,
		ShareRequest:  shareRequest,
	}
}

func ReqDownloadFileWrongData(fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask) *protos.ReqDownloadFileWrong {
	var failedSlices []string
	var failedPPNodes []*protos.PPBaseInfo
	for sliceHash, _ := range dTask.FailedSlice {
		failedSlices = append(failedSlices, sliceHash)
	}
	for _, nodeInfo := range dTask.FailedPPNodes {
		failedPPNodes = append(failedPPNodes, nodeInfo)
	}
	return &protos.ReqDownloadFileWrong{
		FileIndexes: &protos.FileIndexes{
			P2PAddress:    fInfo.P2PAddress,
			WalletAddress: fInfo.WalletAddress,
			SavePath:      fInfo.SavePath,
		},
		FileHash:      fInfo.FileHash,
		Sign:          fInfo.Sign,
		ReqId:         fInfo.ReqId,
		IsVideoStream: fInfo.IsVideoStream,
		FailedSlices:  failedSlices,
		FailedPpNodes: failedPPNodes,
	}
}

func FindMyFileListData(fileName, dir, reqID, keyword string, fileType protos.FileSortType, isUp bool) *protos.ReqFindMyFileList {
	return &protos.ReqFindMyFileList{
		FileName:      fileName,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
		Directory:     dir,
		ReqId:         reqID,
		FileType:      fileType,
		IsUp:          isUp,
		Keyword:       keyword,
	}
}

func RspTransferDownloadResultData(taskId, sliceHash, spP2pAddress string) *protos.RspTransferDownloadResult {
	return &protos.RspTransferDownloadResult{
		TaskId: taskId,
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		SpP2PAddress: spP2pAddress,
		SliceHash:    sliceHash,
	}
}

func RspTransferDownload(data []byte, taskId, sliceHash, spP2pAddress string, offset, sliceSize uint64) *protos.RspTransferDownload {
	return &protos.RspTransferDownload{
		Data:         data,
		TaskId:       taskId,
		Offset:       offset,
		SliceSize:    sliceSize,
		SpP2PAddress: spP2pAddress,
		SliceHash:    sliceHash,
	}
}

func ReqDeleteFileData(fileHash, reqID string) *protos.ReqDeleteFile {
	return &protos.ReqDeleteFile{
		FileHash:      fileHash,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
		Sign:          setting.GetSign(setting.P2PAddress + fileHash),
		ReqId:         reqID,
	}
}

func ReqDownloadSliceWrong(taskID, sliceHash, p2pAddress, walletAddress string, wrongType protos.DownloadWrongType) *protos.ReqDownloadSliceWrong {
	return &protos.ReqDownloadSliceWrong{
		P2PAddress:    p2pAddress,
		WalletAddress: walletAddress,
		TaskId:        taskID,
		SliceHash:     sliceHash,
		Type:          wrongType,
	}
}

func RspDownloadSliceWrong(target *protos.RspDownloadSliceWrong) *msg.RelayMsgBuf {
	sendTager := &protos.ReqDownloadSlice{
		SliceInfo: &protos.SliceOffsetInfo{
			SliceHash:   target.NewSliceInfo.SliceStorageInfo.SliceHash,
			SliceOffset: target.NewSliceInfo.SliceOffset,
		},
		P2PAddress:    target.P2PAddress,
		WalletAddress: target.WalletAddress,
		TaskId:        target.TaskId,
		FileHash:      target.FileHash,
	}
	data, err := proto.Marshal(sendTager)
	if err != nil {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeaderWithoutReqId(data, header.ReqDownloadSlice),
		MSGData: data,
	}
}

func RspGetHDInfoData() *protos.RspGetHDInfo {
	rsp := &protos.RspGetHDInfo{
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
	}

	diskStats, err := utils.GetDiskUsage(setting.Config.StorehousePath)
	if err == nil {
		rsp.DiskSize = diskStats.Total
		rsp.DiskFree = diskStats.Free
	}

	return rsp
}

func RspDeleteSliceData(sliceHash, msg string, result bool) *protos.RspDeleteSlice {
	state := protos.ResultState_RES_SUCCESS
	if !result {
		state = protos.ResultState_RES_FAIL
	}
	return &protos.RspDeleteSlice{
		P2PAddress: setting.P2PAddress,
		SliceHash:  sliceHash,
		Result: &protos.Result{
			State: state,
			Msg:   msg,
		},
	}
}

func ReqShareLinkData(reqID string) *protos.ReqShareLink {
	return &protos.ReqShareLink{
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
	}
}

func ReqShareFileData(reqID, fileHash, pathHash string, isPrivate bool, shareTime int64) *protos.ReqShareFile {
	return &protos.ReqShareFile{
		FileHash:      fileHash,
		IsPrivate:     isPrivate,
		ShareTime:     shareTime,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
		PathHash:      pathHash,
		ReqId:         reqID,
	}
}

func ReqDeleteShareData(reqID, shareID string) *protos.ReqDeleteShare {
	return &protos.ReqDeleteShare{
		ReqId:         reqID,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
		ShareId:       shareID,
	}
}

func ReqGetShareFileData(keyword, sharePassword, saveAs, reqID string) *protos.ReqGetShareFile {
	return &protos.ReqGetShareFile{
		Keyword:       keyword,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
		SharePassword: sharePassword,
		SaveAs:        saveAs,
	}
}

func UploadSpeedOfProgressData(fileHash string, size uint64) *protos.UploadSpeedOfProgress {
	return &protos.UploadSpeedOfProgress{
		FileHash:  fileHash,
		SliceSize: size,
	}
}

func ReqNodeStatusData() *protos.ReqReportNodeStatus {
	// cpu total used percent
	totalPercent, _ := cpu.Percent(3*time.Second, false)
	// num of cpu cores
	coreNum, _ := cpu.Counts(false)
	var cpuPercent float64
	if len(totalPercent) == 0 {
		cpuPercent = 0
	} else {
		cpuPercent = totalPercent[0]
	}
	cpuStat := &protos.CpuStat{NumCores: int64(coreNum), TotalUsedPercent: math.Round(cpuPercent*100) / 100}

	// Memory physical + swap
	virtualMem, _ := mem.VirtualMemory()
	virtualUsedMem := virtualMem.Used
	virtualTotalMem := virtualMem.Total

	swapMemory, _ := mem.SwapMemory()
	swapUsedMem := swapMemory.Used
	swapTotalMem := swapMemory.Total
	memStat := &protos.MemoryStat{
		MemUsed: int64(virtualUsedMem), MemTotal: int64(virtualTotalMem),
		SwapMemUsed: int64(swapUsedMem), SwapMemTotal: int64(swapTotalMem),
	}

	// Disk usage statistics
	diskStat := &protos.DiskStat{}

	info, err := utils.GetDiskUsage(setting.Config.StorehousePath)
	if err == nil {
		diskStat.RootUsed = int64(info.Used)
		diskStat.RootTotal = int64(info.Total)
	} else {
		utils.ErrorLog("Can't fetch disk usage statistics", err)
	}

	// TODO Bandwidth
	bwStat := &protos.BandwidthStat{}

	req := &protos.ReqReportNodeStatus{
		P2PAddress: setting.P2PAddress,
		Cpu:        cpuStat,
		Memory:     memStat,
		Disk:       diskStat,
		Bandwidth:  bwStat,
	}
	return req
}

// PPMsgHeader
func PPMsgHeaderWithoutReqId(data []byte, head string) header.MessageHead {
	return header.MakeMessageHeader(1, uint16(setting.Config.Version.AppVer), uint32(len(data)), head, utils.ZeroId())
}

func UnmarshalData(ctx context.Context, target interface{}) bool {
	msgBuf := core.MessageFromContext(ctx)
	utils.DebugLogf("Received message type = %v msgBuf len = %v", reflect.TypeOf(target), len(msgBuf.MSGData))
	return UnmarshalMessageData(msgBuf.MSGData, target)
}

func UnmarshalMessageData(data []byte, target interface{}) bool {
	if err := proto.Unmarshal(data, target.(proto.Message)); err != nil {
		utils.ErrorLog("protobuf Unmarshal error", err)
		return false
	}
	if _, ok := reflect.TypeOf(target).Elem().FieldByName("Data"); !ok {
		utils.DebugLog("target = ", target)
	}
	return true
}

func GetReqIdFromMessage(ctx context.Context) int64 {
	msgBuf := core.MessageFromContext(ctx)
	return msgBuf.MSGHead.ReqId
}

func VerifySpSignature(spP2PAddress string, message, sign []byte) bool {
	val, ok := setting.SPMap.Load(spP2PAddress)
	if !ok {
		utils.ErrorLog("cannot find sp info by given the SP address ", spP2PAddress)
		return false
	}

	spInfo, ok := val.(setting.SPBaseInfo)
	if !ok {
		utils.ErrorLog("Fail to parse SP info ", spP2PAddress)
		return false
	}

	_, pubKeyRaw, err := bech32.DecodeAndConvert(spInfo.P2PPublicKey)
	if err != nil {
		utils.ErrorLog("Error when trying to decode P2P pubKey bech32", err)
		return false
	}

	p2pPubKey := tmed25519.PubKey{}
	err = relay.Cdc.Amino.UnmarshalBinaryBare(pubKeyRaw, &p2pPubKey)

	if err != nil {
		utils.ErrorLog("Error when trying to read P2P pubKey ed25519 binary", err)
		return false
	}

	return ed25519.Verify(p2pPubKey.Bytes(), message, sign)
}
