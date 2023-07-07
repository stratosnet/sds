package pp

import (
	"sync"

	"github.com/stratosnet/sds/pp/api/rpc"
)

var (
	// key(walletAddress + P2PAddress + reqId) : value(*rpc.GetRPResult)
	rpcRPResultMap = &sync.Map{}

	// key(walletAddress + reqId) : value(*rpc.GetActivateResult)
	rpcActivateResultMap = &sync.Map{}

	// key(walletAddress + reqId) : value(*rpc.GetPrepayResult)
	rpcPrepayResultMap = &sync.Map{}

	// key(P2PAddress + reqId) : value(*rpc.GetStartMiningResult)
	rpcStartMiningResultMap = &sync.Map{}

	// key(walletAddress + reqId) : value(*rpc.GetWithdrawResult)
	rpcWithdrawResultMap = &sync.Map{}

	// key(walletAddress + reqId) : value(*rpc.GetSendResult)
	rpcSendResultMap = &sync.Map{}
)

func GetRPResult(key string) (*rpc.RPResult, bool) {
	result, loaded := rpcRPResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.RPResult), loaded
	}
	return nil, loaded
}

func SetRPResult(key string, result *rpc.RPResult) {
	if result != nil {
		rpcRPResultMap.Store(key, result)
	}
}

func GetActivateResult(key string) (*rpc.ActivateResult, bool) {
	result, loaded := rpcActivateResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.ActivateResult), loaded
	}
	return nil, loaded
}

func SetActivateResult(key string, result *rpc.ActivateResult) {
	if result != nil {
		rpcActivateResultMap.Store(key, result)
	}
}

func GetPrepayResult(key string) (*rpc.PrepayResult, bool) {
	result, loaded := rpcPrepayResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.PrepayResult), loaded
	}
	return nil, loaded
}

func SetPrepayResult(key string, result *rpc.PrepayResult) {
	if result != nil {
		rpcPrepayResultMap.Store(key, result)
	}
}

func GetStartMiningResult(key string) (*rpc.StartMiningResult, bool) {
	result, loaded := rpcStartMiningResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.StartMiningResult), loaded
	}
	return nil, loaded
}

func SetStartMiningResult(key string, result *rpc.StartMiningResult) {
	if result != nil {
		rpcStartMiningResultMap.Store(key, result)
	}
}

func GetWithdrawResult(key string) (*rpc.WithdrawResult, bool) {
	result, loaded := rpcWithdrawResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.WithdrawResult), loaded
	}
	return nil, loaded
}

func SetWithdrawResult(key string, result *rpc.WithdrawResult) {
	if result != nil {
		rpcWithdrawResultMap.Store(key, result)
	}
}

func GetSendResult(key string) (*rpc.SendResult, bool) {
	result, loaded := rpcSendResultMap.LoadAndDelete(key)
	if result != nil && loaded {
		return result.(*rpc.SendResult), loaded
	}
	return nil, loaded
}

func SetSendResult(key string, result *rpc.SendResult) {
	if result != nil {
		rpcSendResultMap.Store(key, result)
	}
}
