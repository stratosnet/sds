package peers

import (
	"context"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
)

// ReportNodeStatus
func ReportNodeStatus(ctx context.Context) func() {
	return func() {
		if setting.IsStartMining && setting.State == types.PP_ACTIVE {
			status := requests.ReqNodeStatusData()
			go doReportNodeStatus(ctx, status)
		}
	}
}

func doReportNodeStatus(ctx context.Context, status *protos.ReqReportNodeStatus) {
	pp.DebugLog(ctx, "Sending RNS message to SP! "+status.String())
	SendMessageToSPServer(ctx, status, header.ReqReportNodeStatus)
	// if current reachable is too less, try refresh the list
	_, total, connected := peerList.GetPPList(ctx)
	pp.Logf(ctx, "#pp_in_list:[%d], #pp_connected:[%d]", total, connected)
	if total > 0 && total <= 2 {
		GetPPListFromSP(ctx)
	}
}
