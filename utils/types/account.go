package types

import (
	"bytes"
	ed25519crypto "crypto/ed25519"
	"encoding/hex"
	"fmt"
	"github.com/tendermint/tendermint/crypto"
	"math/big"
	"strings"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/stratosnet/sds/utils/crypto/sha3"
	"github.com/stratosnet/stratos-chain/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

// Lengths of hashes and addresses in bytes.
const (
	// HashLength is the expected length of the hash
	HashLength = 32
	// AddressLength
	AddressLength = 20
	// AccPublicKeyLength
	AccPublicKeyLength = 33
	// AccPrivateKeyLength
	AccPrivateKeyLength = 32
	// P2pPublicKeyLength
	P2pPublicKeyLength = 32
	// P2pPrivateKeyLength
	P2pPrivateKeyLength = 64
)

// Address
type Address [AddressLength]byte

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

// AccPubKey account(wallet) public key
type AccPubKey [AccPublicKeyLength]byte

// AccPrivKey account(wallet) private key
type AccPrivKey [AccPrivateKeyLength]byte

// P2pPubKey P2P address public key
type P2pPubKey [P2pPublicKeyLength]byte

// P2pPrivKey P2P address private key
type P2pPrivKey [P2pPrivateKeyLength]byte

// BytesToAddress returns Address with value b.
// If b is larger than len(h), b will be cropped from the left.
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// BytesToPubKey sets b to PubKey.
// If b is larger than len(h), b will be cropped from the left.
func BytesToAccPubKey(b []byte) AccPubKey {
	var p AccPubKey
	p.SetBytes(b)
	return p
}

// BytesToAccPriveKey sets b to PrivKey.
// If b is larger than len(h), b will be cropped from the left.
func BytesToAccPriveKey(b []byte) AccPrivKey {
	var p AccPrivKey
	p.SetBytes(b)
	return p
}

// BytesToP2pPubKey sets b to P2pPubKey
// If b is larger than len(h), b will be cropped from the left.
func BytesToP2pPubKey(b []byte) P2pPubKey {
	var p P2pPubKey
	p.SetBytes(b)
	return p
}

// BytesToAccPriveKey sets b to P2pPrivKey.
// If b is larger than len(h), b will be cropped from the left.
func BytesToP2pPrivKey(b []byte) P2pPrivKey {
	var p P2pPrivKey
	p.SetBytes(b)
	return p
}

// Bytes gets the byte representation of the underlying hash.
func (h Hash) Bytes() []byte { return h[:] }

// Big converts a hash to a big integer.
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex converts a hash to a hex string.
func (h Hash) Hex() string { return Encode(h[:]) }

// Encode encodes b as a hex string with 0x prefix.
func Encode(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x…%x", h[:3], h[29:])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()
}

func (h Hash) Float64() float64 {
	b, ok := new(big.Float).SetString(h.String())
	if ok {
		s, _ := b.Float64()
		return s
	}
	return 0
}

// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
func (h Hash) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), h[:])
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// SetBytes sets the address to the value of b.
// If b is larger than len(a) it will panic.
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

// Bytes gets the string representation of the underlying address.
func (a Address) Bytes() []byte { return a[:] }

// Big converts an address to a big integer.
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }

// Hash converts an address to a hash by left-padding it with zeros.
func (a Address) Hash() Hash { return BytesToHash(a[:]) }

