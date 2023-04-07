package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/utils"
)

func RspBadVersion(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspBadVersion
	if err := VerifyMessage(ctx, header.RspBadVersion, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.ErrorLogf("The command [%v] was rejected due to an invalid version [%v] (minimum version [%v]). The connection will be dropped. Please update to a more recent version",
		target.Command, target.Version, target.MinimumVersion)
}
