package grpc

import (
	"context"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"

	pptypes "github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/relay"
	relaytypes "github.com/stratosnet/sds/relay/stratoschain/types"
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
		IsActive:  pptypes.PP_INACTIVE,
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
		state.IsActive = pptypes.PP_ACTIVE
	case stakingtypes.Unbonding:
		state.IsActive = pptypes.PP_UNBONDING
	case stakingtypes.Unbonded:
		state.IsActive = pptypes.PP_INACTIVE
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
