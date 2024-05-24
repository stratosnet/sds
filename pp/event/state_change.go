package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/msg/header"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/sds-msg/protos"
)

func RspStateChange(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspStateChangePP
	if err := VerifyMessage(ctx, header.RspStateChangePP, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}

	success := requests.UnmarshalData(ctx, &target)
	if !success {
		utils.ErrorLog("failed unmarshal the RspStateChangePP message")
		return
	}
	if target.UpdateState != uint32(protos.PPState_OFFLINE) {
		utils.Log("State change hasn't been completed")
		return
	}
	pp.Log(ctx, "State change has been completed, please start mining.")
}
