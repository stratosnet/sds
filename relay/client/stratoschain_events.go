package client

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

func (m *MultiClient) SubscribeToStratosChainEvents() error {
	err := m.SubscribeToStratosChain("message.action='create_resource_node'", m.CreateResourceNodeMsgHandler())
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
	err = m.SubscribeToStratosChain("message.action='remove_indexing_node'", m.RemoveIndexingNodeMsgHandler())
	if err != nil {
		return err
	}
	// TODO: query will probably change when this is implemented in stratos-chain
	err = m.SubscribeToStratosChain("message.action='sp_registration_approved'", m.SPRegistrationApprovedMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='SdsPrepayTx'", m.PrepayMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='FileUploadTx'", m.PrepayMsgHandler())
	return err
}

func (m *MultiClient) CreateResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		//fmt.Printf("%+v\n", result)
		conn := m.GetSdsClientConn()

		nodeAddressList := result.Events["create_resource_node.node_address"]
		if len(nodeAddressList) < 1 {
			fmt.Println("No node address was specified in the create_resource_node message from stratos-chain")
			return
		}
		activatedMsg := &protos.ReqActivated{WalletAddress: nodeAddressList[0]}
		activatedMsgBytes, err := proto.Marshal(activatedMsg)
		if err != nil {
			fmt.Println("Error when trying to marshal activatedMsg proto: " + err.Error())
			return
		}
		msgToSend := &msg.RelayMsgBuf{
			MSGData: activatedMsgBytes,
			MSGHead: header.MakeMessageHeader(1, 1, uint32(len(activatedMsgBytes)), header.ReqActivated),
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
		//fmt.Printf("%+v\n", result)
		conn := m.GetSdsClientConn()

		nodeAddressList := result.Events["remove_resource_node.resource_node"]
		if len(nodeAddressList) < 1 {
			fmt.Println("No node address was specified in the remove_resource_node message from stratos-chain")
			return
		}
		deactivatedMsg := &protos.ReqDeactivated{WalletAddress: nodeAddressList[0]}
		deactivatedMsgBytes, err := proto.Marshal(deactivatedMsg)
		if err != nil {
			fmt.Println("Error when trying to marshal deactivatedMsg proto: " + err.Error())
			return
		}
		msgToSend := &msg.RelayMsgBuf{
			MSGData: deactivatedMsgBytes,
			MSGHead: header.MakeMessageHeader(1, 1, uint32(len(deactivatedMsgBytes)), header.ReqDeactivated),
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

func (m *MultiClient) RemoveIndexingNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	}
}

func (m *MultiClient) SPRegistrationApprovedMsgHandler() func(event coretypes.ResultEvent) {

	return func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	}
}

func (m *MultiClient) PrepayMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
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
			fmt.Println("Error when trying to marshal uploadedMsg proto: " + err.Error())
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
