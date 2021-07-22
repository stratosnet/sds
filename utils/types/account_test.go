package types

import (
	"github.com/stratosnet/sds/relay/stratoschain/prefix"
	"testing"
)

func TestAccountAddressBechConversion(t *testing.T) {
	hrp := "st"
	prefix.SetConfig(hrp)
	addressString := "st1yx3kkx9jnqeck59j744nc5qgtv4lt4dc45jcwz"
	addr, err := BechToAddress(addressString)
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
