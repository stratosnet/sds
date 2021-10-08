package event

// Author j
import (
	"context"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"

	"github.com/alex023/clock"
)

// RspGetPPList
func RspGetPPList(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get GetPPList RSP")
	var target protos.RspGetPPList
	if !types.UnmarshalData(ctx, &target) {
		return
	}
	utils.Log("get GetPPList RSP", target.PpList)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
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

	ppList := setting.PPList
	for _, ppInfo := range ppList {
		if ppInfo.NetworkAddress == setting.NetworkAddress {
			continue
		}
		client.PPConn = client.NewClient(ppInfo.NetworkAddress, true)
		if client.PPConn != nil {
			peers.RegisterChain(false)
			return
		}
		utils.DebugLog("failed to conn PP，delete:", ppInfo)
		setting.DeletePPList(ppInfo.NetworkAddress)
	}
	reloadPPlist()
}

func reloadPPlist() {
	utils.DebugLog("failed to get PPlist. retry after 3 second")
	clock := clock.NewClock()
	clock.AddJobRepeat(time.Second*3, 1, peers.GetPPList)
	// defer job.Cancel()
}
