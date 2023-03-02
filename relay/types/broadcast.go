package types

import (
	"github.com/pkg/errors"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"

	"github.com/stratosnet/sds/relay"
)

const (
	SignatureSecp256k1 = iota
	SignatureEd25519

	DefaultTimeoutHeight = uint64(10)
)

type SignatureKey struct {
	AccountNum      uint64 `json:"account_num,omitempty"`
	AccountSequence uint64 `json:"account_sequence,omitempty"`
	Address         string `json:"address,omitempty"`
	PrivateKey      []byte `json:"private_key,omitempty"`
	Type            int    `json:"type,omitempty"`
}

type UnsignedMsg struct {
	Msg           sdktypes.Msg   `json:"msg,omitempty"`
	SignatureKeys []SignatureKey `json:"signature_keys,omitempty"`
	Type          string         `json:"type,omitempty"`
}

type UnsignedMsgBytes struct {
	Msg           []byte         `json:"msg,omitempty"`
	SignatureKeys []SignatureKey `json:"signature_keys,omitempty"`
	Type          string         `json:"type,omitempty"`
}

type UnsignedMsgs struct {
	Msgs []UnsignedMsgBytes `json:"msgs,omitempty"`
}

func (u UnsignedMsg) ToBytes() UnsignedMsgBytes {
	return UnsignedMsgBytes{
		Msg:           relay.ProtoCdc.MustMarshalJSON(u.Msg),
		SignatureKeys: u.SignatureKeys,
		Type:          u.Type,
	}
}

func (u UnsignedMsgBytes) FromBytes() (UnsignedMsg, error) {
	unsignedMsg := UnsignedMsg{
		SignatureKeys: u.SignatureKeys,
		Type:          u.Type,
	}

	var err error

	switch u.Type {
	case "create_meta_node":
		msg := registertypes.MsgCreateMetaNode{}
		err = relay.ProtoCdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = &msg
	case "meta_node_registration_vote":
		msg := registertypes.MsgMetaNodeRegistrationVote{}
		err = relay.ProtoCdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = &msg
	case "update_meta_node":
		msg := registertypes.MsgUpdateMetaNode{}
		err = relay.ProtoCdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = &msg
	case "slashing_resource_node":
		msg := pottypes.MsgSlashingResourceNode{}
		err = relay.ProtoCdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = &msg
	case "FileUploadTx":
		msg := sdstypes.MsgFileUpload{}
		err = relay.ProtoCdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = &msg
	case "SdsPrepayTx":
		msg := sdstypes.MsgPrepay{}
		err = relay.ProtoCdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = &msg
	case "volume_report":
		msg := pottypes.MsgVolumeReport{}
		err = relay.ProtoCdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = &msg
	case "update_effective_stake":
		msg := registertypes.MsgUpdateEffectiveStake{}
		err = relay.Cdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = &msg
	default:
		return UnsignedMsg{}, errors.Errorf("Unknown msg type [%v]", u.Type)
	}

	if err != nil {
		return UnsignedMsg{}, errors.Wrapf(err, "Cannot unmarshal msg of type [%v]", u.Type)
	}
	return unsignedMsg, nil
}
