package event

import (
	"context"
	"strings"
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
	var target protos.RspGetPPStatus
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.DebugLogf("get GetPPStatus RSP, activation status = %v", target.ActivationState)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog(target.Result.Msg)
		if strings.Contains(target.Result.Msg, "Please register first") {
			return
		}
		utils.Log("failed to query node status, retrying in 10 seconds...")
		peers.ScheduleReloadPPStatus(time.Second * 10)
		return
	}

	setting.State = target.ActivationState
	if setting.State == ppTypes.PP_ACTIVE {
		setting.IsPP = true
	}
	peers.InitPPList()
}
