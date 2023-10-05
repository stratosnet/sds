package namespace

import (
	"context"
	"time"

	"github.com/stratosnet/sds/metrics"
	rpc_api "github.com/stratosnet/sds/relay/rpc"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

const (
	// WAIT_TIMEOUT timeout for waiting result from external source, in seconds
	WAIT_TIMEOUT time.Duration = 10 * time.Second

	SIGNATURE_INFO_TTL = 10 * time.Minute
)

var (
	signatureInfoMap = utils.NewAutoCleanMap(SIGNATURE_INFO_TTL)
)

type signatureInfo struct {
	signature rpc_api.Signature
	reqTime   int64
}

type rpcPubApi struct {
}

func RpcPubApi() *rpcPubApi {
	return &rpcPubApi{}
}

type rpcPrivApi struct {
}

func RpcPrivApi() *rpcPrivApi {
	return &rpcPrivApi{}
}

// apis returns the collection of built-in RPC APIs.
func Apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "owner",
			Version:   "1.0",
			Service:   RpcPrivApi(),
			Public:    false,
		},
		//{
		//	Namespace: "user",
		//	Version:   "1.0",
		//	Service:   RpcPubApi(),
		//	Public:    true,
		//},
	}
}

func (api *rpcPrivApi) RequestSync(ctx context.Context, param rpc_api.ParamReqSync) rpc_api.SyncResult {
	metrics.RpcReqCount.WithLabelValues("RequestSync").Inc()
	txHash := param.TxHash

	// verify if wallet and public key match
	if len(txHash) == 0 {
		return rpc_api.SyncResult{Return: rpc_api.WRONG_INPUT}
	}

	grpc.QueryTxByHash(txHash)
	return rpc_api.SyncResult{Return: rpc_api.SUCCESS}
	// Store initial signature info
	//signatureInfoMap.Store(txHash, signatureInfo{
	//	signature: param.Signature,
	//	reqTime:   reqTime,
	//})
}
