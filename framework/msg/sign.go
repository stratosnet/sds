package msg

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/stratosnet/sds/framework/crypto"
	fwed25519 "github.com/stratosnet/sds/framework/crypto/ed25519"
	fwtypes "github.com/stratosnet/sds/framework/types"
)

const (
	P2pAddressBech32Length    = 44
	MsgSignLen                = fwed25519.SignatureSize + P2pAddressBech32Length + fwed25519.PubKeySize
	ThresholdToHashBeforeSign = 128
)

type Signer func([]byte) ([]byte, error)

// MessageSign
type MessageSign struct {
	Signature  []byte
	P2pAddress string
	P2pPubKey  []byte
	Signer     Signer
}

func (s *MessageSign) Sign(hb []byte) error {
	// add check for the length
	var signMsg []byte
	if len(hb) > ThresholdToHashBeforeSign {
		signMsg = crypto.CalcHashBytes(hb)
	} else {
		signMsg = hb
	}
	signature, err := s.Signer(signMsg)
	if err != nil {
		return err
	}
	s.Signature = signature
	return nil
}

func (s *MessageSign) Verify(hb []byte, remoteP2pAddr string) error {
	if s.P2pAddress == "" || s.P2pPubKey == nil || s.Signature == nil {
		return errors.New("missing information")
	}
	if s.P2pAddress != remoteP2pAddr {
		return errors.New(fmt.Sprintf("wrong source p2p address from msg: %s, loc: %s", s.P2pAddress, remoteP2pAddr))
	}
	// verify node p2p address
	if !fwtypes.VerifyP2pAddrBytes(s.P2pPubKey, s.P2pAddress) {
		return errors.New("p2p address doesn't match public key")
	}
	var signMsg []byte
	if len(hb) > ThresholdToHashBeforeSign {
		signMsg = crypto.CalcHashBytes(hb)
	} else {
		signMsg = hb
	}
	if !fwtypes.VerifyP2pSignBytes(s.P2pPubKey, s.Signature, signMsg) {
		return errors.New("wrong signature")
	}
	return nil
}

func (s *MessageSign) Encode(sign []byte) int {
	var i = 0
	i += copy(sign[i:], s.P2pAddress)
	i += copy(sign[i:], s.P2pPubKey)
	i += copy(sign[i:], s.Signature)
	return i
}

func (s *MessageSign) Decode(sign []byte) int {
	var i = 0
	s.P2pAddress = string(sign[i : i+P2pAddressBech32Length])
	i += P2pAddressBech32Length

	s.P2pPubKey = sign[i : i+fwed25519.PubKeySize]
	i += fwed25519.PubKeySize
	s.Signature = sign[i : i+fwed25519.SignatureSize]
	i += fwed25519.SignatureSize
	return i
}
