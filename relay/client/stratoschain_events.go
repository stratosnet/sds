package client

import (
	"encoding/hex"
	"fmt"
	sdkTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/protobuf/proto"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/utils/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

func (m *MultiClient) SubscribeToStratosChainEvents() error {
	err := m.SubscribeToStratosChain("message.action='create_resource_node'", m.CreateResourceNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='unbonding_resource_node'", m.UnbondingResourceNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='remove_resource_node'", m.RemoveResourceNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='create_indexing_node'", m.CreateIndexingNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='unbonding_indexing_node'", m.UnbondingIndexingNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='remove_indexing_node'", m.RemoveIndexingNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='complete_unbonding_node'", m.CompleteUnbondingNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='indexing_node_reg_vote'", m.IndexingNodeVoteMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='SdsPrepayTx'", m.PrepayMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='FileUploadTx'", m.FileUploadMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='volume_report'", m.VolumeReportHandler())
	return err
}

func (m *MultiClient) CreateResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		conn := m.GetSdsClientConn()

		networkAddressList := result.Events["create_resource_node.network_address"]
		if len(networkAddressList) < 1 {
			fmt.Println("No network address was specified in the create_resource_node message from stratos-chain")
			return
		}

		p2pAddress, err := types.BechToAddress(networkAddressList[0])
		if err != nil {
			fmt.Println("Error when trying to convert P2P address to bytes: " + err.Error())
			return
		}
		p2pAddressString, err := p2pAddress.ToBech(setting.Config.BlockchainInfo.P2PAddressPrefix)
		if err != nil {
			fmt.Println("Error when trying to convert P2P address to bech32: " + err.Error())
			return
		}

		nodePubkeyList := result.Events["create_resource_node.pub_key"]
		if len(nodePubkeyList) < 1 {
			fmt.Println("No node pubkey was specified in the create_resource_node message from stratos-chain")
			return
		}
		p2pPubkeyRaw, err := hex.DecodeString(nodePubkeyList[0])
		if err != nil {
			fmt.Println("Error when trying to decode P2P pubkey hex: " + err.Error())
			return
		}
		p2pPubkey := ed25519.PubKeyEd25519{}
		err = stratoschain.Cdc.UnmarshalBinaryBare(p2pPubkeyRaw, &p2pPubkey)
		if err != nil {
			fmt.Println("Error when trying to read P2P pubkey ed25519 binary: " + err.Error())
			return
		}

		ozoneLimitChangeStr := result.Events["create_resource_node.ozone_limit_changes"]

		activatedMsg := &protos.ReqActivatedPP{
			P2PAddress:        p2pAddressString,
			P2PPubkey:         hex.EncodeToString(p2pPubkey[:]),
			OzoneLimitChanges: ozoneLimitChangeStr[0],
		}
		activatedMsgBytes, err := proto.Marshal(activatedMsg)
		if err != nil {
			fmt.Println("Error when trying to marshal ReqActivatedPP proto: " + err.Error())
			return
		}
		msgToSend := &msg.RelayMsgBuf{
			MSGData: activatedMsgBytes,
			MSGHead: header.MakeMessageHeader(1, 1, uint32(len(activatedMsgBytes)), header.ReqActivatedPP),
		}

		err = conn.Write(msgToSend)
		if err != nil {
			fmt.Println("Error when sending message to SDS: " + err.Error())
			return
		}
	}
}

