package client

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/cmd/relayd/sds"
	"github.com/stratosnet/sds/cmd/relayd/stratoschain"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/msg/protos"
	tmHttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"sync"
)

type MultiClient struct {
	Once *sync.Once
	SdsClientConn *cf.ClientConn
	SdsWebsocketConn *websocket.Conn
	StratosClientConn *tmHttp.HTTP
	StratosEventsChan <-chan coretypes.ResultEvent
}

func NewClient() *MultiClient {
	client := &MultiClient{
		Once: &sync.Once{},
	}
	return client
}

func (m *MultiClient) Start() error {
	// Client to send messages to SDS SP node
	sdsClientUrl := setting.Config.SDS.NetworkAddress + ":" + setting.Config.SDS.ClientPort
	sdsClient := sds.NewClient(sdsClientUrl)
	if sdsClient == nil {
		return errors.New("couldn't start SDS client to send messages to the SP node at " + sdsClientUrl)
	}
	m.SdsClientConn = sdsClient

	// Client to subscribe to events from SDS SP node
	sdsWebsocketUrl := setting.Config.SDS.NetworkAddress + ":" + setting.Config.SDS.WebsocketPort
	fullSdsWebsocketUrl := "ws://" + sdsWebsocketUrl + "/websocket"
	sdsTopics := []string{"test"}
	ws := sds.DialWebsocket(fullSdsWebsocketUrl, sdsTopics)
	if ws == nil {
		return errors.New("couldn't subscribe to SDS websocket at " + fullSdsWebsocketUrl)
	}
	m.SdsWebsocketConn = ws
	go m.sdsEventsReaderLoop()

	// Client to subscribe to stratos-chain events and send messages via websocket
	scWebsocketUrl := setting.Config.StratosChain.NetworkAddress + ":" + setting.Config.StratosChain.WebsocketPort
	stratosQuery := "tm.event = 'Tx'"
	client, eventsChan, err := stratoschain.DialWebsocket(scWebsocketUrl, stratosQuery)
	if err != nil {
		return err
	}
	m.StratosClientConn = client
	m.StratosEventsChan = eventsChan
	go m.stratosEventsReaderLoop()

	return nil
}

func (m *MultiClient) Stop() {
	m.Once.Do(func() {
		if m.SdsClientConn != nil {
			m.SdsClientConn.Close()
		}
		if m.SdsWebsocketConn != nil {
			_ = m.SdsWebsocketConn.Close()
		}
		if m.StratosClientConn != nil {
			_ = m.StratosClientConn.Stop()
		}
		fmt.Println("All client connections have been stopped")
	})
}

func (m *MultiClient) sdsEventsReaderLoop() {
	for {
		_, data, err := m.SdsWebsocketConn.ReadMessage()
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		// TODO: handle messages. Need a proto type that can differentiate between event type
		// Responding to most events will probably involve sending a message to stratos-chain using m.StratosClientConn

		fmt.Println("received: " + string(data))
		msg := protos.RspGetPPList{}
		proto.Unmarshal(data, &msg)
		fmt.Printf("Received: %v\n", msg)
	}
}

func (m *MultiClient) stratosEventsReaderLoop() {
	for message := range m.StratosEventsChan {
		fmt.Println("Received a new message from stratos-chain!")
		// TODO: handle message. Often this will involve sending a message to SDS using m.SdsClientConn
		/*
			msgToSend := &msg.RelayMsgBuf{
				MSGHead: header.MakeMessageHeader(1, 1, 0, header.ReqGetPPList),
			}
			err = m.SdsClientConn.Write(msgToSend)
			if err != nil {
				fmt.Println("Error when sending message to SDS: " + err.Error())
			} else {
				fmt.Println("Sent msg to SDS")
			}
		*/
		fmt.Println(message)
	}
}