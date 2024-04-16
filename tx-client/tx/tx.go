package tx

import (
	"fmt"

	"github.com/cosmos/cosmos-proto/anyutil"
	"google.golang.org/protobuf/types/known/anypb"

	sdked25519 "cosmossdk.io/api/cosmos/crypto/ed25519"
	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
	sdksecp256k1 "github.com/stratosnet/stratos-chain/api/stratos/crypto/v1/ethsecp256k1"

	fwed25519 "github.com/stratosnet/sds/framework/crypto/ed25519"
	fwsecp256k1 "github.com/stratosnet/sds/framework/crypto/secp256k1"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"

	authsigning "github.com/stratosnet/sds/tx-client/types/auth/signing"
	txsigning "github.com/stratosnet/sds/tx-client/types/tx/signing"
)

// SignWithPrivKey signs a given tx with the given private key, and returns the
// corresponding SignatureV2 if the signing is successful.
func SignWithPrivKey(
	signMode signingv1beta1.SignMode, signerData authsigning.SignerData,
	tx *txv1beta1.Tx, priv fwcryptotypes.PrivKey, txConfig TxConfig,
	accSeq uint64,
) (txsigning.SignatureV2, error) {
	var sigV2 txsigning.SignatureV2

	// Generate the bytes to be signed.
	signBytes, err := txConfig.SignModeHandler().GetSignBytes(signMode, signerData, tx)
	if err != nil {
		return sigV2, err
	}

	// Sign those bytes
	signature, err := priv.Sign(signBytes)
	if err != nil {
		return sigV2, err
	}

	// Construct the SignatureV2 struct
	sigData := txsigning.SingleSignatureData{
		SignMode:  signMode,
		Signature: signature,
	}

	pubKeyAny, err := getPackedPubKeyAnyByPrivKey(priv)
	if err != nil {
		return sigV2, err
	}

	sigV2 = txsigning.SignatureV2{
		PubKey:   pubKeyAny,
		Data:     &sigData,
		Sequence: accSeq,
	}

	return sigV2, nil
}

func getPackedPubKeyAnyByPrivKey(priv fwcryptotypes.PrivKey) (pubKeyAny *anypb.Any, err error) {
	switch priv.Type() {
	case fwsecp256k1.KeyType:
		pubKey := &sdksecp256k1.PubKey{Key: priv.PubKey().Bytes()}
		pubKeyAny, err = anyutil.New(pubKey)
	case fwed25519.KeyType:
		pubKey := &sdked25519.PubKey{Key: priv.PubKey().Bytes()}
		pubKeyAny, err = anyutil.New(pubKey)
	default:
		return nil, fmt.Errorf("Key type is not supported. ")
	}
	if err != nil {
		return nil, err
	}

	return pubKeyAny, nil
}