// Hex returns an EIP55-compliant hex string representation of the address.
func (a Address) Hex() string {
	unchecksummed := hex.EncodeToString(a[:])
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

// String implements fmt.Stringer.
func (a Address) String() string {
	return a.Hex()
}

func (a Address) ToBech(hrp string) (string, error) {
	return bech32.ConvertAndEncode(hrp, a.Bytes())
}

func (a Address) WalletAddressToBech() (string, error) {
	return a.ToBech(types.AccountAddressPrefix)
}

func (a Address) P2pAddressToBech() (string, error) {
	return a.ToBech(types.SdsNodeP2PAddressPrefix)
}

func (a Address) P2pPublicKeyToBech() (string, error) {
	return a.ToBech(types.SdsNodeP2PPubkeyPrefix)
}

func (a Address) Compare(b Address) int {
	return bytes.Compare(a.Bytes(), b.Bytes())
}

func WalletAddressFromBech(str string) (Address, error) {
	addr, err := sdktypes.GetFromBech32(str, types.AccountAddressPrefix)
	if err != nil {
		return Address{}, err
	}
	err = sdktypes.VerifyAddressFormat(addr)
	if err != nil {
		return Address{}, err
	}
	return BytesToAddress(addr), err
}

func P2pAddressFromBech(str string) (Address, error) {
	if len(strings.TrimSpace(str)) == 0 {
		return Address{}, nil
	}
	bz, err := sdktypes.GetFromBech32(str, types.SdsNodeP2PAddressPrefix)
	if err != nil {
		return Address{}, err
	}
	err = sdktypes.VerifyAddressFormat(bz)
	if err != nil {
		return Address{}, err
	}
	return BytesToAddress(bz), nil
}

// Bytes2Hex returns the hexadecimal encoding of d.
func Bytes2Hex(d []byte) string {
	return hex.EncodeToString(d)
}

// Hex2Bytes returns the bytes represented by the hexadecimal string str.
func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}

// FromHex returns the bytes represented by the hexadecimal string s.
// s may be prefixed with "0x".
func FromHex(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return Hex2Bytes(s)
}

// BigToAddress returns Address with byte values of b.
// If b is larger than len(h), b will be cropped from the left.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// HexToAddress returns Address with byte values of s.
// If s is larger than len(h), s will be cropped from the left.
func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

// HexToHash sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) Hash { return BytesToHash(FromHex(s)) }

// IsHexAddress verifies whether a string can represent a valid hex-encoded
// Ethereum address or not.
func IsHexAddress(s string) bool {
	if hasHexPrefix(s) {
		s = s[2:]
	}
	return len(s) == 2*AddressLength && isHex(s)
}

// hasHexPrefix validates str begins with '0x' or '0X'.
func hasHexPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// isHexCharacter returns bool of c being a valid hexadecimal.
func isHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// isHex validates whether each byte is valid hexadecimal string.
func isHex(str string) bool {
	if len(str)%2 != 0 {
		return false
	}
	for _, c := range []byte(str) {
		if !isHexCharacter(c) {
			return false
		}
	}
	return true
}

// Bytes gets the byte representation of the underlying hash.
func (p AccPubKey) Bytes() []byte { return p[:] }

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (p *AccPubKey) SetBytes(b []byte) {
	if len(b) > len(p) {
		b = b[len(b)-AccPublicKeyLength:]
	}
	copy(p[AccPublicKeyLength-len(b):], b)
}

func (p AccPubKey) ToBech() (string, error) {
	return bech32.ConvertAndEncode(types.AccountPubKeyPrefix, p.Bytes())
}

// Address generate a wallet address from account public key
func (p AccPubKey) Address() Address {
	pk := secp256k1.PubKey{Key: p.Bytes()}
	return BytesToAddress(pk.Address().Bytes())
}

// WalletPubkeyFromBech create an AccPubKey from Bech of wallet pubkey
func WalletPubkeyFromBech(str string) (AccPubKey, error) {
	pubkey, err := sdktypes.GetFromBech32(str, types.AccountPubKeyPrefix)
	if err != nil {
		fmt.Println(err)
		return AccPubKey{}, err
	}
	err = sdktypes.VerifyAddressFormat(pubkey)
	if err != nil {
		return AccPubKey{}, err
	}
	return BytesToAccPubKey(pubkey), err
}

// VerifyWalletAddr verify the wallet address and public key match
func VerifyWalletAddr(walletPubkey, walletAddr string) int {
	pk, err := WalletPubkeyFromBech(walletPubkey)
	if err != nil {
		return -1
	}
	address := pk.Address()
	address2, err := WalletAddressFromBech(walletAddr)
	if err != nil {
		return -1
	}
	return address.Compare(address2)
}

