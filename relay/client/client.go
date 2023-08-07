package client

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/gorilla/websocket"
	"github.com/stratosnet/sds/cmd/relayd/setting"
	"github.com/stratosnet/sds/relay/stratoschain/grpc"
	relaytypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/sds/utils/types"
)

type MultiClient struct {
	cancel            context.CancelFunc
	Ctx               context.Context
	once              *sync.Once
	sdsWebsocketConn  *websocket.Conn
	txBroadcasterChan chan relaytypes.UnsignedMsg
	sdsConn           connection
	stchainConn       connection
	WalletAddress     string
	WalletPrivateKey  cryptotypes.PrivKey
}

// connection is a generic interface for a client connection to an external service (sds or stchain)
type connection interface {
	stop()
	refresh() // Stop the connection if it was already started, then re-establish the connection
}

func NewClient(spHomePath string) (*MultiClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	newClient := &MultiClient{
		cancel: cancel,
		Ctx:    ctx,
		once:   &sync.Once{},
	}

	newClient.sdsConn = newSdsConnection(newClient)
	newClient.stchainConn = newStchainConnection(newClient)

	err := newClient.loadKeys(spHomePath)
	return newClient, err
}

func (m *MultiClient) loadKeys(spHomePath string) error {
	walletJson, err := os.ReadFile(filepath.Join(spHomePath, setting.Config.Keys.WalletPath))
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
	grpc.URL = setting.Config.StratosChain.GrpcServer.Url
	grpc.INSECURE = setting.Config.StratosChain.GrpcServer.Insecure

	// Start client connections
	go m.sdsConn.refresh()
	go m.stchainConn.refresh()

	return nil
}

func (m *MultiClient) Stop() {
	m.once.Do(func() {
		m.cancel()
		m.sdsConn.stop()
		m.stchainConn.stop()
	})
}
