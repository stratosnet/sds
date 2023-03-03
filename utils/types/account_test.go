package types

import (
	"testing"

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

func TestAccountAddressBechConversion(t *testing.T) {
	hrp := "st"

	addressString := "st1yx3kkx9jnqeck59j744nc5qgtv4lt4dc45jcwz"
	addr, err := WalletAddressFromBech(addressString)
	if err != nil {
		t.Fatal("couldn't convert bech32 string to Address: " + err.Error())
	}

	addressString2, err := addr.ToBech(hrp)
	if err != nil {
		t.Fatal("couldn't convert Address to bech32 string: " + err.Error())
	}

	if addressString != addressString2 {
		t.Fatalf("the bech32 address conversion is broken. Expected [%v] Actual [%v]", addressString, addressString2)
	}
}