// VerifyWalletAddrBytes []byte version of VerifyWalletKey() for the pubkey format
func VerifyWalletAddrBytes(walletPubkey []byte, walletAddr string) int {
	pk := BytesToAccPubKey(walletPubkey)
	address := pk.Address()
	address2, err := WalletAddressFromBech(walletAddr)
	if err != nil {
		return -1
	}
	return address.Compare(address2)
}

// VerifyWalletSign verify the signature by wallet key
func VerifyWalletSign(walletPubkey, signature, message string) bool {
	ds, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	pk, err := WalletPubkeyFromBech(walletPubkey)
	if err != nil {
		return false
	}
	pubkey := secp256k1.PubKey{Key: pk.Bytes()}
	return pubkey.VerifySignature([]byte(message), ds)
}

// VerifyWalletSignBytes []byte version of VerifyWalletKey() for pubkey and signature format
func VerifyWalletSignBytes(walletPubkey []byte, signature []byte, message string) bool {
	pubkey := secp256k1.PubKey{Key: walletPubkey}
	return pubkey.VerifySignature([]byte(message), signature)
}

// Bytes gets the byte representation of the underlying hash.
func (p AccPrivKey) Bytes() []byte { return p[:] }

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (p *AccPrivKey) SetBytes(b []byte) {
	if len(b) > len(p) {
		b = b[len(b)-AccPrivateKeyLength:]
	}
	copy(p[AccPrivateKeyLength-len(b):], b)
}

// Sign secp256k1 sign since account(wallet) uses secp256k1 key pair
func (p AccPrivKey) Sign(b []byte) ([]byte, error) {
	pk := secp256k1.PrivKey{Key: p.Bytes()}
	return pk.Sign(b)
}

// PubKeyFromPrivKey generate a AccPubKey from the AccPrivKey
func (p AccPrivKey) PubKeyFromPrivKey() AccPubKey {
	return BytesToAccPubKey(hd.Secp256k1.Generate()(p.Bytes()).PubKey().Bytes())
}

// VerifyP2pAddrBytes verify whether P2P address matches public key
func VerifyP2pAddrBytes(p2pPubkey []byte, p2pAddr string) bool {
	pk := BytesToP2pPubKey(p2pPubkey)
	address := pk.Address()
	address2, err := P2pAddressFromBech(p2pAddr)
	if err != nil {
		return false
	}
	return (address.Compare(address2) == 0)
}

// VerifyP2pSignBytes verify the signature made by P2P key
func VerifyP2pSignBytes(p2pPubkey []byte, signature []byte, message string) bool {
	pk := ed25519.PubKey(p2pPubkey)
	return pk.VerifySignature([]byte(message), signature)
}

// Bytes gets the byte representation of the underlying hash.
func (p P2pPubKey) Bytes() []byte { return p[:] }

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (p *P2pPubKey) SetBytes(b []byte) {
	if len(b) > len(p) {
		b = b[len(b)-P2pPublicKeyLength:]
	}
	copy(p[P2pPublicKeyLength-len(b):], b)
}

// ToBech the Bech format string
func (p P2pPubKey) ToBech() (string, error) {
	return bech32.ConvertAndEncode(types.SdsNodeP2PPubkeyPrefix, p.Bytes())
}

// Address generate a P2P address from P2P public key
func (p P2pPubKey) Address() Address {
	pk := ed25519.PubKey(p.Bytes()).Address()
	return BytesToAddress(pk)
}

// Bytes gets the byte representation of the underlying hash.
func (p P2pPrivKey) Bytes() []byte { return p[:] }

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (p *P2pPrivKey) SetBytes(b []byte) {
	if len(b) > len(p) {
		b = b[len(b)-P2pPrivateKeyLength:]
	}
	copy(p[P2pPrivateKeyLength-len(b):], b)
}

// Sign p2p address uses ed25519 algo
func (p P2pPrivKey) Sign(b []byte) []byte {
	return ed25519crypto.Sign(p.Bytes(), b)
}

// PubKey generate a P2pPubKey from a P2pPrivKey
func (p P2pPrivKey) PubKey() P2pPubKey {
	return BytesToP2pPubKey(crypto.PrivKey(ed25519.PrivKey(p.Bytes())).PubKey().Bytes())
}
