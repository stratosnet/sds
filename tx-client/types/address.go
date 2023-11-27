package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/stratosnet/tx-client/crypto/ed25519"
	cryptotypes "github.com/stratosnet/tx-client/crypto/types"
	"github.com/stratosnet/tx-client/types/bech32"
)

// Bech32 conversion constants
const (
	// MaxAddrLen is the maximum allowed length (in bytes) for an address.
	MaxAddrLen = 255

	StratosBech32Prefix = "st"

	// PrefixAccount is the prefix for account keys
	PrefixAccount = "acc"
	// PrefixValidator is the prefix for validator keys
	PrefixValidator = "val"
	// PrefixConsensus is the prefix for consensus keys
	PrefixConsensus = "cons"
	// PrefixPublic is the prefix for public keys
	PrefixPublic = "pub"
	// PrefixOperator is the prefix for operator keys
	PrefixOperator = "oper"
	// PrefixSds is the prefix for sds keys
	PrefixSds = "sds"

	// AccountAddressPrefix defines the Bech32 prefix of an account's address (st)
	AccountAddressPrefix = StratosBech32Prefix
	// AccountPubKeyPrefix defines the Bech32 prefix of an account's public key (stpub)
	AccountPubKeyPrefix = StratosBech32Prefix + PrefixPublic
	// ValidatorAddressPrefix defines the Bech32 prefix of a validator's operator address (stvaloper)
	ValidatorAddressPrefix = StratosBech32Prefix + PrefixValidator + PrefixOperator
	// ValidatorPubKeyPrefix defines the Bech32 prefix of a validator's operator public key (stvaloperpub)
	ValidatorPubKeyPrefix = StratosBech32Prefix + PrefixValidator + PrefixOperator + PrefixPublic
	// ConsNodeAddressPrefix defines the Bech32 prefix of a consensus node address (stvalcons)
	ConsNodeAddressPrefix = StratosBech32Prefix + PrefixValidator + PrefixConsensus
	// ConsNodePubKeyPrefix defines the Bech32 prefix of a consensus node public key (stvalconspub)
	ConsNodePubKeyPrefix = StratosBech32Prefix + PrefixValidator + PrefixConsensus + PrefixPublic
	// SdsNodeP2PPubkeyPrefix defines the Bech32 prefix of an sds account's public key (stsdspub)
	SdsNodeP2PPubkeyPrefix = StratosBech32Prefix + PrefixSds + PrefixPublic
	// SdsNodeP2PAddressPrefix defines the Bech32 prefix of an sds account's address (stsds)
	SdsNodeP2PAddressPrefix = StratosBech32Prefix + PrefixSds
)

var (
	_ Address = SdsAddress{}
	_ Address = AccAddress{}
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
}

// ----------------------------------------------------------------------------
// account
// ----------------------------------------------------------------------------

// AccAddress a wrapper around bytes meant to represent an account address.
// When marshaled to a string or JSON, it uses Bech32.
type AccAddress []byte

func AccAddressBytesToBech32(addr []byte) string {
	return addrBytesToBech32(addr, AccountAddressPrefix)
}

// AccAddressFromBech32 creates an AccAddress from a Bech32 string.
func AccAddressFromBech32(address string) (addr AccAddress, err error) {
	if len(strings.TrimSpace(address)) == 0 {
		return AccAddress{}, fmt.Errorf("empty address string is not allowed")
	}

	bz, err := GetFromBech32(address, AccountAddressPrefix)
	if err != nil {
		return nil, err
	}

	err = VerifyAddressFormat(bz)
	if err != nil {
		return nil, err
	}

	return AccAddress(bz), nil
}

// Returns boolean for whether two AccAddresses are Equal
func (aa AccAddress) Equals(aa2 Address) bool {
	if aa.Empty() && aa2.Empty() {
		return true
	}

	return bytes.Equal(aa.Bytes(), aa2.Bytes())
}

// Returns boolean for whether an AccAddress is empty
func (aa AccAddress) Empty() bool {
	return len(aa) == 0
}

// Marshal returns the raw address bytes. It is needed for protobuf
// compatibility.
func (aa AccAddress) Marshal() ([]byte, error) {
	return aa, nil
}

// Unmarshal sets the address to the given data. It is needed for protobuf
// compatibility.
func (aa *AccAddress) Unmarshal(data []byte) error {
	*aa = data
	return nil
}

// MarshalJSON marshals to JSON using Bech32.
func (aa AccAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(aa.String())
}

// MarshalYAML marshals to YAML using Bech32.
func (aa AccAddress) MarshalYAML() (interface{}, error) {
	return aa.String(), nil
}

// UnmarshalJSON unmarshals from JSON assuming Bech32 encoding.
func (aa *AccAddress) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if s == "" {
		*aa = AccAddress{}
		return nil
	}

	aa2, err := AccAddressFromBech32(s)
	if err != nil {
		return err
	}

	*aa = aa2
	return nil
}

