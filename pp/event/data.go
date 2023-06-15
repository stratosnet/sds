package event

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/cosmos-sdk/crypto/ledger"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/relay"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	relaytypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/sds/utils/types"
	stchaintypes "github.com/stratosnet/stratos-chain/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"
)

func reqActivateData(ctx context.Context, amount types.Coin, txFee types.TxFee) (*protos.ReqActivatePP, error) {
	// Create and sign transaction to add new resource node
	ownerAddress, err := types.WalletAddressFromBech(setting.WalletAddress)
	if err != nil {
		return nil, err
	}

	p2pAddress := p2pserver.GetP2pServer(ctx).GetP2PAddrInTypeAddress()
	txMsg, err := stratoschain.BuildCreateResourceNodeMsg(registertypes.STORAGE, p2pserver.GetP2pServer(ctx).GetP2PPublicKey(), amount, ownerAddress, p2pAddress)
	if err != nil {
		return nil, err
	}
	signatureKeys := []relaytypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: relaytypes.SignatureSecp256k1},
	}

	txBytes, err := CreateAndSimulateTx(txMsg, registertypes.TypeMsgCreateResourceNode, txFee, "", signatureKeys)
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

func reqUpdateDepositData(ctx context.Context, depositDelta types.Coin, txFee types.TxFee) (*protos.ReqUpdateDepositPP, error) {
	// Create and sign transaction to update deposit for existing resource node
	networkAddr := ed25519.PubKeyBytesToAddress(p2pserver.GetP2pServer(ctx).GetP2PPublicKey())
	ownerAddr, err := secp256k1.PubKeyToAddress(setting.WalletPublicKey)
	if err != nil {
		return nil, err
	}

	txMsg := stratoschain.BuildUpdateResourceNodeDepositMsg(networkAddr, *ownerAddr, depositDelta)
	signatureKeys := []relaytypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: relaytypes.SignatureSecp256k1},
	}

	txBytes, err := CreateAndSimulateTx(txMsg, registertypes.TypeMsgUpdateResourceNodeDeposit, txFee, "", signatureKeys)
	if err != nil {
		return nil, err
	}

	req := &protos.ReqUpdateDepositPP{
		Tx:         txBytes,
		P2PAddress: p2pserver.GetP2pServer(ctx).GetP2PAddress(),
	}
	return req, nil
}

func reqDeactivateData(ctx context.Context, txFee types.TxFee) (*protos.ReqDeactivatePP, error) {
	// Create and sign transaction to remove a resource node
	nodeAddress := ed25519.PubKeyBytesToAddress(p2pserver.GetP2pServer(ctx).GetP2PPublicKey())
	ownerAddress, err := secp256k1.PubKeyToAddress(setting.WalletPublicKey)
	if err != nil {
		return nil, err
	}

	txMsg := stratoschain.BuildRemoveResourceNodeMsg(nodeAddress, *ownerAddress)
	signatureKeys := []relaytypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: relaytypes.SignatureSecp256k1},
	}

	txBytes, err := CreateAndSimulateTx(txMsg, registertypes.TypeMsgRemoveResourceNode, txFee, "", signatureKeys)
	if err != nil {
		return nil, err
	}

	req := &protos.ReqDeactivatePP{
		Tx:         txBytes,
		P2PAddress: p2pserver.GetP2pServer(ctx).GetP2PAddress(),
	}
	return req, nil
}

func reqPrepayData(ctx context.Context, beneficiary []byte, amount types.Coin, txFee types.TxFee,
	walletAddr string, walletPubkey, wsign []byte, reqTime int64) (*protos.ReqPrepay, error) {
	// Create and sign a prepay transaction
	senderAddress, err := types.WalletAddressFromBech(setting.WalletAddress)
	if err != nil {
		return nil, err
	}

	txMsg := stratoschain.BuildPrepayMsg(senderAddress.Bytes(), beneficiary, amount)
	signatureKeys := []relaytypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: relaytypes.SignatureSecp256k1},
	}

	txBytes, err := CreateAndSimulateTx(txMsg, sdstypes.TypeMsgPrepay, txFee, "", signatureKeys)
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

