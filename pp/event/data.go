package event

import (
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto"
	"path"

	"github.com/golang/protobuf/proto"
)

func reqRegisterData(toSP bool) *protos.ReqRegister {
	return &protos.ReqRegister{
		Address: &protos.PPBaseInfo{
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
		},
		MyAddress: &protos.PPBaseInfo{
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
		},
		PublicKey: setting.PublicKey,
	}
}

func reqRegisterDataTR(target *protos.ReqRegister) *msg.RelayMsgBuf {
	req := target
	req.MyAddress = &protos.PPBaseInfo{
		WalletAddress:  setting.WalletAddress,
		NetworkAddress: setting.NetworkAddress,
	}
	data, err := proto.Marshal(req)
	if utils.CheckError(err) {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeader(data, header.ReqRegister),
		MSGData: data,
	}
}

func reqMiningData() *protos.ReqMining {
	return &protos.ReqMining{
		Address: &protos.PPBaseInfo{
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
		},
		PublicKey: setting.PublicKey,
		Sign:      setting.GetSign(setting.WalletAddress),
	}
}

func reqGetPPlistData() *protos.ReqGetPPList {
	req := &protos.ReqGetPPList{
		MyAddress: &protos.PPBaseInfo{
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
		},
	}
	return req
}

// RequestUploadFileData RequestUploadFileData
func RequestUploadFileData(paths, storagePath, reqID string, isCover bool) *protos.ReqUploadFile {
	info := file.GetFileInfo(paths)
	if info == nil {
		utils.ErrorLog("wrong filePath")
		return nil
	}
	fileName := info.Name()
	utils.Log("fileName~~~~~~~~~~~~~~~~~~~~~~~~", fileName)
	fileHash := file.GetFileHash(paths)
	utils.Log("fileHash~~~~~~~~~~~~~~~~~~~~~~", fileHash)
	req := &protos.ReqUploadFile{
		FileInfo: &protos.FileInfo{
			FileSize:    uint64(info.Size()),
			FileName:    fileName,
			FileHash:    fileHash,
			StoragePath: storagePath,
		},
		MyAddress: &protos.PPBaseInfo{
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
		},
		Sign:    setting.GetSign(setting.WalletAddress + fileHash),
		IsCover: isCover,
		ReqId:   reqID,
	}
	if isCover {
		fileSuffix := path.Ext(paths)
		req.FileInfo.FileName = fileHash + fileSuffix
	}
	utils.DebugLog("setting.WalletAddress + fileHash", []byte(setting.WalletAddress+fileHash))

	puk, _ := crypto.UnmarshalPubkey(setting.PublicKey)
	if utils.ECCVerify([]byte(setting.WalletAddress+fileHash), req.Sign, puk) {
		utils.DebugLog("okkkkkk")
	} else {
		utils.DebugLog("noooooo")
	}
	// info
	p := &task.UpProgress{
		Total:     info.Size(),
		HasUpload: 0,
	}
	task.UpLoadProgressMap.Store(fileHash, p)
	// if isCover {
	// 	os.Remove(path)
	// }
	return req
}

func rspDownloadSliceData(target *protos.ReqDownloadSlice) *protos.RspDownloadSlice {
	slice := task.GetDonwloadSlice(target)
	return &protos.RspDownloadSlice{
		WalletAddress: target.WalletAddress,
		SliceInfo:     target.SliceInfo,
		FileCrc:       slice.FileCrc,
		FileHash:      target.FileHash,
		TaskId:        target.TaskId,
		Data:          slice.Data,
		SliceSize:     uint64(len(slice.Data)),
		SavePath:      target.SavePath,
	}
}

