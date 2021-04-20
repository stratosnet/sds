package events

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"path/filepath"
	"strings"
	"time"
)

// saveFolder is a concrete implementation of event
type saveFolder struct {
	event
}

const saveFolderEvent = "save_folder"

// GetSaveFolderHandler creates event and return handler func for it
func GetSaveFolderHandler(s *net.Server) EventHandleFunc {
	e := saveFolder{newEvent(saveFolderEvent, s, saveFolderCallbackFunc)}
	return e.Handle
}

// saveFolderCallbackFunc is the main process of save folder
func saveFolderCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqSaveFolder)

	rsp := &protos.RspSaveFolder{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
		WalletAddress: body.WalletAddress,
		ReqId:         body.ReqId,
		FolderPath:    "",
		FolderHash:    body.FolderHash,
	}

	if body.WalletAddress == "" || body.FolderHash == "" {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wallet address or file hash can't be empty"
		return rsp, header.RspSaveFolder
	}

	origFolder := &table.UserDirectory{DirHash: body.FolderHash}

	if s.CT.Fetch(origFolder) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "folder not exist"
		return rsp, header.RspSaveFolder
	}

	rsp.FolderPath = origFolder.Path

	files := origFolder.RecursFindFiles(s.CT)

	sPath := ""
	if strings.ContainsRune(origFolder.Path, '/') {
		sPath = filepath.Dir(origFolder.Path)
	}
	dirs := []*protos.FileInfo{
		{
			FileName:           filepath.Base(origFolder.Path),
			FileSize:           0,
			FileHash:           utils.CalcHash([]byte(body.WalletAddress + origFolder.Path)),
			CreateTime:         uint64(origFolder.Time),
			IsDirectory:        true,
			StoragePath:        sPath,
			OwnerWalletAddress: body.WalletAddress,
		},
	}

	folders := origFolder.RecursFindDirs(s.CT)

	if len(folders) > 0 {
		dirs = append(dirs, folders...)
		columns := []string{"dir_hash", "wallet_address", "path", "time"}
		insertFolders := make([][]interface{}, 0)

		for _, dir := range dirs {

			path := dir.FileName
			if dir.StoragePath != "" {
				path = strings.Join([]string{dir.StoragePath, dir.FileName}, "/")
			}
			dirHash := utils.CalcHash([]byte(body.WalletAddress + path))

			err := s.CT.FetchTable(new(table.UserDirectory), map[string]interface{}{
				"where": map[string]interface{}{
					"path = ? AND wallet_address = ?": []interface{}{path, body.WalletAddress},
				},
			})

			if err != nil {
				insertFolders = append(insertFolders, []interface{}{dirHash, body.WalletAddress, path, time.Now().Unix()})
			}
		}
		_, _ = s.CT.GetDriver().Insert(origFolder.TableName(), columns, insertFolders...)
	}

	if len(files) > 0 {
		userMapFileColumns := []string{"wallet_address", "file_hash"}
		dirMapFileColumns := []string{"dir_hash", "file_hash", "owner"}
		dirMapFile := make([][]interface{}, 0)
		userMapFile := make([][]interface{}, 0)
		for _, file := range files {

			err := s.CT.FetchTable(new(table.UserHasFile), map[string]interface{}{
				"where": map[string]interface{}{
					"wallet_address = ? AND file_hash = ?": []interface{}{body.WalletAddress, file.FileHash},
				},
			})
			if err != nil {
				userMapFile = append(userMapFile, []interface{}{body.WalletAddress, file.FileHash})
			}

			dirHash := ""
			if file.StoragePath != "" {
				dirHash = utils.CalcHash([]byte(body.WalletAddress + file.StoragePath))
				err = s.CT.FetchTable(new(table.UserDirectoryMapFile), map[string]interface{}{
					"where": map[string]interface{}{
						"dir_hash = ? AND file_hash = ? AND owner = ?": []interface{}{dirHash, file.FileHash, body.WalletAddress},
					},
				})
				if err != nil && dirHash != "" {
					dirMapFile = append(dirMapFile, []interface{}{dirHash, file.FileHash, body.WalletAddress})
				}
			}
		}
		if len(dirMapFile) > 0 {
			_, _ = s.CT.GetDriver().Insert("user_directory_map_file", dirMapFileColumns, dirMapFile...)
		}
		if len(userMapFile) > 0 {
			_, _ = s.CT.GetDriver().Insert("user_has_file", userMapFileColumns, userMapFile...)
		}
	}

	return rsp, header.RspSaveFolder
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *saveFolder) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqSaveFolder{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
