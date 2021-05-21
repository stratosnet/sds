package spbf

import (
	"context"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/utils"
	"net"

	"sync"
	"time"

	"github.com/alex023/clock"
)

var (
	netID *utils.AtomicInt64
	lock  sync.Mutex
)

func init() {
	netID = utils.CreateAtomicInt64(0)
}

type onConnectFunc func(WriteCloser) bool
type onMessageFunc func(msg.RelayMsgBuf, WriteCloser)
type onCloseFunc func(WriteCloser)
type onErrorFunc func(WriteCloser)

type options struct {
	onConnect      onConnectFunc
	onMessage      onMessageFunc
	onClose        onCloseFunc
	onError        onErrorFunc
	bufferSize     int
	logOpen        bool
	maxConnections int
	maxflow        int
}

// ServerOption
type ServerOption func(*options)

// Server
type Server struct {
	opts             options
	ctx              context.Context
	cancel           context.CancelFunc
	conns            *sync.Map
	wg               *sync.WaitGroup
	mu               sync.Mutex // lock
	lis              map[net.Listener]bool
	interv           time.Duration
	goroutine        int64
	goAtom           *utils.AtomicInt64
	allFlow          int64
	allAtom          *utils.AtomicInt64
	readFlow         int64
	readAtom         *utils.AtomicInt64
	writeFlow        int64
	writeAtom        *utils.AtomicInt64
	secondReadFlowA  int64
	secondReadAtomA  *utils.AtomicInt64
	secondWriteFlowA int64
	secondWriteAtomA *utils.AtomicInt64
	secondReadFlowB  int64
	secondReadAtomB  *utils.AtomicInt64
	secondWriteFlowB int64
	secondWriteAtomB *utils.AtomicInt64
}

// Mylog
func Mylog(b bool, v ...interface{}) {
	if b {
		utils.DebugLog(v...)
	}
}

// CreateServer
func CreateServer(opt ...ServerOption) *Server {
	var opts options
	for _, o := range opt {
		o(&opts)
	}

	// initiates go-routine pool instance
	GlobalTaskPool = makeTaskPool(0)

	s := &Server{
		opts:             opts,
		conns:            &sync.Map{},
		wg:               &sync.WaitGroup{},
		lis:              make(map[net.Listener]bool),
		goAtom:           utils.CreateAtomicInt64(0),
		allAtom:          utils.CreateAtomicInt64(0),
		readAtom:         utils.CreateAtomicInt64(0),
		writeAtom:        utils.CreateAtomicInt64(0),
		secondReadAtomA:  utils.CreateAtomicInt64(0),
		secondWriteAtomA: utils.CreateAtomicInt64(0),
		secondReadAtomB:  utils.CreateAtomicInt64(0),
		secondWriteAtomB: utils.CreateAtomicInt64(0),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s
}

// ConnsSize
func (s *Server) ConnsSize() int {
	var sz int
	s.conns.Range(func(k, v interface{}) bool {
		sz++
		return true
	})
	return sz
}

// Start
func (s *Server) Start(l net.Listener) error {
	s.mu.Lock()
	if s.lis == nil {
		s.mu.Unlock()
		l.Close()
		return utils.ErrServerClosed
	}
	s.lis[l] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if s.lis != nil && s.lis[l] {
			l.Close()
			delete(s.lis, l)
		}
		s.mu.Unlock()
	}()

	Mylog(s.opts.logOpen, "server start, net", l.Addr().Network(), "addr", l.Addr().String(), "\n")
	var (
		myClock = clock.NewClock()
		logFunc = func() {
			// utils.Log("connsSize:", s.ConnsSize(), "routine num:", s.goroutine, "allread:", fmt.Sprintf("%.4f", float64(s.readFlow)/1024/1024), "MB", "allwrite:", fmt.Sprintf("%.4f", float64(s.writeFlow)/1024/1024), "MB", "all:", fmt.Sprintf("%.4f", float64(s.allFlow)/1024/1024), "MB",
			// "read/s:", fmt.Sprintf("%.4f", float64(s.secondReadFlow)/1024/1024), "MB", "write/s:", fmt.Sprintf("%.4f", float64(s.secondWriteFlow)/1024/1024), "MB")
			s.secondReadFlowB = s.secondReadAtomB.GetNewAndSetAtomic(s.secondReadFlowA)
			s.secondWriteFlowB = s.secondWriteAtomB.GetAndSetAtomic(s.secondWriteFlowA)
			s.secondReadFlowA = s.secondReadAtomA.GetNewAndSetAtomic(0)
			s.secondWriteFlowA = s.secondWriteAtomA.GetNewAndSetAtomic(0)
		}
	)
	myClock.AddJobRepeat(time.Second*1, 0, logFunc)
	// var tempDelay time.Duration
	for {
		spbConn, err := l.Accept()
		if err != nil {
			// if ne, ok := err.(net.Error); ok && ne.Temporary() {
			// 	if tempDelay == 0 {
			// 		tempDelay = 5 * time.Millisecond
			// 	} else {
			// 		tempDelay *= 2
			// 	}
			// 	if max := 1 * time.Second; tempDelay >= max {
			// 		tempDelay = max
			// 	}
			// 	holmes.Errorf("accept error %v, retrying in %d\n", err, tempDelay)
			// 	select {
			// 	case <-time.After(tempDelay):
			// 	case <-s.ctx.Done():
			// 	}
			// 	continue
			// }

			utils.ErrorLog("accept err:", err)
			return err
		}
		tcpConn, ok := spbConn.(*net.TCPConn)
		if ok {
			tcpConn.SetNoDelay(false)
		}
		// tempDelay = 0

		sz := s.ConnsSize()
		if s.opts.maxConnections != 0 {
			if sz >= s.opts.maxConnections {
				utils.ErrorLog("max connections size", sz, "refuse\n")
				spbConn.Close()
				continue
			}
		}
		//utils.DebugLog("MaxConnections", s.opts.maxConnections)
		netid := netID.GetOldAndIncrement()
		sc := CreateServerConn(netid, s, spbConn)
		sc.SetConnName(sc.spbConn.RemoteAddr().String())

		// s.mu.Lock()
		// if s.sched != nil {
		// 	sc.RunEvery(s.interv, s.sched)
		// }
		// s.mu.Unlock()

		s.conns.Store(netid, sc)
		// addTotalConn(1)
		s.wg.Add(1) // this will be Done() in ServerConn.Close()
		s.goroutine = s.goAtom.IncrementAndGetNew()
		go func() {
			sc.Start()
		}()

		Mylog(s.opts.logOpen, "accepted client", sc.GetName(), "id:", netid, "total:", s.ConnsSize(), "\n")
		// s.conns.Range(func(k, v interface{}) bool {
		// 	i := k.(int64)
		// 	c := v.(*ServerConn)
		// 	Mylog(s.opts.logOpen,"client(%d) %s", i, c.GetName())
		// 	return true
		// })
	}
}

