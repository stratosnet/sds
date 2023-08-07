package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/gorilla/websocket"
	"github.com/stratosnet/sds/cmd/relayd/setting"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/relay"
	"github.com/stratosnet/sds/relay/sds"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	relaytypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/types"
	"google.golang.org/protobuf/proto"
)

const (
	txBroadcastMaxInterval = 500 // milliseconds
)

type sdsConnection struct {
	client *MultiClient

	sdsWebsocketConn  *websocket.Conn
	txBroadcasterChan chan relaytypes.UnsignedMsg

	cancel context.CancelFunc
	ctx    context.Context
	once   *sync.Once
	wg     *sync.WaitGroup
	mux    sync.Mutex
}

func newSdsConnection(client *MultiClient) *sdsConnection {
	return &sdsConnection{
		client: client,
	}
}

func (s *sdsConnection) stop() {
	if s.once == nil {
		return
	}

	s.once.Do(func() {
		s.cancel()

		if s.sdsWebsocketConn != nil {
			_ = s.sdsWebsocketConn.Close()
		}

		if s.txBroadcasterChan != nil {
			close(s.txBroadcasterChan)
			s.txBroadcasterChan = nil
		}

		s.wg.Wait()
		utils.Log("sds connection has been stopped")
	})
}

func (s *sdsConnection) refresh() {
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
	if setting.Config.SDS.ConnectionRetries.RefreshInterval > 0 {
		go func() {
			s.wg.Add(1)
			defer s.wg.Done()

			select {
			case <-time.After(time.Duration(setting.Config.SDS.ConnectionRetries.RefreshInterval) * time.Second):
				utils.Logf("sds connection has been alive for a long time and will be refreshed")
				go s.refresh()
			case <-s.ctx.Done():
				return
			}
		}()
	}

	sdsWebsocketUrl := setting.Config.SDS.NetworkAddress + ":" + setting.Config.SDS.WebsocketPort

	// Connect to SDS SP node in a loop
	i := 0
	for ; i < setting.Config.SDS.ConnectionRetries.Max; i++ {
		if s.ctx.Err() != nil {
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
		s.sdsWebsocketConn = ws

		go s.sdsEventsReaderLoop()
		go s.txBroadcasterLoop()

		utils.Log("Successfully subscribed to events from SDS SP node and started client to send messages back")
		return
	}

	// This is reached when we couldn't establish the connection to the SP node
	if i == setting.Config.SDS.ConnectionRetries.Max {
		utils.ErrorLog("Couldn't connect to SDS SP node after many tries. Relayd will shutdown")
	} else {
		utils.ErrorLog("Couldn't subscribe to SDS SP node. Relayd will shutdown")
	}
	s.client.cancel() // Cancel global context
}

func (s *sdsConnection) sdsEventsReaderLoop() {
	s.wg.Add(1)

	defer func() {
		if r := recover(); r != nil {
			utils.ErrorLog("Recovering from panic in sds events reader loop", r)
		}

		s.wg.Done()
		go s.refresh()
	}()

	for {
		if s.ctx.Err() != nil {
			return
		}
		_, data, err := s.sdsWebsocketConn.ReadMessage()
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
				s.txBroadcasterChan <- unsignedMsg
			}
		}
	}
}

func (s *sdsConnection) txBroadcasterLoop() {
	if s.txBroadcasterChan != nil {
		// tx broadcaster loop already running
		return
	}

	s.wg.Add(1)

	defer func() {
		if r := recover(); r != nil {
			utils.ErrorLog("Recovering from panic in sds tx broadcaster loop", r)
		}

		s.wg.Done()
		go s.refresh()
	}()

	s.txBroadcasterChan = make(chan relaytypes.UnsignedMsg, setting.Config.StratosChain.Broadcast.ChannelSize)

	var unsignedMsgs []*relaytypes.UnsignedMsg
	broadcastTxs := func() {
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
		txBytes, err := stratoschain.BuildTxBytes(protoConfig, txBuilder, setting.Config.BlockchainInfo.ChainId, unsignedMsgs)
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

		txBytes, err = stratoschain.BuildTxBytes(protoConfig, txBuilder, setting.Config.BlockchainInfo.ChainId, unsignedMsgs)
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

	timeOver := time.After(txBroadcastMaxInterval * time.Millisecond)
	for {
		select {
		case <-s.ctx.Done():
			return
		case msg, ok := <-s.txBroadcasterChan:
			if !ok {
				utils.ErrorLog("The stratos-chain tx broadcaster channel has been closed")
				return
			}
			if msg.Type != "slashing_resource_node" { // Not printing slashing messages, since SP can slash up to 500 PPs at once, polluting the logs
				utils.DebugLogf("Received a new msg of type [%v] to broadcast! ", msg.Type)
			}
			for i := range msg.SignatureKeys {
				// For messages coming from SP, add the wallet private key that was loaded on start-up
				if len(msg.SignatureKeys[i].PrivateKey) == 0 && msg.SignatureKeys[i].Address == s.client.WalletAddress {
					msg.SignatureKeys[i].PrivateKey = s.client.WalletPrivateKey.Bytes()
				}
			}
			unsignedMsgs = append(unsignedMsgs, &msg)
			if len(unsignedMsgs) >= setting.Config.StratosChain.Broadcast.MaxMsgPerTx {
				// Max broadcast size is reached. Broadcasting now
				broadcastTxs()
				timeOver = time.After(txBroadcastMaxInterval * time.Millisecond)
			}
		case <-timeOver:
			// No new messages are waiting to broadcast. Broadcasting existing messages now
			if len(unsignedMsgs) > 0 {
				broadcastTxs()
				timeOver = time.After(txBroadcastMaxInterval * time.Millisecond)
			}
		}
	}
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
