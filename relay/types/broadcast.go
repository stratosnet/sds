package types

import (
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/relay"
	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"
)

const (
	SignatureSecp256k1 = iota
	SignatureEd25519
)

type SignatureKey struct {
	AccountNum      uint64 `json:"account_num,omitempty"`
	AccountSequence uint64 `json:"account_sequence,omitempty"`
	Address         string `json:"address,omitempty"`
	PrivateKey      []byte `json:"private_key,omitempty"`
	Type            int    `json:"type,omitempty"`
}

type UnsignedMsg struct {
	Msg           types.Msg      `json:"msg,omitempty"`
	SignatureKeys []SignatureKey `json:"signature_keys,omitempty"`
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
		Msg:           u.Msg.GetSignBytes(),
		SignatureKeys: u.SignatureKeys,
		Type:          u.Msg.Type(),
	}
}

func (u UnsignedMsgBytes) FromBytes() (UnsignedMsg, error) {
	unsignedMsg := UnsignedMsg{
		SignatureKeys: u.SignatureKeys,
	}

	var err error

	switch u.Type {
	case "create_indexing_node":
		msg := registertypes.MsgCreateIndexingNode{}
		err = relay.Cdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = msg
	case "indexing_node_reg_vote":
		msg := registertypes.MsgIndexingNodeRegistrationVote{}
		err = relay.Cdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = msg
	case "update_indexing_node":
		msg := registertypes.MsgUpdateIndexingNode{}
		err = relay.Cdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = msg
	case "slashing_resource_node":
		msg := pottypes.MsgSlashingResourceNode{}
		err = relay.Cdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = msg
	case "FileUploadTx":
		msg := sdstypes.MsgFileUpload{}
		err = relay.Cdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = msg
	case "SdsPrepayTx":
		msg := sdstypes.MsgPrepay{}
		err = relay.Cdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = msg
	case "volume_report":
		msg := pottypes.MsgVolumeReport{}
		err = relay.Cdc.UnmarshalJSON(u.Msg, &msg)
		unsignedMsg.Msg = msg
	default:
		return UnsignedMsg{}, errors.Errorf("Unknown msg type [%v]", u.Type)
	}

	if err != nil {
		return UnsignedMsg{}, errors.Wrapf(err, "Cannot unmarshal msg of type [%v]", u.Type)
	}
	return unsignedMsg, nil
}