// Stop
func (s *Server) Stop() {
	s.mu.Lock()
	listeners := s.lis
	s.lis = nil
	s.mu.Unlock()

	for l := range listeners {
		l.Close()
		Mylog(s.opts.logOpen, "stop accepting at address \n", l.Addr().String())
	}

	// close all connections
	conns := map[int64]*ServerConn{}

	s.conns.Range(func(k, v interface{}) bool {
		i := k.(int64)
		c := v.(*ServerConn)
		conns[i] = c
		return true
	})
	// let GC do the cleanings
	s.conns = nil

	for _, c := range conns {
		c.spbConn.Close()
		Mylog(s.opts.logOpen, "close client \n", c.GetName())
	}

	s.mu.Lock()
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()
}

// OnConnectOption
func OnConnectOption(cb func(WriteCloser) bool) ServerOption {
	return func(o *options) {
		o.onConnect = cb
	}
}

// OnMessageOption
func OnMessageOption(cb func(msg.RelayMsgBuf, WriteCloser)) ServerOption {
	return func(o *options) {
		o.onMessage = cb
	}
}

// OnCloseOption
func OnCloseOption(cb func(WriteCloser)) ServerOption {
	return func(o *options) {
		o.onClose = cb
	}
}

// OnErrorOption
func OnErrorOption(cb func(WriteCloser)) ServerOption {
	return func(o *options) {
		o.onError = cb
	}
}

// BufferSizeOption
func BufferSizeOption(indicator int) ServerOption {
	return func(o *options) {
		o.bufferSize = indicator
	}
}

// LogOpenOption
func LogOpenOption(b bool) ServerOption {
	return func(o *options) {
		o.logOpen = b
	}
}

// MaxConnectionsOption
func MaxConnectionsOption(indicator int) ServerOption {
	return func(o *options) {
		o.maxConnections = indicator
	}
}

// MaxFlowOption
func MaxFlowOption(indicator int) ServerOption {
	return func(o *options) {
		o.maxflow = indicator
	}
}

// Unicast
func (s *Server) Unicast(netid int64, msg *msg.RelayMsgBuf) error {
	v, ok := s.conns.Load(netid)
	if ok {
		return v.(*ServerConn).Write(msg)
	}
	Mylog(s.opts.logOpen, "conn id not found", msg)
	return nil
}

// Broadcast
func (s *Server) Broadcast(msg *msg.RelayMsgBuf) {
	s.conns.Range(func(k, v interface{}) bool {
		c := v.(*ServerConn)
		if err := c.Write(msg); err != nil {
			Mylog(s.opts.logOpen, "broadcast error:", err, "conn id:", k.(int64))
			return false
		}
		return true
	})
}

// GetWriteFlow
func (s *Server) GetWriteFlow() int64 {
	return s.writeFlow
}

// GetReadFlow
func (s *Server) GetReadFlow() int64 {
	return s.readFlow
}

// GetSecondReadFlow
func (s *Server) GetSecondReadFlow() int64 {
	return s.secondReadFlowB
}

// GetSecondWriteFlow
func (s *Server) GetSecondWriteFlow() int64 {
	return s.secondWriteFlowB
}
