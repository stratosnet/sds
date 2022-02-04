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
func RspGetSPList(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get GetSPList RSP")
	var target protos.RspGetSPList
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.DebugLog("get GetSPList RSP", target.SpList)
	if target.Result.State != protos.ResultState_RES_SUCCESS {
		utils.Log("failed to get any indexing nodes, reloading")
		peers.ScheduleReloadSPlist(time.Second * 3)
		return
	}

	changed := false
	for _, sp := range target.SpList {
		existing, ok := setting.SPMap.Load(sp.P2PAddress)
		if ok {
			existingSp := existing.(setting.SPBaseInfo)
			if sp.P2PPubKey != existingSp.P2PPublicKey || sp.NetworkAddress != existingSp.NetworkAddress {
				changed = true
			}
		} else {
			changed = true
		}

		setting.SPMap.Store(sp.P2PAddress, setting.SPBaseInfo{
			P2PAddress:     sp.P2PAddress,
			P2PPublicKey:   sp.P2PPubKey,
			NetworkAddress: sp.NetworkAddress,
		})
	}
	if changed {
		setting.SPMap.Delete("unknown")
		setting.Config.SPList = nil
		setting.SPMap.Range(func(k, v interface{}) bool {
			sp := v.(setting.SPBaseInfo)
			setting.Config.SPList = append(setting.Config.SPList, sp)
			return true
		})
		if err := utils.WriteConfig(setting.Config, setting.ConfigPath); err != nil {
			utils.ErrorLog("Couldn't write config with updated SP list to yaml file", err)
		}
	}
}