func rspDownloadSliceDataSplit(rsp *protos.RspDownloadSlice, dataStart, dataEnd, offsetStart, offsetEnd uint64, last bool) *protos.RspDownloadSlice {
	if dataEnd == 0 {
		return &protos.RspDownloadSlice{
			SliceInfo: &protos.SliceOffsetInfo{
				SliceHash: rsp.SliceInfo.SliceHash,
				SliceOffset: &protos.SliceOffset{
					SliceOffsetStart: offsetStart,
					SliceOffsetEnd:   offsetEnd,
				},
			},
			FileCrc:       rsp.FileCrc,
			FileHash:      rsp.FileHash,
			Data:          rsp.Data[dataStart:],
			WalletAddress: rsp.WalletAddress,
			TaskId:        rsp.TaskId,
			SliceSize:     rsp.SliceSize,
			Result:        rsp.Result,
			NeedReport:    last,
			SavePath:      rsp.SavePath,
		}
	}
	return &protos.RspDownloadSlice{
		SliceInfo: &protos.SliceOffsetInfo{
			SliceHash: rsp.SliceInfo.SliceHash,
			SliceOffset: &protos.SliceOffset{
				SliceOffsetStart: offsetStart,
				SliceOffsetEnd:   offsetEnd,
			},
		},
		FileCrc:       rsp.FileCrc,
		FileHash:      rsp.FileHash,
		Data:          rsp.Data[dataStart:dataEnd],
		WalletAddress: rsp.WalletAddress,
		TaskId:        rsp.TaskId,
		SliceSize:     rsp.SliceSize,
		Result:        rsp.Result,
		NeedReport:    last,
		SavePath:      rsp.SavePath,
	}

}

func reqUploadFileSliceData(task *task.UploadSliceTask) *protos.ReqUploadFileSlice {
	return &protos.ReqUploadFileSlice{
		TaskId:        task.TaskID,
		FileCrc:       task.FileCRC,
		SliceNumAddr:  task.SliceNumAddr,
		SliceInfo:     task.SliceOffsetInfo,
		Data:          task.Data,
		FileHash:      task.FileHash,
		WalletAddress: setting.WalletAddress,
		SliceSize:     task.SliceTotalSize,
	}
}

func reqReportUploadSliceResultData(target *protos.RspUploadFileSlice) *protos.ReportUploadSliceResult {

	utils.DebugLog("reqReportUploadSliceResultData____________________", target.SliceSize)
	return &protos.ReportUploadSliceResult{
		TaskId:        target.TaskId,
		SliceNumAddr:  target.SliceNumAddr,
		SliceHash:     target.SliceHash,
		IsPP:          false,
		UploadSuccess: true,
		FileHash:      target.FileHash,
		SliceSize:     target.SliceSize,
		Sign:          setting.GetSign(setting.WalletAddress + target.FileHash),
		WalletAddress: setting.WalletAddress,
	}
}
func reqReportUploadSliceResultDataPP(target *protos.ReqUploadFileSlice) *protos.ReportUploadSliceResult {
	utils.DebugLog("____________________", target.SliceSize)
	return &protos.ReportUploadSliceResult{
		TaskId:        target.TaskId,
		SliceNumAddr:  target.SliceNumAddr,
		SliceHash:     target.SliceInfo.SliceHash,
		IsPP:          true,
		UploadSuccess: true,
		FileHash:      target.FileHash,
		SliceSize:     target.SliceSize,
		Sign:          setting.GetSign(setting.WalletAddress + target.FileHash),
		WalletAddress: setting.WalletAddress,
	}
}

func rspUploadFileSliceData(target *protos.ReqUploadFileSlice) *protos.RspUploadFileSlice {
	return &protos.RspUploadFileSlice{
		TaskId:        target.TaskId,
		FileHash:      target.FileHash,
		SliceHash:     target.SliceInfo.SliceHash,
		WalletAddress: target.WalletAddress,
		SliceNumAddr:  target.SliceNumAddr,
		SliceSize:     target.SliceSize,
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}
}

