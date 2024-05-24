package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/stratosnet/sds/framework/crypto/sha3"
	"github.com/stratosnet/sds/framework/types/bech32"
)

// Bech32 conversion constants
const (
	// MaxAddrLen is the maximum allowed length (in bytes) for an address.
	MaxAddrLen = 255

	StratosBech32Prefix = "st"
	// PrefixPublic is the prefix for public keys
	PrefixPublic = "pub"
	// PrefixSds is the prefix for sds keys
	PrefixSds = "sds"

	// WalletAddressPrefix defines the Bech32 prefix of an account's address (st)
	WalletAddressPrefix = StratosBech32Prefix
	// WalletPubKeyPrefix defines the Bech32 prefix of an account's public key (stpub)
	WalletPubKeyPrefix = StratosBech32Prefix + PrefixPublic
	// P2PPubkeyPrefix defines the Bech32 prefix of an sds account's public key (stsdspub)
	P2PPubkeyPrefix = StratosBech32Prefix + PrefixSds + PrefixPublic
	// P2PAddressPrefix defines the Bech32 prefix of an sds account's address (stsds)
	P2PAddressPrefix = StratosBech32Prefix + PrefixSds
)

var (
	_ Address = P2PAddress{}
	_ Address = WalletAddress{}

	ErrEmptyAddress = errors.New("address is empty")
)

// Address is a common interface for different types of addresses used by the SDK
type Address interface {
	Equals(Address) bool
	Empty() bool
	Marshal() ([]byte, error)
	MarshalJSON() ([]byte, error)
	Bytes() []byte
	String() string
	Format(s fmt.State, verb rune)
	Hex() string
}

// ----------------------------------------------------------------------------
// account
// ----------------------------------------------------------------------------

// WalletAddress a wrapper around bytes meant to represent an account address.
// When marshaled to a string or JSON, it uses Bech32.
type WalletAddress []byte

func WalletAddressFromHex(s string) WalletAddress {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	bz, _ := hex.DecodeString(s)
	return WalletAddress(bz)
}

func WalletAddressBytesToBech32(addr []byte) string {
	return addrBytesToBech32(addr, WalletAddressPrefix)
}

// WalletAddressFromBech32 creates an WalletAddress from a Bech32 string.
func WalletAddressFromBech32(address string) (addr WalletAddress, err error) {
	if len(strings.TrimSpace(address)) == 0 {
		return WalletAddress{}, ErrEmptyAddress
	}

	bz, err := GetFromBech32(address, WalletAddressPrefix)
	if err != nil {
		return nil, err
	}

	err = VerifyAddressFormat(bz)
	if err != nil {
		return nil, err
	}

	return WalletAddress(bz), nil
}

// Returns boolean for whether two AccAddresses are Equal
func (aa WalletAddress) Equals(aa2 Address) bool {
	if aa.Empty() && aa2.Empty() {
		return true
	}

	return bytes.Equal(aa.Bytes(), aa2.Bytes())
}

// Returns boolean for whether an WalletAddress is empty
func (aa WalletAddress) Empty() bool {
	return len(aa) == 0
}

// Marshal returns the raw address bytes. It is needed for protobuf
// compatibility.
func (aa WalletAddress) Marshal() ([]byte, error) {
	return aa, nil
}

// Unmarshal sets the address to the given data. It is needed for protobuf
// compatibility.
func (aa *WalletAddress) Unmarshal(data []byte) error {
	*aa = data
	return nil
}

// MarshalJSON marshals to JSON using Bech32.
func (aa WalletAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(aa.String())
}

// MarshalYAML marshals to YAML using Bech32.
func (aa WalletAddress) MarshalYAML() (interface{}, error) {
	return aa.String(), nil
}

// UnmarshalJSON unmarshals from JSON assuming Bech32 encoding.
func (aa *WalletAddress) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if s == "" {
		*aa = WalletAddress{}
		return nil
	}

	aa2, err := WalletAddressFromBech32(s)
	if err != nil {
		return err
	}

	*aa = aa2
	return nil
}

// UnmarshalYAML unmarshals from JSON assuming Bech32 encoding.
func (aa *WalletAddress) UnmarshalYAML(data []byte) error {
	var s string
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if s == "" {
		*aa = WalletAddress{}
		return nil
	}

	aa2, err := WalletAddressFromBech32(s)
	if err != nil {
		return err
	}

	*aa = aa2
	return nil
}

// Bytes returns the raw address bytes.
func (aa WalletAddress) Bytes() []byte {
	return aa
}

// String implements the Stringer interface.
func (aa WalletAddress) String() string {
	if aa.Empty() {
		return ""
	}
	bech32Addr, err := bech32.ConvertAndEncode(WalletAddressPrefix, aa.Bytes())
	if err != nil {
		return ""
	}

	return bech32Addr
}

// Format implements the fmt.Formatter interface.

func (aa WalletAddress) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		s.Write([]byte(aa.String()))
	case 'p':
		s.Write([]byte(fmt.Sprintf("%p", aa)))
	default:
		s.Write([]byte(fmt.Sprintf("%X", []byte(aa))))
	}
}

