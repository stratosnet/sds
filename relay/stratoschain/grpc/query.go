package grpc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	registertypes "github.com/stratosnet/stratos-chain/x/register/types"

	"github.com/stratosnet/sds/relay"
	relaytypes "github.com/stratosnet/sds/relay/stratoschain/types"
	"github.com/stratosnet/sds/relay/utils"
)

func QueryAccount(address string) (*authtypes.BaseAccount, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := authtypes.NewQueryClient(conn)
	ctx := context.Background()
	req := authtypes.QueryAccountRequest{Address: address}

	resp, err := client.Account(ctx, &req)
	if err != nil {
		return nil, err
	}

	err = resp.UnpackInterfaces(relay.ProtoCdc)
	if err != nil {
		return nil, err
	}
	cachedAcc := resp.GetAccount().GetCachedValue()
	account := cachedAcc.(*authtypes.BaseAccount)

	return account, err
}

func QueryResourceNodeState(p2pAddress string) (state relaytypes.ResourceNodeState, err error) {
	state = relaytypes.ResourceNodeState{
		IsActive:  relaytypes.PP_INACTIVE,
		Suspended: true,
	}
	conn, err := CreateGrpcConn()
	if err != nil {
		return state, err
	}
	defer conn.Close()

	client := registertypes.NewQueryClient(conn)
	ctx := context.Background()
	req := registertypes.QueryResourceNodeRequest{NetworkAddr: p2pAddress}
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
	case stakingtypes.Bonded:
		state.IsActive = relaytypes.PP_ACTIVE
	case stakingtypes.Unbonding:
		state.IsActive = relaytypes.PP_UNBONDING
	case stakingtypes.Unbonded:
		state.IsActive = relaytypes.PP_INACTIVE
	}

	state.Tokens = resourceNode.Tokens.BigInt()
	return state, nil
}

func QueryMetaNode(p2pAddress string) (err error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return err
	}
	defer conn.Close()
	client := registertypes.NewQueryClient(conn)
	ctx := context.Background()
	req := registertypes.QueryMetaNodeRequest{NetworkAddr: p2pAddress}
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

func QueryTxByHash(txHash string) (*types.TxResponse, error) {
	conn, err := CreateGrpcConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := sdktx.NewServiceClient(conn)
	ctx := context.Background()
	req := sdktx.GetTxRequest{Hash: txHash}
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
