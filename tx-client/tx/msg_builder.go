package tx

import (
	"math/big"
	"sort"
	"strconv"

	"google.golang.org/protobuf/types/known/anypb"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	basev1beta1 "cosmossdk.io/api/cosmos/base/v1beta1"
	sdkmath "cosmossdk.io/math"

	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
	registerv1 "github.com/stratosnet/stratos-chain/api/stratos/register/v1"
	sdsv1 "github.com/stratosnet/stratos-chain/api/stratos/sds/v1"

	"github.com/stratosnet/sds/framework/crypto/ed25519"
	"github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/types/bech32"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"
)

// Stratos-chain 'pot' module
func BuildVolumeReportMsg(traffic []*txclienttypes.Traffic, reporterAddress, reporterOwnerAddress []byte, epoch uint64,
	reportReference string, blsTxDataHash, blsSignature []byte, blsPubKeys [][]byte) (*potv1.MsgVolumeReport, []byte, error) {

	aggregatedVolume := make(map[string]uint64)
	for _, trafficRecord := range traffic {
		aggregatedVolume[trafficRecord.WalletAddress] += trafficRecord.Volume
	}

	var nodesVolume []*potv1.SingleWalletVolume
	for walletAddressString, volume := range aggregatedVolume {
		_, _, err := bech32.DecodeAndConvert(walletAddressString)
		if err != nil {
			return nil, []byte{}, err
		}
		volumeStr := strconv.FormatUint(volume, 10)
		nodesVolume = append(nodesVolume, &potv1.SingleWalletVolume{
			WalletAddress: walletAddressString,
			Volume:        volumeStr,
		})
	}

	// Map iteration order is not guaranteed. Let's sort the resulting volumes list
	sort.SliceStable(nodesVolume, func(i, j int) bool {
		return nodesVolume[i].WalletAddress < nodesVolume[j].WalletAddress
	})

	blsSignatureInfo := &potv1.BLSSignatureInfo{
		PubKeys:   blsPubKeys,
		Signature: blsSignature,
		TxData:    blsTxDataHash,
	}

	volumeReportMsg := &potv1.MsgVolumeReport{
		WalletVolumes:   nodesVolume,
		Reporter:        types.SdsAddressBytesToBech32(reporterAddress),
		Epoch:           sdkmath.NewIntFromUint64(epoch).String(),
		ReportReference: reportReference,
		ReporterOwner:   types.AccAddressBytesToBech32(reporterOwnerAddress),
		BLSSignature:    blsSignatureInfo,
	}

	msgBytes := txclienttypes.GetVolumeReportMsgBytes(volumeReportMsg)

	return volumeReportMsg, msgBytes, nil
}

func BuildSlashingResourceNodeMsg(spP2pAddress []types.SdsAddress, spWalletAddress []types.AccAddress,
	ppP2pAddress types.SdsAddress, ppWalletAddress types.AccAddress, slashingAmount *big.Int, suspend bool,
) *potv1.MsgSlashingResourceNode {

	var spP2pAddressesBech32 []string
	for _, p2pAddress := range spP2pAddress {
		spP2pAddressesBech32 = append(spP2pAddressesBech32, p2pAddress.String())
	}
	var spWalletAddressesBech32 []string
	for _, walletAddress := range spWalletAddress {
		spWalletAddressesBech32 = append(spWalletAddressesBech32, walletAddress.String())
	}

	return &potv1.MsgSlashingResourceNode{
		Reporters:      spP2pAddressesBech32,
		ReporterOwner:  spWalletAddressesBech32,
		NetworkAddress: ppP2pAddress.String(),
		WalletAddress:  ppWalletAddress.String(),
		Slashing:       slashingAmount.String(),
		Suspend:        suspend,
	}
}

func BuildUpdateEffectiveDepositMsg(spP2pAddress []types.SdsAddress, spWalletAddress []types.AccAddress,
	ppP2pAddress types.SdsAddress, newEffectiveDeposit *big.Int) *registerv1.MsgUpdateEffectiveDeposit {

	var spP2pAddressSdk []string
	for _, p2pAddress := range spP2pAddress {
		spP2pAddressSdk = append(spP2pAddressSdk, p2pAddress.String())
	}
	var spWalletAddressSdk []string
	for _, walletAddress := range spWalletAddress {
		spWalletAddressSdk = append(spWalletAddressSdk, walletAddress.String())
	}

	return &registerv1.MsgUpdateEffectiveDeposit{
		Reporters:       spP2pAddressSdk,
		ReporterOwner:   spWalletAddressSdk,
		NetworkAddress:  ppP2pAddress.String(),
		EffectiveTokens: newEffectiveDeposit.String(),
	}
}

// Stratos-chain 'register' module
func BuildCreateResourceNodeMsg(nodeType txclienttypes.NodeType, pubKey []byte, depositAmount txclienttypes.Coin,
	ownerAddress types.AccAddress, p2pAddress types.SdsAddress) (*registerv1.MsgCreateResourceNode, error) {

	if nodeType == 0 {
		nodeType = txclienttypes.STORAGE
	}

	pk := &ed25519.PubKey{Key: pubKey}
	pkAny, err := anypb.New(pk)
	if err != nil {
		return nil, err
	}

	return &registerv1.MsgCreateResourceNode{
		NetworkAddress: p2pAddress.String(),
		Pubkey:         pkAny,
		Value: &basev1beta1.Coin{
			Denom:  depositAmount.Denom,
			Amount: depositAmount.Amount.String(),
		},
		OwnerAddress: ownerAddress.String(),
		Description: &registerv1.Description{
			Moniker: p2pAddress.String(),
		},
		NodeType: uint32(nodeType),
	}, nil
}

