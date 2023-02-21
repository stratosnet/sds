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
			network.GetPeer(ctx).RunFsm(ctx, network.EVENT_SP_NO_PP_IN_STORE)
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

	formatRspGetPPStatus(ctx, &target)

	if target.IsActive == ppTypes.PP_ACTIVE {
		network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_RSP_ACTIVATE)
		if setting.IsAuto {
			network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_STATUS_ONLINE)
		}
	} else {
		network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_STATUS_INACTIVE)
	}

	if target.InitPpList {
		network.GetPeer(ctx).InitPPList(ctx)
	} else {
		if target.State == int32(protos.PPState_ONLINE) {
			network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_STATUS_ONLINE)
		}
	}
}

func formatRspGetPPStatus(ctx context.Context, response *protos.RspGetPPStatus) {
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
	case int32(protos.PPState_ONLINE):
		state = protos.PPState_ONLINE.String()
		if setting.OnlineTime == 0 {
			setting.OnlineTime = time.Now().Unix()
		}
	case int32(protos.PPState_SUSPEND):
		state = protos.PPState_SUSPEND.String()
		setting.OnlineTime = 0
	case int32(protos.PPState_MAINTENANCE):
		state = protos.PPState_MAINTENANCE.String()
		setting.OnlineTime = 0
	default:
		state = "Unknown"
	}
	pp.Logf(ctx, "*** current node status ***\n"+
		"Activation: %v | Mining: %v | Initial tier: %v | Ongoing tier: %v | Weight score: %v",
		activation, state, response.InitTier, response.OngoingTier, response.WeightScore)
}
