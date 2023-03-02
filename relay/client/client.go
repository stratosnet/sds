package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	tmHttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/stratosnet/sds/cmd/relayd/setting"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/relay"
	"github.com/stratosnet/sds/relay/sds"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	"github.com/stratosnet/sds/relay/stratoschain/handlers"
	"github.com/stratosnet/sds/relay/stratoschain/tx"
	relaytypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/sds/utils/types"
)

type MultiClient struct {
	cancel                context.CancelFunc
	Ctx                   context.Context
	once                  *sync.Once
	sdsWebsocketConn      *websocket.Conn
	stratosWebsocketUrl   string
	stratosEventsChannels *sync.Map
	txBroadcasterChan     chan relaytypes.UnsignedMsg
	WalletAddress         string
	WalletPrivateKey      cryptotypes.PrivKey
	wg                    *sync.WaitGroup
}

type websocketSubscription struct {
	channel <-chan coretypes.ResultEvent
	client  *tmHttp.HTTP
	query   string
}

func NewClient(spHomePath string) (*MultiClient, error) {
	ctx, cancel := context.WithCancel(context.Background())
	newClient := &MultiClient{
		cancel:                cancel,
		Ctx:                   ctx,
		once:                  &sync.Once{},
		stratosEventsChannels: &sync.Map{},
		wg:                    &sync.WaitGroup{},
	}

	err := newClient.loadKeys(spHomePath)
	return newClient, err
}

func (m *MultiClient) loadKeys(spHomePath string) error {
	walletJson, err := ioutil.ReadFile(filepath.Join(spHomePath, setting.Config.Keys.WalletPath))
	if err != nil {
		return err
	}

	walletKey, err := utils.DecryptKey(walletJson, setting.Config.Keys.WalletPassword)
	if err != nil {
		return err
	}

	m.WalletPrivateKey = secp256k1.PrivKeyToSdkPrivKey(walletKey.PrivateKey)
	m.WalletAddress, err = types.BytesToAddress(m.WalletPrivateKey.PubKey().Address()).WalletAddressToBech()
	if err != nil {
		return err
	}

	utils.DebugLogf("verified wallet key successfully! walletAddr is %v", m.WalletAddress)
	return nil
}

func (m *MultiClient) Start() error {
	// GRPC client to send msgs to stratos-chain
	grpc.URL = setting.Config.StratosChain.GrpcServer
	// Client to subscribe to stratos-chain events and receive messages via websocket
	m.stratosWebsocketUrl = setting.Config.StratosChain.WebsocketServer

	go m.connectToSDS()
	go m.connectToStratosChain()

	return nil
}

