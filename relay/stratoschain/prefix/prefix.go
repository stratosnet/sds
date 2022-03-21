package prefix

import (
	"github.com/stratosnet/stratos-chain/types"
)

var sealed = false

func init() {
	setConfig()
}

func setConfig() {
	if !sealed {
		config := types.GetConfig()
		config.Seal()
		sealed = true
	}
}
