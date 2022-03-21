package event

import (
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
)

func reqActivateData(amount, fee, gas int64) (*protos.ReqActivatePP, error) {
	// Create and sign transaction to add new resource node
	ownerAddress, err := types.WalletAddressFromBech(setting.WalletAddress)
	if err != nil {
		return nil, err
	}
	p2pAddress, err := types.P2pAddressFromBech(setting.P2PAddress)
	if err != nil {
		return nil, err
	}

	txMsg := stratoschain.BuildCreateResourceNodeMsg(setting.Config.Token, setting.P2PAddress, registertypes.STORAGE, setting.P2PPublicKey, amount, ownerAddress, p2pAddress)
	signatureKeys := []stratoschain.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: stratoschain.SignatureSecp256k1},
	}
	unsignedMsgs := []*stratoschain.UnsignedMsg{{Msg: txMsg, SignatureKeys: signatureKeys}}
	txBytes, err := stratoschain.BuildTxBytes(setting.Config.Token, setting.Config.ChainId, "", flags.BroadcastBlock, unsignedMsgs, fee, gas)
	if err != nil {
		return nil, err
	}

	req := &protos.ReqActivatePP{
		Tx:            txBytes,
		PpInfo:        setting.GetPPInfo(),
		AlreadyActive: false,
	}
	return req, nil
}

func reqUpdateStakeData(stakeDelta, fee, gas int64, incrStake bool) (*protos.ReqUpdateStakePP, error) {
	// Create and sign transaction to update stake for existing resource node
	networkAddr := ed25519.PubKeyBytesToAddress(setting.P2PPublicKey)
	ownerAddr, err := crypto.PubKeyToAddress(setting.WalletPublicKey)
	if err != nil {
		return nil, err
	}

	txMsg := stratoschain.BuildUpdateResourceNodeStakeMsg(networkAddr, ownerAddr, setting.Config.Token, stakeDelta, incrStake)
	signatureKeys := []stratoschain.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: stratoschain.SignatureSecp256k1},
	}
	unsignedMsgs := []*stratoschain.UnsignedMsg{{Msg: txMsg, SignatureKeys: signatureKeys}}
	txBytes, err := stratoschain.BuildTxBytes(setting.Config.Token, setting.Config.ChainId, "", flags.BroadcastSync, unsignedMsgs, fee, gas)
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

	txMsg := stratoschain.BuildRemoveResourceNodeMsg(nodeAddress, ownerAddress)
	signatureKeys := []stratoschain.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: stratoschain.SignatureSecp256k1},
	}
	unsignedMsgs := []*stratoschain.UnsignedMsg{{Msg: txMsg, SignatureKeys: signatureKeys}}
	txBytes, err := stratoschain.BuildTxBytes(setting.Config.Token, setting.Config.ChainId, "", flags.BroadcastSync, unsignedMsgs, fee, gas)
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

	txMsg := stratoschain.BuildPrepayMsg(setting.Config.Token, amount, senderAddress[:])
	signatureKeys := []stratoschain.SignatureKey{
		{Address: setting.WalletAddress, PrivateKey: setting.WalletPrivateKey, Type: stratoschain.SignatureSecp256k1},
	}
	unsignedMsgs := []*stratoschain.UnsignedMsg{{Msg: txMsg, SignatureKeys: signatureKeys}}
	txBytes, err := stratoschain.BuildTxBytes(setting.Config.Token, setting.Config.ChainId, "", flags.BroadcastSync, unsignedMsgs, fee, gas)
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
