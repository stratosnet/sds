package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/utils"
)

// GetWalletOz queries current ozone balance
func GetWalletOz(walletAddr string) error {
	utils.Logf("Querying current ozone balance of the wallet: %v", walletAddr)
	peers.SendMessageToSPServer(requests.ReqGetWalletOzData(walletAddr), header.ReqGetWalletOz)
	return nil
}

func RspGetWalletOz(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get GetWalletOz RSP")
	var target protos.RspGetWalletOz
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.Logf("get GetWalletOz RSP, the current ozone balance of %v = %v", target.GetWalletAddress(), target.GetWalletOz())
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.Logf("failed to get ozone balance: %v", target.Result.Msg)
		return
	}
}
