package event

import (
	"context"
	"fmt"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// Activate Inactive PP node becomes active
func Activate(amount, fee, gas int64) error {
	activateReq, err := reqActivateData(amount, fee, gas)
	if err != nil {
		utils.ErrorLog("Couldn't build PP activate request: " + err.Error())
		return err
	}
	fmt.Println("Sending activate message to SP! " + activateReq.WalletAddress)
	SendMessageToSPServer(activateReq, header.ReqActivate)
	return nil
}

// RspActivate. Response to asking the SP node to activate this PP node
func RspActivate(ctx context.Context, conn spbf.WriteCloser) {
	utils.Log("get RspActivate")
	var target protos.RspActivate
	success := unmarshalData(ctx, &target)
	if !success {
		return
	}

	utils.Log("get RspActivate", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}

	if target.AlreadyActive {
		fmt.Println("Current node is already active")
		setting.IsActive = true
	} else {
		fmt.Println("The activation transaction was broadcast")
	}
}

// RspActivated. Response when this PP node was successfully activated
func RspActivated(ctx context.Context, conn spbf.WriteCloser) {
	setting.IsActive = true
}
