package events

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/sp/tools"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/hashring"
	"time"

	"github.com/google/uuid"
)

// FileStorageInfo
type FileStorageInfo struct {
	Server *net.Server
}

// GetServer
func (e *FileStorageInfo) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *FileStorageInfo) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *FileStorageInfo) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqFileStorageInfo)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqFileStorageInfo)

		rsp := &protos.RspFileStorageInfo{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.FileIndexes.WalletAddress,
			ReqId:         body.ReqId,
			SavePath:      body.FileIndexes.SavePath,
			VisitCer:      "",
			FileHash:      "",
			FileName:      "",
			SliceInfo:     nil,
			FileSize:      0,
		}

		var fileHash string
		var storageWalletAddress string

		// validate
		if ok, errMsg := e.Validate(body, &fileHash, &storageWalletAddress); !ok {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = errMsg
			return rsp, header.RspFileStorageInfo
		}

		// search file
		file := new(table.File)
		file.Hash = fileHash
		file.WalletAddress = storageWalletAddress
		if e.GetServer().CT.Fetch(file) != nil || file.State == table.STATE_DELETE {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wrong downloading address"
			return rsp, header.RspFileStorageInfo
		}

		rsp.FileHash = fileHash

		//  todo change to read from redis

		res, err := e.GetServer().CT.FetchTables([]table.FileSlice{}, map[string]interface{}{
			"alias":   "e",
			"columns": "e.*, fss.wallet_address, fss.network_address",
			"where":   map[string]interface{}{"e.file_hash = ?": fileHash},
			"join":    []string{"file_slice_storage", "e.slice_hash = fss.slice_hash", "fss", "left"},
			"orderBy": "e.slice_number ASC",
		})
		if err != nil {
			utils.ErrorLog(err.Error())
		}

		fileSlices := res.([]table.FileSlice)

		var sliceInfo []*protos.DownloadSliceInfo
		if len(fileSlices) <= 0 || err != nil {

			rsp.Result.Msg = "file not exist"
			rsp.Result.State = protos.ResultState_RES_FAIL
			return rsp, header.RspFileStorageInfo
		}

		transferWalletAddress := e.GetServer().Who(conn.(*spbf.ServerConn).GetName())
		if transferWalletAddress == "" {
			rsp.Result.Msg = "not miner"
			rsp.Result.State = protos.ResultState_RES_FAIL
			return rsp, header.RspFileStorageInfo
		}

		rsp.FileName = file.Name

		downloadFile := &data.DownloadFile{
			WalletAddress: body.FileIndexes.WalletAddress,
			FileHash:      fileHash,
			FileName:      file.Name,
			SliceNum:      file.SliceNum,
			List:          make(map[uint64]bool),
		}

		// create hashring
		storageRing := make(map[string]*hashring.HashRing)
		fileSliceList := make(map[string]table.FileSlice)

		for _, row := range fileSlices {
			fileSliceList[row.SliceHash] = row
		}

		for s, row := range fileSliceList {
			storageRing[s] = hashring.New(20)
			downloadFile.List[row.SliceNumber] = false
		}

		for _, row := range fileSlices {
			node := &hashring.Node{ID: row.WalletAddress, Host: row.NetworkAddress}
			if e.GetServer().HashRing.IsOnline(node.ID) {
				storageRing[row.SliceHash].AddNode(node)
				storageRing[row.SliceHash].SetOnline(node.ID)
			}
		}

		for s, row := range fileSliceList {

			ring := storageRing[s]
			_, provideNodeID := ring.GetNode(row.SliceHash + "#" + uuid.New().String())

			node := ring.Node(provideNodeID)

			if node == nil {
				rsp.Result.Msg = "no resource to process, try later"
				rsp.Result.State = protos.ResultState_RES_FAIL
				return rsp, header.RspFileStorageInfo
			}

			si := new(protos.DownloadSliceInfo)
			si.TaskId = tools.GenerateTaskID(s)
			si.SliceNumber = row.SliceNumber
			si.SliceStorageInfo = &protos.SliceStorageInfo{
				SliceSize: row.SliceSize,
				SliceHash: row.SliceHash,
			}
			si.StoragePpInfo = &protos.PPBaseInfo{
				WalletAddress:  node.ID,
				NetworkAddress: node.Host,
			}
			si.SliceOffset = new(protos.SliceOffset)
			si.SliceOffset.SliceOffsetStart = row.SliceOffsetStart
			si.SliceOffset.SliceOffsetEnd = row.SliceOffsetEnd
			si.VisitResult = true
			sliceInfo = append(sliceInfo, si)

			task := new(data.DownloadTask)
			task.TaskId = si.TaskId
			task.SliceHash = row.SliceHash
			task.SliceSize = row.SliceSize
			task.StorageWalletAddress = node.ID
			task.WalletAddressList = []string{
				body.FileIndexes.WalletAddress, // download node
				transferWalletAddress,          // transfer node
				node.ID,                        // storage node
				//row.WalletAddress,              // storage node wallet
			}

			task.SliceNumber = row.SliceNumber
			task.Time = uint64(time.Now().Unix())

			// todo change to read from redis
			e.GetServer().Store(task, 3600*time.Second)
		}

		fmt.Println("save download file key = ", downloadFile.GetCacheKey())
		e.GetServer().Store(downloadFile, 7*24*3600*time.Second)

		rsp.FileSize = file.Size
		rsp.SliceInfo = sliceInfo

		return rsp, header.RspFileStorageInfo
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}

// Validate
func (e *FileStorageInfo) Validate(req *protos.ReqFileStorageInfo, fileHash *string, storageWalletAddress *string) (bool, string) {

	if len(req.Sign) <= 0 {
		return false, "signature can't be empty"
	}

	if req.FileIndexes.WalletAddress == "" {
		return false, "wallet address can't be empty"
	}

	filePath := req.FileIndexes.FilePath

	if filePath == "" {
		return false, "file path can't be empty"
	}

	var err error
	_, *storageWalletAddress, *fileHash, _, err = tools.ParseFileHandle(filePath)

	if err != nil {
		return false, "wrong file path format, failed to parse"
	}

	user := &table.User{WalletAddress: req.FileIndexes.WalletAddress}
	if e.GetServer().CT.Fetch(user) != nil {
		return false, "not authorized to process"
	}

	pukInByte, err := hex.DecodeString(user.Puk)
	if err != nil {
		return false, err.Error()
	}

	puk, err := crypto.UnmarshalPubkey(pukInByte)
	if err != nil {
		return false, err.Error()
	}

	data := req.FileIndexes.WalletAddress + *fileHash
	if !utils.ECCVerify([]byte(data), req.Sign, puk) {
		return false, "signature verification failed"
	}

	return true, ""
}
