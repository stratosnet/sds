package types

import (
	"fmt"
	"github.com/stratosnet/sds/relay/stratoschain"
	"io/ioutil"
	"math/rand"
	"testing"
)

func TestAccountAddressBechConversion(t *testing.T) {
	hrp := "st"
	stratoschain.SetConfig(hrp)
	addressString := "st1yx3kkx9jnqeck59j744nc5qgtv4lt4dc45jcwz"
	addr, err := BechToAddress(addressString)
	if err != nil {
		t.Fatal("couldn't convert bech32 string to Address: " + err.Error())
	}

	addressString2 := addr.ToBech()

	if addressString != addressString2 {
		t.Fatalf("the bech32 address conversion is broken. Expected [%v] Actual [%v]", addressString, addressString2)
	}
}

func TestFake(t *testing.T) {
	for i := 0; i < 10; i++ {
		rand.Seed(int64(i))
		data := make([]byte, 1024*1024*8)
		rand.Read(data)
		ioutil.WriteFile(fmt.Sprintf("tmp%v.dat", i), data, 0777)
	}
}
