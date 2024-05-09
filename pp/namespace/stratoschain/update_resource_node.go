package stratoschain

import (
	"context"

	"github.com/cosmos/cosmos-proto/anyutil"
	"github.com/stratosnet/sds/tx-client/grpc"
	registerv1 "github.com/stratosnet/stratos-chain/api/stratos/register/v1"

	"github.com/stratosnet/sds/framework/core"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/tx"
	txclienttx "github.com/stratosnet/sds/tx-client/tx"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"
)

// Broadcast updateResourceNode tx to stratos-chain directly
func UpdateResourceNode(ctx context.Context, moniker, identity, website, securityContact, details string, txFee txclienttypes.TxFee) error {
	ppInfo, err := grpc.QueryResourceNode(setting.Config.Keys.P2PAddress)
	if err != nil {
		pp.ErrorLog(ctx, "Failed to query pp info from stratos-chain: "+err.Error())
		return err
	}

	newDescription := ppInfo.GetDescription()

	if len(moniker) > 0 {
		newDescription.Moniker = moniker
	}
	if len(identity) > 0 {
		newDescription.Identity = identity
	}
	if len(website) > 0 {
		newDescription.Website = website
	}
	if len(securityContact) > 0 {
		newDescription.SecurityContact = securityContact
	}
	if len(details) > 0 {
		newDescription.Details = details
	}

	updateResourceNodeTxBytes, err := reqUpdateResourceNodeData(ctx, newDescription, ppInfo.GetNodeType(), txFee)
	if err != nil {
		pp.ErrorLog(ctx, "Couldn't build updateResourceNode transaction: "+err.Error())
		return err
	}

	err = tx.BroadcastTx(updateResourceNodeTxBytes)
	if err != nil {
		pp.ErrorLog(ctx, "The updateResourceNode transaction couldn't be broadcast", err)
		return err
	}

	reqId := core.GetRemoteReqId(ctx)
	if reqId != "" {
		rpcResult := &rpc.SendResult{
			Return: rpc.SUCCESS,
		}
		defer pp.SetRPCResult(setting.WalletAddress+reqId, rpcResult)
	}

	pp.Log(ctx, "Send transaction delivered.")
	return nil
}

func reqUpdateResourceNodeData(_ context.Context, description *registerv1.Description, nodeType uint32, txFee txclienttypes.TxFee) ([]byte, error) {
	networkAddress, err := fwtypes.P2PAddressFromBech32(setting.Config.Keys.P2PAddress)
	if err != nil {
		return nil, err
	}
	ownerAddress, err := fwtypes.WalletAddressFromBech32(setting.WalletAddress)
	if err != nil {
		return nil, err
	}
	beneficiaryAddress, err := fwtypes.WalletAddressFromBech32(setting.BeneficiaryAddress)
	if err != nil {
		return nil, err
	}

	txMsg := txclienttx.BuildUpdateResourceNodeMsg(networkAddress, ownerAddress, beneficiaryAddress, description, nodeType)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey.Bytes(), Type: txclienttypes.SignatureSecp256k1},
	}

	chainId := setting.Config.Blockchain.ChainId
	gasAdjustment := setting.Config.Blockchain.GasAdjustment

	msgAny, err := anyutil.New(txMsg)
	if err != nil {
		return nil, err
	}

	txBytes, err := txclienttx.CreateAndSimulateTx(msgAny, txFee, "", signatureKeys, chainId, gasAdjustment)
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}
