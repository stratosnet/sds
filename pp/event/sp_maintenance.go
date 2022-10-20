package event

import (
	"context"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/utils"
)

func RspSpUnderMaintenance(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspSpUnderMaintenance
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	utils.Logf("SP[%v] is currently under maintenance, dropping connection and reconnect to another SP",
		target.SpP2PAddress)
	if target.NeedReconnectSp == int32(1) {
		// close connection with SP
		conn.Close()
	}
}