func reqReportDownloadResultData(target *protos.RspDownloadSlice, isPP bool) *protos.ReqReportDownloadResult {

	utils.DebugLog("#################################################################", target.SliceInfo.SliceHash)
	repReq := &protos.ReqReportDownloadResult{
		IsPP:            isPP,
		WalletAddress:   target.WalletAddress,
		FileHash:        target.FileHash,
		Sign:            setting.GetSign(setting.WalletAddress + target.FileHash),
		MyWalletAddress: setting.WalletAddress,
		TaskId:          target.TaskId,
	}
	if isPP {
		utils.Log("PP ReportDownloadResult ")
		if dlTask, ok := task.DownloadTaskMap.Load(target.FileHash + target.WalletAddress); ok {
			donwloadTask := dlTask.(*task.DonwloadTask)
			utils.DebugLog("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^donwloadTask", donwloadTask)
			if sInfo, ok := donwloadTask.SliceInfo[target.SliceInfo.SliceHash]; ok {
				repReq.SliceInfo = sInfo
				repReq.SliceInfo.VisitResult = true
			} else {
				utils.DebugLog("ReportDownloadResult failed~~~~~~~~~~~~~~~~~~~~~~~~~~")
			}

		} else {
			repReq.SliceInfo = &protos.DownloadSliceInfo{
				SliceStorageInfo: &protos.SliceStorageInfo{
					SliceHash: target.SliceInfo.SliceHash,
				},
			}
		}
	} else {
		repReq.SliceInfo = &protos.DownloadSliceInfo{
			SliceStorageInfo: &protos.SliceStorageInfo{
				SliceHash: target.SliceInfo.SliceHash,
			},
		}
	}
	return repReq
}

func reqDownloadSliceData(target *protos.RspFileStorageInfo, rsp *protos.DownloadSliceInfo) *protos.ReqDownloadSlice {
	return &protos.ReqDownloadSlice{
		WalletAddress: setting.WalletAddress,
		FileHash:      target.FileHash,
		TaskId:        rsp.TaskId,
		SliceInfo: &protos.SliceOffsetInfo{
			SliceHash:   rsp.SliceStorageInfo.SliceHash,
			SliceOffset: rsp.SliceOffset,
		},
		SavePath: target.SavePath,
	}
}

func rspFileStorageInfoData(target *protos.RspFileStorageInfo) *msg.RelayMsgBuf {

	utils.DebugLog("download detail，", target)
	sendTarget := target
	sliceInfoArr := []*protos.DownloadSliceInfo{}
	for _, info := range sendTarget.SliceInfo {
		newInfo := protos.DownloadSliceInfo{
			SliceStorageInfo: info.SliceStorageInfo,
			SliceNumber:      info.SliceNumber,
			VisitResult:      info.VisitResult,
			TaskId:           info.TaskId,
			SliceOffset:      info.SliceOffset,
		}
		sliceInfoArr = append(sliceInfoArr, &newInfo)
	}
	sendTarget.SliceInfo = sliceInfoArr
	sendData, err := proto.Marshal(sendTarget)
	if utils.CheckError(err) {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGData: sendData,
		MSGHead: PPMsgHeader(sendData, header.RspFileStorageInfo),
	}
}

func reqRegisterNewPPData() *protos.ReqRegisterNewPP {
	sysInfo := utils.GetSysInfo()
	return &protos.ReqRegisterNewPP{
		WalletAddress: setting.WalletAddress,
		DiskSize:      sysInfo.DiskSize,
		MemorySize:    sysInfo.MemorySize,
		OsAndVer:      sysInfo.OSInfo,
		CpuInfo:       sysInfo.CPUInfo,
		MacAddress:    sysInfo.MacAddress,
		Version:       setting.Config.Version,
		PubKey:        setting.PublicKey,
		Sign:          setting.GetSign(setting.WalletAddress),
	}
}

func reqValidateTransferCerData(target *protos.ReqTransferNotice) *protos.ReqValidateTransferCer {
	return &protos.ReqValidateTransferCer{
		TransferCer: target.TransferCer,
		NewPp:       target.StoragePpInfo,
		OriginalPp: &protos.PPBaseInfo{
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
		},
	}
}

func reqTransferNoticeData(target *protos.ReqTransferNotice) *msg.RelayMsgBuf {
	sendTager := &protos.ReqTransferNotice{
		FromSp:      false,
		TransferCer: target.TransferCer,
		StoragePpInfo: &protos.PPBaseInfo{
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
		},

		SliceStorageInfo: target.SliceStorageInfo,
	}
	data, err := proto.Marshal(sendTager)
	if utils.CheckError(err) {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeader(data, header.ReqTransferNotice),
		MSGData: data,
	}
}

