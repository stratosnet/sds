package events

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/sp/tools"
	"github.com/stratosnet/sds/utils"
	"math"
	"path/filepath"
	"strconv"
	"time"
)

// uploadFile is a concrete implementation of event
type uploadFile struct {
	event
}

const uploadFileEvent = "upload_file"

// GetUploadFileHandler creates event and return handler func for it
func GetUploadFileHandler(s *net.Server) EventHandleFunc {
	e := uploadFile{newEvent(uploadFileEvent, s, uploadFileCallbackFunc)}
	return e.Handle
}

// uploadFileCallbackFunc is the main process of uploading file
func uploadFileCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqUploadFile)

	sliceSize := s.Conf.FileStorage.SliceBlockSize

	rsp := &protos.RspUploadFile{
		StorageCer: "",
		FileHash:   body.FileInfo.FileHash,
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
			Msg:   "ok",
		},
		PpList: nil,
		ReqId:  body.ReqId,
	}

	if ok, msg := validateUploadFileRequest(body, s); !ok {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = msg
		return rsp, header.RspUploadFile
	}

	sliceNum := uint64(math.Ceil(float64(body.FileInfo.FileSize) / float64(sliceSize)))

	// query file slice exist or not todo change to read from redis
	res, err := s.CT.FetchTables([]table.FileSlice{}, map[string]interface{}{
		"where": map[string]interface{}{
			"file_hash = ?": body.FileInfo.FileHash,
		},
	})
	if err != nil {
		utils.ErrorLog(err.Error())
	}

	fileSlices := res.([]table.FileSlice)
	slices := make([]*protos.SliceNumAddr, 0)

	var i uint64
	for i = 1; i <= sliceNum; i++ {

		sliceNumber := i
		sliceOffsetStart := (i - 1) * sliceSize

		var sliceOffsetEnd uint64
		if i == sliceNum {
			sliceOffsetEnd = body.FileInfo.FileSize
		} else {
			sliceOffsetEnd = i * sliceSize
		}

		//

		if len(fileSlices) > 0 {
			if existsFileSlice(fileSlices, sliceNumber, sliceOffsetStart, sliceOffsetEnd) {
				utils.DebugLogf(eventHandleErrorTemplate, uploadFileEvent, "file not uploaded", "file slice already existed")
				continue
			}
		}

		key := body.FileInfo.FileHash + "#" + strconv.FormatUint(i, 10)
		missingNodeIds := []string{body.MyAddress.P2PAddress}
		if s.HashRing.NodeCount <= 1 {
			missingNodeIds = []string{}
		}
		if _, NodeID := s.HashRing.GetNodeExcludedNodeIDs(key, missingNodeIds); NodeID != "" {

			node := s.HashRing.Node(NodeID)

			sliceNumAddr := &protos.SliceNumAddr{
				PpInfo: &protos.PPBaseInfo{
					P2PAddress:     node.ID,
					NetworkAddress: node.Host,
				},
				SliceNumber: sliceNumber,
				SliceOffset: &protos.SliceOffset{
					SliceOffsetStart: sliceOffsetStart,
					SliceOffsetEnd:   sliceOffsetEnd,
				},
			}

			slices = append(slices, sliceNumAddr)
		}
	}

	taskID := tools.GenerateTaskID(body.FileInfo.FileHash)

	walletAddress := body.MyAddress.WalletAddress
	var isCover byte
	if body.IsCover {
		isCover = table.IS_COVER
		walletAddress = s.Conf.FileStorage.PictureLibAddress
	}

	if len(slices) > 0 {
		uploadFile := &data.UploadFile{
			Key:           taskID,
			TaskID:        taskID,
			FileHash:      body.FileInfo.FileHash,
			FileName:      body.FileInfo.FileName,
			FileSize:      body.FileInfo.FileSize,
			FilePath:      body.FileInfo.StoragePath,
			SliceNum:      uint64(len(slices)),
			WalletAddress: walletAddress,
			IsCover:       body.IsCover,
			List:          make(map[uint64]bool),
		}

		for _, fs := range slices {
			uploadFile.List[fs.SliceNumber] = false
		}

		if err = s.Store(uploadFile, 3600*time.Second); err != nil {
			utils.ErrorLogf(eventHandleErrorTemplate, uploadFileEvent, "store file to DB", err)
		}

	} else {

		file := &table.File{Hash: body.FileInfo.FileHash}
		if s.CT.Fetch(file) != nil {
			file.Name = body.FileInfo.FileName
			if len(body.FileInfo.FileName) > 128 {
				suffix := filepath.Ext(body.FileInfo.FileName)
				file.Name = body.FileInfo.FileName[0:(128-len(suffix))] + suffix
			}
			file.Hash = body.FileInfo.FileHash
			file.Size = body.FileInfo.FileSize
			file.SliceNum = sliceNum
			file.State = table.STATE_OK
			file.Time = time.Now().Unix()
			file.WalletAddress = walletAddress
			file.IsCover = isCover
			if err = s.CT.Save(file); err != nil {
				utils.ErrorLogf(eventHandleErrorTemplate, uploadFileEvent, "save file to DB", err)
			}
		} else {

			userHasFile := &table.UserHasFile{}
			err = s.CT.FetchTable(userHasFile, map[string]interface{}{
				"where": map[string]interface{}{
					"wallet_address = ? AND file_hash = ?": []interface{}{
						walletAddress, file.Hash,
					},
				},
			})
			if err != nil {
				userHasFile.WalletAddress = walletAddress
				userHasFile.FileHash = file.Hash
				if _, err = s.CT.StoreTable(userHasFile); err != nil {
					utils.ErrorLogf(eventHandleErrorTemplate, uploadFileEvent, "create table in DB", err)
				}
			}
		}

		s.CT.GetDriver().Delete("user_directory_map_file", map[string]interface{}{
			"owner = ? AND file_hash = ?": []interface{}{
				body.MyAddress.WalletAddress, file.Hash,
			},
		})

		if body.FileInfo.StoragePath != "" {
			dirMapFile := &table.UserDirectoryMapFile{
				UserDirectory: table.UserDirectory{
					Path:          body.FileInfo.StoragePath,
					WalletAddress: body.MyAddress.WalletAddress,
				},
				Owner:    body.MyAddress.WalletAddress,
				FileHash: file.Hash,
			}
			dirMapFile.DirHash = dirMapFile.GenericHash()
			if _, err = s.CT.InsertTable(dirMapFile); err != nil {
				utils.ErrorLogf(eventHandleErrorTemplate, uploadFileEvent, "insert file to DB", err)
			}
		}
	}

	rsp.OwnerWalletAddress = walletAddress
	rsp.FileHash = body.FileInfo.FileHash
	rsp.PpList = slices
	rsp.TaskId = taskID
	rsp.TotalSlice = int64(sliceNum)
	rsp.Result.State = protos.ResultState_RES_SUCCESS
	rsp.ReqId = body.ReqId

	return rsp, header.RspUploadFile
}

