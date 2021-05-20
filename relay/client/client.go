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
	cancel                 context.CancelFunc
	Ctx                    context.Context
	once                   *sync.Once
	sdsClientConn          *cf.ClientConn
	sdsWebsocketConn       *websocket.Conn
	stratosWebsocketClient *tmHttp.HTTP
	stratosEventsChannels  map[<-chan coretypes.ResultEvent]bool
	wg                     *sync.WaitGroup
}

func NewClient() *MultiClient {
	ctx, cancel := context.WithCancel(context.Background())
	client := &MultiClient{
		cancel:                cancel,
		Ctx:                   ctx,
		once:                  &sync.Once{},
		stratosEventsChannels: make(map[<-chan coretypes.ResultEvent]bool),
		wg:                    &sync.WaitGroup{},
	}
	return client
}

func (m *MultiClient) Start() error {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
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
			m.sdsClientConn = sdsClient

			// Client to subscribe to events from SDS SP node
			fullSdsWebsocketUrl := "ws://" + sdsWebsocketUrl + "/websocket"
			sdsTopics := []string{"broadcast"}
			ws := sds.DialWebsocket(fullSdsWebsocketUrl, sdsTopics)
			if ws == nil {
				break
			}
			m.sdsWebsocketConn = ws

			m.wg.Add(1)
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
		m.cancel()
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
	m.stratosWebsocketClient = client
	err = stratoschain.SubscribeToEvents(stratoschain.Client(m))
	if err != nil {
		return err
	}

	fmt.Println("Successfully subscribed to events from stratos-chain and started client to send messages back")
	return nil
}

func (m *MultiClient) Stop() {
	m.once.Do(func() {
		m.cancel()
		m.wg.Wait()

		if m.sdsClientConn != nil {
			m.sdsClientConn.Close()
		}
		if m.sdsWebsocketConn != nil {
			_ = m.sdsWebsocketConn.Close()
		}
		if m.stratosWebsocketClient != nil {
			_ = m.stratosWebsocketClient.Stop()
		}
		fmt.Println("All client connections have been stopped")
	})
}

func (m *MultiClient) sdsEventsReaderLoop() {
	defer m.wg.Done()
	for {
		if m.Ctx.Err() != nil {
			return
		}
		_, data, err := m.sdsWebsocketConn.ReadMessage()
		if err != nil {
			fmt.Println("error when reading from the SDS websocket: " + err.Error())
			return
		}

		fmt.Println("received: " + string(data))
		msg := protos.RelayMessage{}
		err = proto.Unmarshal(data, &msg)
		if err != nil {
			fmt.Println("couldn't unmarshal message to protos.RelayMessage: " + err.Error())
			continue
		}

		switch msg.Type {
		case sds.TypeBroadcast:
			err = stratoschain.BroadcastTxBytes(msg.Data)
			if err != nil {
				fmt.Println("couldn't broadcast transaction: " + err.Error())
				continue
			}
		}
	}
}

func (m *MultiClient) stratosSubscriptionReaderLoop(channel <-chan coretypes.ResultEvent, handler func(coretypes.ResultEvent)) {
	defer delete(m.stratosEventsChannels, channel)
	defer m.wg.Done()
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
			// TODO: handle message. Often this will involve sending a message to SDS using m.sdsClientConn
			/*
				msgToSend := &msg.RelayMsgBuf{
					MSGHead: header.MakeMessageHeader(1, 1, 0, header.ReqGetPPList),
				}
				err = m.sdsClientConn.Write(msgToSend)
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
	out, err := m.stratosWebsocketClient.Subscribe(context.Background(), "", query)
	if err != nil {
		return errors.New("failed to subscribe to query in stratos-chain: " + err.Error())
	}
	m.stratosEventsChannels[out] = true
	m.wg.Add(1)
	go m.stratosSubscriptionReaderLoop(out, handler)
	return nil
}

func (m *MultiClient) GetSdsClientConn() *cf.ClientConn {
	return m.sdsClientConn
}

func (m *MultiClient) GetSdsWebsocketConn() *websocket.Conn {
	return m.sdsWebsocketConn
}

func (m *MultiClient) GetStratosWebsocketClient() *tmHttp.HTTP {
	return m.stratosWebsocketClient
}
