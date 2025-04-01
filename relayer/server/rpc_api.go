package server

import (
	"context"

	authv1beta1 "cosmossdk.io/api/cosmos/auth/v1beta1"
	abciv1beta1 "cosmossdk.io/api/cosmos/base/abci/v1beta1"

	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
	registerv1 "github.com/stratosnet/stratos-chain/api/stratos/register/v1"
	sdsv1 "github.com/stratosnet/stratos-chain/api/stratos/sds/v1"

	"github.com/stratosnet/sds/tx-client/grpc"
	"github.com/stratosnet/sds/tx-client/types"
)

type rpcApi struct {
}

func RpcAPI() *rpcApi {
	return &rpcApi{}
}

func (api *rpcApi) Account(ctx context.Context, walletAddress string) (*authv1beta1.BaseAccount, error) {
	return grpc.QueryAccount(walletAddress)
}

func (api *rpcApi) ResourceNode(ctx context.Context, p2pAddress string) (*registerv1.ResourceNode, error) {
	return grpc.QueryResourceNode(p2pAddress)
}

func (api *rpcApi) ResourceNodeState(ctx context.Context, p2pAddress string) (state types.ResourceNodeState, err error) {
	return grpc.QueryResourceNodeState(p2pAddress)
}

func (api *rpcApi) MetaNode(ctx context.Context, p2pAddress string) (*registerv1.MetaNode, error) {
	val, err := grpc.QueryMetaNode(p2pAddress)
	return val, err
}

func (api *rpcApi) TxHash(ctx context.Context, txHash string) (*abciv1beta1.TxResponse, error) {
	return grpc.QueryTxByHash(txHash)
}

func (api *rpcApi) VolumeReport(ctx context.Context, epoch int64) (*potv1.QueryVolumeReportResponse, error) {
	return grpc.QueryVolumeReport(epoch)
}

func (api *rpcApi) NozSupply(ctx context.Context) (*sdsv1.QueryNozSupplyResponse, error) {
	return grpc.QueryNozSupply()
}

func (api *rpcApi) MerkleRoot(ctx context.Context, commitment string) (*registerv1.QueryMerkleRootResponse, error) {
	return grpc.QueryMerkleRoot(commitment)
}
