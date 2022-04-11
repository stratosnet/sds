package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/relay/sds"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/relay/stratoschain/handlers"
	relaytypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils"
	tmHttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type MultiClient struct {
	cancel                context.CancelFunc
	Ctx                   context.Context
	once                  *sync.Once
	sdsClientConn         *cf.ClientConn
	sdsWebsocketConn      *websocket.Conn
	stratosWebsocketUrl   string
	stratosEventsChannels *sync.Map
	txBroadcasterChan     chan relaytypes.UnsignedMsg
	wg                    *sync.WaitGroup
}

type websocketSubscription struct {
	channel <-chan coretypes.ResultEvent
	client  *tmHttp.HTTP
	query   string
}

func NewClient() *MultiClient {
	ctx, cancel := context.WithCancel(context.Background())
	client := &MultiClient{
		cancel:                cancel,
		Ctx:                   ctx,
		once:                  &sync.Once{},
		stratosEventsChannels: &sync.Map{},
		wg:                    &sync.WaitGroup{},
	}
	return client
}

func (m *MultiClient) Start() error {
	// REST client to send messages to stratos-chain
	stratoschain.Url = setting.Config.StratosChain.RestServer
	// Client to subscribe to stratos-chain events and receive messages via websocket
	m.stratosWebsocketUrl = setting.Config.StratosChain.WebsocketServer

	go m.connectToSDS()
	go m.connectToStratosChain()

	return nil
}

func (m *MultiClient) Stop() {
	m.once.Do(func() {
		m.cancel()
		if m.sdsClientConn != nil {
			m.sdsClientConn.ClientClose()
		}
		if m.sdsWebsocketConn != nil {
			_ = m.sdsWebsocketConn.Close()
		}

		m.stratosEventsChannels.Range(func(k, v interface{}) bool {
			subscription, ok := v.(websocketSubscription)
			if !ok {
				return false
			}

			go func() {
				err := subscription.client.Unsubscribe(context.Background(), "", subscription.query)
				if err != nil {
					utils.ErrorLog("couldn't unsubscribe from "+subscription.query, err)
				} else {
					utils.Log("unsubscribed from " + subscription.query)
				}
				_ = subscription.client.Stop()
			}()
			return false
		})

		m.wg.Wait()
		utils.Log("All client connections have been stopped")
	})
}

func (m *MultiClient) connectToSDS() {
	m.wg.Add(1)
	defer m.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			utils.ErrorLog("Recovering from panic in SDS connection goroutine", r)
		}
	}()

	sdsClientUrl := setting.Config.SDS.NetworkAddress + ":" + setting.Config.SDS.ClientPort
	sdsWebsocketUrl := setting.Config.SDS.NetworkAddress + ":" + setting.Config.SDS.WebsocketPort

	// Connect to SDS SP node in a loop
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
		sdsTopics := []string{sds.TypeBroadcast}
		ws := sds.DialWebsocket(fullSdsWebsocketUrl, sdsTopics)
		if ws == nil {
			break
		}
		m.sdsWebsocketConn = ws

		go m.sdsEventsReaderLoop()
		go m.txBroadcasterLoop()

		utils.Log("Successfully subscribed to events from SDS SP node and started client to send messages back")
		return
	}

	// This is reached when we couldn't establish the connection to the SP node
	if i == setting.Config.SDS.ConnectionRetries.Max {
		utils.ErrorLog("Couldn't connect to SDS SP node after many tries. Relayd will shutdown")
	} else {
		utils.ErrorLog("Couldn't subscribe to SDS events through websockets. Relayd will shutdown")
	}
	m.cancel()
}

func (m *MultiClient) connectToStratosChain() {
	m.wg.Add(1)
	defer m.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			utils.ErrorLog("Recovering from panic in stratos-chain connection goroutine", r)
		}
	}()

	// Connect to stratos-chain in a loop
	i := 0
	for ; i < setting.Config.StratosChain.ConnectionRetries.Max; i++ {
		if m.Ctx.Err() != nil {
			return
		}

		if i != 0 {
			time.Sleep(time.Millisecond * time.Duration(setting.Config.StratosChain.ConnectionRetries.SleepDuration))
		}

		err := m.SubscribeToStratosChainEvents()
		if err != nil {
			utils.ErrorLog(err)
			continue
		}

		utils.Log("Successfully subscribed to events from stratos-chain")
		return
	}

	// This is reached when we couldn't establish the connection to the stratos-chain
	if i == setting.Config.StratosChain.ConnectionRetries.Max {
		utils.ErrorLog("Couldn't connect to stratos-chain after many tries. Relayd will shutdown")
	} else {
		utils.ErrorLog("Couldn't subscribe to stratos-chain events through websockets. Relayd will shutdown")
	}
	m.cancel()
}

func (m *MultiClient) sdsEventsReaderLoop() {
	m.wg.Add(1)
	defer func() {
		if m.Ctx.Err() == nil {
			go m.connectToSDS()
		}
		m.wg.Done()
	}()

	for {
		if m.Ctx.Err() != nil {
			return
		}
		_, data, err := m.sdsWebsocketConn.ReadMessage()
		if err != nil {
			utils.ErrorLog("error when reading from the SDS websocket", err)
			return
		}

		utils.Log("received: " + string(data))
		msg := protos.RelayMessage{}
		err = proto.Unmarshal(data, &msg)
		if err != nil {
			utils.ErrorLog("couldn't unmarshal message to protos.RelayMessage", err)
			continue
		}

		switch msg.Type {
		case sds.TypeBroadcast:
			unsignedMsgs := relaytypes.UnsignedMsgs{}
			err = json.Unmarshal(msg.Data, &unsignedMsgs)
			if err != nil {
				utils.ErrorLog("couldn't unmarshal UnsignedMsgs json", err)
				continue
			}
			for _, msgBytes := range unsignedMsgs.Msgs {
				unsignedMsg, err := msgBytes.FromBytes()
				if err != nil {
					utils.ErrorLog(err)
					continue
				}
				// Add unsignedMsg to tx broadcast channel
				m.txBroadcasterChan <- unsignedMsg
			}
		}
	}
}

