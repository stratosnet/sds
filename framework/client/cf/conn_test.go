package cf

import (
	"fmt"
	"testing"

	pool "github.com/libp2p/go-buffer-pool"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/utils"
)

func TestCreateWallet(t *testing.T) {
	msgH := []byte("This is a header..AB1234")
	fmt.Println("Size of msgH:", len(msgH))
	m := &msg.RelayMsgBuf{
		PacketId: 23123,
		MSGBody:  []byte("This is a test message of "),
	}
	var memory *msg.RelayMsgBuf
	buffer := pool.NewBuffer(msgH)
	buffer.Grow(len(m.MSGData))
	fmt.Println("buffer:", buffer)
	memory.MSGBody = buffer.Bytes()
	copy(memory.MSGData[0:], msgH)
	copy(memory.MSGData[utils.MsgHeaderLen:], m.MSGData)

}
