package events

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"github.com/golang/protobuf/proto"
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

// fileStorageInfo is a concrete implementation of event
type fileStorageInfo struct {
	event
}

const fileStorageInfoEvent = "file_storage_info"

// GetFileStorageInfoHandler creates event and return handler func for it
func GetFileStorageInfoHandler(s *net.Server) EventHandleFunc {
	return fileStorageInfo{
		newEvent(fileStorageInfoEvent, s, fileStorageInfoCallbackFunc),
	}.Handle
}

// fileStorageInfoCallbackFunc is the main process of getting file storage info
func fileStorageInfoCallbackFunc(_ context.Context, s *net.Server, message proto.Message, conn spbf.WriteCloser) (proto.Message, string) {
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

	// validate
	storageWalletAddress, fileHash, err := validateFileStorageInfoRequest(s, body)
	if err != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = err.Error()
		return rsp, header.RspFileStorageInfo
	}

	// search file
	file := &table.File{
		Hash: fileHash,
		UserHasFile: table.UserHasFile{
			WalletAddress: storageWalletAddress,
		},
	}
	if s.CT.Fetch(file) != nil || file.State == table.STATE_DELETE {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "wrong downloading address"
		return rsp, header.RspFileStorageInfo
	}

	rsp.FileHash = fileHash

	//  todo change to read from redis

	res, err := s.CT.FetchTables([]table.FileSlice{}, map[string]interface{}{
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

	transferWalletAddress := s.Who(conn.(*spbf.ServerConn).GetName())
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

	for str, row := range fileSliceList {
		storageRing[str] = hashring.New(20)
		downloadFile.List[row.SliceNumber] = false
	}

	for _, row := range fileSlices {
		node := &hashring.Node{ID: row.WalletAddress, Host: row.NetworkAddress}
		if s.HashRing.IsOnline(node.ID) {
			storageRing[row.SliceHash].AddNode(node)
			storageRing[row.SliceHash].SetOnline(node.ID)
		}
	}

	for str, row := range fileSliceList {

		ring := storageRing[str]
		_, provideNodeID := ring.GetNode(row.SliceHash + "#" + uuid.New().String())

		node := ring.Node(provideNodeID)

		if node == nil {
			rsp.Result.Msg = "no resource to process, try later"
			rsp.Result.State = protos.ResultState_RES_FAIL
			return rsp, header.RspFileStorageInfo
		}

		si := &protos.DownloadSliceInfo{
			TaskId:      tools.GenerateTaskID(str),
			SliceNumber: row.SliceNumber,
			SliceStorageInfo: &protos.SliceStorageInfo{
				SliceSize: row.SliceSize,
				SliceHash: row.SliceHash,
			},
			StoragePpInfo: &protos.PPBaseInfo{
				WalletAddress:  node.ID,
				NetworkAddress: node.Host,
			},
			SliceOffset: &protos.SliceOffset{
				SliceOffsetStart: row.SliceOffsetStart,
				SliceOffsetEnd:   row.SliceOffsetEnd,
			},
			VisitResult: true,
		}

		sliceInfo = append(sliceInfo, si)

		task := &data.DownloadTask{
			TaskId:               si.TaskId,
			SliceHash:            row.SliceHash,
			SliceSize:            row.SliceSize,
			StorageWalletAddress: node.ID,
			SliceNumber:          row.SliceNumber,
			Time:                 uint64(time.Now().Unix()),
			WalletAddressList: []string{
				body.FileIndexes.WalletAddress, // download node
				transferWalletAddress,          // transfer node
				node.ID,                        // storage node
				//row.WalletAddress,              // storage node wallet
			},
		}

		// todo change to read from redis
		if err = s.Store(task, 3600*time.Second); err != nil {
			utils.ErrorLogf(eventHandleErrorTemplate, fileStorageInfoEvent, "store task to db", err)
		}
	}

	utils.DebugLog("save download file key = ", downloadFile.GetCacheKey())
	if err = s.Store(downloadFile, 7*24*3600*time.Second); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, fileStorageInfoEvent, "store download file to db", err)
	}

	rsp.FileSize = file.Size
	rsp.SliceInfo = sliceInfo

	return rsp, header.RspFileStorageInfo
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *fileStorageInfo) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := new(protos.ReqFileStorageInfo)
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}

// validateFileStorageInfoRequest validates request
func validateFileStorageInfoRequest(s *net.Server, req *protos.ReqFileStorageInfo) (storageWalletAddress, fileHash string, err error) {

	if len(req.Sign) <= 0 {
		err = errors.New("signature can't be empty")
		return
	}

	if req.FileIndexes.WalletAddress == "" {
		err = errors.New("wallet address can't be empty")
		return
	}

	filePath := req.FileIndexes.FilePath

	if filePath == "" {
		err = errors.New("file path can't be empty")
		return
	}

	_, storageWalletAddress, fileHash, _, err = tools.ParseFileHandle(filePath)

	if err != nil {
		err = errors.New("wrong file path format, failed to parse")
		return
	}

	user := &table.User{WalletAddress: req.FileIndexes.WalletAddress}
	if s.CT.Fetch(user) != nil {
		err = errors.New("not authorized to process")
		return
	}
	var pukInByte []byte
	pukInByte, err = hex.DecodeString(user.Puk)
	if err != nil {
		return
	}

	var puk *ecdsa.PublicKey
	puk, err = crypto.UnmarshalPubkey(pukInByte)
	if err != nil {
		return
	}

	d := req.FileIndexes.WalletAddress + fileHash
	if !utils.ECCVerify([]byte(d), req.Sign, puk) {
		err = errors.New("signature verification failed")
		return
	}

	return
}
