package event

// Author j
import (
	"context"
	"fmt"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// RegisterNewPP P-SP P register to become PP
func RegisterNewPP() {
	if setting.CheckLogin() {
		peers.SendMessageToSPServer(types.ReqRegisterNewPPData(), header.ReqRegisterNewPP)
	}
}

// RspRegisterNewPP  SP-P
func RspRegisterNewPP(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspRegisterNewPP")
	var target protos.RspRegisterNewPP
	if types.UnmarshalData(ctx, &target) {
		utils.Log("get RspRegisterNewPP", target.Result.State, target.Result.Msg)
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			fmt.Println("register as PP successfully, you can deposit by `activate` ")
			setting.IsPP = true
		}
	}

}
