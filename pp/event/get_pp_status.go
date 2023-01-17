package event

import (
	"context"
	"strings"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/network"
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
	pp.DebugLogf(ctx, "get GetPPStatus RSP, activation status = %v", target.IsActive)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog(target.Result.Msg)
		if strings.Contains(target.Result.Msg, "Please register first") {
			setting.IsPPSyncedWithSP = true
			return
		}
		pp.Log(ctx, "failed to query node status, please retry later")
		return
	}

	setting.State = target.IsActive
	if setting.State == ppTypes.PP_ACTIVE {
		setting.IsPP = true
		setting.IsPPSyncedWithSP = true
	}

	isSuspended := setting.IsSuspended
	formatRspGetPPStatus(ctx, target)

	if target.InitPpList {
		network.GetPeer(ctx).InitPPList(ctx)
		if setting.IsAuto && setting.State == ppTypes.PP_ACTIVE && !setting.IsLoginToSP {
			network.GetPeer(ctx).StartRegisterToSp(ctx)
		}
	} else {
		// after user intervention, pp state changed from suspended to non-suspended, start register process
		if !setting.IsLoginToSP && !setting.IsSuspended && isSuspended != setting.IsSuspended {
			network.GetPeer(ctx).StartRegisterToSp(ctx)
		}
	}
}

func formatRspGetPPStatus(ctx context.Context, response protos.RspGetPPStatus) {
	activation, state := "", ""

	switch response.IsActive {
	case ppTypes.PP_ACTIVE:
		activation = "Active"
	case ppTypes.PP_INACTIVE:
		activation = "Inactive"
	case ppTypes.PP_UNBONDING:
		activation = "Unbonding"
	default:
		activation = "Unknown"
	}

	switch response.State {
	case int32(protos.PPState_OFFLINE):
		state = protos.PPState_OFFLINE.String()
		setting.OnlineTime = 0
		setting.IsSuspended = false
	case int32(protos.PPState_ONLINE):
		state = protos.PPState_ONLINE.String()
		if setting.OnlineTime == 0 {
			setting.OnlineTime = time.Now().Unix()
		}
		setting.IsSuspended = false
	case int32(protos.PPState_SUSPEND):
		state = protos.PPState_SUSPEND.String()
		setting.OnlineTime = 0
		// a just activated pp node should be allowed to register to sp
		// so, a more strict condition to set pp to a "suspended" flag: the value of tier
		if response.InitTier != 0 && response.OngoingTier == 0 {
			setting.IsSuspended = true
		} else {
			setting.IsSuspended = false
		}
	case int32(protos.PPState_MAINTENANCE):
		state = protos.PPState_MAINTENANCE.String()
		setting.OnlineTime = 0
		setting.IsSuspended = false
	default:
		state = "Unknown"
	}
	pp.Logf(ctx, "*** current node status ***\n"+
		"Activation: %v | Mining: %v | Initial tier: %v | Ongoing tier: %v | Weight score: %v",
		activation, state, response.InitTier, response.OngoingTier, response.WeightScore)
}
