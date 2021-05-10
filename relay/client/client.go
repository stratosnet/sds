package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/relay/sds"
	"github.com/stratosnet/sds/relay/stratoschain"
	tmHttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"sync"
	"time"
)

type MultiClient struct {
	Cancel                     context.CancelFunc
	Ctx                        context.Context
	Once                       *sync.Once
	SdsClientConn              *cf.ClientConn
	SdsWebsocketConn           *websocket.Conn
	StratosWebsocketClientConn *tmHttp.HTTP
	StratosEventsChannels      map[<-chan coretypes.ResultEvent]bool
	Wg                         *sync.WaitGroup
}

func NewClient() *MultiClient {
	ctx, cancel := context.WithCancel(context.Background())
	client := &MultiClient{
		Cancel:                cancel,
		Ctx:                   ctx,
		Once:                  &sync.Once{},
		StratosEventsChannels: make(map[<-chan coretypes.ResultEvent]bool),
		Wg:                    &sync.WaitGroup{},
	}
	return client
}

func (m *MultiClient) Start() error {
	m.Wg.Add(1)
	go func() {
		defer m.Wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovering from panic in SDS connection goroutine: %v\n", r)
			}
		}()

		sdsClientUrl := setting.Config.SDS.NetworkAddress + ":" + setting.Config.SDS.ClientPort
		sdsWebsocketUrl := setting.Config.SDS.NetworkAddress + ":" + setting.Config.SDS.WebsocketPort

		i := 0
		for ; i < setting.Config.SDS.ConnectionRetries.Max; i++ {
			if m.Ctx.Err() != nil {
				return
			}

			if i != 0 {
				time.Sleep(time.Millisecond * time.Duration(setting.Config.SDS.ConnectionRetries.SleepDuration))
			}

			// Client to send messages to SDS SP node
			sdsClient := sds.NewClient(sdsClientUrl)
			if sdsClient == nil {
				continue
			}
			m.SdsClientConn = sdsClient

			// Client to subscribe to events from SDS SP node
			fullSdsWebsocketUrl := "ws://" + sdsWebsocketUrl + "/websocket"
			sdsTopics := []string{"todo"} // TODO: fill list of SDS topics to subscribe to
			ws := sds.DialWebsocket(fullSdsWebsocketUrl, sdsTopics)
			if ws == nil {
				break
			}
			m.SdsWebsocketConn = ws

			m.Wg.Add(1)
			go m.sdsEventsReaderLoop()

			fmt.Println("Successfully subscribed to events from SDS SP node and started client to send messages back")
			return
		}

		// This is reached when we couldn't establish the connection to the SP node
		if i == setting.Config.SDS.ConnectionRetries.Max {
			fmt.Println("Couldn't connect to SDS SP node after many tries. Relay will shutdown")
		} else {
			fmt.Println("Couldn't subscribe to SDS events through websockets. Relay will shutdown")
		}
		m.Cancel()
	}()

	// REST client to send messages to stratos-chain
	scRestUrl := setting.Config.StratosChain.NetworkAddress + ":" + setting.Config.StratosChain.RestPort
	stratoschain.Url = scRestUrl

	// Client to subscribe to stratos-chain events and send messages via websocket
	scWebsocketUrl := setting.Config.StratosChain.NetworkAddress + ":" + setting.Config.StratosChain.WebsocketPort
	client, err := stratoschain.DialWebsocket(scWebsocketUrl)
	if err != nil {
		return err
	}
	m.StratosWebsocketClientConn = client
	err = stratoschain.SubscribeToEvents(stratoschain.Client(m))
	if err != nil {
		return err
	}

	fmt.Println("Successfully subscribed to events from stratos-chain and started client to send messages back")
	return nil
}

func (m *MultiClient) Stop() {
	m.Once.Do(func() {
		m.Cancel()
		m.Wg.Wait()

		if m.SdsClientConn != nil {
			m.SdsClientConn.Close()
		}
		if m.SdsWebsocketConn != nil {
			_ = m.SdsWebsocketConn.Close()
		}
		if m.StratosWebsocketClientConn != nil {
			_ = m.StratosWebsocketClientConn.Stop()
		}
		fmt.Println("All client connections have been stopped")
	})
}

func (m *MultiClient) sdsEventsReaderLoop() {
	defer m.Wg.Done()
	for {
		if m.Ctx.Err() != nil {
			return
		}
		_, data, err := m.SdsWebsocketConn.ReadMessage()
		if err != nil {
			fmt.Println("error when reading from the SDS websocket: " + err.Error())
			return
		}

		// TODO: handle messages. Need a proto type that can differentiate between event type
		// Responding to most events will probably involve sending a message to stratos-chain using m.StratosWebsocketClientConn

		fmt.Println("received: " + string(data))
		msg := protos.RspGetPPList{}
		proto.Unmarshal(data, &msg)
		fmt.Printf("Received: %v\n", msg)
	}
}

func (m *MultiClient) stratosSubscriptionReaderLoop(channel <-chan coretypes.ResultEvent, handler func(coretypes.ResultEvent)) {
	defer delete(m.StratosEventsChannels, channel)
	defer m.Wg.Done()
	for {
		select {
		case <-m.Ctx.Done():
			return
		case message, ok := <-channel:
			if !ok {
				fmt.Println("The stratos-chain events websocket channel has been closed")
				return
			}
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
			handler(message)
		}
	}
}

func (m *MultiClient) SubscribeToStratosChain(query string, handler func(coretypes.ResultEvent)) error {
	out, err := m.StratosWebsocketClientConn.Subscribe(context.Background(), "", query)
	if err != nil {
		return errors.New("failed to subscribe to query in stratos-chain: " + err.Error())
	}
	m.StratosEventsChannels[out] = true
	m.Wg.Add(1)
	go m.stratosSubscriptionReaderLoop(out, handler)
	return nil
}
