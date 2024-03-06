package tx

import (
	"math/big"
	"sort"
	"strconv"

	"github.com/cosmos/cosmos-proto/anyutil"
	"github.com/stratosnet/sds/tx-client/utils"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	basev1beta1 "cosmossdk.io/api/cosmos/base/v1beta1"
	sdked25519 "cosmossdk.io/api/cosmos/crypto/ed25519"
	sdkmath "cosmossdk.io/math"

	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
	registerv1 "github.com/stratosnet/stratos-chain/api/stratos/register/v1"
	sdsv1 "github.com/stratosnet/stratos-chain/api/stratos/sds/v1"

	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
	fwtypes "github.com/stratosnet/sds/framework/types"
	msgtypes "github.com/stratosnet/sds/sds-msg/types"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"
)

// Stratos-chain 'pot' module

func BuildVolumeReportMsg(traffic []*txclienttypes.Traffic, reporterAddress fwtypes.P2PAddress, reporterOwnerAddress fwtypes.WalletAddress, epoch uint64,
	reportReference string, blsTxDataHash, blsSignature []byte, blsPubKeys [][]byte) (*potv1.MsgVolumeReport, []byte, error) {

	aggregatedVolume := make(map[string]uint64)
	for _, trafficRecord := range traffic {
		aggregatedVolume[trafficRecord.WalletAddress] += trafficRecord.Volume
	}

	var nodesVolume []*potv1.SingleWalletVolume
	for walletAddressString, volume := range aggregatedVolume {
		_, err := fwtypes.WalletAddressFromBech32(walletAddressString)
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
		Reporter:        fwtypes.P2PAddressBytesToBech32(reporterAddress),
		Epoch:           sdkmath.NewIntFromUint64(epoch).String(),
		ReportReference: reportReference,
		ReporterOwner:   fwtypes.WalletAddressBytesToBech32(reporterOwnerAddress),
		BLSSignature:    blsSignatureInfo,
	}

	msgBytes, err := utils.GetVolumeReportMsgBytes(volumeReportMsg)
	if err != nil {
		return nil, nil, err
	}

	return volumeReportMsg, msgBytes, nil
}

