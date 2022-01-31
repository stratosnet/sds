package event

// Author j
import (
	"context"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"

	"github.com/alex023/clock"
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
		reloadPPlist()
		return
	}
	setting.SavePPList(&target)
	if len(setting.PPList) == 0 {
		// no PP exist, register to SP
		if !setting.IsLoginToSP {
			peers.RegisterChain(true)
			setting.IsLoginToSP = true
		}
		reloadPPlist()
		return
	}

	if success := peers.SendRegisterRequestViaPP(setting.PPList); !success {
		reloadPPlist()
	}
}

func reloadPPlist() {
	utils.DebugLog("failed to get PPlist. retry after 3 second")
	clock := clock.NewClock()
	clock.AddJobRepeat(time.Second*3, 1, peers.GetPPList)
	// defer job.Cancel()
}
