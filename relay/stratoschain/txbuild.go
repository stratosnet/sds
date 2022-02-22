package stratoschain

import (
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	utiltypes "github.com/stratosnet/sds/utils/types"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"
	"github.com/tendermint/tendermint/libs/bech32"
)

type Traffic struct {
	Volume        uint64
	WalletAddress string
}

// Stratos-chain 'pot' module
func BuildVolumeReportMsg(traffic []*Traffic, reporterAddress, reporterOwnerAddress []byte, epoch uint64,
	reportReference string, blsTxData, blsSignature []byte, blsPubKeys [][]byte) (sdktypes.Msg, error) {
	aggregatedVolume := make(map[string]uint64)
	for _, trafficRecord := range traffic {
		aggregatedVolume[trafficRecord.WalletAddress] += trafficRecord.Volume
	}

	var nodesVolume []pottypes.SingleWalletVolume
	for walletAddressString, volume := range aggregatedVolume {
		_, walletAddressBytes, err := bech32.DecodeAndConvert(walletAddressString)
		if err != nil {
			return nil, err
		}
		walletAddress := sdktypes.AccAddress(walletAddressBytes[:])
		nodesVolume = append(nodesVolume, pottypes.SingleWalletVolume{
			WalletAddress: walletAddress,
			Volume:        sdktypes.NewIntFromUint64(volume),
		})
	}

	blsSignatureInfo := pottypes.NewBLSSignatureInfo(blsPubKeys, blsSignature, blsTxData)

	return pottypes.NewMsgVolumeReport(nodesVolume, reporterAddress, sdktypes.NewIntFromUint64(epoch), reportReference, reporterOwnerAddress, blsSignatureInfo), nil
}

// Stratos-chain 'register' module
func BuildCreateResourceNodeMsg(networkID, token, moniker, nodeType string, pubKey []byte, stakeAmount int64, ownerAddress utiltypes.Address) sdktypes.Msg {
	if nodeType == "" {
		nodeType = registertypes.STORAGE.Type()
	}
	return registertypes.NewMsgCreateResourceNode(
		networkID,
		ed25519.PubKeyBytesToPubKey(pubKey),
		sdktypes.NewInt64Coin(token, stakeAmount),
		ownerAddress[:],
		registertypes.Description{
			Moniker: moniker,
		},
		nodeType,
	)
}

func BuildCreateIndexingNodeMsg(networkID, token, moniker string, pubKey []byte, stakeAmount int64, ownerAddress utiltypes.Address) sdktypes.Msg {
	return registertypes.NewMsgCreateIndexingNode(
		networkID,
		ed25519.PubKeyBytesToPubKey(pubKey),
		sdktypes.NewInt64Coin(token, stakeAmount),
		ownerAddress[:],
		registertypes.Description{
			Moniker: moniker,
		},
	)
}

// Stratos-chain 'register' module
func BuildUpdateResourceNodeStakeMsg(networkAddr, ownerAddr utiltypes.Address, token string, stakeDelta int64, incrStake bool) sdktypes.Msg {
	return registertypes.NewMsgUpdateResourceNodeStake(
		networkAddr[:],
		ownerAddr[:],
		sdktypes.NewInt64Coin(token, stakeDelta),
		incrStake,
	)
}

func BuildUpdateIndexingNodeStakeMsg(networkAddr, ownerAddr utiltypes.Address, token string, stakeDelta int64, incrStake bool) sdktypes.Msg {
	return registertypes.NewMsgUpdateIndexingNodeStake(
		networkAddr[:],
		ownerAddr[:],
		sdktypes.NewInt64Coin(token, stakeDelta),
		incrStake,
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

func BuildIndexingNodeRegistrationVoteMsg(candidateNetworkAddress, candidateOwnerAddress, voterNetworkAddress, voterOwnerAddress utiltypes.Address, voteOpinion bool) sdktypes.Msg {
	return registertypes.NewMsgIndexingNodeRegistrationVote(
		candidateNetworkAddress[:],
		candidateOwnerAddress[:],
		registertypes.VoteOpinionFromBool(voteOpinion),
		voterNetworkAddress[:],
		voterOwnerAddress[:],
	)
}

// Stratos-chain 'sds' module
func BuildFileUploadMsg(fileHash string, from, reporterAddress, uploaderAddress []byte) sdktypes.Msg {
	return sdstypes.NewMsgUpload(
		fileHash,
		from,
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
