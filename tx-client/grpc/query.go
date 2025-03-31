package grpc

import (
	"context"
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
	sdsv1 "github.com/stratosnet/stratos-chain/api/stratos/sds/v1"

	authv1beta1 "cosmossdk.io/api/cosmos/auth/v1beta1"
	abciv1beta1 "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	stakingv1beta1 "cosmossdk.io/api/cosmos/staking/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"

	registerv1 "github.com/stratosnet/stratos-chain/api/stratos/register/v1"

	"github.com/stratosnet/sds/framework/utils"
	msgtypes "github.com/stratosnet/sds/sds-msg/types"
	"github.com/stratosnet/sds/tx-client/types"
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

func QueryResourceNode(p2pAddress string) (*registerv1.ResourceNode, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := registerv1.NewQueryClient(conn)
	ctx := context.Background()
	req := registerv1.QueryResourceNodeRequest{NetworkAddr: p2pAddress}
	resp, err := client.ResourceNode(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.GetNode(), nil
}

func QueryResourceNodeState(p2pAddress string) (state types.ResourceNodeState, err error) {
	state = types.ResourceNodeState{
		IsActive:  msgtypes.PP_INACTIVE,
		Suspended: true,
	}
	conn, err := CreateGrpcConn()
	if err != nil {
		return state, err
	}
	defer conn.Close()

	resourceNode, err := QueryResourceNode(p2pAddress)
	if err != nil {
		return state, err
	}

	if resourceNode.GetNetworkAddress() != p2pAddress {
		return state, nil
	}

	state.Suspended = resourceNode.Suspend
	switch resourceNode.GetStatus() {
	case stakingv1beta1.BondStatus_BOND_STATUS_BONDED:
		state.IsActive = msgtypes.PP_ACTIVE
	case stakingv1beta1.BondStatus_BOND_STATUS_UNBONDING:
		state.IsActive = msgtypes.PP_UNBONDING
	case stakingv1beta1.BondStatus_BOND_STATUS_UNBONDED:
		state.IsActive = msgtypes.PP_INACTIVE
	}

	tokens, success := big.NewInt(0).SetString(resourceNode.Tokens, 10)
	if !success {
		return state, errors.Errorf("token amount [%v] is an invalid big.Int string", resourceNode.Tokens)
	}

	state.Tokens = tokens
	return state, nil
}

func QueryMetaNode(p2pAddress string) (*registerv1.MetaNode, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := registerv1.NewQueryClient(conn)
	ctx := context.Background()
	req := registerv1.QueryMetaNodeRequest{NetworkAddr: p2pAddress}
	resp, err := client.MetaNode(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.GetNode(), nil
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

func QueryVolumeReport(epoch int64) (*potv1.QueryVolumeReportResponse, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := potv1.NewQueryClient(conn)
	ctx := context.Background()
	req := potv1.QueryVolumeReportRequest{Epoch: epoch}

	resp, err := client.VolumeReport(ctx, &req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// QueryNozSupply queries the remaining ozone limit and the total ozone supply from stchain
func QueryNozSupply() (*sdsv1.QueryNozSupplyResponse, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := sdsv1.NewQueryClient(conn)
	ctx := context.Background()
	req := sdsv1.QueryNozSupplyRequest{}
	resp, err := client.NozSupply(ctx, &req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// SimPrepay simulate the prepay and get the estimated noz
func SimPrepay(amount types.Coin) (*sdsv1.QuerySimPrepayResponse, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := sdsv1.NewQueryClient(conn)
	ctx := context.Background()
	req := sdsv1.QuerySimPrepayRequest{Amount: amount.String()}
	resp, err := client.SimPrepay(ctx, &req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
