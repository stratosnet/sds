package network

import (
	"time"

	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

const (
	INTERVAL_CHECK_BUFFER_POOL = 5 // in seconds
)

func (p *Network) CheckAndFillUpBufferPool() func() {
	return func() {
		// only when it's traffic free, the buffer pool is refilled
		if file.DataBuffer.TryLock() {
			if !utils.IsPoolFull() {
				for i := 0; i < utils.BufferPoolRefillStep; i++ {
					buffer := make([]byte, 0, setting.MaxData)
					utils.ReleaseBuffer(buffer)
				}
			}
			file.DataBuffer.Unlock()
		}
	}
}

func (p *Network) StartDataBufferPool() {
	p.ppPeerClock.AddJobWithInterval(INTERVAL_CHECK_BUFFER_POOL*time.Second, p.CheckAndFillUpBufferPool())
}
