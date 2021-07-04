package stratoschain

import (
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	utiltypes "github.com/stratosnet/sds/utils/types"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"
)

// Stratos-chain 'pot' module
func BuildVolumeReportMsg(traffic []table.Traffic, reporterAddress []byte, epoch uint64, reportReference string) (sdktypes.Msg, error) {
	aggregatedVolume := make(map[string]uint64)
	for _, trafficReccord := range traffic {
		aggregatedVolume[trafficReccord.ProviderWalletAddress] += trafficReccord.Volume
	}

	var nodesVolume []pottypes.SingleNodeVolume
	for address, volume := range aggregatedVolume {
		addressBytes, err := sdktypes.AccAddressFromBech32(address)
		if err != nil {
			return nil, err
		}
		nodesVolume = append(nodesVolume, pottypes.SingleNodeVolume{
			NodeAddress: addressBytes,
			Volume:      sdktypes.NewIntFromUint64(volume),
		})
	}

	return pottypes.NewMsgVolumeReport(nodesVolume, reporterAddress, sdktypes.NewIntFromUint64(epoch), reportReference), nil
}

// Stratos-chain 'register' module
func BuildCreateResourceNodeMsg(networkAddress, token, moniker, nodeType string, pubKey []byte, amount int64, ownerAddress utiltypes.Address) (sdktypes.Msg, error) {
	tmPubkey, err := secp256k1.PubKeyBytesToTendermint(pubKey)
	if err != nil {
		return nil, err
	}
	return registertypes.NewMsgCreateResourceNode(
		networkAddress,
		tmPubkey,
		sdktypes.NewInt64Coin(token, amount),
		ownerAddress[:],
		registertypes.Description{
			Moniker: moniker,
		},
		nodeType,
	), nil
}

func BuildCreateIndexingNodeMsg(networkAddress, token, moniker string, pubKey []byte, amount int64, ownerAddress utiltypes.Address) (sdktypes.Msg, error) {
	tmPubkey, err := secp256k1.PubKeyBytesToTendermint(pubKey)
	if err != nil {
		return nil, err
	}

	return registertypes.NewMsgCreateIndexingNode(
		networkAddress,
		tmPubkey,
		sdktypes.NewInt64Coin(token, amount),
		ownerAddress[:],
		registertypes.Description{
			Moniker: moniker,
		},
	), nil
}

func BuildRemoveResourceNodeMsg(nodeAddress, ownerAddress utiltypes.Address) sdktypes.Msg {
	return registertypes.NewMsgRemoveResourceNode(
		nodeAddress[:],
		ownerAddress[:],
	)
}

func BuildRemoveIndexingNodeMsg(nodeAddress, ownerAddress utiltypes.Address) sdktypes.Msg {
	return registertypes.NewMsgRemoveIndexingNode(
		nodeAddress[:],
		ownerAddress[:],
	)
}

// Stratos-chain 'sds' module
func BuildFileUploadMsg(fileHash, reporterAddress, uploaderAddress []byte) (sdktypes.Msg, error) {
	return sdstypes.NewMsgUpload(
		fileHash,
		reporterAddress,
		uploaderAddress,
	), nil
}

func BuildPrepayMsg(token string, amount int64, senderAddress []byte) (sdktypes.Msg, error) {
	return sdstypes.NewMsgPrepay(
		senderAddress,
		sdktypes.NewCoins(sdktypes.NewInt64Coin(token, amount)),
	), nil
}
