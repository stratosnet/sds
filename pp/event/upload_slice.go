package event

// Author j
import (
	"context"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
)

// ProgressMap required by API
var ProgressMap = &sync.Map{}

func ReqUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	var target protos.ReqUploadFileSlice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.Sign == nil || !verifyUploadSliceSign(&target) {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "signature validation failed",
			},
		}
		peers.SendMessage(conn, rsp, header.RspUploadFileSlice)
		return
	}

	if target.SliceNumAddr.PpInfo.P2PAddress != setting.P2PAddress {
		rsp := &protos.RspUploadFileSlice{
			Result: &protos.Result{
				State: protos.ResultState_RES_FAIL,
				Msg:   "mismatch between p2p address in the request and node p2p address.",
			},
		}
		peers.SendMessage(conn, rsp, header.RspUploadFileSlice)
		return
	}

	peers.SendMessage(conn, requests.UploadSpeedOfProgressData(target.FileHash, uint64(len(target.Data))), header.UploadSpeedOfProgress)

	if !task.SaveUploadFile(&target) {
		// save failed, not handling yet
		utils.ErrorLog("SaveUploadFile failed")
		return
	}

	utils.DebugLogf("ReqUploadFileSlice saving slice %v  current_size %v  total_size %v", target.SliceInfo.SliceHash, file.GetSliceSize(target.SliceInfo.SliceHash), target.SliceSize)
	if file.GetSliceSize(target.SliceInfo.SliceHash) == int64(target.SliceSize) {
		utils.DebugLog("the slice upload finished", target.SliceInfo.SliceHash)
		// respond to PP in case the size is correct but actually not success
		if utils.CalcSliceHash(file.GetSliceData(target.SliceInfo.SliceHash), target.FileHash, target.SliceNumAddr.SliceNumber) == target.SliceInfo.SliceHash {
			peers.SendMessage(conn, requests.RspUploadFileSliceData(&target), header.RspUploadFileSlice)
			// report upload result to SP
			peers.SendMessageToSPServer(requests.ReqReportUploadSliceResultDataPP(&target), header.ReqReportUploadSliceResult)
			utils.DebugLog("storage PP report to SP upload task finished: ", target.SliceInfo.SliceHash)
		} else {
			utils.ErrorLog("newly stored sliceHash is not equal to target sliceHash!")
		}
	}
}

func RspUploadFileSlice(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspUploadFileSlice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	utils.DebugLogf("get RspUploadFileSlice for file %v  sliceNumber %v  size %v", target.FileHash, target.SliceNumAddr.SliceNumber, target.SliceSize)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog("RspUploadFileSlice failure:", target.Result.Msg)
		return
	}

	peers.SendMessageToSPServer(requests.ReqReportUploadSliceResultData(&target), header.ReqReportUploadSliceResult)
}

// RspUploadSlicesWrong updates the destination of slices for an ongoing upload
func RspUploadSlicesWrong(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspUploadSlicesWrong
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog("RspUploadSlicesWrong failure:", target.Result.Msg)
		// TODO: cancel upload
		return
	}

	if len(target.Slices) == 0 {
		utils.ErrorLogf("No new slices in RspUploadSlicesWrong for file %v. Cannot update slice destinations")
		return
	}

	value, ok := task.UploadFileTaskMap.Load(target.FileHash)
	if !ok {
		utils.ErrorLogf("File upload task cannot be found for file %v", target.FileHash)
		return
	}
	uploadTask := value.(*task.UploadFileTask)
	uploadTask.UpdateSliceDestinations(target.Slices)

	// Start upload for all new destinations
	uploadTask.SignalNewDestinations()
}

// RspReportUploadSliceResult  SP-P OR SP-PP
func RspReportUploadSliceResult(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspReportUploadSliceResult")
	var target protos.RspReportUploadSliceResult
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	if target.Result.State == protos.ResultState_RES_SUCCESS {
		utils.DebugLog("ResultState_RES_SUCCESS, sliceNumber，storageAddress，walletAddress", target.SliceNumAddr.SliceNumber, target.SliceNumAddr.PpInfo.NetworkAddress, target.SliceNumAddr.PpInfo.P2PAddress)
	} else {
		utils.Log("ResultState_RES_FAIL : ", target.Result.Msg)
	}
}

