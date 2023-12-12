package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-proto/anyutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
	sdkmath "cosmossdk.io/math"
	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"

	"github.com/stratosnet/sds/framework/crypto"
	"github.com/stratosnet/sds/framework/crypto/bls"
	fwed25519 "github.com/stratosnet/sds/framework/crypto/ed25519"
	fwsecp256k1 "github.com/stratosnet/sds/framework/crypto/secp256k1"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
	fwtypes "github.com/stratosnet/sds/framework/types"

	"github.com/stratosnet/sds/tx-client/grpc"
	"github.com/stratosnet/sds/tx-client/tx"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"
	"github.com/stratosnet/sds/tx-client/utils"
)

/**
	###################################################
	1, Recover user0 on the stratos-chain side using senderMnemonic
	2, Add initial meta-node to the genesis file and user0 should be the owner
	3, Execute foundation-deposit tx on the chain side.

    $ ./stchaind tx pot foundation-deposit --amount=400000stos --from=user0 --home build/node/stchaind --chain-id=testchain --keyring-backend=test --gas=600000 --gas-prices=1gwei
	###################################################
*/

const (
	retCodeOK        = uint32(0)
	chainId          = "testchain"
	gasAdjustment    = 1.2
	grpcServerTest   = "127.0.0.1:9090"
	grpcInsecureTest = true
	logPath          = "./logs/relayer-tx-client-stdout.log"
	//senderAddrBech32        = "st1edp9gkppxzjvcg9nwheh6tp9rsgafatckfdl6m"
	senderMnemonic          = "lumber mushroom situate mechanic detect cake dune receive pipe source swallow miss original stuff disease baby erosion tomorrow minor salt peace lyrics model win"
	senderBip39Passphrase   = ""
	hdPath                  = "m/44'/606'/0'/0/0"
	initMetaNodeNetworkAddr = "stsds1cw8qhgsxddak8hh8gs7veqmy5ku8f8za6qlq64"
	txBroadcastInterval     = time.Second * 7
)

func TestTxBroadcast(t *testing.T) {
	initGrpcSettings()
	senderPrivKey := generateSenderKey(t)
	initialMetaNodeP2PAddr, err := fwtypes.P2PAddressFromBech32(initMetaNodeNetworkAddr)
	require.NoError(t, err)

	// start test txs
	testSend(t, senderPrivKey)

	time.Sleep(txBroadcastInterval)
	fmt.Println("#### create resource node 1")
	testCreateResourceNode(t, senderPrivKey)

	// create meta node1
	time.Sleep(txBroadcastInterval)
	fmt.Println("#### create meta node 1")
	metaNodeP2PPrivKey1 := testCreateMetaNode(t, senderPrivKey)

	// vote meta node1 by initial meta node
	time.Sleep(txBroadcastInterval)
	fmt.Println("#### vote meta node1 by initial meta node")
	testMetaNodeRegVote(t, senderPrivKey, metaNodeP2PPrivKey1, initialMetaNodeP2PAddr)

	// create meta node2
	time.Sleep(txBroadcastInterval)
	fmt.Println("#### create meta node 2")
	metaNodeP2PPrivKey2 := testCreateMetaNode(t, senderPrivKey)

	// vote meta node2 by initial meta node
	time.Sleep(txBroadcastInterval)
	fmt.Println("#### vote meta node 2 by initial meta node")
	testMetaNodeRegVote(t, senderPrivKey, metaNodeP2PPrivKey2, initialMetaNodeP2PAddr)
	// vote meta node2 by meta node 1
	time.Sleep(txBroadcastInterval)
	fmt.Println("#### vote meta node 2 by meta node 1")
	testMetaNodeRegVote(t, senderPrivKey, metaNodeP2PPrivKey2, fwtypes.P2PAddress(metaNodeP2PPrivKey1.PubKey().Address()))

	// prepay before volume report
	time.Sleep(txBroadcastInterval)
	fmt.Println("### prepay")
	testPrePay(t, senderPrivKey)

	// send volume report tx by meta node 1
	time.Sleep(txBroadcastInterval)
	fmt.Println("#### send volume report tx by meta node 1")
	testVolumeReport(t, senderPrivKey, metaNodeP2PPrivKey1, metaNodeP2PPrivKey2)

}

func getBLSSignBytes(msg *potv1.MsgVolumeReport) ([]byte, error) {
	newMsg := potv1.MsgVolumeReport{
		WalletVolumes:   msg.GetWalletVolumes(),
		Reporter:        msg.GetReporter(),
		Epoch:           msg.GetEpoch(),
		ReportReference: msg.GetReportReference(),
		ReporterOwner:   msg.GetReporterOwner(),
		BLSSignature:    &potv1.BLSSignatureInfo{},
	}
	return proto.Marshal(&newMsg)
}

