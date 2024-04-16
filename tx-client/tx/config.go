package tx

import (
	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"

	authsigning "github.com/stratosnet/sds/tx-client/types/auth/signing"
)

type config struct {
	handler authsigning.SignModeHandler
}

// NewTxConfig returns a new protobuf TxConfig using the provided ProtoCodec and sign modes. The
// first enabled sign mode will become the default sign mode.
// NOTE: Use NewTxConfigWithHandler to provide a custom signing handler in case the sign mode
// is not supported by default (eg: SignMode_SIGN_MODE_EIP_191).
func NewTxConfig(enabledSignModes []signingv1beta1.SignMode) TxConfig {
	return NewTxConfigWithHandler(makeSignModeHandler(enabledSignModes))
}

// NewTxConfig returns a new protobuf TxConfig using the provided ProtoCodec and signing handler.
func NewTxConfigWithHandler(handler authsigning.SignModeHandler) TxConfig {
	return &config{
		handler: handler,
	}
}

func (g config) SignModeHandler() authsigning.SignModeHandler {
	return g.handler
}
