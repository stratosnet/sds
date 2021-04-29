package events

import (
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"testing"
)

func Test_getCustomerAddVolumeCallbackFunc(t *testing.T) {
	s, mysqlCloseFunc, redisCloseFunc := StartMock(header.ReqCAddVolume, GetCAddVolumeHandler)
	defer mysqlCloseFunc()
	defer redisCloseFunc()

	m := &protos.ReqCustomerAddVolume{
		WalletAddress: "abcd",
		ReqId:         "1234",
		Volume:        100000000,
		PublicKey:     []byte{1, 2, 3, 4, 5},
	}

	SendMessageToMock(s.Host, header.ReqCAddVolume, m)

	stop := make(chan bool, 10)

	for {
		select {
		case <-stop:
			return
		}
	}
}
