package tx

import (
	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"

	"github.com/stratosnet/sds/tx-client/types/auth/signing"
)

type (

	// TxConfig defines an interface a client can utilize to generate an
	// application-defined concrete transaction type. The type returned must
	// implement TxBuilder.
	TxConfig interface {
		SignModeHandler() signing.SignModeHandler
	}
)

func CreateTxConfigAndTxBuilder() (TxConfig, *txv1beta1.Tx) {
	txConfig := NewTxConfig([]signingv1beta1.SignMode{signingv1beta1.SignMode_SIGN_MODE_DIRECT})
	tx := &txv1beta1.Tx{}
	return txConfig, tx
}
