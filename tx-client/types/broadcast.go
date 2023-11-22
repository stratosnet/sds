package types

import (
	"fmt"

	"github.com/cosmos/gogoproto/proto"
	"github.com/pkg/errors"
	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
	registerv1 "github.com/stratosnet/stratos-chain/api/stratos/register/v1"
	sdsv1 "github.com/stratosnet/stratos-chain/api/stratos/sds/v1"
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
	unsignedMsg := &UnsignedMsg{
		SignatureKeys: u.SignatureKeys,
		Type:          u.Type,
	}

	var err error

	switch u.Type {
	case "create_meta_node":
		msg := &registerv1.MsgCreateMetaNode{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	case "meta_node_registration_vote":
		msg := &registerv1.MsgMetaNodeRegistrationVote{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	case "update_meta_node":
		msg := &registerv1.MsgUpdateMetaNode{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	case "update_meta_node_deposit":
		msg := &registerv1.MsgUpdateMetaNodeDeposit{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	case "withdraw_meta_node_registration_deposit":
		msg := &registerv1.MsgWithdrawMetaNodeRegistrationDeposit{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	case "slashing_resource_node":
		msg := &potv1.MsgSlashingResourceNode{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	case "FileUploadTx":
		msg := &sdsv1.MsgFileUpload{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	case "SdsPrepayTx":
		msg := &sdsv1.MsgPrepay{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	case "volume_report":
		msg := &potv1.MsgVolumeReport{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	case "update_effective_deposit":
		msg := &registerv1.MsgUpdateEffectiveDeposit{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	case "remove_meta_node":
		msg := &registerv1.MsgRemoveMetaNode{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anypb.New(msg)
	default:
		return nil, fmt.Errorf("Unknown msg type [%v]", u.Type)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "Cannot unmarshal msg of type [%v]", u.Type)
	}
	return unsignedMsg, nil
}
