package requests

import (
	"context"
	"encoding/hex"
	"math"
	"path"
	"path/filepath"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"google.golang.org/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/types/bech32"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/types"
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
		MSGHead: PPMsgHeader(data, header.ReqRegister),
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

func ReqGetWalletOzData(walletAddr, reqId string) *protos.ReqGetWalletOz {
	return &protos.ReqGetWalletOz{
		WalletAddress: walletAddr,
	}
}

// RequestUploadFile a file from an owner instead from a "path" belongs to PP's default wallet
func RequestUploadFile(fileName, fileHash string, fileSize uint64, walletAddress, walletPubkey, signature string, isEncrypted, isVideoStream bool) *protos.ReqUploadFile {
	utils.Log("fileName: ", fileName)
	encryptionTag := ""
	if isEncrypted {
		encryptionTag = utils.GetRandomString(8)
	}

	utils.Log("fileHash: ", fileHash)

	nodeSignMsg := utils.GetReqUploadFileNodeSignMessage(setting.P2PAddress, fileHash, header.ReqUploadFile)
	file.SaveRemoteFileHash(fileHash, "rpc:"+fileName, fileSize)

	// convert wallet pubkey to []byte which format is to be used in protobuf messages
	wpk, err := types.WalletPubkeyFromBech(walletPubkey)
	if err != nil {
		utils.ErrorLog("wrong wallet pubkey")
		return nil
	}
	// decode the hex encoded signature back to []byte which is used in protobuf messages
	wsig, err := hex.DecodeString(signature)
	if err != nil {
		utils.ErrorLog("wrong signature")
		return nil
	}
	req := &protos.ReqUploadFile{
		FileInfo: &protos.FileInfo{
			FileSize:           fileSize,
			FileName:           fileName,
			FileHash:           fileHash,
			StoragePath:        "rpc:" + fileName,
			EncryptionTag:      encryptionTag,
			OwnerWalletAddress: walletAddress,
		},
		MyAddress:     setting.GetPPInfo(),
		NodeSign:      types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(nodeSignMsg)),
		WalletSign:    wsig,
		WalletPubkey:  wpk.Bytes(),
		IsCover:       false,
		IsVideoStream: isVideoStream,
	}

	if isVideoStream {
		duration, err := file.GetVideoDuration(filepath.Join(setting.GetRootPath(), file.TEMP_FOLDER, fileHash, fileName))
		if err != nil {
			utils.Log("Failed to get the length of the video: ", err)
			return nil
		}
		req.FileInfo.Duration = duration
	}

	// info
	p := &task.UploadProgress{
		Total:     int64(fileSize),
		HasUpload: 0,
	}
	task.UploadProgressMap.Store(fileHash, p)
	return req
}