func rspTransferNoticeData(agree bool, cer string) *protos.RspTransferNotice {
	rsp := &protos.RspTransferNotice{
		StoragePpInfo: &protos.PPBaseInfo{
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
		},
		TransferCer: cer,
	}
	if agree {
		rsp.Result = &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		}
	} else {
		rsp.Result = &protos.Result{
			State: protos.ResultState_RES_FAIL,
		}
	}
	return rsp
}

func reqTransferDownloadData(transferCer string) *protos.ReqTransferDownload {
	return &protos.ReqTransferDownload{
		TransferCer: transferCer,
	}
}

func reqReportTaskBPData(taskID string, traffic uint64) *msg.RelayMsgBuf {
	utils.DebugLog("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~reqReportTaskBPDatareqReportTaskBPData  taskID ==", taskID, "traffic == ", traffic)
	sendTager := &protos.ReqReportTaskBP{
		TaskId:  taskID,
		Traffic: traffic,
		Reporter: &protos.PPBaseInfo{
			WalletAddress:  setting.WalletAddress,
			NetworkAddress: setting.NetworkAddress,
		},
	}
	data, err := proto.Marshal(sendTager)
	if utils.CheckError(err) {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeader(data, header.ReqReportTaskBP),
		MSGData: data,
	}
}

func reqFileStorageInfoData(path, savePath, reqID string) *protos.ReqFileStorageInfo {
	return &protos.ReqFileStorageInfo{
		FileIndexes: &protos.FileIndexes{
			WalletAddress: setting.WalletAddress,
			FilePath:      path,
			SavePath:      savePath,
		},
		Sign:  setting.GetSign(setting.WalletAddress + path),
		ReqId: reqID,
	}
}

func findMyFileListData(fileName, dir, reqID, keyword string, fileType protos.FileSortType, isUp bool) *protos.ReqFindMyFileList {
	return &protos.ReqFindMyFileList{
		FileName:      fileName,
		WalletAddress: setting.WalletAddress,
		Directory:     dir,
		ReqId:         reqID,
		FileType:      fileType,
		IsUp:          isUp,
		Keyword:       keyword,
	}
}

func findDirectoryData(reqID string) *protos.ReqFindDirectory {
	return &protos.ReqFindDirectory{
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
	}
}

func fileSortData(files []*protos.FileInfo, reqID, albumID string) *protos.ReqFileSort {
	return &protos.ReqFileSort{
		Files:         files,
		ReqId:         reqID,
		WalletAddress: setting.WalletAddress,
		AlbumId:       albumID,
	}
}

func rspTransferDownloadResultData(transferCer string) *protos.RspTransferDownloadResult {
	return &protos.RspTransferDownloadResult{
		TransferCer: transferCer,
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}
}

func rspTransferDownload(data []byte, transferCer string, offset, sliceSize uint64) *protos.RspTransferDownload {
	return &protos.RspTransferDownload{
		Data:        data,
		TransferCer: transferCer,
		Offset:      offset,
		SliceSize:   sliceSize,
	}
}

func reqDeleteFileData(fileHash, reqID string) *protos.ReqDeleteFile {
	return &protos.ReqDeleteFile{
		FileHash:      fileHash,
		WalletAddress: setting.WalletAddress,
		Sign:          setting.GetSign(setting.WalletAddress + fileHash),
		ReqId:         reqID,
	}
}

func reqDownloadSloceWrong(taskID, sliceHash, walletAddress string, wrongType protos.DownloadWrongType) *protos.ReqDownloadSloceWrong {
	return &protos.ReqDownloadSloceWrong{
		WalletAddress: walletAddress,
		TaskId:        taskID,
		SliceHash:     sliceHash,
		Type:          wrongType,
	}
}

func rspDownloadSloceWrong(target *protos.RspDownloadSloceWrong) *msg.RelayMsgBuf {
	sendTager := &protos.ReqDownloadSlice{
		SliceInfo: &protos.SliceOffsetInfo{
			SliceHash:   target.NewSliceInfo.SliceStorageInfo.SliceHash,
			SliceOffset: target.NewSliceInfo.SliceOffset,
		},
		WalletAddress: target.WalletAddress,
		TaskId:        target.TaskId,
		FileHash:      target.FileHash,
	}
	data, err := proto.Marshal(sendTager)
	if utils.CheckError(err) {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeader(data, header.ReqDownloadSlice),
		MSGData: data,
	}
}

