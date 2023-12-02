package framework

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stratosnet/sds/framework/crypto/ed25519"
	"github.com/stratosnet/sds/framework/crypto/sha3"
	"github.com/stretchr/testify/require"
)

func TestHex(t *testing.T) {
	addr := ed25519.GenPrivKey().PubKey().Address()

	hexFromAddr := HexFromAddress(addr.Bytes())
	hexFromUtils := hex.EncodeToString(addr.Bytes())
	fmt.Println("hexFromAddr = ", hexFromAddr)
	fmt.Println("hexFromUtils = ", hexFromUtils)
	require.Equal(t, hexFromAddr, hexFromUtils)
}

func HexFromAddress(address []byte) string {
	unchecksummed := hex.EncodeToString(address[:])
	sha := sha3.NewKeccak256()
	sha.Write([]byte(unchecksummed))
	hash := sha.Sum(nil)

	result := []byte(unchecksummed)
	for i := 0; i < len(result); i++ {
		hashByte := hash[i/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if result[i] > '9' && hashByte > 7 {
			result[i] -= 32
		}
	}
	return "0x" + string(result)
}
