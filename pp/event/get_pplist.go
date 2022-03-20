package event

// Author j
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

func RspGetPPList(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetPPList
	if !requests.UnmarshalData(ctx, &target) {
		utils.ErrorLog("Couldn't unmarshal protobuf to protos.RspGetPPList")
		return
	}

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.Log("failed to get any peers, reloading")
		peers.ScheduleReloadPPlist()
		return
	}

	err := peers.SavePPList(&target)
	if err != nil {
		utils.ErrorLog("Error when saving PP List", err)
	}

	shouldRegisterToSP := setting.IsAuto && !setting.IsLoginToSP &&
		(setting.State == types.PP_ACTIVE)

	list, total := peers.GetPPList()
	if total == 0 {
		// no PP exist, register to SP
		if shouldRegisterToSP {
			peers.RegisterToSP(true)
		}
		peers.ScheduleReloadPPlist()
		return
	}

	// if gateway pp is nil, go connect one from ppList
	if client.PPConn == nil {
		if success := peers.ConnectToGatewayPP(list); !success {
			peers.ScheduleReloadPPlist()
		}
	}

	if shouldRegisterToSP {
		peers.RegisterToSP(true)
	}
}