// UnmarshalYAML unmarshals from JSON assuming Bech32 encoding.
func (aa *AccAddress) UnmarshalYAML(data []byte) error {
	var s string
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if s == "" {
		*aa = AccAddress{}
		return nil
	}

	aa2, err := AccAddressFromBech32(s)
	if err != nil {
		return err
	}

	*aa = aa2
	return nil
}

// Bytes returns the raw address bytes.
func (aa AccAddress) Bytes() []byte {
	return aa
}

// String implements the Stringer interface.
func (aa AccAddress) String() string {
	if aa.Empty() {
		return ""
	}
	bech32Addr, err := bech32.ConvertAndEncode(AccountAddressPrefix, aa.Bytes())
	if err != nil {
		panic(err)
	}

	return bech32Addr
}

// Format implements the fmt.Formatter interface.

func (aa AccAddress) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		s.Write([]byte(aa.String()))
	case 'p':
		s.Write([]byte(fmt.Sprintf("%p", aa)))
	default:
		s.Write([]byte(fmt.Sprintf("%X", []byte(aa))))
	}
}

// ----------------------------------------------------------------------------
// SdsAddress
// ----------------------------------------------------------------------------

type SdsAddress []byte

// SdsPubKeyFromBech32 returns an ed25519 SdsPublicKey from a Bech32 string.
func SdsPubKeyFromBech32(pubkeyStr string) (cryptotypes.PubKey, error) {
	_, sdsPubKeyBytes, err := bech32.DecodeAndConvert(pubkeyStr)
	if err != nil {
		return nil, err
	}
	pubKey := ed25519.PubKey{Key: sdsPubKeyBytes}
	return &pubKey, nil
}

// SdsPubKeyToBech32 convert a SdsPublicKey to a Bech32 string.
func SdsPubKeyToBech32(pubkey cryptotypes.PubKey) (string, error) {
	bech32Pub, err := bech32.ConvertAndEncode(SdsNodeP2PPubkeyPrefix, pubkey.Bytes())
	if err != nil {
		panic(err)
	}
	return bech32Pub, nil
}

func SdsAddressBytesToBech32(addr []byte) string {
	return addrBytesToBech32(addr, SdsNodeP2PAddressPrefix)
}

// SdsAddressFromBech32 creates an SdsAddress from a Bech32 string.
func SdsAddressFromBech32(address string) (addr SdsAddress, err error) {
	if len(strings.TrimSpace(address)) == 0 {
		return SdsAddress{}, fmt.Errorf("empty address string is not allowed")
	}

	bz, err := GetFromBech32(address, SdsNodeP2PAddressPrefix)
	if err != nil {
		return nil, err
	}

	err = VerifyAddressFormat(bz)
	if err != nil {
		return nil, err
	}

	return SdsAddress(bz), nil
}

// Equals Returns boolean for whether two SdsAddress are Equal
func (aa SdsAddress) Equals(aa2 Address) bool {
	if aa.Empty() && aa2.Empty() {
		return true
	}

	return bytes.Equal(aa.Bytes(), aa2.Bytes())
}

// Empty Returns boolean for whether a SdsAddress is empty
func (aa SdsAddress) Empty() bool {
	return aa == nil || len(aa) == 0
}

// Marshal returns the raw address bytes. It is needed for protobuf
// compatibility.
func (aa SdsAddress) Marshal() ([]byte, error) {
	return aa, nil
}

// Unmarshal sets the address to the given data. It is needed for protobuf
// compatibility.
func (aa *SdsAddress) Unmarshal(data []byte) error {
	*aa = data
	return nil
}

// MarshalJSON marshals to JSON using Bech32.
func (aa SdsAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(aa.String())
}

// MarshalYAML marshals to YAML using Bech32.
func (aa SdsAddress) MarshalYAML() (interface{}, error) {
	return aa.String(), nil
}

// UnmarshalJSON unmarshals from JSON assuming Bech32 encoding.
func (aa *SdsAddress) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)

	if err != nil {
		return err
	}
	if s == "" {
		*aa = SdsAddress{}
		return nil
	}

	aa2, err := SdsAddressFromBech32(s)
	if err != nil {
		return err
	}

	*aa = aa2
	return nil
}

// Bytes returns the raw address bytes.
func (aa SdsAddress) Bytes() []byte {
	return aa
}

// String implements the Stringer interface.
func (aa SdsAddress) String() string {
	if aa.Empty() {
		return ""
	}
	bech32Addr, err := bech32.ConvertAndEncode(SdsNodeP2PAddressPrefix, aa.Bytes())
	if err != nil {
		panic(err)
	}

	return bech32Addr
}

// Format implements the fmt.Formatter interface.
func (aa SdsAddress) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		s.Write([]byte(aa.String()))
	case 'p':
		s.Write([]byte(fmt.Sprintf("%p", aa)))
	default:
		s.Write([]byte(fmt.Sprintf("%X", []byte(aa))))
	}
}

//--------------------------------------------------------------------

func addrBytesToBech32(addr []byte, addrPrefix string) string {
	if addr == nil || len(addr) == 0 {
		return ""
	}
	bech32Addr, err := bech32.ConvertAndEncode(addrPrefix, addr)
	if err != nil {
		panic(err)
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
