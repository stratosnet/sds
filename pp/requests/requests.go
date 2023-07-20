package requests

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/stratosnet/sds/pp/p2pserver"
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

const INVALID_STAT = int64(-1)

func ReqRegisterData(ctx context.Context, walletAddr string, walletPubkey, wsig []byte, reqTime int64) *protos.ReqRegister {
	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsig,
		Type:      protos.SignatureType_WALLET,
	}
	return &protos.ReqRegister{
		Address:   p2pserver.GetP2pServer(ctx).GetPPInfo(),
		MyAddress: p2pserver.GetP2pServer(ctx).GetPPInfo(),
		PublicKey: p2pserver.GetP2pServer(ctx).GetP2PPublicKey(),
		Signature: walletSign,
		ReqTime:   reqTime,
	}
}

func ReqRegisterDataTR(ctx context.Context, target *protos.ReqRegister) *msg.RelayMsgBuf {
	req := target
	req.MyAddress = p2pserver.GetP2pServer(ctx).GetPPInfo()
	body, err := proto.Marshal(req)
	if err != nil {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeader(uint32(len(body)), header.ReqRegister),
		MSGBody: body,
	}
}

func ReqMiningData(ctx context.Context) *protos.ReqMining {
	return &protos.ReqMining{Address: p2pserver.GetP2pServer(ctx).GetPPInfo()}
}

func ReqGetPPlistData(ctx context.Context) *protos.ReqGetPPList {
	return &protos.ReqGetPPList{MyAddress: p2pserver.GetP2pServer(ctx).GetPPInfo()}
}

func ReqGetSPlistData(ctx context.Context, walletAddr string, walletPubkey, wsig []byte, reqTime int64) *protos.ReqGetSPList {
	//nowSec := time.Now().Unix()
	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsig,
		Type:      protos.SignatureType_WALLET,
	}
	req := &protos.ReqGetSPList{
		MyAddress: p2pserver.GetP2pServer(ctx).GetPPInfo(),
		ReqTime:   reqTime,
		Signature: walletSign,
	}
	return req
}

func ReqGetPPStatusData(ctx context.Context, initPPList bool) *protos.ReqGetPPStatus {
	return &protos.ReqGetPPStatus{
		MyAddress:  p2pserver.GetP2pServer(ctx).GetPPInfo(),
		InitPpList: initPPList,
	}
}

func ReqGetWalletOzData(walletAddr, reqId string) *protos.ReqGetWalletOz {
	return &protos.ReqGetWalletOz{
		WalletAddress: walletAddr,
	}
}

// RequestUploadFile a file from an owner instead from a "path" belongs to PP's default wallet
func RequestUploadFile(ctx context.Context, fileName, fileHash string, fileSize uint64, walletAddress, walletPubkey, signature string, reqTime int64,
	slices []*protos.SliceHashAddr, isEncrypted bool, desiredTier uint32, allowHigherTier bool, duration uint64) (*protos.ReqUploadFile, error) {
	utils.Log("fileName: ", fileName)
	encryptionTag := ""
	if isEncrypted {
		encryptionTag = utils.GetRandomString(8)
	}

	utils.Log("fileHash: ", fileHash)

	file.SaveRemoteFileHash(fileHash, fileName, fileSize)

	// convert wallet pubkey to []byte which format is to be used in protobuf messages
	wpk, err := types.WalletPubkeyFromBech(walletPubkey)
	if err != nil {
		utils.ErrorLog("wrong wallet pubkey")
		return nil, errors.New("wrong wallet pubkey")
	}
	// decode the hex encoded signature back to []byte which is used in protobuf messages
	wsig, err := hex.DecodeString(signature)
	if err != nil {
		utils.ErrorLog("wrong signature")
		return nil, errors.New("wrong signature")
	}
	req := &protos.ReqUploadFile{
		FileInfo: &protos.FileInfo{
			FileSize:           fileSize,
			FileName:           fileName,
			FileHash:           fileHash,
			StoragePath:        "rpc:" + fileName,
			EncryptionTag:      encryptionTag,
			OwnerWalletAddress: walletAddress,
			Duration:           duration,
		},
		Slices:    slices,
		MyAddress: p2pserver.GetP2pServer(ctx).GetPPInfo(),
		Signature: &protos.Signature{
			Address:   walletAddress,
			Pubkey:    wpk.Bytes(),
			Signature: wsig,
			Type:      protos.SignatureType_WALLET,
		},
		DesiredTier:     desiredTier,
		AllowHigherTier: allowHigherTier,
		ReqTime:         reqTime,
	}

	// info
	p := &task.UploadProgress{
		Total:     int64(fileSize),
		HasUpload: 0,
	}
	task.UploadProgressMap.Store(fileHash, p)
	return req, nil
}

