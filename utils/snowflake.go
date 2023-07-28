package utils

import (
	"fmt"
	"sync"
	"time"
)

const (
	twepoch      = int64(1685987883000) // default start time epoch,
	NodeIdBits   = uint(8)              // node id bits size
	sequenceBits = uint(16)             // sequence number bits size
	/*
	 * 1b sign |                               39b timestamp | 8b node  |   16b sequence ID |
	 *       0 | 0000000 00000000 00000000 00000000 00000000 | 00000000 | 00000000 00000000 |
	 */
	maxNodeId    = -1 ^ (-1 << NodeIdBits) // max node id
	sequenceMask = -1 ^ (-1 << sequenceBits)

	nodeIdShift    = sequenceBits // left shift times
	timestampShift = sequenceBits + NodeIdBits
	maxNextIdsNum  = 100 //max number of Ids per retrieve
)

var MyIdWorker *IdWorker

type IdWorker struct {
	sequence      int64
	lastTimestamp int64
	nodeId        int64
	twepoch       int64
	mutex         sync.Mutex
}

// NewIdWorker new a snowflake id generator object.
func NewIdWorker(NodeId int64) (*IdWorker, error) {
	idWorker := &IdWorker{}
	if NodeId > maxNodeId || NodeId < 0 {
		ErrorLogf("NodeId Id can't be greater than %d or less than 0", maxNodeId)
		return nil, fmt.Errorf("NodeId Id: %d error", NodeId)
	}
	idWorker.nodeId = NodeId
	idWorker.lastTimestamp = -1
	idWorker.sequence = 0
	idWorker.twepoch = twepoch
	idWorker.mutex = sync.Mutex{}
	return idWorker, nil
}

// timeGen generate a unix millisecond.
func timeGen() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// tilNextMillis spin wait till next millisecond.
func tilNextMillis(lastTimestamp int64) int64 {
	timestamp := timeGen()
	for timestamp <= lastTimestamp {
		timestamp = timeGen()
	}
	return timestamp
}

// NextId get a snowflake id.
func (id *IdWorker) NextId() (int64, error) {
	id.mutex.Lock()
	defer id.mutex.Unlock()
	return id.nextid()
}

// NextIds get snowflake ids.
func (id *IdWorker) NextIds(num int) ([]int64, error) {
	if num > maxNextIdsNum || num < 0 {
		ErrorLogf("NextIds num can't be greater than %d or less than 0", maxNextIdsNum)
		return nil, fmt.Errorf("NextIds num: %d error", num)
	}
	ids := make([]int64, num)
	id.mutex.Lock()
	defer id.mutex.Unlock()
	for i := 0; i < num; i++ {
		ids[i], _ = id.nextid()
	}
	return ids, nil
}

func (id *IdWorker) nextid() (int64, error) {
	timestamp := timeGen()
	if timestamp < id.lastTimestamp {
		//    fmt.Sprintf("clock is moving backwards.  Rejecting requests until %d.", id.lastTimestamp)
		return 0, fmt.Errorf("Clock moved backwards.  Refusing to generate id for %d milliseconds", id.lastTimestamp-timestamp)
	}
	if id.lastTimestamp == timestamp {
		id.sequence = (id.sequence + 1) & sequenceMask
		if id.sequence == 0 {
			timestamp = tilNextMillis(id.lastTimestamp)
		}
	} else {
		id.sequence = 0
	}
	id.lastTimestamp = timestamp
	return ((timestamp - id.twepoch) << timestampShift) | (id.nodeId << nodeIdShift) | id.sequence, nil
}

func InitIdWorker(nodeid uint8) error {
	idWorker, err := NewIdWorker(int64(nodeid))
	MyIdWorker = idWorker
	return err
}

func NextSnowFlakeId() (int64, error) {
	return MyIdWorker.NextId()
}

func ZeroId() int64 {
	return int64(0)
}
