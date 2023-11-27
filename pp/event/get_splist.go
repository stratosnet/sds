package event

import (
	"context"

	"github.com/stratosnet/framework/core"
	"github.com/stratosnet/framework/utils"
	"github.com/stratosnet/sds-api/header"
	"github.com/stratosnet/sds-api/protos"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/tx-client/grpc"
)

func RspGetSPList(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetSPList
	if err := VerifyMessage(ctx, header.RspGetSPList, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
		return
	}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.DebugLog("get GetSPList RSP", target.SpList)

	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.Log("failed to get any indexing nodes, ", target.Result.Msg)
		return
	}

	srcP2pAddress := core.GetSrcP2pAddrFromContext(ctx)
	err := grpc.QueryMetaNode(srcP2pAddress)
	if err != nil {
		utils.Log("failed to verify SP, ", err.Error())
		return
	}

	if checkSpListChanged(target.SpList) {
		setting.UpdateSpMap(target.SpList)
		if err = setting.SaveSPMapToFile(); err != nil {
			utils.ErrorLogf("Couldn't save SP list to file: %v", err)
		}
	}
	network.GetPeer(ctx).RunFsm(ctx, network.EVENT_GET_SP_LIST)
}

func checkSpListChanged(list []*protos.SPBaseInfo) bool {
	listMap := make(map[string]bool)

	// Are all elements from the list in the map and unchanged?
	for _, spInList := range list {
		listMap[spInList.P2PAddress] = true
		if sp, ok := setting.SPMap.Load(spInList.P2PAddress); ok {
			spInMap := sp.(setting.SPBaseInfo)
			if spInList.NetworkAddress != spInMap.NetworkAddress || spInList.P2PPubKey != spInMap.P2PPublicKey {
				return true
			}
		} else {
			return true
		}
	}

	// Are there elements in the map but not in the list?
	changed := false
	setting.SPMap.Range(func(key, value interface{}) bool {
		spInfo := value.(setting.SPBaseInfo)
		if !listMap[spInfo.P2PAddress] {
			changed = true
			return false
		}
		return true
	})
	return changed
}
