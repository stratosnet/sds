package client

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/relay/sds"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/relay/stratoschain/handlers"
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
			err = stratoschain.BroadcastTxBytes(msg.Data)
			if err != nil {
				utils.ErrorLog("couldn't broadcast transaction", err)
				continue
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
			utils.Log("Received a new message from stratos-chain!", subscription.query)
			handler(message)
		}
	}
}

func (m *MultiClient) SubscribeToStratosChain(query string, handler func(coretypes.ResultEvent)) error {
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
	err := m.SubscribeToStratosChain("message.action='create_resource_node'", handlers.CreateResourceNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='update_resource_node_stake'", handlers.UpdateResourceNodeStakeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='unbonding_resource_node'", handlers.UnbondingResourceNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='remove_resource_node'", handlers.RemoveResourceNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='complete_unbonding_resource_node'", handlers.CompleteUnbondingResourceNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='create_indexing_node'", handlers.CreateIndexingNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='update_indexing_node_stake'", handlers.UpdateIndexingNodeStakeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='unbonding_indexing_node'", handlers.UnbondingIndexingNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='remove_indexing_node'", handlers.RemoveIndexingNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='complete_unbonding_indexing_node'", handlers.CompleteUnbondingIndexingNodeMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='indexing_node_reg_vote'", handlers.IndexingNodeVoteMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='SdsPrepayTx'", handlers.PrepayMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='FileUploadTx'", handlers.FileUploadMsgHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='volume_report'", handlers.VolumeReportHandler())
	if err != nil {
		return err
	}
	err = m.SubscribeToStratosChain("message.action='slashing_resource_node'", handlers.SlashingResourceNodeHandler())
	return err
}

func (m *MultiClient) GetSdsClientConn() *cf.ClientConn {
	return m.sdsClientConn
}

func (m *MultiClient) GetSdsWebsocketConn() *websocket.Conn {
	return m.sdsWebsocketConn
}
