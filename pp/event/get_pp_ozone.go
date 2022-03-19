package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// GetPPOzone queries current ozone balance
func GetPPOzone() error {
	utils.Log("Sending get ozone balance message to SP from " + setting.WalletAddress)
	peers.SendMessageToSPServer(requests.ReqGetPPStatusData(), header.ReqGetPPOzone)
	return nil
}

func RspGetPPOzone(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get GetPPOzone RSP")
	var target protos.RspGetPPStatus
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.Logf("get GetPPOzone RSP, PP ozone balance = %v", target.GetUoz())
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.Logf("failed to get ozone balance: %v", target.Result.Msg)
		return
	}
}
