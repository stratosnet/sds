package stratoschain

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Client interface {
	SubscribeToStratosChain(query string, handler func(coretypes.ResultEvent)) error
	GetSdsClientConn() *cf.ClientConn
}

func SubscribeToEvents(c Client) error {
	err := subscribeToCreateResourceNodeMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to create resource node msg: " + err.Error())
	}
	err = subscribeToRemoveResourceNodeMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to remove resource node msg: " + err.Error())
	}
	err = subscribeToCreateIndexingNodeMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to create indexing node msg: " + err.Error())
	}
	err = subscribeToRemoveIndexingNodeMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to remove indexing node msg: " + err.Error())
	}
	err = subscribeToSPRegistrationApprovedMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to SP registration approved msg: " + err.Error())
	}
	err = subscribeToPrepayMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to prepay msg: " + err.Error())
	}
	return nil
}

func subscribeToCreateResourceNodeMsg(c Client) error {
	err := c.SubscribeToStratosChain("message.action='create_resource_node'", func(result coretypes.ResultEvent) {
		//fmt.Printf("%+v\n", result)
		conn := c.GetSdsClientConn()

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
	})
	return err
}

func subscribeToRemoveResourceNodeMsg(c Client) error {
	err := c.SubscribeToStratosChain("message.action='remove_resource_node'", func(result coretypes.ResultEvent) {
		//fmt.Printf("%+v\n", result)
		conn := c.GetSdsClientConn()

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
	})
	return err
}

func subscribeToCreateIndexingNodeMsg(c Client) error {
	err := c.SubscribeToStratosChain("message.action='create_indexing_node'", func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	})
	return err
}

func subscribeToRemoveIndexingNodeMsg(c Client) error {
	err := c.SubscribeToStratosChain("message.action='remove_indexing_node'", func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	})
	return err
}

func subscribeToSPRegistrationApprovedMsg(c Client) error {
	// TODO: name will probably change when this is implemented in stratos-chain
	err := c.SubscribeToStratosChain("message.action='sp_registration_approved'", func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	})
	return err
}

func subscribeToPrepayMsg(c Client) error {
	// TODO: name will probably change when this is implemented in stratos-chain
	err := c.SubscribeToStratosChain("message.action='prepay'", func(result coretypes.ResultEvent) {
		// TODO
		fmt.Printf("%+v\n", result)
	})
	return err
}
