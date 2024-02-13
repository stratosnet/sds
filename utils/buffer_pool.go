package utils

import (
	"fmt"
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
	start := time.Now().UnixMilli()
	//DebugLog(len(globalBufferPool.pool), "-")
	globalBufferPool.mutex.Lock()
	if len(globalBufferPool.pool) == 0 {
		if job == nil {
			fmt.Println("Adding Job")
			job, _ = timer.AddJobWithInterval(BufferPoolRefillInterval*time.Second, refillPool())
		} else {
			fmt.Println("Updating Job")
			if !timer.UpdateJobTimeout(job, BufferPoolRefillInterval*time.Second) {
				fmt.Println("failed updating job")
			}
		}
	}
	globalBufferPool.mutex.Unlock()
	buffer := globalBufferPool.requestBuffer()
	costTime := time.Now().UnixMilli() - start
	// TO BE DELETED
	DebugLog(len(globalBufferPool.pool), "-", "cost_time= ", costTime, " ms")
	return buffer
}

func ReleaseBuffer(buffer []byte) {
	DebugLog(len(globalBufferPool.pool), "+")
	globalBufferPool.mutex.Lock()
	timer.Reset()
	job = nil
	globalBufferPool.mutex.Unlock()

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
	} else {
		ErrorLogf("Buffer not released, ACTUAL: len(buffer) = %d, cap(buffer) = %d, EXPECTED: cap(buffer) = %d, len(bp.pool) = %d, bp.poolMaxSize = %d",
			len(buffer), cap(buffer), bp.bufferSize, len(bp.pool), bp.poolMaxSize)
	}
}
