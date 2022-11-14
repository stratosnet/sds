package event

// Author j
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
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
		utils.Log("failed to get any network")
		return
	}

	err := p2pserver.GetP2pServer(ctx).SavePPList(ctx, &target)
	if err != nil {
		utils.ErrorLog("Error when saving PP List", err)
	}

	shouldRegisterToSP := setting.IsAuto && !setting.IsLoginToSP &&
		(setting.State == types.PP_ACTIVE)

	if shouldRegisterToSP {
		network.GetPeer(ctx).RegisterToSP(ctx, true)
	}
}
