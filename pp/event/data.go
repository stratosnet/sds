package event

import (
	"context"

	"github.com/cosmos/cosmos-proto/anyutil"

	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/sds-msg/protos"
	msgtypes "github.com/stratosnet/sds/sds-msg/types"
	txclienttx "github.com/stratosnet/sds/tx-client/tx"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"
)

func reqActivateData(ctx context.Context, amount txclienttypes.Coin, txFee txclienttypes.TxFee) (*protos.ReqActivatePP, error) {
	// Create and sign transaction to add new resource node
	ownerAddress, err := fwtypes.WalletAddressFromBech32(setting.WalletAddress)
	if err != nil {
		return nil, err
	}

	beneficiaryAddress, err := fwtypes.WalletAddressFromBech32(setting.BeneficiaryAddress)
	if err != nil {
		return nil, err
	}

	txMsg, err := txclienttx.BuildCreateResourceNodeMsg(msgtypes.STORAGE, p2pserver.GetP2pServer(ctx).GetP2PPublicKey(), amount, ownerAddress, beneficiaryAddress)
	if err != nil {
		return nil, err
	}
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

	req := &protos.ReqActivatePP{
		Tx:             txBytes,
		PpInfo:         p2pserver.GetP2pServer(ctx).GetPPInfo(),
		AlreadyActive:  false,
		InitialDeposit: amount.String(),
	}
	return req, nil
}

func reqUpdateDepositData(ctx context.Context, depositDelta txclienttypes.Coin, txFee txclienttypes.TxFee) (*protos.ReqUpdateDepositPP, error) {
	// Create and sign transaction to update deposit for existing resource node
	networkAddr := p2pserver.GetP2pServer(ctx).GetP2PAddress()
	ownerAddr := fwtypes.WalletAddress(setting.WalletPublicKey.Address())

	txMsg := txclienttx.BuildUpdateResourceNodeDepositMsg(networkAddr, ownerAddr, depositDelta)
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

	req := &protos.ReqUpdateDepositPP{
		Tx:           txBytes,
		P2PAddress:   p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
		DepositDelta: depositDelta.String(),
	}
	return req, nil
}

func reqDeactivateData(ctx context.Context, txFee txclienttypes.TxFee) (*protos.ReqDeactivatePP, error) {
	// Create and sign transaction to remove a resource node
	nodeAddress := p2pserver.GetP2pServer(ctx).GetP2PAddress()
	ownerAddress := fwtypes.WalletAddress(setting.WalletPublicKey.Address())

	txMsg := txclienttx.BuildRemoveResourceNodeMsg(nodeAddress, ownerAddress)
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

	req := &protos.ReqDeactivatePP{
		Tx:         txBytes,
		P2PAddress: p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
	}
	return req, nil
}

func reqPrepayData(ctx context.Context, beneficiary fwtypes.WalletAddress, amount txclienttypes.Coin, txFee txclienttypes.TxFee,
	walletAddr string, walletPubkey, wsign []byte, reqTime int64) (*protos.ReqPrepay, error) {
	// Create and sign a prepay transaction
	senderAddress, err := fwtypes.WalletAddressFromBech32(setting.WalletAddress)
	if err != nil {
		return nil, err
	}

	txMsg := txclienttx.BuildPrepayMsg(senderAddress, beneficiary, amount)
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

	walletSign := &protos.Signature{
		Address:   walletAddr,
		Pubkey:    walletPubkey,
		Signature: wsign,
		Type:      protos.SignatureType_WALLET,
	}
	req := &protos.ReqPrepay{
		Tx:         txBytes,
		P2PAddress: p2pserver.GetP2pServer(ctx).GetP2PAddress().String(),
		Signature:  walletSign,
		ReqTime:    reqTime,
	}
	return req, nil
}
