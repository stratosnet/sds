package event

// Author j
import (
	"context"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
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
		peers.ScheduleReloadPPlist(3 * time.Second)
		return
	}

	err := setting.Peers.SavePPList(&target)
	if err != nil {
		utils.ErrorLog("Error when saving PP List", err)
	}

	if len(setting.Peers.GetPPList()) == 0 {
		// no PP exist, register to SP
		if !setting.IsLoginToSP {
			peers.RegisterToSP(true)
		}
		peers.ScheduleReloadPPlist(3 * time.Second)
		return
	}

	// if gateway pp is nil, go connect one from ppList
	if client.PPConn == nil {
		if success := peers.SendRegisterRequestViaPP(setting.Peers.GetPPList()); !success {
			peers.ScheduleReloadPPlist(3 * time.Second)
		}
	}
}
