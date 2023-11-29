package types

import (
	"fmt"

	"github.com/cosmos/cosmos-proto/anyutil"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	potv1 "github.com/stratosnet/stratos-chain/api/stratos/pot/v1"
	registerv1 "github.com/stratosnet/stratos-chain/api/stratos/register/v1"
	sdsv1 "github.com/stratosnet/stratos-chain/api/stratos/sds/v1"
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
	case MSG_TYPE_CREATE_META_NODE:
		msg := &registerv1.MsgCreateMetaNode{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	case MSG_TYPE_META_NODE_REG_VOTE:
		msg := &registerv1.MsgMetaNodeRegistrationVote{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	case MSG_TYPE_UPDATE_META_NODE:
		msg := &registerv1.MsgUpdateMetaNode{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	case MSG_TYPE_UPDATE_META_NODE_DEPOSIT:
		msg := &registerv1.MsgUpdateMetaNodeDeposit{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	case MSG_TYPE_WITHDRAWN_META_NODE_REG_DEPOSIT:
		msg := &registerv1.MsgWithdrawMetaNodeRegistrationDeposit{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	case MSG_TYPE_SLASHING_RESOURCE_NODE:
		msg := &potv1.MsgSlashingResourceNode{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	case MSG_TYPE_FILE_UPLOAD:
		msg := &sdsv1.MsgFileUpload{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	case MSG_TYPE_PREPAY:
		msg := &sdsv1.MsgPrepay{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	case MSG_TYPE_VOLUME_REPORT:
		msg := &potv1.MsgVolumeReport{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	case MSG_TYPE_UPDATE_EFFECTIVE_DEPOSIT:
		msg := &registerv1.MsgUpdateEffectiveDeposit{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	case MSG_TYPE_REMOVE_META_NODE:
		msg := &registerv1.MsgRemoveMetaNode{}
		err = proto.Unmarshal(u.Msg, msg)
		unsignedMsg.Msg, err = anyutil.New(msg)
	default:
		return nil, fmt.Errorf("Unknown msg type [%v]", u.Type)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "Cannot unmarshal msg of type [%v]", u.Type)
	}
	return unsignedMsg, nil
}
