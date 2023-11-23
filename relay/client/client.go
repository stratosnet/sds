package client

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/stratosnet/tx-client/crypto/secp256k1"
	cryptotypes "github.com/stratosnet/tx-client/crypto/types"
	"github.com/stratosnet/tx-client/grpc"
	txclienttypes "github.com/stratosnet/tx-client/types"

	"github.com/stratosnet/framework/utils"

	"github.com/stratosnet/relay/cmd/relayd/setting"
	relaytypes "github.com/stratosnet/relay/types"
)

type MultiClient struct {
	cancel context.CancelFunc
	Ctx    context.Context
	once   *sync.Once

	sdsConn     connection
	stchainConn connection

	WalletAddress    string
	WalletPrivateKey cryptotypes.PrivKey
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

	walletKey, err := relaytypes.DecryptKey(walletJson, setting.Config.Keys.WalletPassword)
	if err != nil {
		return err
	}

	m.WalletPrivateKey = secp256k1.Generate(walletKey.PrivateKey)
	m.WalletAddress = txclienttypes.AccAddress(m.WalletPrivateKey.PubKey().Address()).String()

	utils.DebugLogf("verified wallet key successfully! walletAddr is %v", m.WalletAddress)
	return nil
}

func (m *MultiClient) Start() error {
	// GRPC client to send msgs to stratos-chain
	grpc.SERVER = setting.Config.StratosChain.GrpcServer.GrpcServer
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
