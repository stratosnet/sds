package event

import (
	"context"
	"github.com/stratosnet/sds/pp/setting"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"

	"github.com/alex023/clock"
)

// GetPPList P node get PPList
func GetSPList() {
	utils.DebugLog("SendMessage(client.SPConn, req, header.ReqGetSPList)")
	SendMessageToSPServer(reqGetSPlistData(), header.ReqGetSPList)
}

// RspGetPPList
func RspGetSPList(ctx context.Context, conn core.WriteCloser) {
	utils.Log("get GetSPList RSP")
	var target protos.RspGetSPList
	if !unmarshalData(ctx, &target) {
		return
	}
	utils.Log("get GetSPList RSP", target.SpList)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		reloadSPlist()
		return
	}

	spMap := make(map[string][]byte)
	for _, sp := range target.SpList {
		spMap[sp.P2PAddress] = sp.P2PPubKey
	}

	setting.SPPublicKey = spMap
}

func reloadSPlist() {
	utils.DebugLog("failed to get SPlist. retry after 3 second")
	newClock := clock.NewClock()
	newClock.AddJobRepeat(time.Second*3, 1, GetSPList)
}
