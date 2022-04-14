package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/utils"
)

func RspBadVersion(ctx context.Context, _ core.WriteCloser) {
	var target protos.RspBadVersion
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.ErrorLogf("The command [%v] was rejected due to an invalid version [%v] (minimum version [%v]). The connection will be dropped. Please update to a more recent version",
		target.Command, target.Version, target.MinimumVersion)
}
