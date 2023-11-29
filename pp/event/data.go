package event

import (
	"context"

	"github.com/stratosnet/sds/framework/utils/crypto/ed25519"
	"github.com/stratosnet/sds/framework/utils/crypto/secp256k1"
	"github.com/stratosnet/sds/framework/utils/types"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/sds-msg/protos"
	txclienttx "github.com/stratosnet/sds/tx-client/tx"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"
	"google.golang.org/protobuf/types/known/anypb"
)

func reqActivateData(ctx context.Context, amount txclienttypes.Coin, txFee txclienttypes.TxFee) (*protos.ReqActivatePP, error) {
	// Create and sign transaction to add new resource node
	ownerAddress, err := types.WalletAddressFromBech(setting.WalletAddress)
	if err != nil {
		return nil, err
	}

	p2pAddress := p2pserver.GetP2pServer(ctx).GetP2PAddrInTypeAddress()
	txMsg, err := txclienttx.BuildCreateResourceNodeMsg(txclienttypes.STORAGE, p2pserver.GetP2pServer(ctx).GetP2PPublicKey(), amount, ownerAddress.Bytes(), p2pAddress.Bytes())
	if err != nil {
		return nil, err
	}
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: txclienttypes.SignatureSecp256k1},
	}

	chainId := setting.Config.Blockchain.ChainId
	gasAdjustment := setting.Config.Blockchain.GasAdjustment

	msgAny, err := anypb.New(txMsg)
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
	networkAddr := ed25519.PubKeyBytesToAddress(p2pserver.GetP2pServer(ctx).GetP2PPublicKey())
	ownerAddr, err := secp256k1.PubKeyToAddress(setting.WalletPublicKey)
	if err != nil {
		return nil, err
	}

	txMsg := txclienttx.BuildUpdateResourceNodeDepositMsg(networkAddr.Bytes(), ownerAddr.Bytes(), depositDelta)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: txclienttypes.SignatureSecp256k1},
	}

	chainId := setting.Config.Blockchain.ChainId
	gasAdjustment := setting.Config.Blockchain.GasAdjustment

	msgAny, err := anypb.New(txMsg)
	if err != nil {
		return nil, err
	}

	txBytes, err := txclienttx.CreateAndSimulateTx(msgAny, txFee, "", signatureKeys, chainId, gasAdjustment)
	if err != nil {
		return nil, err
	}

	req := &protos.ReqUpdateDepositPP{
		Tx:         txBytes,
		P2PAddress: p2pserver.GetP2pServer(ctx).GetP2PAddress(),
	}
	return req, nil
}

func reqDeactivateData(ctx context.Context, txFee txclienttypes.TxFee) (*protos.ReqDeactivatePP, error) {
	// Create and sign transaction to remove a resource node
	nodeAddress := ed25519.PubKeyBytesToAddress(p2pserver.GetP2pServer(ctx).GetP2PPublicKey())
	ownerAddress, err := secp256k1.PubKeyToAddress(setting.WalletPublicKey)
	if err != nil {
		return nil, err
	}

	txMsg := txclienttx.BuildRemoveResourceNodeMsg(nodeAddress.Bytes(), ownerAddress.Bytes())
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: txclienttypes.SignatureSecp256k1},
	}

	chainId := setting.Config.Blockchain.ChainId
	gasAdjustment := setting.Config.Blockchain.GasAdjustment

	msgAny, err := anypb.New(txMsg)
	if err != nil {
		return nil, err
	}

	txBytes, err := txclienttx.CreateAndSimulateTx(msgAny, txFee, "", signatureKeys, chainId, gasAdjustment)
	if err != nil {
		return nil, err
	}

	req := &protos.ReqDeactivatePP{
		Tx:         txBytes,
		P2PAddress: p2pserver.GetP2pServer(ctx).GetP2PAddress(),
	}
	return req, nil
}

func reqPrepayData(ctx context.Context, beneficiary []byte, amount txclienttypes.Coin, txFee txclienttypes.TxFee,
	walletAddr string, walletPubkey, wsign []byte, reqTime int64) (*protos.ReqPrepay, error) {
	// Create and sign a prepay transaction
	senderAddress, err := types.WalletAddressFromBech(setting.WalletAddress)
	if err != nil {
		return nil, err
	}

	txMsg := txclienttx.BuildPrepayMsg(senderAddress.Bytes(), beneficiary, amount)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: txclienttypes.SignatureSecp256k1},
	}

	chainId := setting.Config.Blockchain.ChainId
	gasAdjustment := setting.Config.Blockchain.GasAdjustment

	msgAny, err := anypb.New(txMsg)
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
		P2PAddress: p2pserver.GetP2pServer(ctx).GetP2PAddress(),
		Signature:  walletSign,
		ReqTime:    reqTime,
	}
	return req, nil
}
