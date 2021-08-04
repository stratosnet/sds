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

// Deactivate Request that an active PP node becomes inactive
func Deactivate(fee, gas int64) error {
	deactivateReq, err := reqDeactivateData(fee, gas)
	if err != nil {
		utils.ErrorLog("Couldn't build PP deactivate request: " + err.Error())
		return err
	}
	fmt.Println("Sending deactivate message to SP! " + deactivateReq.P2PAddress)
	SendMessageToSPServer(deactivateReq, header.ReqDeactivate)
	return nil
}

// RspDeactivate. Response to asking the SP node to deactivate this PP node
func RspDeactivate(ctx context.Context, conn spbf.WriteCloser) {
	var target protos.RspDeactivate
	success := unmarshalData(ctx, &target)
	if !success {
		return
	}

	utils.Log("get RspActivate", target.Result.State, target.Result.Msg)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		return
	}

	setting.State = byte(target.ActivationState)

	if target.ActivationState == setting.PP_INACTIVE {
		fmt.Println("Current node is already inactive")
	} else {
		fmt.Println("The deactivation transaction was broadcast")
	}
}

// RspActivated. Response when this PP node was successfully activated
func RspDeactivated(ctx context.Context, conn spbf.WriteCloser) {
	setting.State = setting.PP_INACTIVE
	fmt.Println("This PP node is now inactive")
}
