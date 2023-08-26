package utils

import (
	"sync"
)

const (
	BufferPoolRefillStep = 10
)

type bufferPool struct {
	bufferSize  int
	poolMaxSize int
	pool        chan []byte
	counter     int64
	mutex       sync.Mutex
}

var globalBufferPool *bufferPool

func InitBufferPool(bufferSize, poolSize int) {
	globalBufferPool = &bufferPool{
		bufferSize:  bufferSize,
		poolMaxSize: poolSize,
		pool:        make(chan []byte, poolSize),
		counter:     0,
	}
	// fill up the pool
	for i := 0; i < cap(globalBufferPool.pool); i++ {
		buffer := make([]byte, 0, bufferSize)
		globalBufferPool.pool <- buffer
	}
}

func IsPoolFull() bool {
	return len(globalBufferPool.pool) >= globalBufferPool.poolMaxSize
}

func RequestBuffer() []byte {
	DebugLog("-", len(globalBufferPool.pool))
	return globalBufferPool.requestBuffer()
}

func ReleaseBuffer(buffer []byte) {
	DebugLog("+", len(globalBufferPool.pool))
	globalBufferPool.releaseBuffer(buffer)
}

func (bp *bufferPool) requestBuffer() []byte {

	// Check if a buffer is available in the pool
	if len(bp.pool) > 0 {
		return <-bp.pool
	}

	// No buffer available, block the caller
	buffer := <-bp.pool
	return buffer
}

func (bp *bufferPool) releaseBuffer(buffer []byte) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	if cap(buffer) == bp.bufferSize && len(bp.pool) < bp.poolMaxSize {
		bp.pool <- buffer
	}
}
