package utils

import (
	"sync"
	"time"

	"github.com/alex023/clock"
)

const (
	BufferPoolRefillInterval = 3 // in second
)

type bufferPool struct {
	bufferSize  int
	poolMaxSize int
	pool        chan []byte
	counter     int64
	mutex       sync.Mutex
}

var (
	globalBufferPool *bufferPool
	timer            *clock.Clock
	job              clock.Job
)

func InitBufferPool(bufferSize, poolSize int) {
	globalBufferPool = &bufferPool{
		bufferSize:  bufferSize,
		poolMaxSize: poolSize,
		pool:        make(chan []byte, poolSize),
		counter:     0,
	}
	timer = clock.NewClock()
	refillPool()()
}

func IsPoolFull() bool {
	return len(globalBufferPool.pool) >= globalBufferPool.poolMaxSize
}

func refillPool() func() {
	return func() {
		globalBufferPool.mutex.Lock()
		// fill up the pool
		for i := 0; i < cap(globalBufferPool.pool); i++ {
			buffer := make([]byte, 0, globalBufferPool.bufferSize)
			globalBufferPool.pool <- buffer
		}
		DebugLog("refilled pool:", len(globalBufferPool.pool))
		globalBufferPool.mutex.Unlock()
	}
}

func RequestBuffer() []byte {
	//DebugLog(len(globalBufferPool.pool), "-")
	globalBufferPool.mutex.Lock()
	if len(globalBufferPool.pool) == 0 {
		if job == nil {
			job, _ = timer.AddJobWithInterval(BufferPoolRefillInterval*time.Second, refillPool())
		} else {
			timer.UpdateJobTimeout(job, BufferPoolRefillInterval*time.Second)
		}
	}
	globalBufferPool.mutex.Unlock()
	buffer := globalBufferPool.requestBuffer()
	return buffer
}

func ReleaseBuffer(buffer []byte) {
	globalBufferPool.mutex.Lock()
	timer.Reset()
	job = nil
	globalBufferPool.mutex.Unlock()

	if buffer == nil {
		return
	}
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

	if cap(buffer) != bp.bufferSize {
		ErrorLogf("release buffer at %v with wrong capaity %v ", &buffer[0:globalBufferPool.bufferSize][0], len(buffer))
		return
	}
	if len(bp.pool) >= bp.poolMaxSize {
		ErrorLogf("buffer pool is full when release buffer at %v, current pool size %v ", &buffer[0:globalBufferPool.bufferSize][0], len(bp.pool))
		return
	}
	bp.pool <- buffer
}