func rspGetHDInfoData() *protos.RspGetHDInfo {
	size, free := setting.GetDHInfo()
	return &protos.RspGetHDInfo{
		WalletAddress: setting.WalletAddress,
		DiskSize:      size,
		DiskFree:      free,
	}
}

func rspDeleteSliceData(sliceHash, msg string, result bool) *protos.RspDeleteSlice {
	state := protos.ResultState_RES_SUCCESS
	if !result {
		state = protos.ResultState_RES_FAIL
	}
	return &protos.RspDeleteSlice{
		WalletAddress: setting.WalletAddress,
		SliceHash:     sliceHash,
		Result: &protos.Result{
			State: state,
			Msg:   msg,
		},
	}
}

func reqMakeDirectoryData(path, reqID string) *protos.ReqMakeDirectory {
	return &protos.ReqMakeDirectory{
		WalletAddress: setting.WalletAddress,
		Directory:     path,
		ReqId:         reqID,
	}
}

func reqRemoveDirectoryData(path, reqID string) *protos.ReqRemoveDirectory {
	return &protos.ReqRemoveDirectory{
		WalletAddress: setting.WalletAddress,
		Directory:     path,
		ReqId:         reqID,
	}
}

func reqShareLinkData(reqID string) *protos.ReqShareLink {
	return &protos.ReqShareLink{
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
	}
}

func reqShareFileData(reqID, fileHash, pathHash string, isPrivate bool, shareTime int64) *protos.ReqShareFile {
	return &protos.ReqShareFile{
		FileHash:      fileHash,
		IsPrivate:     isPrivate,
		ShareTime:     shareTime,
		WalletAddress: setting.WalletAddress,
		PathHash:      pathHash,
		ReqId:         reqID,
	}
}

func reqDeleteShareData(reqID, shareID string) *protos.ReqDeleteShare {
	return &protos.ReqDeleteShare{
		ReqId:         reqID,
		WalletAddress: setting.WalletAddress,
		ShareId:       shareID,
	}
}

func reqSaveFileData(fileHash, reqID, ownerAddress string) *protos.ReqSaveFile {
	return &protos.ReqSaveFile{
		FileHash:               fileHash,
		FileOwnerWalletAddress: ownerAddress,
		WalletAddress:          setting.WalletAddress,
		ReqId:                  reqID,
	}

}

func reqSaveFolderData(folderHash, reqID, ownerAddress string) *protos.ReqSaveFolder {
	return &protos.ReqSaveFolder{
		FolderHash:               folderHash,
		FolderOwnerWalletAddress: ownerAddress,
		WalletAddress:            setting.WalletAddress,
		ReqId:                    reqID,
	}

}

func reqMoveFileDirectoryData(fileHash, originalDir, targetDir, reqID string) *protos.ReqMoveFileDirectory {
	return &protos.ReqMoveFileDirectory{
		FileHash:          fileHash,
		WalletAddress:     setting.WalletAddress,
		ReqId:             reqID,
		DirectoryTarget:   targetDir,
		DirectoryOriginal: originalDir,
	}
}

func reqGetMyConfig(walletAddress, reqID string) *protos.ReqConfig {
	return &protos.ReqConfig{
		WalletAddress:  walletAddress,
		ReqId:          reqID,
		NetworkAddress: setting.NetworkAddress,
	}
}

func reqDownloadSlocePause(fileHash, reqID string) *protos.ReqDownloadSlocePause {
	return &protos.ReqDownloadSlocePause{
		FileHash:      fileHash,
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
	}
}

func rspDownloadSlocePauseData(target *protos.ReqDownloadSlocePause) *msg.RelayMsgBuf {
	sendTager := &protos.RspDownloadSlocePause{
		WalletAddress: target.WalletAddress,
		FileHash:      target.FileHash,
		ReqId:         target.ReqId,
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}
	data, err := proto.Marshal(sendTager)
	if utils.CheckError(err) {
		utils.ErrorLog(err)
	}
	return &msg.RelayMsgBuf{
		MSGHead: PPMsgHeader(data, header.RspDownloadSlicePause),
		MSGData: data,
	}
}

