package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/common"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/hashring"
	"time"

	"github.com/google/uuid"
)

// DownloadFailed
type DownloadFailed struct {
	Server *net.Server
}

// GetServer
func (e *DownloadFailed) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *DownloadFailed) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *DownloadFailed) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqDownloadSloceWrong)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqDownloadSloceWrong)

		rsp := &protos.RspDownloadSloceWrong{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress: body.WalletAddress,
			TaskId:        body.TaskId,
			NewSliceInfo:  nil,
			FileHash:      "",
		}

		if body.TaskId == "" || body.SliceHash == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "task ID or slice hash can't be empty"
			return rsp, header.RspDownloadSliceWrong
		}

		task := &data.DownloadTask{
			TaskId: body.TaskId,
		}
		if e.GetServer().Load(task) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "task is finished or not exist"
			return rsp, header.RspDownloadSliceWrong
		}

		res, err := e.GetServer().CT.FetchTables([]table.FileSliceStorage{}, map[string]interface{}{
			"where": map[string]interface{}{
				"slice_hash = ?": body.SliceHash,
			},
		})

		if err != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "no resource to process, try later"
			return rsp, header.RspDownloadSliceWrong
		}

		sliceStorage := res.([]table.FileSliceStorage)

		if len(sliceStorage) > 0 {

			ring := hashring.New(e.GetServer().Conf.HashRing.VirtualNodeNum)
			for _, storage := range sliceStorage {
				if storage.WalletAddress != task.StorageWalletAddress {
					if e.GetServer().HashRing.IsOnline(storage.WalletAddress) {
						ring.AddNode(&hashring.Node{ID: storage.WalletAddress, Host: storage.NetworkAddress})
						ring.SetOnline(storage.WalletAddress)
					}
				}
			}

			_, anotherWalletAddress := ring.GetNode(utils.CalcHash([]byte(uuid.New().String() + body.SliceHash)))

			if anotherWalletAddress == "" {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "no resource to process, try later"
				return rsp, header.RspDownloadSliceWrong
			}

			fileSlice := new(table.FileSlice)
			fileSlice.SliceHash = task.SliceHash
			fileSlice.WalletAddress = anotherWalletAddress

			if e.GetServer().CT.Fetch(fileSlice) == nil {


				fileSliceStorage := new(table.FileSliceStorage)
				fileSliceStorage.SliceHash = task.SliceHash
				fileSliceStorage.WalletAddress = task.StorageWalletAddress

				e.GetServer().CT.DeleteTable(fileSliceStorage)

				e.GetServer().HandleMsg(&common.MsgTransferNotice{
					SliceHash:         fileSlice.SliceHash,
					FromWalletAddress: fileSlice.WalletAddress,
					ToWalletAddress:   task.StorageWalletAddress,
				})

				task.StorageWalletAddress = anotherWalletAddress

				e.GetServer().Store(task, 3600*time.Second)

				rsp.FileHash = fileSlice.FileHash
				rsp.NewSliceInfo = &protos.DownloadSliceInfo{
					SliceStorageInfo: &protos.SliceStorageInfo{
						SliceHash: fileSlice.SliceHash,
						SliceSize: fileSlice.SliceSize,
					},
					SliceNumber: fileSlice.SliceNumber,
					StoragePpInfo: &protos.PPBaseInfo{
						WalletAddress:  fileSlice.WalletAddress,
						NetworkAddress: fileSlice.NetworkAddress,
					},
					SliceOffset: &protos.SliceOffset{
						SliceOffsetStart: fileSlice.SliceOffsetStart,
						SliceOffsetEnd:   fileSlice.SliceOffsetEnd,
					},
				}
			}
		}

		return rsp, header.RspDownloadSliceWrong
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)

}