func UploadFileSlice(tk *task.UploadSliceTask, sign []byte) error {
	tkDataLen := len(tk.Data)
	fileHash := tk.FileHash
	storageP2pAddress := tk.SliceNumAddr.PpInfo.P2PAddress
	storageNetworkAddress := tk.SliceNumAddr.PpInfo.NetworkAddress

	if tkDataLen <= setting.MAXDATA {
		tk.SliceOffsetInfo.SliceOffset.SliceOffsetStart = 0
		return sendSlice(requests.ReqUploadFileSliceData(tk, sign), fileHash, storageP2pAddress, storageNetworkAddress)
	}

	dataStart := 0
	dataEnd := setting.MAXDATA
	for {
		newTask := &task.UploadSliceTask{
			TaskID:         tk.TaskID,
			FileHash:       tk.FileHash,
			SliceNumAddr:   tk.SliceNumAddr,
			FileCRC:        tk.FileCRC,
			SliceTotalSize: tk.SliceTotalSize,
			SliceOffsetInfo: &protos.SliceOffsetInfo{
				SliceHash: tk.SliceOffsetInfo.SliceHash,
				SliceOffset: &protos.SliceOffset{
					SliceOffsetStart: uint64(dataStart),
					SliceOffsetEnd:   uint64(dataEnd),
				},
			},
			SpP2pAddress: tk.SpP2pAddress,
		}
		if dataEnd < (tkDataLen + 1) {
			newTask.Data = tk.Data[dataStart:dataEnd]
			utils.DebugLogf("Uploading slice data %v-%v (total %v)", dataStart, dataEnd, newTask.SliceTotalSize)
			err := sendSlice(requests.ReqUploadFileSliceData(newTask, sign), fileHash, storageP2pAddress, storageNetworkAddress)
			if err != nil {
				return err
			}
			dataStart += setting.MAXDATA
			dataEnd += setting.MAXDATA
		} else {
			utils.DebugLogf("Uploading slice data %v-%v (total %v)", dataStart, tkDataLen, newTask.SliceTotalSize)
			newTask.Data = tk.Data[dataStart:]
			return sendSlice(requests.ReqUploadFileSliceData(newTask, sign), fileHash, storageP2pAddress, storageNetworkAddress)
		}
	}
}

func sendSlice(pb proto.Message, fileHash, p2pAddress, networkAddress string) error {
	utils.DebugLog("sendSlice(pb proto.Message, fileHash, p2pAddress, networkAddress string)",
		fileHash, p2pAddress, networkAddress)

	key := fileHash + p2pAddress

	if c, ok := client.UpConnMap.Load(key); ok {
		conn := c.(*cf.ClientConn)
		err := peers.SendMessage(conn, pb, header.ReqUploadFileSlice)
		if err == nil {
			utils.DebugLog("SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
			return nil
		}
	}

	conn, err := client.NewClient(networkAddress, false)
	if err != nil {
		return errors.Wrap(err, "Failed to create connection with "+networkAddress)
	}

	err = peers.SendMessage(conn, pb, header.ReqUploadFileSlice)
	if err == nil {
		utils.DebugLog("SendMessage(conn, pb, header.ReqUploadFileSlice) ", conn)
		client.UpConnMap.Store(key, conn)
	} else {
		utils.ErrorLog("Fail to send upload slice request to" + networkAddress)
	}
	return err
}

func UploadSpeedOfProgress(ctx context.Context, _ core.WriteCloser) {
	var target protos.UploadSpeedOfProgress
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	prg, ok := task.UploadProgressMap.Load(target.FileHash)
	if !ok {
		utils.DebugLog("paused!!")
		return
	}

	progress := prg.(*task.UploadProgress)
	progress.HasUpload += int64(target.SliceSize)
	p := float32(progress.HasUpload) / float32(progress.Total) * 100
	utils.Logf("fileHash: %v  uploaded：%.2f %% ", target.FileHash, p)
	setting.ShowProgress(p)
	ProgressMap.Store(target.FileHash, p)
	if progress.HasUpload >= progress.Total {
		task.UploadProgressMap.Delete(target.FileHash)
		task.CleanUpConnMap(target.FileHash)
		ScheduleReqBackupStatus(target.FileHash)
		if file.IsFileRpcRemote(target.FileHash) {
			file.SetRemoteFileResult(target.FileHash, rpc.Result{Return: rpc.SUCCESS})
		}
	}
}

func verifyUploadSliceSign(target *protos.ReqUploadFileSlice) bool {
	return requests.VerifySpSignature(target.SpP2PAddress,
		[]byte(target.P2PAddress+target.FileHash+header.ReqUploadFileSlice), target.Sign)
}
