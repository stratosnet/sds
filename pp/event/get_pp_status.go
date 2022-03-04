package event

import (
	"context"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	ppTypes "github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// RspGetPPStatus
func RspGetPPStatus(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get GetPPStatus RSP")
	var target protos.RspGetPPStatus
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.DebugLogf("get GetPPStatus RSP, activation status = %v", target.ActivationState)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.Log("failed to get any indexing nodes, reloading")
		peers.ScheduleReloadPPStatus(time.Second * 3)
		return
	}

	setting.State = byte(target.ActivationState)
	if setting.State == ppTypes.PP_ACTIVE {
		setting.IsPP = true
	}
	peers.InitPPList()
}
