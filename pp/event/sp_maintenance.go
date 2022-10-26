package event

import (
	"context"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

func RspSpUnderMaintenance(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspSpUnderMaintenance
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	switch conn.(type) {
	case *core.ServerConn:
		utils.DebugLog("Ignore RspSpUnderMaintenance in SeverConn")
		return
	case *cf.ClientConn:
		if conn.(*cf.ClientConn).GetName() != client.SPConn.GetName() {
			utils.DebugLog("Ignore RspSpUnderMaintenance from non SP node")
			return
		}

		if target.MaintenanceType == int32(protos.SpMaintenanceType_CONSENSUS) {
			utils.Logf("SP[%v] is currently under maintenance, maintenance_type: %v",
				target.SpP2PAddress, protos.SpMaintenanceType_CONSENSUS.String())

			// record SpMaintenance
			triggerSpSwitch := client.RecordSpMaintenance(target.SpP2PAddress, target.Time)
			if setting.Config.IsSwitchIfSpMaintenance && triggerSpSwitch {
				ReqHBLatencyCheckSpList(ctx, conn)
			}
		}
	}
}
