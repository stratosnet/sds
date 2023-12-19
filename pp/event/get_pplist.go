package event

// Author j
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/sds-msg/protos"
)

func RspGetPPList(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetPPList
	if err := VerifyMessage(ctx, header.RspGetPPList, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}

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

}
