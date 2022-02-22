package event

// Author j
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

// RegisterNewPP P-SP P register to become PP
func RegisterNewPP() {
	if setting.CheckLogin() {
		peers.SendMessageToSPServer(requests.ReqRegisterNewPPData(), header.ReqRegisterNewPP)
	}
}

// RspRegisterNewPP  SP-P
func RspRegisterNewPP(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspRegisterNewPP")
	var target protos.RspRegisterNewPP
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	utils.Log("get RspRegisterNewPP", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		if target.AlreadyPp {
			setting.IsPP = true
		}
		return
	}

	utils.Log("registered as PP successfully, you can deposit by `activate` ")
	setting.IsPP = true
}