func reqCreateAlbumData(albumName, albumBlurb, albumCoverHash, reqID string, albumType protos.AlbumType, files []*protos.FileInfo, isPrivate bool) *protos.ReqCreateAlbum {
	return &protos.ReqCreateAlbum{
		WalletAddress:  setting.WalletAddress,
		ReqId:          reqID,
		AlbumName:      albumName,
		AlbumBlurb:     albumBlurb,
		AlbumCoverHash: albumCoverHash,
		AlbumType:      albumType,
		FileInfo:       files,
		IsPrivate:      isPrivate,
	}
}

func reqGetShareFileData(keyword, sharePassword, reqID string) *protos.ReqGetShareFile {
	return &protos.ReqGetShareFile{
		Keyword:       keyword,
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
		SharePassword: sharePassword,
	}
}

func reqFindMyAlbumData(albumType protos.AlbumType, reqID string, page, number uint64, keyword string) *protos.ReqFindMyAlbum {
	return &protos.ReqFindMyAlbum{
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
		AlbumType:     albumType,
		Page:          page,
		Number:        number,
		Keyword:       keyword,
	}
}

func reqEditAlbumData(albumID, albumCoverHash, albumName, albumBlurb, reqID string, changeFiles []*protos.FileInfo, isPrivate bool) *protos.ReqEditAlbum {
	return &protos.ReqEditAlbum{
		WalletAddress:  setting.WalletAddress,
		ReqId:          reqID,
		AlbumId:        albumID,
		AlbumCoverHash: albumCoverHash,
		AlbumName:      albumName,
		AlbumBlurb:     albumBlurb,
		ChangeFiles:    changeFiles,
		IsPrivate:      isPrivate,
	}
}

func reqAlbumContentData(albumID, reqID string) *protos.ReqAlbumContent {
	return &protos.ReqAlbumContent{
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
		AlbumId:       albumID,
	}
}

func reqCollectionAlbumData(albumID, reqID string, isCollection bool) *protos.ReqCollectionAlbum {
	return &protos.ReqCollectionAlbum{
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
		AlbumId:       albumID,
		IsCollection:  isCollection,
	}
}

func reqDeleteAlbumData(albumID, reqID string) *protos.ReqDeleteAlbum {
	return &protos.ReqDeleteAlbum{
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
		AlbumId:       albumID,
	}
}
func reqSearchAlbumData(keyword, reqID string, aType protos.AlbumType, sType protos.AlbumSortType, page, number uint64) *protos.ReqSearchAlbum {
	return &protos.ReqSearchAlbum{
		AlbumType:     aType,
		Keyword:       keyword,
		AlbumSortType: sType,
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
		Page:          page,
		Number:        number,
	}
}

func reqInviteData(code, reqID string) *protos.ReqInvite {
	return &protos.ReqInvite{
		WalletAddress:  setting.WalletAddress,
		ReqId:          reqID,
		InvitationCode: code,
	}
}
func reqGetRewardData(reqID string) *protos.ReqGetReward {
	return &protos.ReqGetReward{
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
	}
}

func reqAbstractAlbumData(reqID string) *protos.ReqAbstractAlbum {
	return &protos.ReqAbstractAlbum{
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
	}
}

func reqMyCollectionAlbumData(aType protos.AlbumType, reqID string, page, number uint64, keyword string) *protos.ReqMyCollectionAlbum {
	return &protos.ReqMyCollectionAlbum{
		AlbumType:     aType,
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
		Page:          page,
		Number:        number,
		Keyword:       keyword,
	}

}

func reqFindDirectoryTreeData(reqID, pathHash string) *protos.ReqFindDirectoryTree {
	return &protos.ReqFindDirectoryTree{
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
		PathHash:      pathHash,
	}

}

func uploadSpeedOfProgressData(fileHash string, size uint64) *protos.UploadSpeedOfProgress {
	return &protos.UploadSpeedOfProgress{
		FileHash:  fileHash,
		SliceSize: size,
	}
}

func reqGetCapacityData(reqID string) *protos.ReqGetCapacity {
	return &protos.ReqGetCapacity{
		WalletAddress: setting.WalletAddress,
		ReqId:         reqID,
	}
}
