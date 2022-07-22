package event

import (
	"context"
	"strings"

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
	utils.DebugLogf("get GetPPStatus RSP, activation status = %v", target.IsActive)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog(target.Result.Msg)
		if strings.Contains(target.Result.Msg, "Please register first") {
			return
		}
		utils.Log("failed to query node status, please retry later")
		return
	}

	setting.State = target.IsActive
	if setting.State == ppTypes.PP_ACTIVE {
		setting.IsPP = true
	}

	formatRspGetPPStatus(target)

	if target.InitPpList {
		peers.InitPPList()
	}
}

func formatRspGetPPStatus(response protos.RspGetPPStatus) {
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
	case int32(protos.PPState_ONLINE):
		state = protos.PPState_ONLINE.String()
	case int32(protos.PPState_SUSPEND):
		state = protos.PPState_SUSPEND.String()
	default:
		state = "Unknown"
	}
	utils.Logf("*** current node status ***\n"+
		"Activation: %v | Mining: %v | Initial tier: %v | Ongoing tier: %v | Weight score: %v",
		activation, state, response.InitTier, response.OngoingTier, response.WeightScore)
}
