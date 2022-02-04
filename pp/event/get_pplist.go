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

// RspGetPPList
func RspGetPPList(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetPPList
	if !requests.UnmarshalData(ctx, &target) {
		utils.ErrorLog("Couldn't unmarshal protobuf to protos.RspGetPPList")
		return
	}
	utils.DebugLog("get GetPPList RSP", target.PpList)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.Log("failed to get any peers, reloading")
		peers.ScheduleReloadPPlist(3 * time.Second)
		return
	}
	setting.SavePPList(&target)
	if len(setting.GetLocalPPList()) == 0 {
		// no PP exist, register to SP
		if !setting.IsLoginToSP {
			peers.RegisterChain(true)
		}
		peers.ScheduleReloadPPlist(3 * time.Second)
		return
	}
	// if gateway pp is nil, go connect one from ppList
	if client.PPConn == nil {
		if success := peers.SendRegisterRequestViaPP(setting.GetLocalPPList()); !success {
			peers.ScheduleReloadPPlist(3 * time.Second)
		}
	}

}