// RequestUploadFileData assume the PP's current wallet is the owner, otherwise RequestUploadFile() should be used instead
func RequestUploadFileData(ctx context.Context, paths, storagePath string, isCover, isVideoStream, isEncrypted bool) *protos.ReqUploadFile {
	info, err := file.GetFileInfo(paths)
	if err != nil {
		pp.ErrorLog(ctx, "wrong filePath", err.Error())
		return nil
	}
	fileName := info.Name()
	pp.Log(ctx, "fileName~~~~~~~~~~~~~~~~~~~~~~~~", fileName)
	encryptionTag := ""
	if isEncrypted {
		encryptionTag = utils.GetRandomString(8)
	}
	fileHash := file.GetFileHash(paths, encryptionTag)
	pp.Log(ctx, "fileHash~~~~~~~~~~~~~~~~~~~~~~", fileHash)

	nodeSignMsg := utils.GetReqUploadFileNodeSignMessage(setting.P2PAddress, fileHash, header.ReqUploadFile)
	walletString := utils.GetFileUploadWalletSignMessage(fileHash, setting.WalletAddress)
	wsign, err := types.BytesToAccPriveKey(setting.WalletPrivateKey).Sign([]byte(walletString))
	if err != nil {
		return nil
	}
	req := &protos.ReqUploadFile{
		FileInfo: &protos.FileInfo{
			FileSize:           uint64(info.Size()),
			FileName:           fileName,
			FileHash:           fileHash,
			StoragePath:        storagePath,
			EncryptionTag:      encryptionTag,
			OwnerWalletAddress: setting.WalletAddress,
		},
		MyAddress: &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
			RestAddress:    setting.RestAddress,
		},
		NodeSign:      types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(nodeSignMsg)),
		WalletSign:    wsign,
		WalletPubkey:  setting.WalletPublicKey,
		IsCover:       isCover,
		IsVideoStream: isVideoStream,
	}
	if isCover {
		fileSuffix := path.Ext(paths)
		req.FileInfo.FileName = fileHash + fileSuffix
	}
	if isVideoStream {
		duration, err := file.GetVideoDuration(paths)
		if err != nil {
			pp.ErrorLog(ctx, "Failed to get the length of the video: ", err)
			return nil
		}
		req.FileInfo.Duration = duration
	}

	// info
	p := &task.UploadProgress{
		Total:     info.Size(),
		HasUpload: 0,
	}
	task.UploadProgressMap.Store(fileHash, p)
	// if isCover {
	//	os.Remove(path)
	// }
	return req
}

// RequestDownloadFile the entry for rpc remote download
func RequestDownloadFile(fileHash, sdmPath, walletAddr string, reqId string, walletSign, walletPubkey []byte, shareRequest *protos.ReqGetShareFile) *protos.ReqFileStorageInfo {
	// file's request id is used for identifying the download session
	fileReqId := reqId
	if reqId == "" {
		fileReqId = uuid.New().String()
	}

	// download file uses fileHash + fileReqId as the key
	file.SaveRemoteFileHash(fileHash+fileReqId, "rpc:", 0)

	// path: mesh network address
	metrics.DownloadPerformanceLogNow(fileHash + ":SND_STORAGE_INFO_SP:")
	req := ReqFileStorageInfoData(sdmPath, "", "", walletAddr, walletSign, walletPubkey, false, shareRequest)
	return req
}