func (m *MultiClient) UnbondingResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		conn := m.GetSdsClientConn()

		nodeAddressList := result.Events["unbonding_resource_node.resource_node"]
		if len(nodeAddressList) < 1 {
			fmt.Println("No node address was specified in the remove_resource_node message from stratos-chain")
			return
		}

		p2pAddress, err := types.BechToAddress(nodeAddressList[0])
		if err != nil {
			fmt.Println("Error when trying to convert P2P address to bytes: " + err.Error())
			return
		}
		p2pAddressString, err := p2pAddress.ToBech(setting.Config.BlockchainInfo.P2PAddressPrefix)
		if err != nil {
			fmt.Println("Error when trying to convert P2P address to bech32: " + err.Error())
			return
		}

		// get ozone limit change
		ozoneLimitChange := result.Events["unbonding_resource_node.ozone_limit_changes"]
		ozoneLimitChangeStr := ozoneLimitChange[0]
		// get mature time
		ubdMatureTime := result.Events["unbonding_resource_node.unbonding_mature_time"]
		ubdMatureTimeStr := ubdMatureTime[0]
		ubdMsg := &protos.ReqUnbondingPP{
			P2PAddress:          p2pAddressString,
			OzoneLimitChanges:   ozoneLimitChangeStr,
			UnbondingMatureTime: ubdMatureTimeStr,
		}
		ubdMsgBytes, err := proto.Marshal(ubdMsg)
		if err != nil {
			fmt.Println("Error when trying to marshal ReqDeactivatedPP proto: " + err.Error())
			return
		}
		msgToSend := &msg.RelayMsgBuf{
			MSGData: ubdMsgBytes,
			MSGHead: header.MakeMessageHeader(1, 1, uint32(len(ubdMsgBytes)), header.ReqUnbondingPP),
		}

		err = conn.Write(msgToSend)
		if err != nil {
			fmt.Println("Error when sending message to SDS: " + err.Error())
			return
		}
	}
}

func (m *MultiClient) RemoveResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		conn := m.GetSdsClientConn()

		nodeAddressList := result.Events["remove_resource_node.resource_node"]
		if len(nodeAddressList) < 1 {
			fmt.Println("No node address was specified in the remove_resource_node message from stratos-chain")
			return
		}

		p2pAddress, err := types.BechToAddress(nodeAddressList[0])
		if err != nil {
			fmt.Println("Error when trying to convert P2P address to bytes: " + err.Error())
			return
		}
		p2pAddressString, err := p2pAddress.ToBech(setting.Config.BlockchainInfo.P2PAddressPrefix)
		if err != nil {
			fmt.Println("Error when trying to convert P2P address to bech32: " + err.Error())
			return
		}

		deactivatedMsg := &protos.ReqDeactivatedPP{P2PAddress: p2pAddressString}
		deactivatedMsgBytes, err := proto.Marshal(deactivatedMsg)
		if err != nil {
			fmt.Println("Error when trying to marshal ReqDeactivatedPP proto: " + err.Error())
			return
		}
		msgToSend := &msg.RelayMsgBuf{
			MSGData: deactivatedMsgBytes,
			MSGHead: header.MakeMessageHeader(1, 1, uint32(len(deactivatedMsgBytes)), header.ReqDeactivatedPP),
		}

		err = conn.Write(msgToSend)
		if err != nil {
			fmt.Println("Error when sending message to SDS: " + err.Error())
			return
		}
	}
}

func (m *MultiClient) CreateIndexingNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	}
}

func (m *MultiClient) UnbondingIndexingNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	}
}
func (m *MultiClient) RemoveIndexingNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	}
}
func (m *MultiClient) CompleteUnbondingNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	}
}

