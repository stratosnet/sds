package types

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// SignatureKey --------------------------------------
type SignatureKey struct {
	AccountNum      uint64 `json:"account_num,omitempty"`
	AccountSequence uint64 `json:"account_sequence,omitempty"`
	Address         string `json:"address,omitempty"`
	PrivateKey      []byte `json:"private_key,omitempty"`
	Type            int    `json:"type,omitempty"`
}

// UnsignedMsgs --------------------------------------
type UnsignedMsgs struct {
	Msgs []*UnsignedMsgBytes `json:"msgs,omitempty"`
}

// UnsignedMsg ---------------------------------------
type UnsignedMsg struct {
	Msg           *anypb.Any      `json:"msg,omitempty"`
	SignatureKeys []*SignatureKey `json:"signature_keys,omitempty"`
	Type          string          `json:"type,omitempty"`
}

func (u *UnsignedMsg) ToBytes() (*UnsignedMsgBytes, error) {
	msgBz, err := proto.Marshal(u.Msg)
	if err != nil {
		return nil, err
	}

	return &UnsignedMsgBytes{
		Msg:           msgBz,
		SignatureKeys: u.SignatureKeys,
		Type:          u.Type,
	}, nil
}

// UnsignedMsgBytes ----------------------------------
type UnsignedMsgBytes struct {
	Msg           []byte          `json:"msg,omitempty"`
	SignatureKeys []*SignatureKey `json:"signature_keys,omitempty"`
	Type          string          `json:"type,omitempty"`
}

func (u *UnsignedMsgBytes) FromBytes() (*UnsignedMsg, error) {
	msgAny := &anypb.Any{}
	err := proto.Unmarshal(u.Msg, msgAny)
	if err != nil {
		return nil, err
	}

	return &UnsignedMsg{
		Msg:           msgAny,
		SignatureKeys: u.SignatureKeys,
		Type:          u.Type,
	}, nil
}