func RspDownloadSliceData(target *protos.ReqDownloadSlice) *protos.RspDownloadSlice {
	slice := task.GetDownloadSlice(target)
	return &protos.RspDownloadSlice{
		P2PAddress:        target.P2PAddress,
		WalletAddress:     target.WalletAddress,
		SliceInfo:         target.SliceInfo,
		FileCrc:           slice.FileCrc,
		FileHash:          target.FileHash,
		TaskId:            target.TaskId,
		Data:              slice.Data,
		SliceSize:         uint64(len(slice.Data)),
		SavePath:          target.SavePath,
		Result:            &protos.Result{State: protos.ResultState_RES_SUCCESS, Msg: ""},
		IsEncrypted:       target.IsEncrypted,
		SpP2PAddress:      target.SpP2PAddress,
		IsVideoCaching:    target.IsVideoCaching,
		StorageP2PAddress: target.StorageP2PAddress,
		SliceNumber:       target.SliceNumber,
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
		FileCrc:           rsp.FileCrc,
		FileHash:          rsp.FileHash,
		Data:              rsp.Data[dataStart:],
		P2PAddress:        rsp.P2PAddress,
		WalletAddress:     rsp.WalletAddress,
		TaskId:            rsp.TaskId,
		SliceSize:         rsp.SliceSize,
		Result:            rsp.Result,
		NeedReport:        last,
		SavePath:          rsp.SavePath,
		SpP2PAddress:      rsp.SpP2PAddress,
		IsEncrypted:       rsp.IsEncrypted,
		IsVideoCaching:    rsp.IsVideoCaching,
		StorageP2PAddress: rsp.StorageP2PAddress,
		SliceNumber:       rsp.SliceNumber,
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

func ReqUploadFileSliceData(task *task.UploadSliceTask, destP2pAddr string) *protos.ReqUploadFileSlice {
	msg := utils.GetReqUploadFileSlicePpNodeSignMessage(setting.P2PAddress, destP2pAddr, header.ReqUploadFileSlice)
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
		PpP2PPubkey:   setting.P2PPublicKey,
		PpNodeSign:    types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
	}
}

func RspUploadFileSliceData(target *protos.ReqUploadFileSlice) *protos.RspUploadFileSlice {
	msg := utils.GetRspUploadFileSliceNodeSignMessage(setting.P2PAddress, target.P2PAddress, header.RspUploadFileSlice)
	return &protos.RspUploadFileSlice{
		TaskId:        target.TaskId,
		FileHash:      target.FileHash,
		SliceHash:     target.SliceInfo.SliceHash,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: target.WalletAddress,
		SliceNumAddr:  target.SliceNumAddr,
		SliceSize:     target.SliceSize,
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		SpP2PAddress: target.SpP2PAddress,
		SpNodeSign:   target.SliceNumAddr.SpNodeSign,
		PpP2PPubkey:  setting.P2PPublicKey,
		PpNodeSign:   types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
	}
}

func ReqUploadSlicesWrong(uploadTask *task.UploadFileTask, spP2pAddress string, slicesToDownload []*protos.SliceHashAddr, failedSlices []bool) *protos.ReqUploadSlicesWrong {
	return &protos.ReqUploadSlicesWrong{
		FileHash:             uploadTask.FileHash,
		TaskId:               uploadTask.TaskID,
		UploadType:           uploadTask.Type,
		MyAddress:            setting.GetPPInfo(),
		SpP2PAddress:         spP2pAddress,
		ExcludedDestinations: uploadTask.GetExcludedDestinations(),
		Slices:               slicesToDownload,
		FailedSlices:         failedSlices,
	}
}

func ReqReportUploadSliceResultData(target *protos.RspUploadFileSlice, costTime int64) *protos.ReportUploadSliceResult {
	utils.DebugLog("reqReportUploadSliceResultData____________________", target.SliceSize)
	msg := utils.GetReportUploadSliceResultPpNodeSignMessage(setting.P2PAddress, target.FileHash, target.SliceHash, target.TaskId, header.ReqReportUploadSliceResult)
	return &protos.ReportUploadSliceResult{
		TaskId:             target.TaskId,
		SliceNumAddr:       target.SliceNumAddr,
		SliceHash:          target.SliceHash,
		IsPP:               false,
		UploadSuccess:      true,
		FileHash:           target.FileHash,
		SliceSize:          target.SliceSize,
		PpNodeSign:         types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
		P2PAddress:         setting.P2PAddress,
		WalletAddress:      setting.WalletAddress,
		SpP2PAddress:       target.SpP2PAddress,
		PpP2PPubkey:        setting.P2PPublicKey,
		CostTime:           costTime,
		OpponentP2PAddress: target.SliceNumAddr.PpInfo.P2PAddress,
	}
}

func ReqReportUploadSliceResultDataPP(target *protos.ReqUploadFileSlice, costTime int64) *protos.ReportUploadSliceResult {
	utils.DebugLog("____________________", target.SliceSize)
	msg := utils.GetReportUploadSliceResultPpNodeSignMessage(setting.P2PAddress, target.FileHash, target.SliceInfo.SliceHash, target.TaskId, header.ReqReportUploadSliceResult)
	return &protos.ReportUploadSliceResult{
		TaskId:             target.TaskId,
		SliceNumAddr:       target.SliceNumAddr,
		SliceHash:          target.SliceInfo.SliceHash,
		IsPP:               true,
		UploadSuccess:      true,
		FileHash:           target.FileHash,
		SliceSize:          target.SliceSize,
		PpNodeSign:         types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
		P2PAddress:         setting.P2PAddress,
		WalletAddress:      setting.WalletAddress,
		SpP2PAddress:       target.SpP2PAddress,
		PpP2PPubkey:        setting.P2PPublicKey,
		CostTime:           costTime,
		OpponentP2PAddress: target.P2PAddress,
	}
}

func ReqReportDownloadResultData(target *protos.RspDownloadSlice, costTime int64, isPP bool) *protos.ReqReportDownloadResult {
	utils.DebugLog("#################################################################", target.SliceInfo.SliceHash)
	msg := utils.GetReportDownloadSliceResultPpNodeSignMessage(setting.P2PAddress, target.SliceInfo.SliceHash, target.TaskId, header.ReqReportDownloadResult)
	repReq := &protos.ReqReportDownloadResult{
		IsPP:                 isPP,
		DownloaderP2PAddress: target.P2PAddress,
		WalletAddress:        target.WalletAddress,
		PpP2PAddress:         setting.P2PAddress,
		PpWalletAddress:      setting.WalletAddress,
		FileHash:             target.FileHash,
		NodeSign:             types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
		TaskId:               target.TaskId,
		SpP2PAddress:         target.SpP2PAddress,
		CostTime:             costTime,
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
				SliceNumber: target.SliceNumber,
				SliceStorageInfo: &protos.SliceStorageInfo{
					SliceHash: target.SliceInfo.SliceHash,
					SliceSize: target.SliceSize,
				},
			}
		}
		repReq.OpponentP2PAddress = target.P2PAddress
	} else {
		repReq.SliceInfo = &protos.DownloadSliceInfo{
			SliceNumber: target.SliceNumber,
			SliceStorageInfo: &protos.SliceStorageInfo{
				SliceHash: target.SliceInfo.SliceHash,
				SliceSize: target.SliceSize,
			},
		}
		repReq.OpponentP2PAddress = target.StorageP2PAddress
	}
	return repReq
}

