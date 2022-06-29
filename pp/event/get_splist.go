package event

import (
	"context"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// RspGetPPList
func RspGetIndexNodeList(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get GetIndexNodeList RSP")
	var target protos.RspGetIndexNodeList
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.DebugLog("get GetIndexNodeList RSP", target.IndexNodeList)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.Log("failed to get any indexing nodes, reloading")
		peers.ScheduleReloadIndexNodelist(time.Second * 3)
		return
	}

	changed := false
	for _, indexNode := range target.IndexNodeList {
		existing, ok := setting.IndexNodeMap.Load(indexNode.P2PAddress)
		if ok {
			existingIndexNode := existing.(setting.IndexNodeBaseInfo)
			if indexNode.P2PPubKey != existingIndexNode.P2PPublicKey || indexNode.NetworkAddress != existingIndexNode.NetworkAddress {
				changed = true
			}
		} else {
			changed = true
		}

		setting.IndexNodeMap.Store(indexNode.P2PAddress, setting.IndexNodeBaseInfo{
			P2PAddress:     indexNode.P2PAddress,
			P2PPublicKey:   indexNode.P2PPubKey,
			NetworkAddress: indexNode.NetworkAddress,
		})
	}
	if changed {
		setting.IndexNodeMap.Delete("unknown")
		setting.Config.IndexNodeList = nil
		setting.IndexNodeMap.Range(func(k, v interface{}) bool {
			indexNode := v.(setting.IndexNodeBaseInfo)
			setting.Config.IndexNodeList = append(setting.Config.IndexNodeList, indexNode)
			return true
		})
		if err := utils.WriteTomlConfig(setting.Config, setting.ConfigPath); err != nil {
			utils.ErrorLog("Couldn't write config with updated Index Node list to yaml file", err)
		}
	}
}
