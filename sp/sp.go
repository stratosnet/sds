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

// StartSP starts SP node
func StartSP(conf string) {

	server := net.NewServer(conf)

	spbf.Register(header.ReqMining, events.GetMiningHandler(server))
	spbf.Register(header.ReqRegister, events.GetRegisterHandler(server))
	spbf.Register(header.ReqActivate, events.GetActivateHandler(server))
	spbf.Register(header.ReqActivated, events.GetActivatedHandler(server))
	spbf.Register(header.ReqGetPPList, events.GetPPListHandler(server))
	spbf.Register(header.ReqRegisterNewPP, events.GetRegisterNewPPHandler(server))
	spbf.Register(header.ReqUploadFile, events.GetUploadFileHandler(server))
	spbf.Register(header.ReqReportUploadSliceResult, events.GetReportUploadSliceResultHandler(server))
	spbf.Register(header.ReqFindMyFileList, events.GetFindMyFileListHandler(server))
	spbf.Register(header.ReqFileStorageInfo, events.GetFileStorageInfoHandler(server))
	spbf.Register(header.ReqReportDownloadResult, events.GetReportDownloadResultHandler(server))
	spbf.Register(header.RspTransferNotice, events.GetTransferNoticeHandler(server))
	spbf.Register(header.ReqValidateTransferCer, events.GetTransferCerValidateHandler(server))
	spbf.Register(header.ReqReportTransferResult, events.GetReportTransferResultHandler(server))
	spbf.Register(header.ReqDownloadTaskInfo, events.GetDownloadTaskInfoHandler(server))
	spbf.Register(header.ReqDeleteFile, events.GetDeleteFileHandler(server))
	spbf.Register(header.ReqDownloadSliceWrong, events.GetDownloadFailedHandler(server))
	spbf.Register(header.RspDeleteSlice, events.GetDeleteSliceHandler(server))
	spbf.Register(header.RspGetHDInfo, events.GetHDInfoHandler(server))
	spbf.Register(header.ReqMakeDirectory, events.GetMakeDirHandler(server))
	spbf.Register(header.ReqRemoveDirectory, events.GetRmDirHandler(server))
	spbf.Register(header.ReqMoveFileDirectory, events.GetMoveFileDirHandler(server))
	spbf.Register(header.ReqSaveFile, events.GetSaveFileHandler(server))
	spbf.Register(header.ReqSaveFolder, events.GetSaveFolderHandler(server))
	spbf.Register(header.ReqShareFile, events.GetShareFileHandler(server))
	spbf.Register(header.ReqGetShareFile, events.GetGetShareFileHandler(server))
	spbf.Register(header.ReqShareLink, events.GetShareLinkHandler(server))
	spbf.Register(header.ReqConfig, events.GetGetConfigHandler(server))
	spbf.Register(header.ReqDeleteShare, events.GetDeleteShareHandler(server))
	spbf.Register(header.ReqCreateAlbum, events.GetCreateAlbumHandler(server))
	spbf.Register(header.ReqEditAlbum, events.GetEditAlbumHandler(server))
	spbf.Register(header.ReqAlbumContent, events.GetAlbumContentHandler(server))
	spbf.Register(header.ReqFindMyAlbum, events.GetFindMyAlbumHandler(server))
	spbf.Register(header.ReqSearchAlbum, events.GetSearchAlbumHandler(server))
	spbf.Register(header.ReqInvite, events.GetInviteHandler(server))
	spbf.Register(header.ReqGetReward, events.GetGetRewardHandler(server))
	spbf.Register(header.ReqCollectionAlbum, events.GetCollectAlbumHandler(server))
	spbf.Register(header.ReqMyCollectionAlbum, events.GetMyCollectAlbumHandler(server))
	spbf.Register(header.ReqAbstractAlbum, events.AbstractAlbumHandler(server))
	spbf.Register(header.ReqFindDirectoryTree, events.GetFindDirectoryTreeHandler(server))
	spbf.Register(header.ReqDeleteAlbum, events.GetDeleteAlbumHandler(server))
	spbf.Register(header.ReqGetCapacity, events.GetGetCapacityHandler(server))
	spbf.Register(header.ReqFileSort, events.GetFileSortHandler(server))
	spbf.Register(header.ReqFindDirectory, events.GetFindDirectoryHandler(server))
	spbf.Register(header.ReqCAddVolume, events.GetCAddVolumeHandler(server))
	spbf.Register(header.ReqCUseVolume, events.GetCUseVolumeHandler(server))

	server.Start()
}

// StartAPI start the SP user API
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
