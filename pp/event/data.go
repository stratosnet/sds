package event

import (
	"math/big"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/legacy/legacytx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/relay"
	"github.com/stratosnet/sds/relay/stratoschain"
	relaytypes "github.com/stratosnet/sds/relay/types"
	//authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
)

func reqActivateData(amount, fee, gas, height int64) (*protos.ReqActivatePP, error) {
	// Create and sign transaction to add new resource node
	ownerAddress, err := types.WalletAddressFromBech(setting.WalletAddress)
	if err != nil {
		return nil, err
	}
	p2pAddress, err := types.P2pAddressFromBech(setting.P2PAddress)
	if err != nil {
		return nil, err
	}

	protoConfig, txBuilder := createTxConfigAndTxBuilder()

	//protoConfig := authtx.NewTxConfig(relay.ProtoCdc, []signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_DIRECT})
	//txBuilder := protoConfig.NewTxBuilder()

	txMsg, err := stratoschain.BuildCreateResourceNodeMsg(setting.Config.Token, setting.P2PAddress, registertypes.STORAGE, setting.P2PPublicKey, amount, ownerAddress, p2pAddress)
	if err != nil {
		return nil, err
	}

	txBuilder, err = setMsgInfoToTxBuilder(txBuilder, txMsg, fee, gas)
	//err = txBuilder.SetMsgs(txMsg)
	if err != nil {
		return nil, err
	}

	//txBuilder.SetFeeAmount(sdktypes.NewCoins(sdktypes.NewInt64Coin(setting.Config.Token, fee)))
	////txBuilder.SetFeeGranter(tx.FeeGranter())
	//txBuilder.SetGasLimit(uint64(gas))
	//txBuilder.SetMemo("")

	signatureKeys := []relaytypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: relaytypes.SignatureSecp256k1},
	}
	unsignedMsgs := []*relaytypes.UnsignedMsg{{Msg: txMsg.(legacytx.LegacyMsg), SignatureKeys: signatureKeys}}
	txBytes, err := stratoschain.BuildTxBytesNew(protoConfig, txBuilder, setting.Config.Token, setting.Config.ChainId, "", flags.BroadcastBlock, unsignedMsgs, fee, gas, height)
	if err != nil {
		return nil, err
	}

	req := &protos.ReqActivatePP{
		Tx:            txBytes,
		PpInfo:        setting.GetPPInfo(),
		AlreadyActive: false,
		InitialStake:  big.NewInt(amount).String(),
	}
	return req, nil
}

func setMsgInfoToTxBuilder(txBuilder client.TxBuilder, txMsg sdktypes.Msg, fee int64, gas int64) (client.TxBuilder, error) {
	err := txBuilder.SetMsgs(txMsg)
	if err != nil {
		return nil, err
	}

	txBuilder.SetFeeAmount(sdktypes.NewCoins(sdktypes.NewInt64Coin(setting.Config.Token, fee)))
	//txBuilder.SetFeeGranter(tx.FeeGranter())
	txBuilder.SetGasLimit(uint64(gas))
	txBuilder.SetMemo("")
	return txBuilder, nil
}

func createTxConfigAndTxBuilder() (client.TxConfig, client.TxBuilder) {
	protoConfig := authtx.NewTxConfig(relay.ProtoCdc, []signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_DIRECT})
	txBuilder := protoConfig.NewTxBuilder()
	return protoConfig, txBuilder
}

func reqUpdateStakeData(stakeDelta, fee, gas int64, incrStake bool) (*protos.ReqUpdateStakePP, error) {
	// Create and sign transaction to update stake for existing resource node
	networkAddr := ed25519.PubKeyBytesToAddress(setting.P2PPublicKey)
	ownerAddr, err := crypto.PubKeyToAddress(setting.WalletPublicKey)
	if err != nil {
		return nil, err
	}

	protoConfig, txBuilder := createTxConfigAndTxBuilder()

	txMsg := stratoschain.BuildUpdateResourceNodeStakeMsg(networkAddr, ownerAddr, setting.Config.Token, stakeDelta, incrStake)
	txBuilder, err = setMsgInfoToTxBuilder(txBuilder, txMsg, fee, gas)
	if err != nil {
		return nil, err
	}
	signatureKeys := []relaytypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: relaytypes.SignatureSecp256k1},
	}
	unsignedMsgs := []*relaytypes.UnsignedMsg{{Msg: txMsg.(legacytx.LegacyMsg), SignatureKeys: signatureKeys}}
	txBytes, err := stratoschain.BuildTxBytesNew(protoConfig, txBuilder, setting.Config.Token, setting.Config.ChainId, "", flags.BroadcastSync, unsignedMsgs, fee, gas, int64(0))
	if err != nil {
		return nil, err
	}

	req := &protos.ReqUpdateStakePP{
		Tx:         txBytes,
		P2PAddress: setting.P2PAddress,
	}
	return req, nil
}

func reqDeactivateData(fee, gas int64) (*protos.ReqDeactivatePP, error) {
	// Create and sign transaction to remove a resource node
	nodeAddress := ed25519.PubKeyBytesToAddress(setting.P2PPublicKey)
	ownerAddress, err := crypto.PubKeyToAddress(setting.WalletPublicKey)
	if err != nil {
		return nil, err
	}

	protoConfig, txBuilder := createTxConfigAndTxBuilder()
	txMsg := stratoschain.BuildRemoveResourceNodeMsg(nodeAddress, ownerAddress)
	txBuilder, err = setMsgInfoToTxBuilder(txBuilder, txMsg, fee, gas)
	if err != nil {
		return nil, err
	}
	signatureKeys := []relaytypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: relaytypes.SignatureSecp256k1},
	}
	unsignedMsgs := []*relaytypes.UnsignedMsg{{Msg: txMsg.(legacytx.LegacyMsg), SignatureKeys: signatureKeys}}
	txBytes, err := stratoschain.BuildTxBytesNew(protoConfig, txBuilder, setting.Config.Token, setting.Config.ChainId, "", flags.BroadcastSync, unsignedMsgs, fee, gas, int64(0))
	if err != nil {
		return nil, err
	}

	req := &protos.ReqDeactivatePP{
		Tx:         txBytes,
		P2PAddress: setting.P2PAddress,
	}
	return req, nil
}

func reqPrepayData(amount, fee, gas int64) (*protos.ReqPrepay, error) {
	// Create and sign a prepay transaction
	senderAddress, err := types.WalletAddressFromBech(setting.WalletAddress)
	if err != nil {
		return nil, err
	}
	protoConfig, txBuilder := createTxConfigAndTxBuilder()
	txMsg := stratoschain.BuildPrepayMsg(setting.Config.Token, amount, senderAddress[:])
	txBuilder, err = setMsgInfoToTxBuilder(txBuilder, txMsg, fee, gas)
	if err != nil {
		return nil, err
	}
	signatureKeys := []relaytypes.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: relaytypes.SignatureSecp256k1},
	}
	unsignedMsgs := []*relaytypes.UnsignedMsg{{Msg: txMsg.(legacytx.LegacyMsg), SignatureKeys: signatureKeys}}
	txBytes, err := stratoschain.BuildTxBytesNew(protoConfig, txBuilder, setting.Config.Token, setting.Config.ChainId, "", flags.BroadcastSync, unsignedMsgs, fee, gas, int64(0))
	if err != nil {
		return nil, err
	}

	req := &protos.ReqPrepay{
		Tx:            txBytes,
		P2PAddress:    setting.P2PAddress,
		WalletAddress: setting.WalletAddress,
	}
	return req, nil
}
