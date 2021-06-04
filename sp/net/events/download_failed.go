package events

import (
	"context"
	"github.com/golang/protobuf/proto"
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

// downloadFailed is a concrete implementation of event
type downloadFailed struct {
	event
}

const downloadFailedEvent = "download_failed"

// GetDownloadFailedHandler create event and return handler func for it
func GetDownloadFailedHandler(s *net.Server) EventHandleFunc {
	e := downloadFailed{newEvent(downloadFailedEvent, s, downloadFailedCallbackFunc)}
	return e.Handle
}

// downloadFailedCallbackFunc is the main process of download fail event
func downloadFailedCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqDownloadSliceWrong)

	rsp := &protos.RspDownloadSliceWrong{
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

	task := &data.DownloadTask{TaskId: body.TaskId}

	if s.Load(task) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "task is finished or not exist"
		return rsp, header.RspDownloadSliceWrong
	}

	res, err := s.CT.FetchTables([]table.FileSliceStorage{}, map[string]interface{}{
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

	if len(sliceStorage) <= 0 {
		return rsp, header.RspDownloadSliceWrong
	}

	ring := hashring.New(s.Conf.HashRing.VirtualNodeNum)
	for _, storage := range sliceStorage {
		if storage.WalletAddress != task.StorageWalletAddress {
			if s.HashRing.IsOnline(storage.WalletAddress) {
				ring.AddNode(&hashring.Node{ID: storage.WalletAddress, NetworkId: &protos.NetworkId{
					PublicKey: storage.PublicKey,
					NetworkAddress: storage.NetworkAddress,
				}})
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

	fileSlice := &table.FileSlice{
		SliceHash: task.SliceHash,
		FileSliceStorage: table.FileSliceStorage{
			WalletAddress: anotherWalletAddress,
		},
	}

	if s.CT.Fetch(fileSlice) != nil {
		return rsp, header.RspDownloadSliceWrong
	}

	fileSliceStorage := &table.FileSliceStorage{
		SliceHash:     task.SliceHash,
		WalletAddress: task.StorageWalletAddress,
	}

	if _, err = s.CT.DeleteTable(fileSliceStorage); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, downloadFailedEvent, "delete file slice storage table", err)
	}

	s.HandleMsg(&common.MsgTransferNotice{
		SliceHash:         fileSlice.SliceHash,
		FromWalletAddress: fileSlice.WalletAddress,
		ToWalletAddress:   task.StorageWalletAddress,
	})

	task.StorageWalletAddress = anotherWalletAddress

	if err := s.Store(task, 3600*time.Second); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, downloadFailedEvent, "store task to db", err)
	}

	rsp.FileHash = fileSlice.FileHash
	rsp.NewSliceInfo = &protos.DownloadSliceInfo{
		SliceStorageInfo: &protos.SliceStorageInfo{
			SliceHash: fileSlice.SliceHash,
			SliceSize: fileSlice.SliceSize,
		},
		SliceNumber: fileSlice.SliceNumber,
		StoragePpInfo: &protos.PPBaseInfo{
			WalletAddress:  fileSlice.WalletAddress,
			NetworkId: &protos.NetworkId{
				PublicKey: fileSlice.PublicKey,
				NetworkAddress: fileSlice.NetworkAddress,

			},
		},
		SliceOffset: &protos.SliceOffset{
			SliceOffsetStart: fileSlice.SliceOffsetStart,
			SliceOffsetEnd:   fileSlice.SliceOffsetEnd,
		},
	}
	return rsp, header.RspDownloadSliceWrong
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *downloadFailed) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqDownloadSliceWrong{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
