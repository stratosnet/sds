package events

import (
	"context"
	"encoding/hex"
	"math"
	"path/filepath"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
	"github.com/qsnetwork/qsds/sp/storages/data"
	"github.com/qsnetwork/qsds/sp/storages/table"
	"github.com/qsnetwork/qsds/sp/tools"
	"github.com/qsnetwork/qsds/utils"
	"github.com/qsnetwork/qsds/utils/crypto"
	"strconv"
	"time"
)

// UploadFile
type UploadFile struct {
	Server *net.Server
}

// GetServer
func (e *UploadFile) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *UploadFile) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *UploadFile) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqUploadFile)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqUploadFile)

		sliceSize := e.GetServer().Conf.FileStorage.SliceBlockSize

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


		if ok, msg := e.Validate(body); !ok {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = msg
			return rsp, header.RspUploadFile
		}

		sliceNum := uint64(math.Ceil(float64(body.FileInfo.FileSize) / float64(sliceSize)))

		// query file slice exist or not todo change to read from redis
		res, err := e.GetServer().CT.FetchTables([]table.FileSlice{}, map[string]interface{}{
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
				existsFileSlice := func(fileSlices []table.FileSlice, no, start, end uint64) bool {
					for _, fs := range fileSlices {
						if fs.SliceNumber == no &&
							fs.SliceOffsetStart == start &&
							fs.SliceOffsetEnd == end {

							return true
						}
					}
					return false
				}
				if existsFileSlice(fileSlices, sliceNumber, sliceOffsetStart, sliceOffsetEnd) {
					continue
				}
			}

			key := body.FileInfo.FileHash + "#" + strconv.FormatUint(i, 10)
			missingNodeIds := []string{body.MyAddress.WalletAddress}
			if e.GetServer().HashRing.NodeCount <= 1 {
				missingNodeIds = []string{}
			}
			if _, NodeID := e.GetServer().HashRing.GetNodeMissNodeIDs(key, missingNodeIds); NodeID != "" {

				node := e.GetServer().HashRing.Node(NodeID)

				sliceNumAddr := &protos.SliceNumAddr{
					PpInfo: &protos.PPBaseInfo{
						WalletAddress:  node.ID,
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
			walletAddress = e.GetServer().Conf.FileStorage.PictureLibAddress
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

			e.GetServer().Store(uploadFile, 3600*time.Second)

		} else {



			file := new(table.File)
			file.Hash = body.FileInfo.FileHash
			if e.GetServer().CT.Fetch(file) != nil {
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
				e.GetServer().CT.Save(file)
			} else {

				userHasFile := new(table.UserHasFile)
				err := e.GetServer().CT.FetchTable(userHasFile, map[string]interface{}{
					"where": map[string]interface{}{
						"wallet_address = ? AND file_hash = ?": []interface{}{
							walletAddress, file.Hash,
						},
					},
				})
				if err != nil {
					userHasFile.WalletAddress = walletAddress
					userHasFile.FileHash = file.Hash
					e.GetServer().CT.StoreTable(userHasFile)
				}
			}


			e.GetServer().CT.GetDriver().Delete("user_directory_map_file", map[string]interface{}{
				"owner = ? AND file_hash = ?": []interface{}{
					body.MyAddress.WalletAddress, file.Hash,
				},
			})


			if body.FileInfo.StoragePath != "" {

				dirMapFile := new(table.UserDirectoryMapFile)

				dirMapFile.Owner = body.MyAddress.WalletAddress
				dirMapFile.Path = body.FileInfo.StoragePath
				dirMapFile.FileHash = file.Hash
				dirMapFile.WalletAddress = body.MyAddress.WalletAddress
				dirMapFile.DirHash = dirMapFile.GenericHash()
				e.GetServer().CT.InsertTable(dirMapFile)
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

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}

// Validate
func (e *UploadFile) Validate(req *protos.ReqUploadFile) (bool, string) {

	// check hashring
	if e.GetServer().HashRing.NodeCount <= 0 {
		return false, "no online PP node, try later"
	}


	if req.FileInfo.FileHash == "" ||
		req.FileInfo.FileSize <= 0 {
		return false, "file info invalid"
	}

	if req.MyAddress.WalletAddress == "" {
		return false, "wallet address can't be empty"
	}

	if len(req.Sign) <= 0 {
		return false, "signature can't be empty"
	}

	user := &table.User{WalletAddress: req.MyAddress.WalletAddress}
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

	data := req.MyAddress.WalletAddress + req.FileInfo.FileHash
	if !utils.ECCVerify([]byte(data), req.Sign, puk) {
		return false, "signature verification failed"
	}

	return true, ""
}
