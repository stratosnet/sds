package sp

import (
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/sp/api/core"
	"github.com/qsnetwork/sds/sp/api/handlers"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/net/events"
	"github.com/qsnetwork/sds/utils"
)

// StartSP
func StartSP(conf string) *net.Server {

	server := net.NewServer(conf)

	server.ListenEvent(header.ReqMining, new(events.Mining))
	server.ListenEvent(header.ReqRegister, new(events.Register))
	server.ListenEvent(header.ReqGetPPList, new(events.GetPPList))
	server.ListenEvent(header.ReqRegisterNewPP, new(events.RegisterNewPP))
	server.ListenEvent(header.ReqUploadFile, new(events.UploadFile))
	server.ListenEvent(header.ReqReportUploadSliceResult, new(events.ReportUploadSliceResult))
	server.ListenEvent(header.ReqFindMyFileList, new(events.FindMyFileList))
	server.ListenEvent(header.ReqFileStorageInfo, new(events.FileStorageInfo))
	server.ListenEvent(header.ReqReportDownloadResult, new(events.ReportDownloadResult))
	server.ListenEvent(header.RspTransferNotice, new(events.TransferNotice))
	server.ListenEvent(header.ReqValidateTransferCer, new(events.TransferCerValidate))
	server.ListenEvent(header.ReqReportTransferResult, new(events.ReportTransferResult))
	server.ListenEvent(header.ReqGetBPList, new(events.GetBPList))
	server.ListenEvent(header.ReqDownloadTaskInfo, new(events.DownloadTaskInfo))
	server.ListenEvent(header.ReqDeleteFile, new(events.DeleteFile))
	server.ListenEvent(header.ReqDownloadSliceWrong, new(events.DownloadFailed))
	server.ListenEvent(header.RspDeleteSlice, new(events.DeleteSlice))
	server.ListenEvent(header.RspGetHDInfo, new(events.GetHDInfo))
	server.ListenEvent(header.ReqMakeDirectory, new(events.MakeDirectory))
	server.ListenEvent(header.ReqRemoveDirectory, new(events.RemoveDirectory))
	server.ListenEvent(header.ReqMoveFileDirectory, new(events.MoveFileDirectory))
	server.ListenEvent(header.ReqSaveFile, new(events.SaveFile))
	server.ListenEvent(header.ReqSaveFolder, new(events.SaveFolder))
	server.ListenEvent(header.ReqShareFile, new(events.ShareFile))
	server.ListenEvent(header.ReqGetShareFile, new(events.GetShareFile))
	server.ListenEvent(header.ReqShareLink, new(events.ShareLink))
	server.ListenEvent(header.ReqConfig, new(events.GetConfig))
	server.ListenEvent(header.ReqDeleteShare, new(events.DeleteShare))
	server.ListenEvent(header.ReqCreateAlbum, new(events.CreateAlbum))
	server.ListenEvent(header.ReqEditAlbum, new(events.EditAlbum))
	server.ListenEvent(header.ReqAlbumContent, new(events.AlbumContent))
	server.ListenEvent(header.ReqFindMyAlbum, new(events.FindMyAlbum))
	server.ListenEvent(header.ReqSearchAlbum, new(events.SearchAlbum))
	server.ListenEvent(header.ReqInvite, new(events.Invite))
	server.ListenEvent(header.ReqGetReward, new(events.GetReward))
	server.ListenEvent(header.ReqCollectionAlbum, new(events.CollectAlbum))
	server.ListenEvent(header.ReqMyCollectionAlbum, new(events.MyCollectAlbum))
	server.ListenEvent(header.ReqAbstractAlbum, new(events.AbstractAlbum))
	server.ListenEvent(header.ReqFindDirectoryTree, new(events.FindDirectoryTree))
	server.ListenEvent(header.ReqDeleteAlbum, new(events.DeleteAlbum))
	server.ListenEvent(header.ReqGetCapacity, new(events.GetCapacity))
	server.ListenEvent(header.ReqFileSort, new(events.AlbumFileSort))
	server.ListenEvent(header.ReqFindDirectory, new(events.FindDirectory))

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