// RequestUploadFileData assume the PP's current wallet is the owner, otherwise RequestUploadFile() should be used instead
func RequestUploadFileData(ctx context.Context, fileInfo *protos.FileInfo, slices []*protos.SliceHashAddr, desiredTier uint32, allowHigherTier bool,
	walletAddr string, walletPubkey, walletSign []byte, reqTime int64) *protos.ReqUploadFile {

	req := &protos.ReqUploadFile{
		FileInfo:  fileInfo,
		MyAddress: p2pserver.GetP2pServer(ctx).GetPPInfo(),
		Signature: &protos.Signature{
			Address:   walletAddr,
			Pubkey:    walletPubkey,
			Signature: walletSign,
			Type:      protos.SignatureType_WALLET,
		},
		DesiredTier:     desiredTier,
		AllowHigherTier: allowHigherTier,
		Slices:          slices,
		ReqTime:         reqTime,
	}

	// info
	p := &task.UploadProgress{
		Total:     int64(fileInfo.FileSize),
		HasUpload: 0,
	}
	task.UploadProgressMap.Store(fileInfo.FileHash, p)
	return req
}

// RequestDownloadFile the entry for rpc remote download
func RequestDownloadFile(ctx context.Context, fileHash, sdmPath, walletAddr string, reqId string, walletSign, walletPubkey []byte, shareRequest *protos.ReqGetShareFile, reqTime int64) *protos.ReqFileStorageInfo {
	// file's request id is used for identifying the download session
	fileReqId := reqId
	if reqId == "" {
		fileReqId = uuid.New().String()
	}

	// download file uses fileHash + fileReqId as the key
	file.SaveRemoteFileHash(fileHash+fileReqId, "", 0)

	// path: mesh network address
	metrics.DownloadPerformanceLogNow(fileHash + ":SND_STORAGE_INFO_SP:")
	req := ReqFileStorageInfoData(ctx, sdmPath, "", "", walletAddr, walletPubkey, walletSign, shareRequest, reqTime)
	return req
}

