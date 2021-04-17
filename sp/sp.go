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

	spbf.Register(header.ReqMining, events.MiningHandler(server))       // server.ListenEvent(header.ReqMining, new(events.Mining))
	spbf.Register(header.ReqRegister, events.RegisterHandler(server))   // server.ListenEvent(header.ReqRegister, new(events.Register))
	spbf.Register(header.ReqGetPPList, events.GetPPListHandler(server)) // server.ListenEvent(header.ReqGetPPList, new(events.GetPPList))

	server.ListenEvent(header.ReqRegisterNewPP, new(events.RegisterNewPP))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqUploadFile, new(events.UploadFile))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqReportUploadSliceResult, new(events.ReportUploadSliceResult))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqFindMyFileList, new(events.FindMyFileList))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqFileStorageInfo, new(events.FileStorageInfo))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqReportDownloadResult, new(events.ReportDownloadResult))
	spbf.Register(header, events)

	server.ListenEvent(header.RspTransferNotice, new(events.TransferNotice))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqValidateTransferCer, new(events.TransferCerValidate))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqReportTransferResult, new(events.ReportTransferResult))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqGetBPList, new(events.GetBPList))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqDownloadTaskInfo, new(events.DownloadTaskInfo))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqDeleteFile, new(events.DeleteFile))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqDownloadSliceWrong, new(events.DownloadFailed))
	spbf.Register(header, events)

	server.ListenEvent(header.RspDeleteSlice, new(events.DeleteSlice))
	spbf.Register(header, events)

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

	server.ListenEvent(header.ReqDeleteShare, new(events.DeleteShare))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqCreateAlbum, new(events.CreateAlbum))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqEditAlbum, new(events.EditAlbum))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqAlbumContent, new(events.AlbumContent))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqFindMyAlbum, new(events.FindMyAlbum))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqSearchAlbum, new(events.SearchAlbum))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqInvite, new(events.Invite))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqGetReward, new(events.GetReward))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqCollectionAlbum, new(events.CollectAlbum))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqMyCollectionAlbum, new(events.MyCollectAlbum))
	spbf.Register(header, events)

	spbf.Register(header.ReqAbstractAlbum, events.AbstractAlbumHandler(server)) // server.ListenEvent(header.ReqAbstractAlbum, new(events.AbstractAlbum))

	server.ListenEvent(header.ReqFindDirectoryTree, new(events.FindDirectoryTree))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqDeleteAlbum, new(events.DeleteAlbum))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqGetCapacity, new(events.GetCapacity))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqFileSort, new(events.AlbumFileSort))
	spbf.Register(header, events)

	server.ListenEvent(header.ReqFindDirectory, new(events.FindDirectory))
	spbf.Register(header, events)

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
