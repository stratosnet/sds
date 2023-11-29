package tx

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"

	authsigning "github.com/stratosnet/sds/tx-client/types/auth/signing"
)

// signModeDirectHandler defines the SIGN_MODE_DIRECT SignModeHandler
type signModeDirectHandler struct{}

var _ authsigning.SignModeHandler = signModeDirectHandler{}

// DefaultMode implements SignModeHandler.DefaultMode
func (signModeDirectHandler) DefaultMode() signingv1beta1.SignMode {
	return signingv1beta1.SignMode_SIGN_MODE_DIRECT
}

// Modes implements SignModeHandler.Modes
func (signModeDirectHandler) Modes() []signingv1beta1.SignMode {
	return []signingv1beta1.SignMode{signingv1beta1.SignMode_SIGN_MODE_DIRECT}
}

// GetSignBytes implements SignModeHandler.GetSignBytes
func (signModeDirectHandler) GetSignBytes(mode signingv1beta1.SignMode, data authsigning.SignerData, tx *txv1beta1.Tx) ([]byte, error) {
	if mode != signingv1beta1.SignMode_SIGN_MODE_DIRECT {
		return nil, fmt.Errorf("expected %s, got %s", signingv1beta1.SignMode_SIGN_MODE_DIRECT, mode)
	}

	txBodyToSign := &txv1beta1.TxBody{
		Messages: tx.GetBody().GetMessages(),
		Memo:     tx.GetBody().GetMemo(),
	}

	bodyBz, err := proto.Marshal(txBodyToSign)
	if err != nil {
		return nil, err
	}

	authInfoBz, err := proto.Marshal(tx.GetAuthInfo())
	if err != nil {
		return nil, err
	}

	return DirectSignBytes(bodyBz, authInfoBz, data.ChainID, data.AccountNumber)
}

// DirectSignBytes returns the SIGN_MODE_DIRECT sign bytes for the provided TxBody bytes, AuthInfo bytes, chain ID,
// account number and sequence.
func DirectSignBytes(bodyBytes, authInfoBytes []byte, chainID string, accnum uint64) ([]byte, error) {
	signDoc := &txv1beta1.SignDoc{
		BodyBytes:     bodyBytes,
		AuthInfoBytes: authInfoBytes,
		ChainId:       chainID,
		AccountNumber: accnum,
	}
	signDocBz, err := proto.Marshal(signDoc)
	if err != nil {
		return nil, err
	}
	return signDocBz, nil
}
