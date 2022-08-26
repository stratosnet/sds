package event

// Author j
import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
)

// RegisterNewPP P-SP P register to become PP
func RegisterNewPP(ctx context.Context) {
	if setting.CheckLogin() {
		peers.SendMessageToSPServer(ctx, requests.ReqRegisterNewPPData(), header.ReqRegisterNewPP)
	}
}

// RspRegisterNewPP  SP-P
func RspRegisterNewPP(ctx context.Context, conn core.WriteCloser) {
	pp.Log(ctx, "get RspRegisterNewPP")
	var target protos.RspRegisterNewPP
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	pp.Log(ctx, "get RspRegisterNewPP", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		if target.AlreadyPp {
			setting.IsPP = true
		}
		return
	}

	pp.Log(ctx, "registered as PP successfully, you can deposit by `activate` ")
	setting.IsPP = true
}
