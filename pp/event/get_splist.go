package event

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	"github.com/stratosnet/sds/utils"
)

const (
	SPLIST_INTERVAL_BASE = 60 // In second
	SPLIST_MAX_JITTER    = 10 // In second
)

func RspGetSPList(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspGetSPList
	if err := VerifyMessage(ctx, header.RspGetSPList, &target); err != nil {
		utils.ErrorLog("failed verifying the message, ", err.Error())
	}

	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	defer network.GetPeer(ctx).ScheduleReloadSPlist(ctx, time.Second*time.Duration(SPLIST_INTERVAL_BASE+rand.Intn(SPLIST_MAX_JITTER)))
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

	if checkSpListChanged(target.SpList, setting.SPMap) {
		setting.SPMap = &sync.Map{}
		updateSpMap(target.SpList, setting.SPMap)
		setting.Config.SPList = nil
		if err := setting.FlushConfig(); err != nil {
			utils.ErrorLog("Couldn't write config with updated SP list to yaml file", err)
		}
	}
	network.GetPeer(ctx).RunFsm(ctx, network.EVENT_GET_SP_LIST)
}

func updateSpMap(lst []*protos.SPBaseInfo, mp *sync.Map) {
	for _, spInList := range lst {
		spInMap := &setting.SPBaseInfo{
			P2PAddress:     spInList.P2PAddress,
			P2PPublicKey:   spInList.P2PPubKey,
			NetworkAddress: spInList.NetworkAddress,
		}
		mp.Store(spInList.P2PAddress, spInMap)
	}
}

func checkSpListChanged(lst []*protos.SPBaseInfo, mp *sync.Map) bool {
	// Compare the elements in the slice and sync.Map
	for _, spInList := range lst {
		if sp, ok := mp.Load(spInList.P2PAddress); ok {
			spInMap := sp.(setting.SPBaseInfo)
			if spInList.NetworkAddress != spInMap.NetworkAddress || spInList.P2PPubKey != spInMap.P2PPublicKey {
				return true
			}
		} else {
			return true
		}
	}

	changed := false
	mp.Range(func(key, value interface{}) bool {
		if _, ok := func(list []*protos.SPBaseInfo, spInfo setting.SPBaseInfo) (int, bool) {
			for i, spInList := range list {
				if spInList.P2PAddress == spInfo.P2PAddress &&
					spInList.NetworkAddress == spInfo.NetworkAddress &&
					spInList.P2PPubKey == spInfo.P2PPublicKey {
					return i, true
				}
			}
			changed = true
			return -1, false
		}(lst, value.(setting.SPBaseInfo)); !ok {
			return false
		}
		return true
	})
	return changed
}
