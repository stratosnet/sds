package common

import (
	"encoding/json"
)

const (
	MSG_LOGOUT            = 0x01
	MSG_MIMING            = 0x02
	MSG_TRANSFER_NOTICE   = 0x03
	MSG_BACKUP_SLICE      = 0x04
	MSG_BACKUP_PP         = 0x05
	MSG_DELETE_SLICE      = 0x06
	MSG_AGGREGATE_TRAFFIC = 0x07
)

type Msg interface {
	GetType() uint32
}

type MsgWrapper struct {
	MsgType uint32
	Msg     Msg
}

type MsgMining struct {
	P2PAddress     string
	NetworkAddress string
	Name           string
	Puk            []byte
}

func (m *MsgMining) GetType() uint32 {
	return MSG_MIMING
}

type MsgLogout struct {
	Name string
}

func (m *MsgLogout) GetType() uint32 {
	return MSG_LOGOUT
}

type MsgTransferNotice struct {
	SliceHash      string
	FromP2PAddress string
	ToP2PAddress   string
	DeleteOrigin   bool
}

func (m *MsgTransferNotice) GetType() uint32 {
	return MSG_TRANSFER_NOTICE
}

type MsgBackupSlice struct {
	SliceHash      string
	FromP2PAddress string
}

func (m *MsgBackupSlice) GetType() uint32 {
	return MSG_BACKUP_SLICE
}

type MsgBackupPP struct {
	P2PAddress string
}

func (m *MsgBackupPP) GetType() uint32 {
	return MSG_BACKUP_PP
}

type MsgDeleteSlice struct {
	P2PAddress string
	SliceHash  string
}

func (m *MsgDeleteSlice) GetType() uint32 {
	return MSG_DELETE_SLICE
}

type MsgAggregateTraffic struct {
	Time int64
}

func (m *MsgAggregateTraffic) GetType() uint32 {
	return MSG_AGGREGATE_TRAFFIC
}

func (w *MsgWrapper) UnmarshalJSON(data []byte) error {
	m := map[string]json.RawMessage{}
	var typeValue uint32

	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	if err := json.Unmarshal(m["MsgType"], &typeValue); err != nil {
		return err
	}

	w.MsgType = typeValue

	switch typeValue {
	case MSG_TRANSFER_NOTICE:
		var msgTransferNotice MsgTransferNotice
		if err := json.Unmarshal(m["Msg"], &msgTransferNotice); err != nil {
			return err
		}
		w.Msg = &msgTransferNotice
	case MSG_AGGREGATE_TRAFFIC:
		var msgAggregateTraffic MsgAggregateTraffic
		if err := json.Unmarshal(m["Msg"], &msgAggregateTraffic); err != nil {
			return err
		}
		w.Msg = &msgAggregateTraffic
	case MSG_BACKUP_PP:
		var msgBackupPP MsgBackupPP
		if err := json.Unmarshal(m["Msg"], &msgBackupPP); err != nil {
			return err
		}
		w.Msg = &msgBackupPP
	case MSG_BACKUP_SLICE:
		var msgBackupSlice MsgBackupSlice
		if err := json.Unmarshal(m["Msg"], &msgBackupSlice); err != nil {
			return err
		}
		w.Msg = &msgBackupSlice
	}

	return nil
}
