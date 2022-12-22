package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// RspReportNodeStatus
func RspReportNodeStatus(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspReportNodeStatus
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.DebugLogf("get RspReportNodeStatus RSP = %v", target.GetPpstate())
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog(target.Result.Msg)
		return
	}
	// finished register->startmining->first_status_report process
	if !setting.IsLoginToSP && setting.IsStartMining {
		utils.DebugLog("@#@#@#@#@#@#@#@#@#@#@#@#@#@#")
		setting.IsLoginToSP = true
	}

	if target.GetPpstate() == 2 {
		setting.IsStartMining = false
	}
}
