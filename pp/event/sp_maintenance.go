package event

import (
	"context"
	"time"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/utils"
)

func NoticeSpUnderMaintenance(ctx context.Context, conn core.WriteCloser) {
	var target protos.NoticeSpUnderMaintenance
	if err := VerifyMessage(ctx, header.NoticeSpUnderMaintenance, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	switch conn := conn.(type) {
	case *core.ServerConn:
		utils.DebugLog("Ignore NoticeSpUnderMaintenance in ServerConn")
		return
	case *cf.ClientConn:
		if conn.GetName() != p2pserver.GetP2pServer(ctx).GetSpName() {
			utils.DebugLog("Ignore NoticeSpUnderMaintenance from non SP node")
			return
		}

		if target.MaintenanceType == int32(protos.SpMaintenanceType_CONSENSUS) {
			utils.Logf("SP[%v] is currently under maintenance, maintenance_type: %v",
				target.SpP2PAddress, protos.SpMaintenanceType_CONSENSUS.String())

			// record SpMaintenance
			triggerSpSwitch := p2pserver.GetP2pServer(ctx).RecordSpMaintenance(target.SpP2PAddress, time.Now())
			if triggerSpSwitch {
				network.GetPeer(ctx).SpLatencyCheck(ctx)()
			}
		}
	}
}
