package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/utils"
)

func RspMessageForward(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get RspMessageForward")
	var target protos.RspMessageForward
	if err := VerifyMessage(ctx, header.RspMessageForward, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	if target.Result != nil && target.Result.State == protos.ResultState_RES_FAIL {
		utils.DebugLog("ReqMessageForward failure received,", target.Result.Msg)
		return
	}

	message := &msg.RelayMsgBuf{
		MSGBody: target.Msg,
	}
	message.MSGHead.ReqId = requests.GetReqIdFromMessage(ctx)
	ctx = core.CreateContextWithMessage(ctx, message)
	if handler := core.GetHandlerFunc(uint8(target.CmdType)); handler != nil {
		handler(ctx, conn)
	}
}