func (m *MultiClient) IndexingNodeVoteMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		conn := m.GetSdsClientConn()

		candidateNetworkAddressList := result.Events["indexing_node_reg_vote.candidate_network_address"]
		if len(candidateNetworkAddressList) < 1 {
			fmt.Println("No candidate network address was specified in the indexing_node_reg_vote message from stratos-chain")
			return
		}
		p2pAddress, err := types.BechToAddress(candidateNetworkAddressList[0])
		if err != nil {
			fmt.Println("Error when trying to convert P2P address to bytes: " + err.Error())
			return
		}
		p2pAddressString, err := p2pAddress.ToBech(setting.Config.BlockchainInfo.P2PAddressPrefix)
		if err != nil {
			fmt.Println("Error when trying to convert P2P address to bech32: " + err.Error())
			return
		}

		candidateStatusList := result.Events["indexing_node_reg_vote.candidate_status"]
		if len(candidateStatusList) < 1 {
			fmt.Println("No candidate status was specified in the indexing_node_reg_vote message from stratos-chain")
			return
		}
		if candidateStatusList[0] != sdkTypes.BondStatusBonded {
			// The candidate needs more votes before being considered active
			return
		}

		activatedMsg := &protos.ReqActivatedSP{
			P2PAddress: p2pAddressString,
		}
		activatedMsgBytes, err := proto.Marshal(activatedMsg)
		if err != nil {
			fmt.Println("Error when trying to marshal ReqActivatedSP proto: " + err.Error())
			return
		}
		msgToSend := &msg.RelayMsgBuf{
			MSGData: activatedMsgBytes,
			MSGHead: header.MakeMessageHeader(1, 1, uint32(len(activatedMsgBytes)), header.ReqActivatedSP),
		}

		err = conn.Write(msgToSend)
		if err != nil {
			fmt.Println("Error when sending message to SDS: " + err.Error())
			return
		}
	}
}

func (m *MultiClient) PrepayMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		fmt.Printf("%+v\n", result)
		conn := m.GetSdsClientConn()

		reporterList := result.Events["Prepay.reporter"]
		if len(reporterList) < 1 {
			fmt.Println("No reporter address was specified in the prepay message from stratos-chain")
			return
		}

		purchasedUozList := result.Events["Prepay.purchased"]
		if len(purchasedUozList) < 1 {
			fmt.Println("No purchased ozone amount was specified in the prepay message from stratos-chain")
			return
		}

		prepaidMsg := &protos.ReqPrepaid{
			WalletAddress: reporterList[0],
			PurchasedUoz:  purchasedUozList[0],
		}
		prepaidMsgBytes, err := proto.Marshal(prepaidMsg)
		if err != nil {
			fmt.Println("Error when trying to marshal ReqPrepaid proto: " + err.Error())
			return
		}
		msgToSend := &msg.RelayMsgBuf{
			MSGData: prepaidMsgBytes,
			MSGHead: header.MakeMessageHeader(1, 1, uint32(len(prepaidMsgBytes)), header.ReqPrepaid),
		}

		err = conn.Write(msgToSend)
		if err != nil {
			fmt.Println("Error when sending message to SDS: " + err.Error())
			return
		}
	}
}

func (m *MultiClient) FileUploadMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		conn := m.GetSdsClientConn()

		reporterAddressList := result.Events["FileUploadTx.reporter"]
		if len(reporterAddressList) < 1 {
			fmt.Println("No reporter address was specified in the FileUploadTx message from stratos-chain")
			return
		}

		uploaderAddressList := result.Events["FileUploadTx.uploader"]
		if len(uploaderAddressList) < 1 {
			fmt.Println("No uploader address was specified in the FileUploadTx message from stratos-chain")
			return
		}

		fileHashList := result.Events["FileUploadTx.file_hash"]
		if len(fileHashList) < 1 {
			fmt.Println("No file hash was specified in the FileUploadTx message from stratos-chain")
			return
		}

		uploadedMsg := &protos.Uploaded{
			ReporterAddress: reporterAddressList[0],
			UploaderAddress: uploaderAddressList[0],
			FileHash:        fileHashList[0],
		}
		uploadedMsgBytes, err := proto.Marshal(uploadedMsg)
		if err != nil {
			fmt.Println("Error when trying to marshal Uploaded proto: " + err.Error())
			return
		}
		msgToSend := &msg.RelayMsgBuf{
			MSGData: uploadedMsgBytes,
			MSGHead: header.MakeMessageHeader(1, 1, uint32(len(uploadedMsgBytes)), header.Uploaded),
		}

		err = conn.Write(msgToSend)
		if err != nil {
			fmt.Println("Error when sending message to SDS: " + err.Error())
			return
		}
	}
}

func (m *MultiClient) VolumeReportHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	}
}
