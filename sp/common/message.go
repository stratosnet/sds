package common

const (
	MSG_LOGOUT          = 0x01
	MSG_MIMING          = 0x02
	MSG_TRANSFER_NOTICE = 0x03
	MSG_BACKUP_SLICE    = 0x04
	MSG_BACKUP_PP       = 0x05
	MSG_DELETE_SLICE    = 0x06
)

type Msg interface {
	GetType() uint32
}

type MsgMining struct {
	WalletAddress  string
	NetworkId      string
	Name           string
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
	SliceHash         string
	FromWalletAddress string
	ToWalletAddress   string
}

func (m *MsgTransferNotice) GetType() uint32 {
	return MSG_TRANSFER_NOTICE
}

type MsgBackupSlice struct {
	SliceHash         string
	FromWalletAddress string
}

func (m *MsgBackupSlice) GetType() uint32 {
	return MSG_BACKUP_SLICE
}

type MsgBackupPP struct {
	WalletAddress string
}

func (m *MsgBackupPP) GetType() uint32 {
	return MSG_BACKUP_PP
}

type MsgDeleteSlice struct {
	WalletAddress string
	SliceHash     string
}

func (m *MsgDeleteSlice) GetType() uint32 {
	return MSG_DELETE_SLICE
}
