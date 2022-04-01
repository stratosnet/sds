package core

import (
	"sync"

	"github.com/stratosnet/sds/utils"
)

type connPool struct {
	conns   *sync.Map
	connCnt *utils.AtomicInt64
}

func newConnPool() *connPool {
	return &connPool{
		conns:   &sync.Map{},
		connCnt: utils.CreateAtomicInt64(0),
	}
}

func (cp *connPool) Store(id int64, conn *ServerConn) {
	cp.conns.Store(id, conn)
	cp.connCnt.IncrementAndGetNew()
}

func (cp *connPool) Delete(id int64) {
	cp.conns.Delete(id)
	cp.connCnt.DecrementAndGetNew()
}

func (cp *connPool) Load(id int64) (*ServerConn, bool) {
	conn, ok := cp.conns.Load(id)
	return conn.(*ServerConn), ok
}

func (cp *connPool) Range(fn func(id int64, conn *ServerConn) bool) {
	cp.conns.Range(func(key, value interface{}) bool {
		conn := value.(*ServerConn)
		id := key.(int64)
		return fn(id, conn)
	})
}

func (cp *connPool) Count() int64 {
	return cp.connCnt.GetAtomic()
}
