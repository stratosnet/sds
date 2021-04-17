package sp

import (
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/sp/api/core"
	"github.com/stratosnet/sds/sp/api/handlers"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/net/events"
	"github.com/stratosnet/sds/utils"
)

// StartSP
func StartSP(conf string) *net.Server {

	server := net.NewServer(conf)

	spbf.Register(header.ReqMining, events.GetMiningHandler(server))                                   // server.ListenEvent(header.ReqMining, new(events.mining))
	spbf.Register(header.ReqRegister, events.GetRegisterHandler(server))                               // server.ListenEvent(header.ReqRegister, new(events.register))
	spbf.Register(header.ReqGetPPList, events.GetPPListHandler(server))                                // server.ListenEvent(header.ReqGetPPList, new(events.getPPList))
	spbf.Register(header.ReqRegisterNewPP, events.GetRegisterNewPPHandler(server))                     // server.ListenEvent(header.ReqRegisterNewPP, new(events.registerNewPP))
	spbf.Register(header.ReqUploadFile, events.GetUploadFileHandler(server))                           // server.ListenEvent(header.ReqUploadFile, new(events.uploadFile))
	spbf.Register(header.ReqReportUploadSliceResult, events.GetReportUploadSliceResultHandler(server)) // server.ListenEvent(header.ReqReportUploadSliceResult, new(events.reportUploadSliceResult))
	spbf.Register(header.ReqFindMyFileList, events.GetFindMyFileListHandler(server))                   // server.ListenEvent(header.ReqFindMyFileList, new(events.findMyFileList))
	spbf.Register(header.ReqFileStorageInfo, events.GetFileStorageInfoHandler(server))                 // server.ListenEvent(header.ReqFileStorageInfo, new(events.fileStorageInfo))
	spbf.Register(header.ReqReportDownloadResult, events.GetReportDownloadResultHandler(server))       // server.ListenEvent(header.ReqReportDownloadResult, new(events.reportDownloadResult))
	spbf.Register(header.RspTransferNotice, events.GetTransferNoticeHandler(server))                   // server.ListenEvent(header.RspTransferNotice, new(events.transferNotice))
	spbf.Register(header.ReqValidateTransferCer, events.GetTransferCerValidateHandler(server))         // server.ListenEvent(header.ReqValidateTransferCer, new(events.transferCerValidate))
	spbf.Register(header.ReqReportTransferResult, events.GetReportTransferResultHandler(server))       // server.ListenEvent(header.ReqReportTransferResult, new(events.reportTransferResult))
	spbf.Register(header.ReqGetBPList, events.GetBPListHandler(server))                                // server.ListenEvent(header.ReqGetBPList, new(events.getBPList))
	spbf.Register(header.ReqDownloadTaskInfo, events.GetDownloadTaskInfoHandler(server))               // server.ListenEvent(header.ReqDownloadTaskInfo, new(events.downloadTaskInfo))
	spbf.Register(header.ReqDeleteFile, events.GetDeleteFileHandler(server))                           //	server.ListenEvent(header.ReqDeleteFile, new(events.deleteFile))
	spbf.Register(header.ReqDownloadSliceWrong, events.GetDownloadFailedHandler(server))               //	server.ListenEvent(header.ReqDownloadSliceWrong, new(events.downloadFailed))
	spbf.Register(header.RspDeleteSlice, events.GetDeleteSliceHandler(server))                         // server.ListenEvent(header.RspDeleteSlice, new(events.deleteSlice))

	server.ListenEvent(header.RspGetHDInfo, new(events.GetHDInfo))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqMakeDirectory, new(events.MakeDirectory))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqRemoveDirectory, new(events.RemoveDirectory))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqMoveFileDirectory, new(events.MoveFileDirectory))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqSaveFile, new(events.SaveFile))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqSaveFolder, new(events.SaveFolder))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqShareFile, new(events.ShareFile))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqGetShareFile, new(events.GetShareFile))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqShareLink, new(events.ShareLink))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqConfig, new(events.GetConfig))
	spbf.Register(header, events)

	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqReportTransferResult, new(events.reportTransferResult))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqGetBPList, new(events.getBPList))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqShareFile, new(events.ShareFile))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqDeleteFile, new(events.deleteFile))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqDownloadSliceWrong, new(events.downloadFailed))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.RspDeleteSlice, new(events.deleteSlice))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.RspGetHDInfo, new(events.GetHDInfo))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqMakeDirectory, new(events.MakeDirectory))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqRemoveDirectory, new(events.RemoveDirectory))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqMoveFileDirectory, new(events.MoveFileDirectory))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqSaveFile, new(events.SaveFile))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqSaveFolder, new(events.SaveFolder))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqShareFile, new(events.ShareFile))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqGetShareFile, new(events.GetShareFile))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqShareLink, new(events.ShareLink))
	spbf.Register(header, events)                                               // 	server.ListenEvent(header.ReqConfig, new(events.GetConfig))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqDeleteShare, new(events.DeleteShare))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqCreateAlbum, new(events.CreateAlbum))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqEditAlbum, new(events.EditAlbum))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqAlbumContent, new(events.AlbumContent))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqFindMyAlbum, new(events.FindMyAlbum))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqSearchAlbum, new(events.SearchAlbum))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqInvite, new(events.Invite))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqGetReward, new(events.GetReward))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqCollectionAlbum, new(events.CollectAlbum))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqMyCollectionAlbum, new(events.MyCollectAlbum))
	spbf.Register(header.ReqAbstractAlbum, events.AbstractAlbumHandler(server)) // server.ListenEvent(header.ReqAbstractAlbum, new(events.AbstractAlbum))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqFindDirectoryTree, new(events.FindDirectoryTree))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqGetCapacity, new(events.GetCapacity))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqFileSort, new(events.AlbumFileSort))
	spbf.Register(header, events)                                               //	server.ListenEvent(header.ReqFindDirectory, new(events.FindDirectory))

	server.Start()

	return server
}

// StartAPI
func StartAPI(conf string) *core.APIServer {

	if conf == "" {
		utils.ErrorLog("empty configuration specified")
		return nil
	}

	config := new(core.Config)
	utils.LoadYamlConfig(config, conf)

	server := core.NewAPIServer(config)

	server.AddHandler("POST", "/login", new(handlers.User), "Login")
	server.AddHandler("GET", "/data/statistics", new(handlers.Data), "Statistics")
	server.AddHandler("GET", "/pp", new(handlers.PP), "List")
	server.AddHandler("POST", "/pp/{wa:[0-9a-zA-Z]+}/backup", new(handlers.PP), "Backup")
	server.AddHandler("GET", "/file", new(handlers.File), "List")
	server.AddHandler("GET", "/file/{hash:[0-9a-zA-Z]+}/slice", new(handlers.File), "Slice")
	server.AddHandler("GET", "/sys", new(handlers.Sys), "Setting")
	server.AddHandler("POST", "/sys", new(handlers.Sys), "Save")
	server.AddHandler("POST", "/sys/client_download", new(handlers.Sys), "ClientDownload")

	server.Start()

	return server
}
