package core

import (
	"github.com/stratosnet/sds/utils"
	"hash/fnv"
	"unsafe"
)

type taskFunc func()

// TaskPool
type TaskPool struct {
	tasks     []*task
	closeChan chan struct{}
}

// GlobalTaskPool
var GlobalTaskPool *TaskPool

// task
type task struct {
	index        int
	callbackChan chan taskFunc
	closeChan    chan struct{}
}

func makeTaskPool(count int) *TaskPool {
	if count <= 0 {
		count = utils.DefTaskPoolCount
	}

	taskPool := TaskPool{
		tasks:     make([]*task, count),
		closeChan: make(chan struct{}),
	}
	for i := range taskPool.tasks {
		taskPool.tasks[i] = makeTask(i, utils.TaskSize, taskPool.closeChan)
		if taskPool.tasks[i] == nil {
			utils.Log("task nil")
		}
	}
	return &taskPool
}

//  make a task and start a go routine
func makeTask(index int, size int, close chan struct{}) *task {
	t := &task{
		index:        index,
		callbackChan: make(chan taskFunc, size),
		closeChan:    close,
	}
	go t.start()
	return t
}
func (t *task) start() {
	for {
		select {
		case <-t.closeChan:
			return
		case fc := <-t.callbackChan:
			fc()
			// todo: add time management
		}
	}
}

// Job: add job to the pool
func (tp *TaskPool) Job(id int64, fc func()) error {
	var hashCode uint32
	h := fnv.New32a()
	h.Write((*((*[8]byte)(unsafe.Pointer(&id))))[:])
	hashCode = h.Sum32()
	// make sure that the msg from the same netId is allocate to the same routine to be processed in sequence
	return tp.tasks[hashCode&uint32(len(tp.tasks)-1)].job(taskFunc(fc))
}

func (t *task) job(fc taskFunc) error {
	select {
	case t.callbackChan <- fc:
		return nil
	default:
		return utils.ErrNotFoundCallBack
	}
}
