package stratoschain

import (
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	utiltypes "github.com/stratosnet/sds/utils/types"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"
)

// Stratos-chain 'pot' module
func BuildVolumeReportMsg(traffic []*core.Traffic, reporterAddress, reporterOwnerAddress []byte, epoch uint64, reportReference string) (sdktypes.Msg, error) {
	aggregatedVolume := make(map[string]uint64)
	for _, trafficReccord := range traffic {
		aggregatedVolume[trafficReccord.P2PAddress] += trafficReccord.Volume
	}

	var nodesVolume []pottypes.SingleNodeVolume
	for p2pAddressString, volume := range aggregatedVolume {
		p2pAddressBytes, err := utiltypes.BechToAddress(p2pAddressString)
		p2pAddress := sdktypes.AccAddress(p2pAddressBytes[:])
		if err != nil {
			return nil, err
		}
		nodesVolume = append(nodesVolume, pottypes.SingleNodeVolume{
			NodeAddress: p2pAddress,
			Volume:      sdktypes.NewIntFromUint64(volume),
		})
	}

	return pottypes.NewMsgVolumeReport(nodesVolume, reporterAddress, sdktypes.NewIntFromUint64(epoch), reportReference, reporterOwnerAddress), nil
}

// Stratos-chain 'register' module
func BuildCreateResourceNodeMsg(networkID, token, moniker, nodeType string, pubKey []byte, amount int64, ownerAddress utiltypes.Address) sdktypes.Msg {
	return registertypes.NewMsgCreateResourceNode(
		networkID,
		ed25519.PubKeyBytesToPubKey(pubKey),
		sdktypes.NewInt64Coin(token, amount),
		ownerAddress[:],
		registertypes.Description{
			Moniker: moniker,
		},
		nodeType,
	)
}

func BuildCreateIndexingNodeMsg(networkAddress, token, moniker string, pubKey []byte, amount int64, ownerAddress utiltypes.Address) sdktypes.Msg {
	return registertypes.NewMsgCreateIndexingNode(
		networkAddress,
		ed25519.PubKeyBytesToPubKey(pubKey),
		sdktypes.NewInt64Coin(token, amount),
		ownerAddress[:],
		registertypes.Description{
			Moniker: moniker,
		},
	)
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
func BuildFileUploadMsg(fileHash, reporterAddress, uploaderAddress []byte) sdktypes.Msg {
	return sdstypes.NewMsgUpload(
		fileHash,
		reporterAddress,
		uploaderAddress,
	)
}

func BuildPrepayMsg(token string, amount int64, senderAddress []byte) sdktypes.Msg {
	return sdstypes.NewMsgPrepay(
		senderAddress,
		sdktypes.NewCoins(sdktypes.NewInt64Coin(token, amount)),
	)
}
