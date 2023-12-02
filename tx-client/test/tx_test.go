package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-proto/anyutil"

	"github.com/stratosnet/sds/framework/crypto/ed25519"
	"github.com/stratosnet/sds/framework/crypto/secp256k1"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
	"github.com/stratosnet/sds/tx-client/grpc"
	"github.com/stratosnet/sds/tx-client/tx"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"
	"github.com/stratosnet/sds/tx-client/utils"
)

const (
	retCodeOK             = uint32(0)
	chainId               = "testchain"
	gasAdjustment         = 1.2
	grpcServerTest        = "127.0.0.1:9090"
	grpcInsecureTest      = true
	logPath               = "./logs/relayer-tx-client-stdout.log"
	senderAddrBech32      = "st1edp9gkppxzjvcg9nwheh6tp9rsgafatckfdl6m"
	senderMnemonic        = "lumber mushroom situate mechanic detect cake dune receive pipe source swallow miss original stuff disease baby erosion tomorrow minor salt peace lyrics model win"
	senderBip39Passphrase = ""
	hdPath                = "m/44'/606'/0'/0/0"
)

func TestTxBroadcast(t *testing.T) {
	//testSend(t)
	testCreateResourceNode(t)
}

func testSend(t *testing.T) {
	fmt.Println("------------------ TestSend() start ------------------")
	initGrpcSettings()
	senderKey := generateSenderKey(t)

	amount := txclienttypes.Coin{Denom: txclienttypes.Wei, Amount: sdkmath.NewInt(100)}
	receiverKey, err := secp256k1.GenerateKey()
	require.NoError(t, err)

	senderAddr := senderKey.PubKey().Address().Bytes()
	receiverAddr := receiverKey.PubKey().Address().Bytes()

	msg := tx.BuildSendMsg(senderAddr, receiverAddr, amount)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: senderAddrBech32, PrivateKey: senderKey.Bytes(), Type: txclienttypes.SignatureSecp256k1},
	}

	msgAny, err := anyutil.New(msg)
	require.NoError(t, err)
	fmt.Println("msgAny = ", msgAny)

	txBytes, err := tx.CreateAndSimulateTx(msgAny, defaultTxFee(), "", signatureKeys, chainId, gasAdjustment)
	require.NoError(t, err)

	resp, err := grpc.BroadcastTx(txBytes, txv1beta1.BroadcastMode_BROADCAST_MODE_SYNC)
	require.NoError(t, err)
	fmt.Println(resp)

	require.Equal(t, retCodeOK, resp.GetTxResponse().GetCode())
}

func testCreateResourceNode(t *testing.T) {
	fmt.Println("------------------ TestCreateResourceNode() start ------------------")
	initGrpcSettings()
	senderKey := generateSenderKey(t)
	p2pKey := ed25519.GenPrivKey()
	depositAmt := txclienttypes.Coin{
		Denom:  txclienttypes.Wei,
		Amount: sdkmath.NewInt(5e18),
	}
	senderAddr := senderKey.PubKey().Address()

	msg, err := tx.BuildCreateResourceNodeMsg(txclienttypes.STORAGE, p2pKey.PubKey(), depositAmt, senderAddr.Bytes())
	require.NoError(t, err)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: senderAddrBech32, PrivateKey: senderKey.Bytes(), Type: txclienttypes.SignatureSecp256k1},
	}

	msgAny, err := anyutil.New(msg)
	require.NoError(t, err)
	fmt.Println("msgAny = ", msgAny)

	txBytes, err := tx.CreateAndSimulateTx(msgAny, defaultTxFee(), "", signatureKeys, chainId, gasAdjustment)
	require.NoError(t, err)

	resp, err := grpc.BroadcastTx(txBytes, txv1beta1.BroadcastMode_BROADCAST_MODE_SYNC)
	require.NoError(t, err)
	fmt.Println(resp)

	require.Equal(t, retCodeOK, resp.GetTxResponse().GetCode())
}

func generateSenderKey(t *testing.T) fwcryptotypes.PrivKey {
	senderKeyBytes, err := secp256k1.Derive(senderMnemonic, senderBip39Passphrase, hdPath)
	require.NoError(t, err)

	senderKey := secp256k1.Generate(senderKeyBytes)
	return senderKey
}

func initGrpcSettings() {
	grpc.SERVER = grpcServerTest
	grpc.INSECURE = grpcInsecureTest
	_ = utils.NewDefaultLogger(logPath, true, true)
}

func defaultTxFee() txclienttypes.TxFee {
	gas := uint64(1e7)
	gasPrice := sdkmath.NewInt(1e9)
	fee := gasPrice.MulRaw(int64(gas))
	txFee := txclienttypes.TxFee{
		Fee:      txclienttypes.Coin{Denom: txclienttypes.Wei, Amount: fee},
		Gas:      gas,
		Simulate: true,
	}
	return txFee
}