func (aa WalletAddress) Hex() string {
	unchecksummed := hex.EncodeToString(aa[:])
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

// ----------------------------------------------------------------------------
// P2PAddress
// ----------------------------------------------------------------------------

type P2PAddress []byte

func P2PAddressFromHex(s string) P2PAddress {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	bz, _ := hex.DecodeString(s)
	return P2PAddress(bz)
}

func P2PAddressBytesToBech32(addr []byte) string {
	return addrBytesToBech32(addr, P2PAddressPrefix)
}

// P2PAddressFromBech32 creates an P2PAddress from a Bech32 string.
func P2PAddressFromBech32(address string) (addr P2PAddress, err error) {
	if len(strings.TrimSpace(address)) == 0 {
		return P2PAddress{}, ErrEmptyAddress
	}

	bz, err := GetFromBech32(address, P2PAddressPrefix)
	if err != nil {
		return nil, err
	}

	err = VerifyAddressFormat(bz)
	if err != nil {
		return nil, err
	}

	return P2PAddress(bz), nil
}

// Equals Returns boolean for whether two P2PAddress are Equal
func (aa P2PAddress) Equals(aa2 Address) bool {
	if aa.Empty() && aa2.Empty() {
		return true
	}

	return bytes.Equal(aa.Bytes(), aa2.Bytes())
}

// Empty Returns boolean for whether a P2PAddress is empty
func (aa P2PAddress) Empty() bool {
	return aa == nil || len(aa) == 0
}

// Marshal returns the raw address bytes. It is needed for protobuf
// compatibility.
func (aa P2PAddress) Marshal() ([]byte, error) {
	return aa, nil
}

// Unmarshal sets the address to the given data. It is needed for protobuf
// compatibility.
func (aa *P2PAddress) Unmarshal(data []byte) error {
	*aa = data
	return nil
}

// MarshalJSON marshals to JSON using Bech32.
func (aa P2PAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(aa.String())
}

// MarshalYAML marshals to YAML using Bech32.
func (aa P2PAddress) MarshalYAML() (interface{}, error) {
	return aa.String(), nil
}

// UnmarshalJSON unmarshals from JSON assuming Bech32 encoding.
func (aa *P2PAddress) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)

	if err != nil {
		return err
	}
	if s == "" {
		*aa = P2PAddress{}
		return nil
	}

	aa2, err := P2PAddressFromBech32(s)
	if err != nil {
		return err
	}

	*aa = aa2
	return nil
}

// Bytes returns the raw address bytes.
func (aa P2PAddress) Bytes() []byte {
	return aa
}

// String implements the Stringer interface.
func (aa P2PAddress) String() string {
	if aa.Empty() {
		return ""
	}
	bech32Addr, err := bech32.ConvertAndEncode(P2PAddressPrefix, aa.Bytes())
	if err != nil {
		return ""
	}

	return bech32Addr
}

// Format implements the fmt.Formatter interface.
func (aa P2PAddress) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		s.Write([]byte(aa.String()))
	case 'p':
		s.Write([]byte(fmt.Sprintf("%p", aa)))
	default:
		s.Write([]byte(fmt.Sprintf("%X", []byte(aa))))
	}
}

func (aa P2PAddress) Hex() string {
	unchecksummed := hex.EncodeToString(aa[:])
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

//--------------------------------------------------------------------

func addrBytesToBech32(addr []byte, addrPrefix string) string {
	if addr == nil || len(addr) == 0 {
		return ""
	}
	bech32Addr, err := bech32.ConvertAndEncode(addrPrefix, addr)
	if err != nil {
		return ""
	}

	return bech32Addr
}

// GetFromBech32 decodes a bytestring from a Bech32 encoded string.
func GetFromBech32(bech32str, prefix string) ([]byte, error) {
	if len(bech32str) == 0 {
		return nil, fmt.Errorf("decoding Bech32 address failed: must provide a non empty address")
	}

	hrp, bz, err := bech32.DecodeAndConvert(bech32str)
	if err != nil {
		return nil, err
	}

	if hrp != prefix {
		return nil, fmt.Errorf("invalid Bech32 prefix; expected %s, got %s", prefix, hrp)
	}

	return bz, nil
}

// VerifyAddressFormat verifies that the provided bytes form a valid address
// according to the default address rules or a custom address verifier set by
// GetConfig().SetAddressVerifier().
// TODO make an issue to get rid of global Config
// ref: https://github.com/cosmos/cosmos-sdk/issues/9690
func VerifyAddressFormat(bz []byte) error {
	//verifier := GetConfig().GetAddressVerifier()
	//if verifier != nil {
	//	return verifier(bz)
	//}

	if len(bz) == 0 {
		return fmt.Errorf("addresses cannot be empty")
	}

	if len(bz) > MaxAddrLen {
		return fmt.Errorf("address max length is %d, got %d", MaxAddrLen, len(bz))
	}

	return nil
}