func BuildCreateMetaNodeMsg(pubKey []byte, depositAmount txclienttypes.Coin, ownerAddress types.AccAddress,
	p2pAddress types.SdsAddress) (*registerv1.MsgCreateMetaNode, error) {

	pk := &ed25519.PubKey{Key: pubKey}
	pkAny, err := anypb.New(pk)
	if err != nil {
		return nil, err
	}

	//pk := ed25519.PubKeyBytesToSdkPubKey(pubKey)

	return &registerv1.MsgCreateMetaNode{
		NetworkAddress: p2pAddress.String(),
		Pubkey:         pkAny,
		Value: &basev1beta1.Coin{
			Denom:  depositAmount.Denom,
			Amount: depositAmount.Amount.String(),
		},
		OwnerAddress: ownerAddress.String(),
		Description: &registerv1.Description{
			Moniker: p2pAddress.String(),
		},
	}, nil
}

// Stratos-chain 'register' module
func BuildUpdateResourceNodeDepositMsg(networkAddr types.SdsAddress, ownerAddr types.AccAddress,
	depositDelta txclienttypes.Coin) *registerv1.MsgUpdateResourceNodeDeposit {

	return &registerv1.MsgUpdateResourceNodeDeposit{
		NetworkAddress: networkAddr.String(),
		OwnerAddress:   ownerAddr.String(),
		DepositDelta: &basev1beta1.Coin{
			Denom:  depositDelta.Denom,
			Amount: depositDelta.Amount.String(),
		},
	}
}

func BuildUpdateMetaNodeDepositMsg(networkAddr types.SdsAddress, ownerAddr types.AccAddress,
	depositDelta txclienttypes.Coin) *registerv1.MsgUpdateMetaNodeDeposit {

	return &registerv1.MsgUpdateMetaNodeDeposit{
		NetworkAddress: networkAddr.String(),
		OwnerAddress:   ownerAddr.String(),
		DepositDelta: &basev1beta1.Coin{
			Denom:  depositDelta.Denom,
			Amount: depositDelta.Amount.String(),
		},
	}
}

func BuildRemoveResourceNodeMsg(nodeAddress types.SdsAddress, ownerAddress types.AccAddress,
) *registerv1.MsgRemoveResourceNode {

	return &registerv1.MsgRemoveResourceNode{
		ResourceNodeAddress: nodeAddress.String(),
		OwnerAddress:        ownerAddress.String(),
	}
}

func BuildRemoveMetaNodeMsg(nodeAddress types.SdsAddress, ownerAddress types.AccAddress,
) *registerv1.MsgRemoveMetaNode {

	return &registerv1.MsgRemoveMetaNode{
		MetaNodeAddress: nodeAddress.String(),
		OwnerAddress:    ownerAddress.String(),
	}

}

func BuildMetaNodeRegistrationVoteMsg(candidateNetworkAddress types.SdsAddress, candidateOwnerAddress types.AccAddress,
	voterNetworkAddress types.SdsAddress, voterOwnerAddress types.AccAddress, voteOpinion bool,
) *registerv1.MsgMetaNodeRegistrationVote {

	return &registerv1.MsgMetaNodeRegistrationVote{
		CandidateNetworkAddress: candidateNetworkAddress.String(),
		CandidateOwnerAddress:   candidateOwnerAddress.String(),
		Opinion:                 voteOpinion,
		VoterNetworkAddress:     voterNetworkAddress.String(),
		VoterOwnerAddress:       voterOwnerAddress.String(),
	}
}

func BuildWithdrawMetaNodeRegistrationDepositMsg(networkAddress types.SdsAddress, ownerAddress types.AccAddress,
) *registerv1.MsgWithdrawMetaNodeRegistrationDeposit {

	return &registerv1.MsgWithdrawMetaNodeRegistrationDeposit{
		NetworkAddress: networkAddress.String(),
		OwnerAddress:   ownerAddress.String(),
	}
}

// Stratos-chain 'sds' module
func BuildFileUploadMsg(fileHash string, from types.AccAddress, reporterAddress types.SdsAddress,
	uploaderAddress types.AccAddress) *sdsv1.MsgFileUpload {

	return &sdsv1.MsgFileUpload{
		FileHash: fileHash,
		From:     from.String(),
		Reporter: reporterAddress.String(),
		Uploader: uploaderAddress.String(),
	}
}

func BuildPrepayMsg(senderAddress types.AccAddress, beneficiaryAddress types.AccAddress, amount txclienttypes.Coin,
) *sdsv1.MsgPrepay {

	return &sdsv1.MsgPrepay{
		Sender:      senderAddress.String(),
		Beneficiary: beneficiaryAddress.String(),
		Amount: []*basev1beta1.Coin{
			{
				Denom:  amount.Denom,
				Amount: amount.Amount.String(),
			},
		},
	}

}

func BuildWithdrawMsg(amount txclienttypes.Coin, senderAddress types.AccAddress, targetAddress types.AccAddress,
) *potv1.MsgWithdraw {

	return &potv1.MsgWithdraw{
		Amount: []*basev1beta1.Coin{
			{
				Denom:  amount.Denom,
				Amount: amount.Amount.String(),
			},
		},
		WalletAddress: senderAddress.String(),
		TargetAddress: targetAddress.String(),
	}
}

func BuildSendMsg(senderAddress types.AccAddress, toAddress types.AccAddress, amount txclienttypes.Coin) *bankv1beta1.MsgSend {

	return &bankv1beta1.MsgSend{
		FromAddress: senderAddress.String(),
		ToAddress:   toAddress.String(),
		Amount: []*basev1beta1.Coin{
			{
				Denom:  amount.Denom,
				Amount: amount.Amount.String(),
			},
		},
	}
}
