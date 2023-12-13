package grpc

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/pkg/errors"
	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"

	authv1beta1 "cosmossdk.io/api/cosmos/auth/v1beta1"
	abciv1beta1 "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	stakingv1beta1 "cosmossdk.io/api/cosmos/staking/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"

	registerv1 "github.com/stratosnet/stratos-chain/api/stratos/register/v1"

	"github.com/stratosnet/sds/tx-client/types"
	"github.com/stratosnet/sds/tx-client/utils"
)

func QueryAccount(address string) (*authv1beta1.BaseAccount, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := authv1beta1.NewQueryClient(conn)
	ctx := context.Background()
	req := authv1beta1.QueryAccountRequest{Address: address}

	resp, err := client.Account(ctx, &req)
	if err != nil {
		return nil, err
	}

	account := &authv1beta1.BaseAccount{}
	if err = resp.Account.UnmarshalTo(account); err != nil {
		return nil, err
	}

	return account, err
}

func QueryResourceNodeState(p2pAddress string) (state types.ResourceNodeState, err error) {
	state = types.ResourceNodeState{
		IsActive:  types.PP_INACTIVE,
		Suspended: true,
	}
	conn, err := CreateGrpcConn()
	if err != nil {
		return state, err
	}
	defer conn.Close()

	client := registerv1.NewQueryClient(conn)
	ctx := context.Background()
	req := registerv1.QueryResourceNodeRequest{NetworkAddr: p2pAddress}
	resp, err := client.ResourceNode(ctx, &req)
	if err != nil {
		return state, err
	}

	resourceNode := resp.GetNode()
	if resourceNode.GetNetworkAddress() != p2pAddress {
		return state, nil
	}

	state.Suspended = resourceNode.Suspend
	switch resourceNode.GetStatus() {
	case stakingv1beta1.BondStatus_BOND_STATUS_BONDED:
		state.IsActive = types.PP_ACTIVE
	case stakingv1beta1.BondStatus_BOND_STATUS_UNBONDING:
		state.IsActive = types.PP_UNBONDING
	case stakingv1beta1.BondStatus_BOND_STATUS_UNBONDED:
		state.IsActive = types.PP_INACTIVE
	}

	tokenInt64, err := strconv.ParseInt(resourceNode.Tokens, 10, 64)
	if err != nil {
		return state, err
	}
	state.Tokens = big.NewInt(tokenInt64)
	return state, nil
}

func QueryMetaNode(p2pAddress string) (err error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return err
	}
	defer conn.Close()
	client := registerv1.NewQueryClient(conn)
	ctx := context.Background()
	req := registerv1.QueryMetaNodeRequest{NetworkAddr: p2pAddress}
	resp, err := client.MetaNode(ctx, &req)
	if err != nil {
		return err
	}

	metaNode := resp.GetNode()
	if metaNode.GetNetworkAddress() != p2pAddress {
		return errors.New("")
	}

	if metaNode.Suspend {
		return errors.New("")
	}
	return nil
}

func QueryTxByHash(txHash string) (*abciv1beta1.TxResponse, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := txv1beta1.NewServiceClient(conn)
	ctx := context.Background()
	req := txv1beta1.GetTxRequest{Hash: txHash}
	resp, err := client.GetTx(ctx, &req)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		errMsg := fmt.Sprintf("QueryTxByHash returned nil response for transaction hash [%v]", txHash)
		return nil, errors.New(errMsg)
	}
	utils.Logf("--- resp is %v", *resp.TxResponse)
	// skip non-successful tx
	if resp.GetTxResponse().Code != 0 {
		errMsg := fmt.Sprintf("Tx with hash[%v] failed: [%v]", txHash, resp.GetTxResponse().String())
		return nil, errors.New(errMsg)
	}
	return resp.TxResponse, nil
}

func QueryVolumeReport(epoch *big.Int) (*potv1.QueryVolumeReportResponse, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := potv1.NewQueryClient(conn)
	ctx := context.Background()
	req := potv1.QueryVolumeReportRequest{Epoch: epoch.Int64()}

	resp, err := client.VolumeReport(ctx, &req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func QueryRemainingOzoneLimit() (*big.Int, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := registerv1.NewQueryClient(conn)
	ctx := context.Background()
	req := registerv1.QueryRemainingOzoneLimitRequest{}
	resp, err := client.RemainingOzoneLimit(ctx, &req)
	if err != nil {
		return nil, err
	}

	if resp.GetOzoneLimit() == "" {
		return nil, errors.New("remaining ozone limit is nil in the response from stchain")
	}

	limit, err := strconv.ParseInt(resp.GetOzoneLimit(), 10, 64)
	if err != nil {
		return nil, err
	}

	return big.NewInt(limit), nil
}
