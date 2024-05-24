package pp

import (
	"sync"

	"github.com/stratosnet/sds/pp/api/rpc"
)

var (
	/**
	RPResult:                             key(walletAddress + P2PAddress + reqId) - value(*rpc.RPResult)
	ActivateResult:                       key(walletAddress + reqId)              - value(*rpc.ActivateResult)
	PrepayResult:                         key(walletAddress + reqId)              - value(*rpc.PrepayResult)
	StartMiningResult:                    key(P2PAddress + reqId)                 - value(*rpc.StartMiningResult)
	WithdrawResult:                       key(walletAddress + reqId)              - value(*rpc.WithdrawResult)
	SendResult:                           key(walletAddress + reqId)              - value(*rpc.SendResult)
	StatusResult:                         key(P2PAddress + reqId)                 - value(*rpc.StatusResult)
	ClearExpiredShareLinksResult:         key(walletAddress + reqId)              - value(*rpc.ClearExpiredShareLinksResult)
	*/
	rpcResultMap = &sync.Map{}
)

func SetRPCResult(key string, result interface{}) {
	if result != nil {
		rpcResultMap.Store(key, result)
	}
}

func GetRPResult(key string) (*rpc.RPResult, bool) {
	result, loaded := rpcResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.RPResult), loaded
	}
	return nil, loaded
}

func GetActivateResult(key string) (*rpc.ActivateResult, bool) {
	result, loaded := rpcResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.ActivateResult), loaded
	}
	return nil, loaded
}

func GetPrepayResult(key string) (*rpc.PrepayResult, bool) {
	result, loaded := rpcResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.PrepayResult), loaded
	}
	return nil, loaded
}

func GetStartMiningResult(key string) (*rpc.StartMiningResult, bool) {
	result, loaded := rpcResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.StartMiningResult), loaded
	}
	return nil, loaded
}

func GetStatusResult(key string) (*rpc.StatusResult, bool) {
	result, loaded := rpcResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.StatusResult), loaded
	}
	return nil, loaded
}

func GetWithdrawResult(key string) (*rpc.WithdrawResult, bool) {
	result, loaded := rpcResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.WithdrawResult), loaded
	}
	return nil, loaded
}

func GetSendResult(key string) (*rpc.SendResult, bool) {
	result, loaded := rpcResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.SendResult), loaded
	}
	return nil, loaded
}

func GetUpdatePPInfoResult(key string) (*rpc.UpdatePPInfoResult, bool) {
	result, loaded := rpcResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.UpdatePPInfoResult), loaded
	}
	return nil, loaded
}