func testVolumeReport(t *testing.T, senderKey fwcryptotypes.PrivKey, metaP2PPrivKeys ...*fwed25519.PrivKey) {

	fmt.Println("------------------ TestVolumeReport() start ------------------")
	senderAddr := fwtypes.WalletAddress(senderKey.PubKey().Address())
	senderAddrBech32 := fwtypes.WalletAddressBytesToBech32(senderAddr)

	traffic := []*txclienttypes.Traffic{
		{
			Volume:        100000,
			WalletAddress: senderAddrBech32,
		},
	}

	metaP2PAddrs := make([]fwtypes.P2PAddress, len(metaP2PPrivKeys))
	for i, metaP2PPrivKey := range metaP2PPrivKeys {
		metaP2PAddrs[i] = fwtypes.P2PAddress(metaP2PPrivKey.PubKey().Address())
	}

	msg, _, err := tx.BuildVolumeReportMsg(traffic, metaP2PAddrs[0], senderAddr, 1, "report_reference", nil, nil, nil)
	require.NoError(t, err)

	// ------------------------ bls sign start ------------------------
	blsSignBytes, err := getBLSSignBytes(msg)
	require.NoError(t, err)
	blsSignBytesHash := crypto.Keccak256(blsSignBytes)

	var blsSignatures = make([][]byte, len(metaP2PPrivKeys))
	var blsPrivKeys = make([][]byte, len(metaP2PPrivKeys))
	var blsPubKeys = make([][]byte, len(metaP2PPrivKeys))

	for i, privKey := range metaP2PPrivKeys {
		blsPrivKeys[i], blsPubKeys[i], err = bls.NewKeyPairFromBytes(privKey.Bytes())
		require.NoError(t, err)
	}

	for i, blsPrivKey := range blsPrivKeys {
		blsSignatures[i], err = bls.Sign(blsSignBytesHash, blsPrivKey)
		require.NoError(t, err)
	}

	finalBlsSignature, err := bls.AggregateSignatures(blsSignatures...)
	signature := &potv1.BLSSignatureInfo{PubKeys: blsPubKeys, Signature: finalBlsSignature, TxData: blsSignBytesHash}
	msg.BLSSignature = signature
	// ------------------------ bls sign end ------------------------

	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: senderAddr.String(), PrivateKey: senderKey.Bytes(), Type: txclienttypes.SignatureSecp256k1},
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

func testMetaNodeRegVote(t *testing.T, senderKey fwcryptotypes.PrivKey, canP2PKey *fwed25519.PrivKey, voterP2PAddr fwtypes.P2PAddress) {
	fmt.Println("------------------ TestMetaNodeRegVote() start ------------------")
	candidateP2PAddr := fwtypes.P2PAddress(canP2PKey.PubKey().Address())
	senderAddr := fwtypes.WalletAddress(senderKey.PubKey().Address())

	msg := tx.BuildMetaNodeRegistrationVoteMsg(candidateP2PAddr, senderAddr, voterP2PAddr, senderAddr, true)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: senderAddr.String(), PrivateKey: senderKey.Bytes(), Type: txclienttypes.SignatureSecp256k1},
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

func testSend(t *testing.T, senderKey fwcryptotypes.PrivKey) {
	fmt.Println("------------------ TestSend() start ------------------")

	amount := txclienttypes.Coin{Denom: txclienttypes.Wei, Amount: sdkmath.NewInt(100)}
	receiverKey, err := fwsecp256k1.GenerateKey()
	require.NoError(t, err)

	senderAddr := fwtypes.WalletAddress(senderKey.PubKey().Address())
	receiverAddr := fwtypes.WalletAddress(receiverKey.PubKey().Address())

	msg := tx.BuildSendMsg(senderAddr, receiverAddr, amount)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: senderAddr.String(), PrivateKey: senderKey.Bytes(), Type: txclienttypes.SignatureSecp256k1},
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

func testCreateResourceNode(t *testing.T, senderKey fwcryptotypes.PrivKey) {
	fmt.Println("------------------ TestCreateResourceNode() start ------------------")

	p2pKey := fwed25519.GenPrivKey()
	depositAmt := txclienttypes.Coin{
		Denom:  txclienttypes.Wei,
		Amount: sdkmath.NewInt(5e18),
	}
	senderAddr := fwtypes.WalletAddress(senderKey.PubKey().Address())

	msg, err := tx.BuildCreateResourceNodeMsg(txclienttypes.STORAGE, p2pKey.PubKey(), depositAmt, senderAddr)
	require.NoError(t, err)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: senderAddr.String(), PrivateKey: senderKey.Bytes(), Type: txclienttypes.SignatureSecp256k1},
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

func testCreateMetaNode(t *testing.T, senderKey fwcryptotypes.PrivKey) *fwed25519.PrivKey {
	fmt.Println("------------------ TestCreateMetaNode() start ------------------")

	p2pKey := fwed25519.GenPrivKey()
	depositAmt := txclienttypes.Coin{
		Denom:  txclienttypes.Wei,
		Amount: sdkmath.NewInt(5e18),
	}
	senderAddr := fwtypes.WalletAddress(senderKey.PubKey().Address())

	msg, err := tx.BuildCreateMetaNodeMsg(p2pKey.PubKey(), depositAmt, senderAddr, senderAddr)
	require.NoError(t, err)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: senderAddr.String(), PrivateKey: senderKey.Bytes(), Type: txclienttypes.SignatureSecp256k1},
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

	return p2pKey
}

func testPrePay(t *testing.T, senderKey fwcryptotypes.PrivKey) {
	fmt.Println("------------------ TestPrePay() start ------------------")
	senderAddr := fwtypes.WalletAddress(senderKey.PubKey().Address())
	amount := txclienttypes.Coin{
		Denom:  txclienttypes.Wei,
		Amount: sdkmath.NewInt(5e18),
	}
	msg := tx.BuildPrepayMsg(senderAddr, senderAddr, amount)
	signatureKeys := []*txclienttypes.SignatureKey{
		{Address: senderAddr.String(), PrivateKey: senderKey.Bytes(), Type: txclienttypes.SignatureSecp256k1},
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
	senderKeyBytes, err := fwsecp256k1.Derive(senderMnemonic, senderBip39Passphrase, hdPath)
	require.NoError(t, err)

	senderKey := fwsecp256k1.Generate(senderKeyBytes)
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
