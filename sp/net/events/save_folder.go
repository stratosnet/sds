package events

import (
	"context"
	"path/filepath"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
	"github.com/qsnetwork/qsds/sp/storages/table"
	"github.com/qsnetwork/qsds/utils"
	"strings"
	"time"
)

// SaveFolder
type SaveFolder struct {
	Server *net.Server
}

// GetServer
func (e *SaveFolder) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *SaveFolder) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *SaveFolder) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqSaveFolder)

	callback := func(message interface{}) (interface{}, string) {

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

		if body.WalletAddress == "" ||
			body.FolderHash == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address or filehash can't be empty"
			return rsp, header.RspSaveFolder
		}

		origFolder := new(table.UserDirectory)
		origFolder.DirHash = body.FolderHash
		if e.GetServer().CT.Fetch(origFolder) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "folder not exist"
			return rsp, header.RspSaveFolder
		}
		rsp.FolderPath = origFolder.Path

		files := origFolder.RecursFindFiles(e.GetServer().CT)

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

		folders := origFolder.RecursFindDirs(e.GetServer().CT)

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


				err := e.GetServer().CT.FetchTable(new(table.UserDirectory), map[string]interface{}{
					"where": map[string]interface{}{
						"path = ? AND wallet_address = ?": []interface{}{path, body.WalletAddress},
					},
				})
				if err != nil {

					insertFolders = append(insertFolders, []interface{}{dirHash, body.WalletAddress, path, time.Now().Unix()})
				}
			}
			e.GetServer().CT.GetDriver().Insert(origFolder.TableName(), columns, insertFolders...)
		}

		if len(files) > 0 {
			userMapFileColumns := []string{"wallet_address", "file_hash"}
			dirMapFileColumns := []string{"dir_hash", "file_hash", "owner"}
			dirMapFile := make([][]interface{}, 0)
			userMapFile := make([][]interface{}, 0)
			for _, file := range files {

				err := e.GetServer().CT.FetchTable(new(table.UserHasFile), map[string]interface{}{
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
					err = e.GetServer().CT.FetchTable(new(table.UserDirectoryMapFile), map[string]interface{}{
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
				e.GetServer().CT.GetDriver().Insert("user_directory_map_file", dirMapFileColumns, dirMapFile...)
			}
			if len(userMapFile) > 0 {
				e.GetServer().CT.GetDriver().Insert("user_has_file", userMapFileColumns, userMapFile...)
			}
		}

		return rsp, header.RspSaveFolder
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
