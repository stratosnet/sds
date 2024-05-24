package signing

import (
	"fmt"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
)

// SignModeHandlerMap is SignModeHandler that aggregates multiple SignModeHandler's into
// a single handler
type SignModeHandlerMap struct {
	defaultMode      signingv1beta1.SignMode
	modes            []signingv1beta1.SignMode
	signModeHandlers map[signingv1beta1.SignMode]SignModeHandler
}

var _ SignModeHandler = SignModeHandlerMap{}

// NewSignModeHandlerMap returns a new SignModeHandlerMap with the provided defaultMode and handlers
func NewSignModeHandlerMap(defaultMode signingv1beta1.SignMode, handlers []SignModeHandler) SignModeHandlerMap {
	handlerMap := make(map[signingv1beta1.SignMode]SignModeHandler)
	var modes []signingv1beta1.SignMode

	for _, h := range handlers {
		for _, m := range h.Modes() {
			if _, have := handlerMap[m]; have {
				panic(fmt.Errorf("duplicate sign mode handler for mode %s", m))
			}
			handlerMap[m] = h
			modes = append(modes, m)
		}
	}

	return SignModeHandlerMap{
		defaultMode:      defaultMode,
		modes:            modes,
		signModeHandlers: handlerMap,
	}
}

// DefaultMode implements SignModeHandler.DefaultMode
func (h SignModeHandlerMap) DefaultMode() signingv1beta1.SignMode {
	return h.defaultMode
}

// Modes implements SignModeHandler.Modes
func (h SignModeHandlerMap) Modes() []signingv1beta1.SignMode {
	return h.modes
}

// DefaultMode implements SignModeHandler.GetSignBytes
func (h SignModeHandlerMap) GetSignBytes(mode signingv1beta1.SignMode, data SignerData, tx *txv1beta1.Tx) ([]byte, error) {
	handler, found := h.signModeHandlers[mode]
	if !found {
		return nil, fmt.Errorf("can't verify sign mode %s", mode.String())
	}
	return handler.GetSignBytes(mode, data, tx)
}