func ReqReportStreamResultData(target *protos.RspDownloadSlice, isPP bool) *protos.ReqReportDownloadResult {
	utils.DebugLog("#################################################################", target.SliceInfo.SliceHash)
	msg := utils.GetReportDownloadSliceResultPpNodeSignMessage(setting.P2PAddress, target.SliceInfo.SliceHash, target.TaskId, header.ReqReportDownloadResult)
	repReq := &protos.ReqReportDownloadResult{
		IsPP:                 isPP,
		DownloaderP2PAddress: target.P2PAddress,
		WalletAddress:        target.WalletAddress,
		PpP2PAddress:         setting.P2PAddress,
		PpWalletAddress:      setting.WalletAddress,
		FileHash:             target.FileHash,
		NodeSign:             types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
		TaskId:               target.TaskId,
		SpP2PAddress:         target.SpP2PAddress,
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
		repReq.OpponentP2PAddress = target.P2PAddress
	} else {
		repReq.SliceInfo = &protos.DownloadSliceInfo{
			SliceStorageInfo: &protos.SliceStorageInfo{
				SliceHash: target.SliceInfo.SliceHash,
				SliceSize: target.SliceSize,
			},
		}
		repReq.OpponentP2PAddress = target.StorageP2PAddress
	}
	return repReq
}