func BuildSlashingResourceNodeMsg(spP2pAddress []fwtypes.P2PAddress, spWalletAddress []fwtypes.WalletAddress,
	ppP2pAddress fwtypes.P2PAddress, ppWalletAddress fwtypes.WalletAddress, slashingAmount *big.Int, suspend bool,
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

func BuildUpdateEffectiveDepositMsg(spP2pAddress []fwtypes.P2PAddress, spWalletAddress []fwtypes.WalletAddress,
	ppP2pAddress fwtypes.P2PAddress, newEffectiveDeposit *big.Int) *registerv1.MsgUpdateEffectiveDeposit {

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

func BuildCreateResourceNodeMsg(nodeType msgtypes.NodeType, p2pPubKey fwcryptotypes.PubKey, depositAmount txclienttypes.Coin,
	ownerAddress fwtypes.WalletAddress) (*registerv1.MsgCreateResourceNode, error) {

	if nodeType == 0 {
		nodeType = msgtypes.STORAGE
	}

	p2pAddress := fwtypes.P2PAddress(p2pPubKey.Address())

	pk := &sdked25519.PubKey{Key: p2pPubKey.Bytes()}
	pkAny, err := anyutil.New(pk)
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

func BuildCreateMetaNodeMsg(p2pPubKey fwcryptotypes.PubKey, depositAmount txclienttypes.Coin,
	ownerAddress fwtypes.WalletAddress, beneficiaryAddress fwtypes.WalletAddress) (*registerv1.MsgCreateMetaNode, error) {

	p2pAddress := fwtypes.P2PAddress(p2pPubKey.Address())

	pk := &sdked25519.PubKey{Key: p2pPubKey.Bytes()}
	pkAny, err := anyutil.New(pk)
	if err != nil {
		return nil, err
	}

	return &registerv1.MsgCreateMetaNode{
		NetworkAddress: p2pAddress.String(),
		Pubkey:         pkAny,
		Value: &basev1beta1.Coin{
			Denom:  depositAmount.Denom,
			Amount: depositAmount.Amount.String(),
		},
		OwnerAddress:       ownerAddress.String(),
		BeneficiaryAddress: beneficiaryAddress.String(),
		Description: &registerv1.Description{
			Moniker: p2pAddress.String(),
		},
	}, nil
}

// Stratos-chain 'register' module

func BuildUpdateResourceNodeDepositMsg(networkAddr fwtypes.P2PAddress, ownerAddr fwtypes.WalletAddress,
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

func BuildUpdateMetaNodeDepositMsg(networkAddr fwtypes.P2PAddress, ownerAddr fwtypes.WalletAddress,
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

func BuildRemoveResourceNodeMsg(nodeAddress fwtypes.P2PAddress, ownerAddress fwtypes.WalletAddress,
) *registerv1.MsgRemoveResourceNode {

	return &registerv1.MsgRemoveResourceNode{
		ResourceNodeAddress: nodeAddress.String(),
		OwnerAddress:        ownerAddress.String(),
	}
}

func BuildRemoveMetaNodeMsg(nodeAddress fwtypes.P2PAddress, ownerAddress fwtypes.WalletAddress,
) *registerv1.MsgRemoveMetaNode {

	return &registerv1.MsgRemoveMetaNode{
		MetaNodeAddress: nodeAddress.String(),
		OwnerAddress:    ownerAddress.String(),
	}

}

func BuildMetaNodeRegistrationVoteMsg(candidateNetworkAddress fwtypes.P2PAddress, candidateOwnerAddress fwtypes.WalletAddress,
	voterNetworkAddress fwtypes.P2PAddress, voterOwnerAddress fwtypes.WalletAddress, voteOpinion bool,
) *registerv1.MsgMetaNodeRegistrationVote {

	return &registerv1.MsgMetaNodeRegistrationVote{
		CandidateNetworkAddress: candidateNetworkAddress.String(),
		CandidateOwnerAddress:   candidateOwnerAddress.String(),
		Opinion:                 voteOpinion,
		VoterNetworkAddress:     voterNetworkAddress.String(),
		VoterOwnerAddress:       voterOwnerAddress.String(),
	}
}

//func BuildWithdrawMetaNodeRegistrationDepositMsg(networkAddress fwtypes.P2PAddress, ownerAddress fwtypes.WalletAddress,
//) *registerv1.MsgWithdrawMetaNodeRegistrationDeposit {
//
//	return &registerv1.MsgWithdrawMetaNodeRegistrationDeposit{
//		NetworkAddress: networkAddress.String(),
//		OwnerAddress:   ownerAddress.String(),
//	}
//}

func BuildKickMetaNodeVoteMsg(targetNetworkAddress fwtypes.P2PAddress, opinion bool, voterNetworkAddress fwtypes.P2PAddress,
	voterOwnerAddress fwtypes.WalletAddress) *registerv1.MsgKickMetaNodeVote {

	return &registerv1.MsgKickMetaNodeVote{
		TargetNetworkAddress: targetNetworkAddress.String(),
		Opinion:              opinion,
		VoterNetworkAddress:  voterNetworkAddress.String(),
		VoterOwnerAddress:    voterOwnerAddress.String(),
	}
}

// Stratos-chain 'sds' module

func BuildFileUploadMsg(fileHash string, from fwtypes.WalletAddress, reporterAddress fwtypes.P2PAddress,
	uploaderAddress fwtypes.WalletAddress) *sdsv1.MsgFileUpload {

	return &sdsv1.MsgFileUpload{
		FileHash: fileHash,
		From:     from.String(),
		Reporter: reporterAddress.String(),
		Uploader: uploaderAddress.String(),
	}
}

func BuildPrepayMsg(senderAddress fwtypes.WalletAddress, beneficiaryAddress fwtypes.WalletAddress, amount txclienttypes.Coin,
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

func BuildWithdrawMsg(amount txclienttypes.Coin, senderAddress fwtypes.WalletAddress, targetAddress fwtypes.WalletAddress,
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

func BuildSendMsg(senderAddress fwtypes.WalletAddress, toAddress fwtypes.WalletAddress, amount txclienttypes.Coin) *bankv1beta1.MsgSend {

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