func RspDownloadSliceData(ctx context.Context, target *protos.ReqDownloadSlice, slice *protos.DownloadSliceInfo) *protos.RspDownloadSlice {
	sliceData := task.GetDownloadSlice(target, slice)
	return &protos.RspDownloadSlice{
		P2PAddress:    target.P2PAddress,
		WalletAddress: target.RspFileStorageInfo.WalletAddress,
		SliceInfo: &protos.SliceOffsetInfo{
			SliceHash:   slice.SliceStorageInfo.SliceHash,
			SliceOffset: slice.SliceOffset,
		},
		FileCrc:           sliceData.FileCrc,
		FileHash:          target.RspFileStorageInfo.FileHash,
		TaskId:            slice.TaskId,
		Data:              sliceData.Data,
		SliceSize:         uint64(len(sliceData.Data)),
		SavePath:          target.RspFileStorageInfo.SavePath,
		Result:            &protos.Result{State: protos.ResultState_RES_SUCCESS, Msg: ""},
		IsEncrypted:       target.RspFileStorageInfo.EncryptionTag != "",
		SpP2PAddress:      target.RspFileStorageInfo.SpP2PAddress,
		StorageP2PAddress: p2pserver.GetP2pServer(ctx).GetP2PAddress(),
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

func ReqUploadFileSliceData(ctx context.Context, task *task.UploadSliceTask, pieceOffset *protos.SliceOffset, data []byte) *protos.ReqUploadFileSlice {
	return &protos.ReqUploadFileSlice{
		RspUploadFile: task.RspUploadFile,
		SliceNumber:   task.SliceNumber,
		SliceHash:     task.SliceHash,
		Data:          data,
		WalletAddress: setting.WalletAddress,
		PieceOffset:   pieceOffset,
		P2PAddress:    p2pserver.GetP2pServer(ctx).GetP2PAddress(),
	}
}

func RspUploadFileSliceData(ctx context.Context, target *protos.ReqUploadFileSlice) *protos.RspUploadFileSlice {
	var slice *protos.SliceHashAddr
	for _, slice = range target.RspUploadFile.Slices {
		if slice.SliceNumber == target.SliceNumber {
			break
		}
	}
	return &protos.RspUploadFileSlice{
		TaskId:        target.RspUploadFile.TaskId,
		FileHash:      target.RspUploadFile.FileHash,
		SliceHash:     target.SliceHash,
		P2PAddress:    p2pserver.GetP2pServer(ctx).GetP2PAddress(),
		WalletAddress: target.WalletAddress,
		Slice:         slice,
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		SpP2PAddress: target.RspUploadFile.SpP2PAddress,
	}
}

func ReqBackupFileSliceData(ctx context.Context, task *task.UploadSliceTask, pieceOffset *protos.SliceOffset, data []byte) *protos.ReqBackupFileSlice {
	return &protos.ReqBackupFileSlice{
		RspBackupFile: task.RspBackupFile,
		SliceNumber:   task.SliceNumber,
		SliceHash:     task.SliceHash,
		Data:          data,
		WalletAddress: setting.WalletAddress,
		PieceOffset:   pieceOffset,
		P2PAddress:    p2pserver.GetP2pServer(ctx).GetP2PAddress(),
	}
}

func RspBackupFileSliceData(target *protos.ReqBackupFileSlice) *protos.RspBackupFileSlice {
	var slice *protos.SliceHashAddr
	for _, slice = range target.RspBackupFile.Slices {
		if slice.SliceNumber == target.SliceNumber {
			break
		}
	}
	return &protos.RspBackupFileSlice{
		TaskId:        target.RspBackupFile.TaskId,
		FileHash:      target.RspBackupFile.FileHash,
		SliceHash:     target.SliceHash,
		WalletAddress: target.WalletAddress,
		Slice:         slice,
		SliceSize:     slice.SliceOffset.SliceOffsetEnd - slice.SliceOffset.SliceOffsetStart,
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		SpP2PAddress: target.RspBackupFile.SpP2PAddress,
	}
}
func ReqUploadSlicesWrong(ctx context.Context, uploadTask *task.UploadFileTask, spP2pAddress string, slicesToDownload []*protos.SliceHashAddr, failedSlices []bool) *protos.ReqUploadSlicesWrong {
	return &protos.ReqUploadSlicesWrong{
		FileHash:             uploadTask.RspUploadFile.FileHash,
		TaskId:               uploadTask.RspUploadFile.TaskId,
		UploadType:           uploadTask.Type,
		MyAddress:            p2pserver.GetP2pServer(ctx).GetPPInfo(),
		SpP2PAddress:         spP2pAddress,
		ExcludedDestinations: uploadTask.GetExcludedDestinations(),
		Slices:               slicesToDownload,
		FailedSlices:         failedSlices,
	}
}

func ReqReportUploadSliceResultData(ctx context.Context, taskId, fileHash, spP2pAddr, opponentP2pAddress string, isPp bool, slice *protos.SliceHashAddr, costTime int64) *protos.ReportUploadSliceResult {
	utils.DebugLog("reqReportUploadSliceResultData____________________", slice.SliceSize)
	return &protos.ReportUploadSliceResult{
		TaskId:             taskId,
		Slice:              slice,
		IsPP:               isPp,
		UploadSuccess:      true,
		FileHash:           fileHash,
		P2PAddress:         p2pserver.GetP2pServer(ctx).GetP2PAddress(),
		WalletAddress:      setting.WalletAddress,
		SpP2PAddress:       spP2pAddr,
		CostTime:           costTime,
		OpponentP2PAddress: opponentP2pAddress,
	}
}

func ReqReportDownloadResultData(ctx context.Context, target *protos.RspDownloadSlice, costTime int64, isPP bool) *protos.ReqReportDownloadResult {
	utils.DebugLog("#################################################################", target.SliceInfo.SliceHash)
	repReq := &protos.ReqReportDownloadResult{
		IsPP:                 isPP,
		DownloaderP2PAddress: target.P2PAddress,
		WalletAddress:        target.WalletAddress,
		PpP2PAddress:         p2pserver.GetP2pServer(ctx).GetP2PAddress(),
		PpWalletAddress:      setting.WalletAddress,
		FileHash:             target.FileHash,
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

func ReqReportStreamResultData(ctx context.Context, target *protos.RspDownloadSlice, isPP bool) *protos.ReqReportDownloadResult {
	utils.DebugLog("#################################################################", target.SliceInfo.SliceHash)
	repReq := &protos.ReqReportDownloadResult{
		IsPP:                 isPP,
		DownloaderP2PAddress: target.P2PAddress,
		WalletAddress:        target.WalletAddress,
		PpP2PAddress:         p2pserver.GetP2pServer(ctx).GetP2PAddress(),
		PpWalletAddress:      setting.WalletAddress,
		FileHash:             target.FileHash,
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

func ReqDownloadSliceData(ctx context.Context, target *protos.RspFileStorageInfo, slice *protos.DownloadSliceInfo) *protos.ReqDownloadSlice {
	return &protos.ReqDownloadSlice{
		RspFileStorageInfo: target,
		SliceNumber:        slice.SliceNumber,
		P2PAddress:         p2pserver.GetP2pServer(ctx).GetP2PAddress(),
	}
}

func ReqRegisterNewPPData(ctx context.Context, walletAddr string, walletPubkey, wsig []byte, reqTime int64) *protos.ReqRegisterNewPP {
	sysInfo := utils.GetSysInfo(setting.Config.Home.StoragePath)
	sysInfo.DiskSize = setting.GetDiskSizeSoftCap(sysInfo.DiskSize)
	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsig,
		Type:      protos.SignatureType_WALLET,
	}
	return &protos.ReqRegisterNewPP{
		P2PAddress:     p2pserver.GetP2pServer(ctx).GetP2PAddress(),
		Signature:      walletSign,
		DiskSize:       sysInfo.DiskSize,
		FreeDisk:       sysInfo.FreeDisk,
		MemorySize:     sysInfo.MemorySize,
		OsAndVer:       sysInfo.OSInfo,
		CpuInfo:        sysInfo.CPUInfo,
		MacAddress:     sysInfo.MacAddress,
		Version:        uint32(setting.Config.Version.AppVer),
		PubKey:         p2pserver.GetP2pServer(ctx).GetP2PPublicKey(),
		NetworkAddress: setting.NetworkAddress,
		ReqTime:        reqTime,
	}
}

func ReqTransferDownloadData(ctx context.Context, notice *protos.NoticeFileSliceBackup) *msg.RelayMsgBuf {

	protoMsg := &protos.ReqTransferDownload{
		NoticeFileSliceBackup: notice,
		NewPp:                 p2pserver.GetP2pServer(ctx).GetPPInfo(),
		P2PAddress:            p2pserver.GetP2pServer(ctx).GetP2PAddress(),
	}
	body, err := proto.Marshal(protoMsg)
	if err != nil {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeader(uint32(len(body)), header.ReqTransferDownload),
		MSGBody: body,
	}
}

func ReqTransferDownloadWrongData(ctx context.Context, notice *protos.NoticeFileSliceBackup) *protos.ReqTransferDownloadWrong {
	return &protos.ReqTransferDownloadWrong{
		TaskId:           notice.TaskId,
		NewPp:            p2pserver.GetP2pServer(ctx).GetPPInfo(),
		OriginalPp:       notice.PpInfo,
		SliceStorageInfo: notice.SliceStorageInfo,
		FileHash:         notice.FileHash,
		SpP2PAddress:     notice.SpP2PAddress,
	}
}

// ReqFileStorageInfoData encode ReqFileStorageInfo message. If it's not a "share request", walletAddr should keep the same
// as the wallet from the "path".
func ReqFileStorageInfoData(ctx context.Context, path, savePath, saveAs, walletAddr string, walletPUbkey, wsig []byte, shareRequest *protos.ReqGetShareFile, reqTime int64) *protos.ReqFileStorageInfo {
	return &protos.ReqFileStorageInfo{
		FileIndexes: &protos.FileIndexes{
			P2PAddress:    p2pserver.GetP2pServer(ctx).GetP2PAddress(),
			WalletAddress: walletAddr,
			FilePath:      path,
			SavePath:      savePath,
			SaveAs:        saveAs,
		},
		Signature: &protos.Signature{
			Address:   walletAddr,
			Pubkey:    walletPUbkey,
			Signature: wsig,
			Type:      protos.SignatureType_WALLET,
		},
		ShareRequest: shareRequest,
		ReqTime:      reqTime,
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
		FailedSlices:  failedSlices,
		FailedPpNodes: failedPPNodes,
		TaskId:        dTask.TaskId,
	}
}

func FindFileListData(fileName string, walletAddr, p2pAddress string, pageId uint64, keyword string, fileType protos.FileSortType, isUp bool, walletPubkey, wsign []byte, reqTime int64) *protos.ReqFindMyFileList {
	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsign,
		Type:      protos.SignatureType_WALLET,
	}
	return &protos.ReqFindMyFileList{
		FileName:   fileName,
		P2PAddress: p2pAddress,
		Signature:  walletSign,
		PageId:     pageId,
		FileType:   fileType,
		IsUp:       isUp,
		Keyword:    keyword,
		ReqTime:    reqTime,
	}
}

func ClearExpiredShareLinksData(p2pAddress, walletAddr string, walletPubkey, wsign []byte, reqTime int64) *protos.ReqClearExpiredShareLinks {
	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsign,
		Type:      protos.SignatureType_WALLET,
	}
	return &protos.ReqClearExpiredShareLinks{
		P2PAddress: p2pAddress,
		Signature:  walletSign,
		ReqTime:    reqTime,
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

func RspTransferDownload(data []byte, taskId, sliceHash, spP2pAddress, p2pAddress string, offset, sliceSize uint64) *protos.RspTransferDownload {
	return &protos.RspTransferDownload{
		Data:         data,
		TaskId:       taskId,
		Offset:       offset,
		SliceSize:    sliceSize,
		SpP2PAddress: spP2pAddress,
		SliceHash:    sliceHash,
		P2PAddress:   p2pAddress,
	}
}

func ReqDeleteFileData(fileHash, p2pAddress string, walletAddr string, walletPubkey, wsign []byte, reqTime int64) *protos.ReqDeleteFile {
	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsign,
	}
	return &protos.ReqDeleteFile{
		FileHash:   fileHash,
		P2PAddress: p2pAddress,
		Signature:  walletSign,
		ReqTime:    reqTime,
	}
}

func RspGetHDInfoData(p2pAddress string) *protos.RspGetHDInfo {
	rsp := &protos.RspGetHDInfo{
		P2PAddress:    p2pAddress,
		WalletAddress: setting.WalletAddress,
	}

	diskStats, err := utils.GetDiskUsage(setting.Config.Home.StoragePath)
	if err == nil {
		diskStats.Total = setting.GetDiskSizeSoftCap(diskStats.Total)
		rsp.DiskSize = int64(diskStats.Total)
		rsp.DiskFree = int64(diskStats.Free)
	} else {
		utils.ErrorLog("Can't fetch disk usage statistics", err)
		rsp.DiskSize = INVALID_STAT
		rsp.DiskFree = INVALID_STAT
	}

	return rsp
}

func ReqShareLinkData(walletAddr, p2pAddress string, page uint64, walletPubkey, wsign []byte, reqTime int64) *protos.ReqShareLink {
	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsign,
		Type:      protos.SignatureType_WALLET,
	}
	return &protos.ReqShareLink{
		P2PAddress: p2pAddress,
		Signature:  walletSign,
		PageId:     page,
		ReqTime:    reqTime,
	}
}

func ReqShareFileData(fileHash, pathHash, walletAddr, p2pAddress string, isPrivate bool, shareTime int64, walletPubkey, wsign []byte, reqTime int64) *protos.ReqShareFile {
	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsign,
		Type:      protos.SignatureType_WALLET,
	}
	return &protos.ReqShareFile{
		FileHash:   fileHash,
		IsPrivate:  isPrivate,
		ShareTime:  shareTime,
		P2PAddress: p2pAddress,
		Signature:  walletSign,
		PathHash:   pathHash,
		ReqTime:    reqTime,
	}
}

func ReqDeleteShareData(shareID, walletAddr, p2pAddress string, walletPubkey, wsign []byte, reqTime int64) *protos.ReqDeleteShare {
	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsign,
		Type:      protos.SignatureType_WALLET,
	}
	return &protos.ReqDeleteShare{
		P2PAddress: p2pAddress,
		Signature:  walletSign,
		ShareId:    shareID,
		ReqTime:    reqTime,
	}
}

func ReqGetShareFileData(keyword, sharePassword, saveAs, walletAddr, p2pAddress string, walletPubkey, wsign []byte, isVideoStream bool, reqTime int64) *protos.ReqGetShareFile {
	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsign,
		Type:      protos.SignatureType_WALLET,
	}
	return &protos.ReqGetShareFile{
		Keyword:       keyword,
		P2PAddress:    p2pAddress,
		Signature:     walletSign,
		SharePassword: sharePassword,
		SaveAs:        saveAs,
		ReqTime:       reqTime,
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

func ReqNodeStatusData(p2pAddress string) *protos.ReqReportNodeStatus {
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
	memStat := &protos.MemoryStat{}
	virtualMem, err := mem.VirtualMemory()
	if err == nil {
		memStat.MemUsed = int64(virtualMem.Used)
		memStat.MemTotal = int64(virtualMem.Total)
	} else {
		utils.ErrorLog("Can't fetch memory statistics when reporting node status", err)
		memStat.MemUsed = INVALID_STAT
		memStat.MemTotal = INVALID_STAT
	}

	swapMemory, err := mem.SwapMemory()
	if err == nil {
		memStat.SwapMemUsed = int64(swapMemory.Used)
		memStat.SwapMemTotal = int64(swapMemory.Total)
	} else {
		utils.ErrorLog("Can't fetch swap memory statistics when reporting node status", err)
		memStat.SwapMemUsed = INVALID_STAT
		memStat.SwapMemTotal = INVALID_STAT
	}

	// Disk usage statistics
	diskStat := &protos.DiskStat{}
	info, err := utils.GetDiskUsage(setting.Config.Home.StoragePath)
	if err == nil {
		diskStat.RootUsed = int64(info.Used)
		info.Total = setting.GetDiskSizeSoftCap(info.Total)
		diskStat.RootTotal = int64(info.Total)
	} else {
		utils.ErrorLog(
			"Can't fetch disk usage statistics when reporting node status, this might cause score deduction", err)
		diskStat.RootUsed = INVALID_STAT
		diskStat.RootTotal = INVALID_STAT
	}

	// TODO Bandwidth
	bwStat := &protos.BandwidthStat{}

	req := &protos.ReqReportNodeStatus{
		P2PAddress: p2pAddress,
		Cpu:        cpuStat,
		Memory:     memStat,
		Disk:       diskStat,
		Bandwidth:  bwStat,
	}
	return req
}

func ReqStartMaintenance(ctx context.Context, duration uint64) *protos.ReqStartMaintenance {
	return &protos.ReqStartMaintenance{
		Address:  p2pserver.GetP2pServer(ctx).GetPPInfo(),
		Duration: duration,
	}
}

func ReqStopMaintenance(ctx context.Context) *protos.ReqStopMaintenance {
	return &protos.ReqStopMaintenance{Address: p2pserver.GetP2pServer(ctx).GetPPInfo()}
}

func ReqDowngradeInfo(ctx context.Context) *protos.ReqGetPPDowngradeInfo {
	return &protos.ReqGetPPDowngradeInfo{MyAddress: p2pserver.GetP2pServer(ctx).GetPPInfo()}
}

func ReqFileReplicaInfo(path, walletAddr, p2pAddress string, replicaIncreaseNum uint32, walletPubkey, walletSign []byte, reqTime int64) *protos.ReqFileReplicaInfo {
	return &protos.ReqFileReplicaInfo{
		P2PAddress:         p2pAddress,
		FilePath:           path,
		ReplicaIncreaseNum: replicaIncreaseNum,
		Signature: &protos.Signature{
			Address:   walletAddr,
			Pubkey:    walletPubkey,
			Signature: walletSign,
			Type:      protos.SignatureType_WALLET,
		},
		ReqTime: reqTime,
	}
}

func ReqFileStatus(fileHash, walletAddr string, walletPubkey, walletSign []byte, reqTime int64) *protos.ReqFileStatus {
	return &protos.ReqFileStatus{
		FileHash: fileHash,
		Signature: &protos.Signature{
			Address:   walletAddr,
			Pubkey:    walletPubkey,
			Signature: walletSign,
			Type:      protos.SignatureType_WALLET,
		},
		ReqTime: reqTime,
	}
}

func GetSliceOffset(sliceNumber, sliceCount, sliceSize, fileSize uint64) *protos.SliceOffset {
	var sliceOffsetStart uint64
	var sliceOffsetEnd uint64
	sliceOffsetStart = (sliceNumber - 1) * sliceSize

	if sliceNumber == sliceCount {
		sliceOffsetEnd = fileSize
	} else {
		sliceOffsetEnd = sliceNumber * sliceSize
	}

	return &protos.SliceOffset{
		SliceOffsetStart: sliceOffsetStart,
		SliceOffsetEnd:   sliceOffsetEnd,
	}
}

func PPMsgHeader(dataLen uint32, msgType header.MsgType) header.MessageHead {
	return header.MakeMessageHeader(1, setting.Config.Version.AppVer, dataLen, msgType)
}

func UnmarshalData(ctx context.Context, target interface{}) bool {
	msgBuf := core.MessageFromContext(ctx)
	pp.DebugLogf(ctx, "Received message type = %v msgBuf len = %v", reflect.TypeOf(target), len(msgBuf.MSGBody))
	ret := UnmarshalMessageData(msgBuf.MSGBody, target)
	if ret {
		switch reflect.TypeOf(target) {
		case reflect.TypeOf(&protos.ReqUploadFileSlice{}):
			target.(*protos.ReqUploadFileSlice).Data = msgBuf.MSGData
		case reflect.TypeOf(&protos.ReqBackupFileSlice{}):
			target.(*protos.ReqBackupFileSlice).Data = msgBuf.MSGData
		case reflect.TypeOf(&protos.RspDownloadSlice{}):
			target.(*protos.RspDownloadSlice).Data = msgBuf.MSGData
		case reflect.TypeOf(&protos.RspTransferDownload{}):
			target.(*protos.RspTransferDownload).Data = msgBuf.MSGData
		}
	}
	return ret
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
		return nil, errors.New(fmt.Sprintf("couldn't find sp info by the given SP address: %s", spP2pAddr))
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
