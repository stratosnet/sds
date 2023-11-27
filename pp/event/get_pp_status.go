package event

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/stratosnet/framework/core"
	"github.com/stratosnet/framework/utils"
	"github.com/stratosnet/sds-api/header"
	"github.com/stratosnet/sds-api/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	ppTypes "github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils/environment"
)

// RspGetPPStatus
func RspGetPPStatus(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetPPStatus
	if err := VerifyMessage(ctx, header.RspGetPPStatus, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	rpcResult := &rpc.StatusResult{Return: rpc.SUCCESS}
	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		defer pp.SetRPCResult(p2pserver.GetP2pServer(ctx).GetP2PAddress()+reqId, rpcResult)
	}
	pp.DebugLogf(ctx, "get GetPPStatus RSP, activation status = %v", target.IsActive)
	setSoftMemoryCap(target.OngoingTier)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.ErrorLog(target.Result.Msg)
		rpcResult.Return = rpc.INTERNAL_COMM_FAILURE
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

	rpcResult.Message = formatRspGetPPStatus(ctx, &target)

	if target.IsActive == ppTypes.PP_ACTIVE {
		network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_RSP_ACTIVATED)
	} else {
		network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_STATUS_INACTIVE)
	}

	if target.InitPpList {
		network.GetPeer(ctx).InitPPList(ctx)
	} else {
		if target.State == int32(protos.PPState_SUSPEND) {
			network.GetPeer(ctx).RunFsm(ctx, network.EVENT_RCV_STATUS_SUSPEND)
		}
	}
}

func formatRspGetPPStatus(ctx context.Context, response *protos.RspGetPPStatus) string {
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

	regStatStr := ""
	regStat := network.GetPeer(ctx).GetStateFromFsm()
	switch regStat.Id {
	case network.STATE_NOT_REGISTERED:
		regStatStr = "Unregistered"
	case network.STATE_REGISTERING:
		regStatStr = "Registering"
	case network.STATE_REGISTERED:
		regStatStr = "Registered"
	default:
		regStatStr = "Unknown"
	}

	spStatus := "disconnected"
	if spInfo := p2pserver.GetP2pServer(ctx).GetSpConn(); spInfo != nil {
		spStatus = fmt.Sprintf("%v (%v)", spInfo.GetRemoteP2pAddress(), spInfo.GetName())
	}

	msgStr := fmt.Sprintf("*** current node status ***\n"+
		"Activation: %v | Registration Status: %v | Mining: %v | Initial tier: %v | Ongoing tier: %v | Weight score: %v | Meta node: %v",
		activation, regStatStr, state, response.InitTier, response.OngoingTier, response.WeightScore, spStatus)
	pp.Log(ctx, msgStr)
	return msgStr
}

func setSoftMemoryCap(tier uint32) {
	if environment.IsDev() {
		switch tier {
		case 0:
			debug.SetMemoryLimit(setting.SoftRamLimitTier0Dev)
		case 1:
			debug.SetMemoryLimit(setting.SoftRamLimitTier1Dev)
		case 2:
			debug.SetMemoryLimit(setting.SoftRamLimitTier2Dev)
		default:
			debug.SetMemoryLimit(setting.SoftRamLimitTier2Dev)
		}
	} else {
		switch tier {
		case 0:
			debug.SetMemoryLimit(setting.SoftRamLimitTier0)
		case 1:
			debug.SetMemoryLimit(setting.SoftRamLimitTier1)
		case 2:
			debug.SetMemoryLimit(setting.SoftRamLimitTier2)
		default:
			debug.SetMemoryLimit(setting.SoftRamLimitTier2)
		}
	}
}
