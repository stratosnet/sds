package utils

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	twepoch        = int64(1417937700000) // default start time epoch,
	DistrictIdBits = uint(5)              // district id bits size
	NodeIdBits     = uint(9)              // node id bits size
	sequenceBits   = uint(10)             // sequence number bits size
	/*
	 * 1 sign   |  39 timestamp                                 | 5 district  |  9 node      | 10 （milliSecond）self sequence ID
	 * 0        |  0000000 00000000 00000000 00000000 00000000  | 00000       | 000000 000   |  000000 0000
	 *
	 */
	maxNodeId     = -1 ^ (-1 << NodeIdBits)     // max node id
	maxDistrictId = -1 ^ (-1 << DistrictIdBits) // max district id

	nodeIdShift        = sequenceBits // left shift times
	DistrictIdShift    = sequenceBits + NodeIdBits
	timestampLeftShift = sequenceBits + NodeIdBits + DistrictIdBits
	sequenceMask       = -1 ^ (-1 << sequenceBits)
	maxNextIdsNum      = 100 //max number of Ids per retrieve
)

var MyIdWorker *IdWorker

type IdWorker struct {
	sequence      int64
	lastTimestamp int64
	nodeId        int64
	twepoch       int64
	districtId    int64
	mutex         sync.Mutex
}

// NewIdWorker new a snowflake id generator object.
func NewIdWorker(NodeId int64) (*IdWorker, error) {
	var districtId int64
	districtId = 1 // default to 1, for future extension
	idWorker := &IdWorker{}
	if NodeId > maxNodeId || NodeId < 0 {
		fmt.Sprintf("NodeId Id can't be greater than %d or less than 0", maxNodeId)
		return nil, errors.New(fmt.Sprintf("NodeId Id: %d error", NodeId))
	}
	if districtId > maxDistrictId || districtId < 0 {
		fmt.Sprintf("District Id can't be greater than %d or less than 0", maxDistrictId)
		return nil, errors.New(fmt.Sprintf("District Id: %d error", districtId))
	}
	idWorker.nodeId = NodeId
	idWorker.districtId = districtId
	idWorker.lastTimestamp = -1
	idWorker.sequence = 0
	idWorker.twepoch = twepoch
	idWorker.mutex = sync.Mutex{}
	fmt.Sprintf("worker starting. timestamp left shift %d, District id bits %d, worker id bits %d, sequence bits %d, workerid %d", timestampLeftShift, DistrictIdBits, NodeIdBits, sequenceBits, NodeId)
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
		fmt.Sprintf("NextIds num can't be greater than %d or less than 0", maxNextIdsNum)
		return nil, errors.New(fmt.Sprintf("NextIds num: %d error", num))
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
		return 0, errors.New(fmt.Sprintf("Clock moved backwards.  Refusing to generate id for %d milliseconds", id.lastTimestamp-timestamp))
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
	return ((timestamp - id.twepoch) << timestampLeftShift) | (id.districtId << DistrictIdShift) | (id.nodeId << nodeIdShift) | id.sequence, nil
}

func InitIdWorker() error {
	idWorker, err := NewIdWorker(int64(0))
	MyIdWorker = idWorker
	return err
}

func NextSnowFakeId() (int64, error) {
	return MyIdWorker.NextId()
}

func ZeroId() int64 {
	return int64(0)
}