func (m *MultiClient) txBroadcasterLoop() {
	if m.txBroadcasterChan != nil {
		// tx broadcaster loop already running
		return
	}

	m.wg.Add(1)
	defer func() {
		// Close existing channel
		close(m.txBroadcasterChan)
		m.txBroadcasterChan = nil

		// If the ctx is not done yet, restart the Tx broadcaster channel
		if m.Ctx.Err() == nil {
			go m.txBroadcasterLoop()
		}
		m.wg.Done()
	}()

	m.txBroadcasterChan = make(chan relaytypes.UnsignedMsg, setting.Config.StratosChain.Broadcast.ChannelSize)

	var unsignedMsgs []*relaytypes.UnsignedMsg
	broadcastTx := func() {
		utils.Logf("Tx broadcaster loop will try to broadcast %v msgs %v", len(unsignedMsgs), countMsgsByType(unsignedMsgs))
		txBytes, err := stratoschain.BuildTxBytes(setting.Config.BlockchainInfo.Token, setting.Config.BlockchainInfo.ChainId, "",
			flags.BroadcastBlock, unsignedMsgs, setting.Config.BlockchainInfo.Transactions.Fee,
			setting.Config.BlockchainInfo.Transactions.Gas)
		unsignedMsgs = nil // Clearing msg list
		if err != nil {
			utils.ErrorLog("couldn't build tx bytes", err)
			return
		}

		err = stratoschain.BroadcastTxBytes(txBytes)
		if err != nil {
			utils.ErrorLog("couldn't broadcast transaction", err)
			return
		}
	}

	for {
		select {
		case <-m.Ctx.Done():
			return
		case msg, ok := <-m.txBroadcasterChan:
			if !ok {
				utils.ErrorLog("The stratos-chain tx broadcaster channel has been closed")
				return
			}
			if msg.Msg.Type() != "slashing_resource_node" { // Not printing slashing messages, since SP can slash up to 500 PPs at once, polluting the logs
				utils.DebugLogf("Received a new msg of type [%v] to broadcast! ", msg.Msg.Type())
			}
			unsignedMsgs = append(unsignedMsgs, &msg)
			if len(unsignedMsgs) >= setting.Config.StratosChain.Broadcast.MaxMsgPerTx {
				// Max broadcast size is reached. Broadcasting now
				broadcastTx()
			}
		case <-time.After(500 * time.Millisecond):
			// No new messages are waiting to broadcast. Broadcasting existing messages now
			if len(unsignedMsgs) > 0 {
				broadcastTx()
			}
		}
	}
}

func (m *MultiClient) stratosSubscriptionReaderLoop(subscription websocketSubscription, handler func(coretypes.ResultEvent)) {
	m.wg.Add(1)
	defer func() {
		if subscription.client != nil {
			go func() {
				err := subscription.client.Unsubscribe(context.Background(), "", subscription.query)
				if err != nil {
					utils.ErrorLog("couldn't unsubscribe from "+subscription.query, err)
				} else {
					utils.Log("unsubscribed from " + subscription.query)
				}
				_ = subscription.client.Stop()
			}()
		}
		m.stratosEventsChannels.Delete(subscription.query)
		m.wg.Done()
	}()

	for {
		select {
		case <-m.Ctx.Done():
			return
		case message, ok := <-subscription.channel:
			if !ok {
				utils.Log("The stratos-chain events websocket channel has been closed")
				return
			}
			utils.Logf("Received a new message of type [%v] from stratos-chain!", subscription.query)
			handler(message)
		}
	}
}

func (m *MultiClient) SubscribeToStratosChain(msgType string) error {
	handler, ok := handlers.Handlers[msgType]
	if !ok {
		return errors.Errorf("Cannot subscribe to message [%v] in stratos-chain: missing handler function")
	}

	query := fmt.Sprintf("message.action='%v'", msgType)
	if _, ok := m.stratosEventsChannels.Load(query); ok {
		return nil
	}

	client, err := stratoschain.DialWebsocket(m.stratosWebsocketUrl)
	if err != nil {
		return err
	}

	out, err := client.Subscribe(context.Background(), "", query, 20)
	if err != nil {
		return errors.New("failed to subscribe to query in stratos-chain: " + err.Error())
	}

	subscription := websocketSubscription{
		channel: out,
		client:  client,
		query:   query,
	}
	m.stratosEventsChannels.Store(query, subscription)
	go m.stratosSubscriptionReaderLoop(subscription, handler)

	return nil
}

func (m *MultiClient) SubscribeToStratosChainEvents() error {
	for msgType := range handlers.Handlers {
		err := m.SubscribeToStratosChain(msgType)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiClient) GetSdsClientConn() *cf.ClientConn {
	return m.sdsClientConn
}

func (m *MultiClient) GetSdsWebsocketConn() *websocket.Conn {
	return m.sdsWebsocketConn
}

func countMsgsByType(unsignedMsgs []*relaytypes.UnsignedMsg) string {
	msgCount := make(map[string]int)
	for _, msg := range unsignedMsgs {
		msgCount[msg.Msg.Type()]++
	}

	countString := ""
	for msgType, count := range msgCount {
		if countString != "" {
			countString += ", "
		}
		countString += fmt.Sprintf("%v: %v", msgType, count)
	}
	return "[" + countString + "]"
}
