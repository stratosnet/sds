package stratoschain

import (
	"math/big"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"

	"github.com/stratosnet/stratos-chain/types"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"

	"github.com/stratosnet/sds/utils/crypto/ed25519"
	utiltypes "github.com/stratosnet/sds/utils/types"
)

type Traffic struct {
	Volume        uint64
	WalletAddress string
}

// Stratos-chain 'pot' module
func BuildVolumeReportMsg(traffic []*Traffic, reporterAddress, reporterOwnerAddress []byte, epoch uint64,
	reportReference string, blsTxDataHash, blsSignature []byte, blsPubKeys [][]byte) (sdktypes.Msg, []byte, error) {
	aggregatedVolume := make(map[string]uint64)
	for _, trafficRecord := range traffic {
		aggregatedVolume[trafficRecord.WalletAddress] += trafficRecord.Volume
	}

	var nodesVolume []*pottypes.SingleWalletVolume
	for walletAddressString, volume := range aggregatedVolume {
		_, _, err := bech32.DecodeAndConvert(walletAddressString)
		if err != nil {
			return nil, []byte{}, err
		}
		volume := sdktypes.NewIntFromUint64(volume)
		nodesVolume = append(nodesVolume, &pottypes.SingleWalletVolume{
			WalletAddress: walletAddressString,
			Volume:        &volume,
		})
	}

	blsSignatureInfo := pottypes.NewBLSSignatureInfo(blsPubKeys, blsSignature, blsTxDataHash)

	volumeReportMsg := pottypes.NewMsgVolumeReport(nodesVolume, reporterAddress, sdktypes.NewIntFromUint64(epoch), reportReference, reporterOwnerAddress, blsSignatureInfo)
	signBytes := volumeReportMsg.GetSignBytes()
	return volumeReportMsg, signBytes, nil
}

func BuildSlashingResourceNodeMsg(spP2pAddress, spWalletAddress []utiltypes.Address, ppP2pAddress, ppWalletAddress utiltypes.Address, slashingAmount *big.Int, suspend bool) sdktypes.Msg {
	var spP2pAddressSdk []types.SdsAddress
	for _, p2pAddress := range spP2pAddress {
		spP2pAddressSdk = append(spP2pAddressSdk, p2pAddress[:])
	}
	var spWalletAddressSdk []sdktypes.AccAddress
	for _, walletAddress := range spWalletAddress {
		spWalletAddressSdk = append(spWalletAddressSdk, walletAddress[:])
	}

	return pottypes.NewMsgSlashingResourceNode(
		spP2pAddressSdk,
		spWalletAddressSdk,
		ppP2pAddress[:],
		ppWalletAddress[:],
		sdktypes.NewIntFromBigInt(slashingAmount),
		suspend,
	)
}

func BuildUpdateEffectiveStakeMsg(spP2pAddress, spWalletAddress []utiltypes.Address, ppP2pAddress utiltypes.Address, newEffectiveStake *big.Int) sdktypes.Msg {
	var spP2pAddressSdk []types.SdsAddress
	for _, p2pAddress := range spP2pAddress {
		spP2pAddressSdk = append(spP2pAddressSdk, p2pAddress[:])
	}
	var spWalletAddressSdk []sdktypes.AccAddress
	for _, walletAddress := range spWalletAddress {
		spWalletAddressSdk = append(spWalletAddressSdk, walletAddress[:])
	}

	return registertypes.NewMsgUpdateEffectiveStake(
		spP2pAddressSdk,
		spWalletAddressSdk,
		ppP2pAddress[:],
		sdktypes.NewIntFromBigInt(newEffectiveStake),
	)

}

// Stratos-chain 'register' module
func BuildCreateResourceNodeMsg(moniker string, nodeType registertypes.NodeType, pubKey []byte, stakeAmount utiltypes.Coin, ownerAddress, p2pAddress utiltypes.Address) (sdktypes.Msg, error) {
	if nodeType == 0 {
		nodeType = registertypes.STORAGE
	}

	pk := ed25519.PubKeyBytesToSdkPubKey(pubKey)

	return registertypes.NewMsgCreateResourceNode(
		p2pAddress[:],
		pk,
		sdktypes.Coin{
			Denom:  stakeAmount.Denom,
			Amount: stakeAmount.Amount,
		},
		ownerAddress[:],
		&registertypes.Description{
			Moniker: moniker,
		},
		uint32(nodeType),
	)
}

