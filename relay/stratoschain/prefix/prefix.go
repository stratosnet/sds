package prefix

import "github.com/cosmos/cosmos-sdk/types"

var (
	AccountPubKeyPrefix    = "pub"
	ValidatorAddressPrefix = "valoper"
	ValidatorPubKeyPrefix  = "valoperpub"
	ConsNodeAddressPrefix  = "valcons"
	ConsNodePubKeyPrefix   = "valconspub"
)

var sealed = false

func SetConfig(addressPrefix string) {
	if !sealed {
		config := types.GetConfig()
		config.SetBech32PrefixForAccount(addressPrefix, addressPrefix+AccountPubKeyPrefix)
		config.SetBech32PrefixForValidator(addressPrefix+ValidatorAddressPrefix, addressPrefix+ValidatorPubKeyPrefix)
		config.SetBech32PrefixForConsensusNode(addressPrefix+ConsNodeAddressPrefix, addressPrefix+ConsNodePubKeyPrefix)
		config.Seal()

		sealed = true
	}
}
