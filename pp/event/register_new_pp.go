package event

// Author j
import (
	"context"
	"fmt"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// RegisterNewPP P-SP P register to become PP
func RegisterNewPP() {
	if setting.CheckLogin() {
		SendMessageToSPServer(reqRegisterNewPPData(), header.ReqRegisterNewPP)
	}
}

// RspRegisterNewPP  SP-P
func RspRegisterNewPP(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get RspRegisterNewPP")
	var target protos.RspRegisterNewPP
	if unmarshalData(ctx, &target) {
		utils.Log("get RspRegisterNewPP", target.Result.State, target.Result.Msg)
		if target.Result.State == protos.ResultState_RES_SUCCESS {
			fmt.Println("register as PP successfully, input start to mining")
			setting.IsPP = true
		}
	}

}
