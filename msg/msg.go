package msg

import (
	"reflect"
	"unsafe"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/cmem"
)

// RelayMsgBuf application layer internal buffer for msg，
type RelayMsgBuf struct {
	PacketId int64
	MSGHead  header.MessageHead
	MSGSign  MessageSign
	MSGBody  []byte
	MSGData  []byte

	Alloc *[]byte
}

func (r *RelayMsgBuf) PutIntoBuffer(msg *RelayMsgBuf) int {
	var i = 0
	totalLength := r.MSGHead.Len + header.MsgHeaderLen + MsgSignLen + uint32(len(msg.MSGData))
	// allocate memory
	r.Alloc = cmem.Alloc(uintptr(totalLength))
	r.MSGData = (*[1 << 30]byte)(unsafe.Pointer(r.Alloc))[:totalLength]
	(*reflect.SliceHeader)(unsafe.Pointer(&r.MSGData)).Cap = int(totalLength)

	// encode the message into the field MSGData
	i += r.MSGHead.Encode(r.MSGData[i : i+header.MsgHeaderLen])
	i += copy(r.MSGData[i:], msg.MSGBody[:])
	if err := r.MSGSign.Sign(r.MSGData[:i]); err != nil {
		utils.ErrorLog("failed sign the message, ", err.Error())
		return 0
	}

	i += r.MSGSign.Encode(r.MSGData[i:])
	if len(msg.MSGData) != 0 {
		utils.DebugLogf("%d bytes data to send.....", len(msg.MSGData))
	}
	i += copy(r.MSGData[i:], msg.MSGData[:])
	return i
}

func (r *RelayMsgBuf) GetHeader() []byte {
	return r.MSGData[:header.MsgHeaderLen]
}

func (r *RelayMsgBuf) GetBytesAfterHeader() []byte {
	return r.MSGData[header.MsgHeaderLen:]
}