func (m *MultiClient) Stop() {
	m.once.Do(func() {
		m.cancel()
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

	sdsWebsocketUrl := setting.Config.SDS.NetworkAddress + ":" + setting.Config.SDS.WebsocketPort

	// Connect to SDS SP node in a loop
	for i := 0; i < setting.Config.SDS.ConnectionRetries.Max; i++ {
		if m.Ctx.Err() != nil {
			return
		}

		if i != 0 {
			time.Sleep(time.Millisecond * time.Duration(setting.Config.SDS.ConnectionRetries.SleepDuration))
		}

		// Client to subscribe to events from SDS SP node
		fullSdsWebsocketUrl := "ws://" + sdsWebsocketUrl + "/websocket"
		sdsTopics := []string{sds.TypeBroadcast}
		ws := sds.DialWebsocket(fullSdsWebsocketUrl, sdsTopics)
		if ws == nil {
			continue
		}
		m.sdsWebsocketConn = ws

		go m.sdsEventsReaderLoop()
		go m.txBroadcasterLoop()

		utils.Log("Successfully subscribed to events from SDS SP node and started client to send messages back")
		return
	}

	// This is reached when we couldn't establish the connection to the SP node
	utils.ErrorLog("Couldn't connect to SDS SP node after many tries. Relayd will shutdown")
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

		var unsignedSdkMsgs []sdktypes.Msg
		protoConfig, txBuilder := createTxConfigAndTxBuilder()
		for _, unsignedMsg := range unsignedMsgs {
			unsignedSdkMsgs = append(unsignedSdkMsgs, unsignedMsg.Msg)
		}
		defer func() {
			unsignedMsgs = nil // Clearing msg list
		}()

		err := setMsgInfoToTxBuilder(txBuilder, unsignedSdkMsgs, 0, "")
		if err != nil {
			utils.ErrorLog("couldn't set tx builder", err)
			return
		}
		txBytes, err := tx.BuildTxBytes(protoConfig, txBuilder, setting.Config.BlockchainInfo.ChainId, unsignedMsgs)
		if err != nil {
			utils.ErrorLog("couldn't build tx bytes", err)
			return
		}

		gasInfo, err := grpc.Simulate(txBytes)
		if err != nil {
			utils.ErrorLog("couldn't simulate tx bytes", err)
			return
		}
		gasLimit := uint64(float64(gasInfo.GasUsed) * setting.Config.BlockchainInfo.Transactions.GasAdjustment)
		txBuilder.SetGasLimit(gasLimit)

		gasPrice, err := types.ParseCoinNormalized(setting.Config.BlockchainInfo.Transactions.GasPrice)
		if err != nil {
			utils.ErrorLog("couldn't parse gas price", err)
			return
		}
		feeAmount := gasPrice.Amount.Mul(sdktypes.NewIntFromUint64(gasLimit))
		fee := sdktypes.NewCoin(gasPrice.Denom, feeAmount)
		txBuilder.SetFeeAmount(sdktypes.NewCoins(
			sdktypes.Coin{
				Denom:  fee.Denom,
				Amount: fee.Amount,
			}),
		)

		txBytes, err = tx.BuildTxBytes(protoConfig, txBuilder, setting.Config.BlockchainInfo.ChainId, unsignedMsgs)
		if err != nil {
			utils.ErrorLog("couldn't build tx bytes", err)
			return
		}

		err = grpc.BroadcastTx(txBytes, sdktx.BroadcastMode_BROADCAST_MODE_BLOCK)
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
			if msg.Type != "slashing_resource_node" { // Not printing slashing messages, since SP can slash up to 500 PPs at once, polluting the logs
				utils.DebugLogf("Received a new msg of type [%v] to broadcast! ", msg.Type)
			}
			for i := range msg.SignatureKeys {
				// For messages coming from SP, add the wallet private key that was loaded on start-up
				if len(msg.SignatureKeys[i].PrivateKey) == 0 && msg.SignatureKeys[i].Address == m.WalletAddress {
					msg.SignatureKeys[i].PrivateKey = m.WalletPrivateKey.Bytes()
				}
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
		return errors.Errorf("Cannot subscribe to message [%v] in stratos-chain: missing handler function", msgType)
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

func (m *MultiClient) GetSdsWebsocketConn() *websocket.Conn {
	return m.sdsWebsocketConn
}

func countMsgsByType(unsignedMsgs []*relaytypes.UnsignedMsg) string {
	msgCount := make(map[string]int)
	for _, msg := range unsignedMsgs {
		msgCount[msg.Type]++
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

func setMsgInfoToTxBuilder(txBuilder client.TxBuilder, txMsg []sdktypes.Msg, gas uint64, memo string) error {
	err := txBuilder.SetMsgs(txMsg...)
	if err != nil {
		return err
	}

	//txBuilder.SetFeeGranter(tx.FeeGranter())
	txBuilder.SetGasLimit(gas)
	txBuilder.SetMemo(memo)
	return nil
}

func createTxConfigAndTxBuilder() (client.TxConfig, client.TxBuilder) {
	protoConfig := authtx.NewTxConfig(relay.ProtoCdc, []signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_DIRECT})
	txBuilder := protoConfig.NewTxBuilder()
	return protoConfig, txBuilder
}