func ReqSendMsg(ctx context.Context, useLedger bool, toAddress types.Address, amount types.Coin, txFee types.TxFee) ([]byte, error) {
	var fromAddress types.Address
	var err error
	var txBytes []byte
	var signatureKeys []relaytypes.SignatureKey
	if useLedger {
		path := *hd.NewFundraiserParams(0, stchaintypes.CoinType, 0)
		_, addr, err := ledger.NewPrivKeySecp256k1(path, stchaintypes.AccountAddressPrefix)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting address, ")
		}
		pp.DebugLog(ctx, "Wallet Address Read from ledger:", addr)
		fromAddress, err = types.WalletAddressFromBech(addr)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting wallet address, ")
		}

		signatureKeys = []relaytypes.SignatureKey{
			{Address: addr, Type: relaytypes.SignatureSecp256k1},
		}
	} else {
		fromAddress, err = types.WalletAddressFromBech(setting.WalletAddress)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting wallet address, ")
		}
		signatureKeys = []relaytypes.SignatureKey{
			{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: relaytypes.SignatureSecp256k1},
		}
	}
	txMsg := stratoschain.BuildSendMsg(fromAddress.Bytes(), toAddress.Bytes(), amount)
	txBytes, err = CreateAndSimulateTx(txMsg, banktypes.TypeMsgSend, txFee, "", signatureKeys)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating simulate tx, ")
	}

	return txBytes, nil
}

func CreateAndSimulateTx(txMsg sdktypes.Msg, msgType string, txFee types.TxFee, memo string, signatureKeys []relaytypes.SignatureKey) ([]byte, error) {
	protoConfig, txBuilder := createTxConfigAndTxBuilder()
	err := setMsgInfoToTxBuilder(txBuilder, txMsg, txFee.Fee, txFee.Gas, memo)
	if err != nil {
		return nil, errors.Wrap(err, "failed setting msg info to the builder, ")
	}

	unsignedMsgs := []*relaytypes.UnsignedMsg{{Msg: txMsg, SignatureKeys: signatureKeys, Type: msgType}}
	txBytes, err := stratoschain.BuildTxBytes(protoConfig, txBuilder, setting.Config.Blockchain.ChainId, unsignedMsgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed building tx bytes, ")
	}

	if txFee.Simulate {
		gasInfo, err := grpc.Simulate(txBytes)
		if err != nil {
			return nil, errors.Wrap(err, "failed simulating, ")
		}
		txBuilder.SetGasLimit(uint64(float64(gasInfo.GasUsed) * setting.Config.Blockchain.GasAdjustment))
		txBytes, err = stratoschain.BuildTxBytes(protoConfig, txBuilder, setting.Config.Blockchain.ChainId, unsignedMsgs)
		if err != nil {
			return nil, errors.Wrap(err, "failed building tx bytes after simulation, ")
		}
	}
	return txBytes, nil
}

func createTxConfigAndTxBuilder() (client.TxConfig, client.TxBuilder) {
	protoConfig := authtx.NewTxConfig(relay.ProtoCdc, []signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_DIRECT})
	txBuilder := protoConfig.NewTxBuilder()
	return protoConfig, txBuilder
}

func setMsgInfoToTxBuilder(txBuilder client.TxBuilder, txMsg sdktypes.Msg, fee types.Coin, gas uint64, memo string) error {
	err := txBuilder.SetMsgs(txMsg)
	if err != nil {
		return err
	}

	txBuilder.SetFeeAmount(sdktypes.NewCoins(
		sdktypes.Coin{
			Denom:  fee.Denom,
			Amount: fee.Amount,
		}),
	)
	txBuilder.SetGasLimit(gas)
	txBuilder.SetMemo(memo)
	return nil
}
