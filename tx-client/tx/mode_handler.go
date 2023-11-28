package tx

import (
	"fmt"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"

	authsigning "github.com/stratosnet/sds/tx-client/types/auth/signing"
)

// DefaultSignModes are the default sign modes enabled for protobuf transactions.
var DefaultSignModes = []signingv1beta1.SignMode{
	signingv1beta1.SignMode_SIGN_MODE_DIRECT,
}

// makeSignModeHandler returns the default protobuf SignModeHandler
// SIGN_MODE_DIRECT supported
// SIGN_MODE_LEGACY_AMINO_JSON not supported
func makeSignModeHandler(modes []signingv1beta1.SignMode) authsigning.SignModeHandler {
	if len(modes) < 1 {
		panic(fmt.Errorf("no sign modes enabled"))
	}

	handlers := make([]authsigning.SignModeHandler, len(modes))

	for i, mode := range modes {
		switch mode {
		case signingv1beta1.SignMode_SIGN_MODE_DIRECT:
			handlers[i] = signModeDirectHandler{}
		default:
			panic(fmt.Errorf("unsupported sign mode %+v", mode))
		}
	}

	return authsigning.NewSignModeHandlerMap(
		modes[0],
		handlers,
	)
}
