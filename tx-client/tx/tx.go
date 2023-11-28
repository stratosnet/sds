package tx

import (
	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"

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

	sigV2 = txsigning.SignatureV2{
		PubKey:   priv.PubKey(),
		Data:     &sigData,
		Sequence: accSeq,
	}

	return sigV2, nil
}
