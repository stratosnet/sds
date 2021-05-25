package stratoschain

import (
	"github.com/cosmos/cosmos-sdk/types"
)

var (
	AccountPubKeyPrefix    = "pub"
	ValidatorAddressPrefix = "valoper"
	ValidatorPubKeyPrefix  = "valoperpub"
	ConsNodeAddressPrefix  = "valcons"
	ConsNodePubKeyPrefix   = "valconspub"
)

func SetConfig(addressPrefix string) {
	config := types.GetConfig()
	config.SetBech32PrefixForAccount(addressPrefix, addressPrefix + AccountPubKeyPrefix)
	config.SetBech32PrefixForValidator(addressPrefix + ValidatorAddressPrefix, addressPrefix + ValidatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(addressPrefix + ConsNodeAddressPrefix, addressPrefix + ConsNodePubKeyPrefix)
	config.Seal()
}