// existsFileSlice checks
func existsFileSlice(fileSlices []table.FileSlice, no, start, end uint64) bool {
	for _, fs := range fileSlices {
		if fs.SliceNumber == no && fs.SliceOffsetStart == start && fs.SliceOffsetEnd == end {
			return true
		}
	}
	return false
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *uploadFile) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqUploadFile{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}

// validateUploadFileRequest validates request
func validateUploadFileRequest(req *protos.ReqUploadFile, s *net.Server) (bool, string) {

	// check hashring
	if s.HashRing.NodeCount <= 0 {
		return false, "no online PP node, try later"
	}

	if req.FileInfo.FileHash == "" || req.FileInfo.FileSize <= 0 {
		return false, "file info invalid"
	}

	if req.MyAddress.P2PAddress == "" {
		return false, "P2P key address can't be empty"
	}

	if len(req.Sign) <= 0 {
		return false, "signature can't be empty"
	}

	user := &table.User{P2PAddress: req.MyAddress.P2PAddress}
	if err := s.CT.Fetch(user); err != nil {
		return false, "not authorized to process"
	}

	puk, err := hex.DecodeString(user.Puk)
	if err != nil {
		return false, err.Error()
	}

	d := req.MyAddress.P2PAddress + req.FileInfo.FileHash
	if !ed25519.Verify(puk, []byte(d), req.Sign) {
		return false, "signature verification failed"
	}

	return true, ""
}
