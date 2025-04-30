package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	tmlog "github.com/cometbft/cometbft/libs/log"
	cmtpubsub "github.com/cometbft/cometbft/libs/pubsub"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	wsclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/relayer/cmd/relayd/setting"

	"github.com/cometbft/cometbft/libs/service"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/relayer/stratoschain/handlers"
)

const (
	ENABLE_WSCLIENT_LOG = false
	NEW_BLOCK_QUERY     = "tm.event='NewBlock'"
)

// stchainConnection is used to subscribe to stratos-chain events and receive messages via websocket
type stchainConnection struct {
	service.BaseService
	client                *MultiClient
	stratosEventsChannels *sync.Map
	ws                    *wsclient.WSClient
}

func newStchainConnection(client *MultiClient) *stchainConnection {
	url, err := utils.ParseUrl(setting.Config.StratosChain.WebsocketServer)
	if err != nil {
		return nil
	}

	wsClient, err := wsclient.NewWS(url.String(true, true, false, false), "/websocket")
	if err != nil {
		return nil
	}

	s := &stchainConnection{
		client:                client,
		stratosEventsChannels: &sync.Map{},
		ws:                    wsClient,
	}

	if ENABLE_WSCLIENT_LOG {
		logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
		logger.With("module", "stchain-channel")
		s.ws.SetLogger(logger)
	}

	wsclient.OnReconnect(s.onReconnect)(s.ws)
	wsclient.PingPeriod(30 * time.Second)(s.ws)
	wsclient.WriteWait(25 * time.Second)(s.ws)
	return s
}

func (s *stchainConnection) subscribeAllQueries() error {
	utils.DebugLog("==== subscribe queries ====")
	err := s.ws.Subscribe(context.Background(), NEW_BLOCK_QUERY)
	if err != nil {
		return err
	}

	for msgType := range handlers.Handlers {
		_, ok := handlers.Handlers[msgType]
		if !ok {
			return errors.Errorf("Cannot subscribe to message [%v] in stratos-chain: missing handler function", msgType)
		}
		query := fmt.Sprintf("message.action='%v'", msgType)
		utils.DebugLog("subscribe:", query)
		err := s.ws.Subscribe(context.Background(), query)
		if err != nil {
			return err
		}
	}
	utils.DebugLog("==== end sub queries ====")
	return nil
}

func (s *stchainConnection) onReconnect() {
	// wsclient doesn't take care of the re-subscription operation when reconnect to ws conn
	err := s.subscribeAllQueries()
	if err != nil {
		utils.ErrorLog("Failed subscribing queries:", err.Error())
	}
}

func (s *stchainConnection) start() error {
	s.ws.OnStart()
	if err := s.subscribeAllQueries(); err != nil {
		utils.ErrorLog("Failed subscribing queries:", err.Error())
	}
	utils.Log("Successfully subscribed to events from stratos-chain")
	go s.readerLoop()
	return nil
}

func (s *stchainConnection) stop() {
	utils.DebugLog("stchainConnection.Stop ... ")
	s.ws.Stop()
}

func (s *stchainConnection) readerLoop() {
EventLoop:
	for {
		select {
		case resp, ok := <-s.ws.ResponsesCh:
			if !ok {
				return
			}
			if resp.Error != nil {
				utils.ErrorLog("WS error", resp.Error.Error())
				// Error can be ErrAlreadySubscribed or max client (subscriptions per
				// client) reached or CometBFT exited.
				// We can ignore ErrAlreadySubscribed, but need to retry in other
				// cases.
				if !strings.Contains(resp.Error.Error(), cmtpubsub.ErrAlreadySubscribed.Error()) {
					// Resubscribe after 1 second to give CometBFT time to restart (if
					// crashed).
					time.Sleep(1 * time.Second)
					go s.subscribeAllQueries()
				}
				continue
			}
			result := new(coretypes.ResultEvent)
			err := cmtjson.Unmarshal(resp.Result, result)
			if err != nil {
				//Logger.Error("failed to unmarshal response", "err", err)
				continue
			}
			// Notify tx broadcaster that a new block was processed
			if result.Query == NEW_BLOCK_QUERY {
				select {
				case s.client.NewBlockChan <- true:
				default:
				}
				// Check if there is something to handle in the new block event
				for name := range result.Events {
					if strings.HasPrefix(name, handlers.EventTypeNewFilesUploaded) {
						handler, ok := handlers.Handlers[handlers.EventTypeNewFilesUploaded]
						if ok && handler != nil {
							cleanEventStrings(*result)
							utils.Logf("Received a new event of type [%v] from stratos-chain!", handlers.EventTypeNewFilesUploaded)
							handler(*result)
						}
						continue EventLoop
					}
				}
				continue
			}
			msgType := ""
			n, err := fmt.Sscanf(result.Query, "message.action='%v'", &msgType)
			if n <= 0 && err != nil {
				continue
			}
			msgType = strings.TrimRight(msgType, "'")

			handler, ok := handlers.Handlers[msgType]
			if ok && handler != nil {
				cleanEventStrings(*result)
				utils.Logf("Received a new message of type [%v] from stratos-chain!", msgType)
				handler(*result)
			}
		}
	}
}

func (s *stchainConnection) refresh() {
	s.start()
}

func cleanEventStrings(resultEvent coretypes.ResultEvent) {
	for name, values := range resultEvent.Events {
		for i, value := range values {
			if len(value) >= 2 && value[0:1] == "\"" && value[len(value)-1:] == "\"" {
				values[i] = value[1 : len(value)-1]
			}
		}
		resultEvent.Events[name] = values
	}

	eventDataTx, ok := resultEvent.Data.(comettypes.EventDataTx)
	if !ok {
		return
	}
	for i, event := range eventDataTx.Result.Events {
		for j, attribute := range event.Attributes {
			if len(attribute.Value) >= 2 && attribute.Value[0:1] == "\"" && attribute.Value[len(attribute.Value)-1:] == "\"" {
				event.Attributes[j].Value = attribute.Value[1 : len(attribute.Value)-1]
			}
		}
		eventDataTx.Result.Events[i] = event
	}
}