func ReqDownloadSliceData(target *protos.RspFileStorageInfo, slice *protos.DownloadSliceInfo) *protos.ReqDownloadSlice {
	msg := utils.GetReqDownloadSlicePpNodeSignMessage(setting.P2PAddress, slice.StoragePpInfo.P2PAddress, slice.SliceStorageInfo.SliceHash, header.ReqDownloadSlice)
	return &protos.ReqDownloadSlice{
		P2PAddress:    setting.P2PAddress,
		WalletAddress: target.WalletAddress,
		FileHash:      target.FileHash,
		TaskId:        slice.TaskId,
		SliceInfo: &protos.SliceOffsetInfo{
			SliceHash:   slice.SliceStorageInfo.SliceHash,
			SliceOffset: slice.SliceOffset,
		},
		SavePath:          target.SavePath,
		IsEncrypted:       target.EncryptionTag != "",
		SliceNumber:       slice.SliceNumber,
		SpNodeSign:        slice.SpNodeSign,
		SpP2PAddress:      target.SpP2PAddress,
		PpNodeSign:        types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
		PpP2PPubkey:       setting.P2PPublicKey,
		StorageP2PAddress: slice.StoragePpInfo.P2PAddress,
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

func ReqTransferDownloadData(notice *protos.ReqFileSliceBackupNotice) *msg.RelayMsgBuf {
	protoMsg := &protos.ReqTransferDownload{
		TaskId:           notice.TaskId,
		NewPp:            setting.GetPPInfo(),
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
		MSGHead: PPMsgHeader(data, header.ReqTransferDownload),
		MSGData: data,
	}
}

func ReqTransferDownloadWrongData(notice *protos.ReqFileSliceBackupNotice) *protos.ReqTransferDownloadWrong {
	return &protos.ReqTransferDownloadWrong{
		TaskId:           notice.TaskId,
		NewPp:            setting.GetPPInfo(),
		OriginalPp:       notice.PpInfo,
		SliceStorageInfo: notice.SliceStorageInfo,
		FileHash:         notice.FileHash,
		Sign:             notice.Sign,
		SpP2PAddress:     notice.SpP2PAddress,
	}
}

// ReqFileStorageInfoData encode ReqFileStorageInfo message. If it's not a "share request", walletAddr should keep the same
// as the wallet from the "path".
func ReqFileStorageInfoData(path, savePath, saveAs, walletAddr string, walletSign, walletPUbkey []byte, isVideoStream bool, shareRequest *protos.ReqGetShareFile) *protos.ReqFileStorageInfo {
	msg := utils.GetReqFileStorageInfoNodeSignMessage(setting.P2PAddress, path, header.ReqFileStorageInfo)
	return &protos.ReqFileStorageInfo{
		FileIndexes: &protos.FileIndexes{
			P2PAddress:    setting.P2PAddress,
			WalletAddress: walletAddr,
			FilePath:      path,
			SavePath:      savePath,
			SaveAs:        saveAs,
		},
		NodeSign:      types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
		WalletSign:    walletSign,
		WalletPubkey:  walletPUbkey,
		IsVideoStream: isVideoStream,
		ShareRequest:  shareRequest,
	}
}

func ReqDownloadFileWrongData(fInfo *protos.RspFileStorageInfo, dTask *task.DownloadTask) *protos.ReqDownloadFileWrong {
	var failedSlices []string
	var failedPPNodes []*protos.PPBaseInfo
	for sliceHash := range dTask.FailedSlice {
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
		Sign:          fInfo.NodeSign,
		IsVideoStream: fInfo.IsVideoStream,
		FailedSlices:  failedSlices,
		FailedPpNodes: failedPPNodes,
	}
}

func FindFileListData(fileName string, walletAddr string, pageId uint64, keyword string, fileType protos.FileSortType, isUp bool) *protos.ReqFindMyFileList {
	return &protos.ReqFindMyFileList{
		FileName:      fileName,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: walletAddr,
		PageId:        pageId,
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

func ReqDeleteFileData(fileHash string) *protos.ReqDeleteFile {
	return &protos.ReqDeleteFile{
		FileHash:      fileHash,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
		Sign:          setting.GetSign(setting.P2PAddress + fileHash),
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
		MSGHead: PPMsgHeader(data, header.ReqDownloadSlice),
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

func ReqShareLinkData(walletAddr string, page uint64) *protos.ReqShareLink {
	return &protos.ReqShareLink{
		P2PAddress:    setting.P2PAddress,
		WalletAddress: walletAddr,
		PageId:        page,
	}
}

func ReqShareFileData(fileHash, pathHash, walletAddr string, isPrivate bool, shareTime int64) *protos.ReqShareFile {
	return &protos.ReqShareFile{
		FileHash:      fileHash,
		IsPrivate:     isPrivate,
		ShareTime:     shareTime,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: walletAddr,
		PathHash:      pathHash,
	}
}

func ReqDeleteShareData(shareID, walletAddr string) *protos.ReqDeleteShare {
	return &protos.ReqDeleteShare{
		P2PAddress:    setting.P2PAddress,
		WalletAddress: walletAddr,
		ShareId:       shareID,
	}
}

func ReqGetShareFileData(keyword, sharePassword, saveAs, walletAddr string, walletPubkey []byte) *protos.ReqGetShareFile {
	msg := utils.GetFileDownloadShareNodeSignMessage(setting.P2PAddress, keyword, header.ReqGetShareFile)
	return &protos.ReqGetShareFile{
		Keyword:       keyword,
		P2PAddress:    setting.P2PAddress,
		NodeSign:      types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
		WalletAddress: walletAddr,
		WalletPubkey:  walletPubkey,
		SharePassword: sharePassword,
		SaveAs:        saveAs,
	}
}

func UploadSpeedOfProgressData(fileHash string, size uint64, start uint64, t int64) *protos.UploadSpeedOfProgress {
	return &protos.UploadSpeedOfProgress{
		FileHash:      fileHash,
		SliceSize:     size,
		SliceOffStart: start,
		HandleTime:    t,
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

func ReqStartMaintenance(duration uint64) *protos.ReqStartMaintenance {
	return &protos.ReqStartMaintenance{
		Address:  setting.GetPPInfo(),
		Duration: duration,
	}
}

func ReqStopMaintenance() *protos.ReqStopMaintenance {
	return &protos.ReqStopMaintenance{
		Address: setting.GetPPInfo(),
	}
}

func ReqDowngradeInfo() *protos.ReqGetPPDowngradeInfo {
	return &protos.ReqGetPPDowngradeInfo{
		MyAddress: &protos.PPBaseInfo{
			P2PAddress:     setting.P2PAddress,
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
			RestAddress:    setting.RestAddress,
		},
	}
}

func ReqFileReplicaInfo(path, walletAddr string, replicaIncreaseNum uint32, walletSign, walletPUbkey []byte) *protos.ReqFileReplicaInfo {
	msg := utils.GetFileReplicaInfoNodeSignMessage(setting.P2PAddress, path, header.ReqFileReplicaInfo)
	return &protos.ReqFileReplicaInfo{
		P2PAddress:         setting.P2PAddress,
		WalletAddress:      walletAddr,
		FilePath:           path,
		ReplicaIncreaseNum: replicaIncreaseNum,
		NodeSign:           types.BytesToP2pPrivKey(setting.P2PPrivateKey).Sign([]byte(msg)),
		WalletSign:         walletSign,
		WalletPubkey:       walletPUbkey,
	}
}

func PPMsgHeader(data []byte, head string) header.MessageHead {
	return header.MakeMessageHeader(1, setting.Config.Version.AppVer, uint32(len(data)), head)
}

func UnmarshalData(ctx context.Context, target interface{}) bool {
	msgBuf := core.MessageFromContext(ctx)
	pp.DebugLogf(ctx, "Received message type = %v msgBuf len = %v", reflect.TypeOf(target), len(msgBuf.MSGData))
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

func GetSpPubkey(spP2pAddr string) ([]byte, error) {
	// find the stored SP public key
	val, ok := setting.SPMap.Load(spP2pAddr)
	if !ok {
		return nil, errors.New("couldn't find sp info by the given SP address")
	}
	spInfo, ok := val.(setting.SPBaseInfo)
	if !ok {
		return nil, errors.New("failed to parse SP info")
	}
	_, spP2pPubkey, err := bech32.DecodeAndConvert(spInfo.P2PPublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding P2P pubKey from bech32")
	}
	return spP2pPubkey, nil
}
