package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	tmHttp "github.com/tendermint/tendermint/rpc/client/http"
	tmrpccoretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/stratosnet/framework/utils"

	"github.com/stratosnet/relay/cmd/relayd/setting"
	"github.com/stratosnet/relay/stratoschain"
	"github.com/stratosnet/relay/stratoschain/handlers"
)

// stchainConnection is used to subscribe to stratos-chain events and receive messages via websocket
type stchainConnection struct {
	client *MultiClient

	stratosEventsChannels *sync.Map

	cancel context.CancelFunc
	ctx    context.Context
	once   *sync.Once
	wg     *sync.WaitGroup
	mux    sync.Mutex
}

type websocketSubscription struct {
	channel <-chan tmrpccoretypes.ResultEvent
	client  *tmHttp.HTTP
	query   string
}

func newStchainConnection(client *MultiClient) *stchainConnection {
	return &stchainConnection{
		client:                client,
		stratosEventsChannels: &sync.Map{},
	}
}

func (s *stchainConnection) stop() {
	if s.once == nil {
		return
	}

	s.once.Do(func() {
		s.cancel()

		s.stratosEventsChannels.Range(func(k, v interface{}) bool {
			subscription, ok := v.(websocketSubscription)
			if !ok {
				return false
			}

			s.wg.Add(1)
			go func() {
				defer s.wg.Done()

				if subscription.client == nil {
					return
				}

				err := subscription.client.Unsubscribe(context.Background(), "", subscription.query)
				if err != nil {
					utils.ErrorLog("couldn't unsubscribe from "+subscription.query, err)
				} else {
					utils.DetailLog("unsubscribed from " + subscription.query)
				}
				_ = subscription.client.Stop()
			}()
			return false
		})

		s.wg.Wait()
		utils.Log("stchain connection has been stopped")
	})
}

func (s *stchainConnection) refresh() {
	if !s.mux.TryLock() {
		return // Refresh procedure already started
	}
	defer s.mux.Unlock()

	s.stop() // Stop the connection if it was started before

	s.ctx, s.cancel = context.WithCancel(s.client.Ctx)
	s.once = &sync.Once{}
	s.wg = &sync.WaitGroup{}

	s.wg.Add(1)
	defer s.wg.Done()

	// Schedule next connection refresh
	if setting.Config.StratosChain.ConnectionRetries.RefreshInterval > 0 {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			select {
			case <-time.After(time.Duration(setting.Config.StratosChain.ConnectionRetries.RefreshInterval) * time.Second):
				utils.Logf("stchain connection has been alive for a long time and will be refreshed")
				go s.refresh()
			case <-s.ctx.Done():
				return
			}
		}()
	}

	// Connect to stratos-chain in a loop
	i := 0
	for ; i < setting.Config.StratosChain.ConnectionRetries.Max; i++ {
		if s.ctx.Err() != nil {
			return
		}

		if i != 0 {
			time.Sleep(time.Millisecond * time.Duration(setting.Config.StratosChain.ConnectionRetries.SleepDuration))
		}

		err := s.subscribeToStratosChainEvents()
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
	s.client.cancel() // Cancel global context
}

func (s *stchainConnection) stratosSubscriptionReaderLoop(subscription websocketSubscription, handler func(tmrpccoretypes.ResultEvent)) {
	s.wg.Add(1)

	defer func() {
		if r := recover(); r != nil {
			utils.ErrorLog("Recovering from panic in stratos-chain subscription reader loop", r)
		}

		s.wg.Done()
		go s.refresh()
	}()

	for {
		select {
		case <-s.ctx.Done():
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

func (s *stchainConnection) subscribeToStratosChain(msgType string) error {
	handler, ok := handlers.Handlers[msgType]
	if !ok {
		return errors.Errorf("Cannot subscribe to message [%v] in stratos-chain: missing handler function", msgType)
	}

	query := fmt.Sprintf("message.action='%v'", msgType)
	if _, ok := s.stratosEventsChannels.Load(query); ok {
		return nil
	}

	client, err := stratoschain.DialWebsocket(setting.Config.StratosChain.WebsocketServer)
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
	s.stratosEventsChannels.Store(query, subscription)
	go s.stratosSubscriptionReaderLoop(subscription, handler)

	return nil
}

func (s *stchainConnection) subscribeToStratosChainEvents() error {
	for msgType := range handlers.Handlers {
		err := s.subscribeToStratosChain(msgType)
		if err != nil {
			return err
		}
	}
	return nil
}