func BuildCreateMetaNodeMsg(moniker string, pubKey []byte, stakeAmount utiltypes.Coin, ownerAddress, p2pAddress utiltypes.Address) (sdktypes.Msg, error) {
	pk := ed25519.PubKeyBytesToSdkPubKey(pubKey)
	return registertypes.NewMsgCreateMetaNode(
		p2pAddress[:],
		pk,
		sdktypes.Coin{
			Denom:  stakeAmount.Denom,
			Amount: stakeAmount.Amount,
		},
		ownerAddress[:],
		&registertypes.Description{
			Moniker: moniker,
		},
	)
}

// Stratos-chain 'register' module
func BuildUpdateResourceNodeStakeMsg(networkAddr, ownerAddr utiltypes.Address, stakeDelta utiltypes.Coin, incrStake bool) sdktypes.Msg {
	coin := sdktypes.Coin{
		Denom:  stakeDelta.Denom,
		Amount: stakeDelta.Amount,
	}
	return registertypes.NewMsgUpdateResourceNodeStake(
		networkAddr[:],
		ownerAddr[:],
		coin,
		incrStake,
	)
}

func BuildUpdateMetaNodeStakeMsg(networkAddr, ownerAddr utiltypes.Address, stakeDelta utiltypes.Coin, incrStake bool) sdktypes.Msg {
	coin := sdktypes.Coin{
		Denom:  stakeDelta.Denom,
		Amount: stakeDelta.Amount,
	}
	return registertypes.NewMsgUpdateMetaNodeStake(
		networkAddr[:],
		ownerAddr[:],
		coin,
		incrStake,
	)
}

func BuildRemoveResourceNodeMsg(nodeAddress, ownerAddress utiltypes.Address) sdktypes.Msg {
	return registertypes.NewMsgRemoveResourceNode(
		nodeAddress[:],
		ownerAddress[:],
	)
}

func BuildRemoveMetaNodeMsg(nodeAddress, ownerAddress utiltypes.Address) sdktypes.Msg {
	return registertypes.NewMsgRemoveMetaNode(
		nodeAddress[:],
		ownerAddress[:],
	)
}

func BuildMetaNodeRegistrationVoteMsg(candidateNetworkAddress, candidateOwnerAddress, voterNetworkAddress, voterOwnerAddress utiltypes.Address, voteOpinion bool) sdktypes.Msg {
	return registertypes.NewMsgMetaNodeRegistrationVote(
		candidateNetworkAddress[:],
		candidateOwnerAddress[:],
		voteOpinion,
		voterNetworkAddress[:],
		voterOwnerAddress[:],
	)
}

// Stratos-chain 'sds' module
func BuildFileUploadMsg(fileHash string, from, reporterAddress, uploaderAddress []byte) sdktypes.Msg {
	walletPrefix := types.GetConfig().GetBech32AccountAddrPrefix()
	p2pAddrPrefix := types.GetConfig().GetBech32SdsNodeP2PAddrPrefix()
	return sdstypes.NewMsgUpload(
		fileHash,
		sdktypes.MustBech32ifyAddressBytes(walletPrefix, from),
		sdktypes.MustBech32ifyAddressBytes(p2pAddrPrefix, reporterAddress),
		sdktypes.MustBech32ifyAddressBytes(walletPrefix, uploaderAddress),
	)
}

func BuildPrepayMsg(senderAddress []byte, beneficiaryAddress []byte, amount utiltypes.Coin) sdktypes.Msg {
	walletPrefix := types.GetConfig().GetBech32AccountAddrPrefix()
	return sdstypes.NewMsgPrepay(
		sdktypes.MustBech32ifyAddressBytes(walletPrefix, senderAddress),
		sdktypes.MustBech32ifyAddressBytes(walletPrefix, beneficiaryAddress),
		sdktypes.NewCoins(sdktypes.Coin{
			Denom:  amount.Denom,
			Amount: amount.Amount,
		}),
	)
}
