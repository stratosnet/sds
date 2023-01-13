package event

import (
	"context"
	"fmt"
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
)

const (
	taskMonitorInterval = 30 * time.Second
)

var (
	taskMonitorClock = clock.NewClock()
	taskMonitorJob   clock.Job
)

// StartMaintenance sends a request to SP to temporarily put the current node into maintenance mode
func StartMaintenance(ctx context.Context, duration uint64) error {
	req := requests.ReqStartMaintenance(duration)
	pp.Log(ctx, "Sending maintenance start request to SP!")
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqStartMaintenance)
	return nil
}

func StopMaintenance(ctx context.Context) error {
	req := requests.ReqStopMaintenance()
	pp.Log(ctx, "Sending maintenance stop request to SP!")
	p2pserver.GetP2pServer(ctx).SendMessageToSPServer(ctx, req, header.ReqStopMaintenance)
	return nil
}

func RspStartMaintenance(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspStartMaintenance
	if !requests.UnmarshalData(ctx, &target) {
		pp.DebugLog(ctx, "Cannot unmarshal start maintenance response")
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.Logf(ctx, "cannot start maintenance: %v", target.Result.Msg)
		return
	}

	pp.Logf(ctx, "Do not stop the pp service until all tasks are completed, otherwise score will be deducted.")
	pp.Logf(ctx, "Checking ongoing tasks... ")
	taskMonitorJob, _ = taskMonitorClock.AddJobRepeat(taskMonitorInterval, 0, taskMonitorFunc(ctx))

	setting.IsStartMining = false
}

func RspStopMaintenance(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspStopMaintenance
	if !requests.UnmarshalData(ctx, &target) {
		pp.DebugLog(ctx, "Cannot unmarshal stop maintenance response")
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		pp.Logf(ctx, "failed to stop maintenance: %v", target.Result.Msg)
		return
	}

	// if PP process is killed, then setting.IsLoginToSP would be false after restarted
	if setting.IsAuto && !setting.IsLoginToSP {
		pp.DebugLog(ctx, "PP is not login to SP, register to SP")
		network.GetPeer(ctx).StartRegisterToSp(ctx)
		return
	}

	// if PP process is not killed, then setting.IsLoginToSP would keep true
	if setting.IsAuto && setting.IsLoginToSP {
		pp.DebugLog(ctx, "PP is login to SP, start mining")
		network.GetPeer(ctx).StartMining(ctx)
		return
	}
}

func taskMonitorFunc(ctx context.Context) func() {
	return func() {
		uploadTaskCnt := GetOngoingUploadTaskCount()
		downloadTaskCnt := GetOngoingDownloadTaskCount()

		transferTasksCnt := task.GetTransferTaskCnt()

		pp.DebugLog(ctx, fmt.Sprintf("Ongoing tasks: upload--%v  download--%v  transfer--%v ",
			uploadTaskCnt, downloadTaskCnt, transferTasksCnt))

		totalTaskCnt := uploadTaskCnt + downloadTaskCnt + transferTasksCnt
		if totalTaskCnt == 0 {
			pp.Logf(ctx, "All tasks have been completed, pp service can be stopped.")
			taskMonitorJob.Cancel()
		} else {
			pp.Logf(ctx, fmt.Sprintf("%v ongoing task remaining, do not stop pp service...", totalTaskCnt))
		}

	}
}
