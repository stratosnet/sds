package core

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alex023/clock"

	"github.com/stratosnet/sds/framework/metrics"
	"github.com/stratosnet/sds/framework/msg"
	"github.com/stratosnet/sds/framework/utils"
)

var (
	netID *utils.AtomicInt64
)

func init() {
	netID = utils.CreateAtomicInt64(0)
}

type onConnectFunc func(WriteCloser) bool
type onMessageFunc func(msg.RelayMsgBuf, WriteCloser)
type onCloseFunc func(WriteCloser)
type onErrorFunc func(WriteCloser)
type onBadAppVerFunc func(version uint16, cmd uint8, minAppVer uint16) []byte
type ContextKV struct {
	Key   interface{}
	Value interface{}
}
type options struct {
	onConnect      onConnectFunc
	onMessage      onMessageFunc
	onClose        onCloseFunc
	onError        onErrorFunc
	onBadAppVer    onBadAppVerFunc
	bufferSize     int
	logOpen        bool
	maxConnections int
	maxflow        int
	minAppVersion  uint16
	p2pAddress     string
	contextkv      []ContextKV
	readTimeout    int64
}

type ServerOption func(*options)

type Server struct {
	opts       options
	ctx        context.Context
	cancel     context.CancelFunc
	conns      *connPool
	wg         *sync.WaitGroup
	mu         sync.Mutex // lock
	lis        map[net.Listener]bool
	goroutine  int64
	goAtom     *utils.AtomicInt64
	volRecOpts volRecOpts
}

const (
	LOG_MODULE_SERVER     = "server: "
	LOG_MODULE_START      = "start: "
	LOG_MODULE_WRITELOOP  = "writeLoop: "
	LOG_MODULE_READLOOP   = "readLoop: "
	LOG_MODULE_HANDLELOOP = "handleLoop: "
	LOG_MODULE_CLOSE      = "close: "
)

func Mylog(b bool, module string, v ...interface{}) {
	if b {
		utils.DebugLogfWithCalldepth(5, "Server Conn: "+module+"%v", v...)
	}
}

func CreateServer(opt ...ServerOption) *Server {
	var opts options
	for _, o := range opt {
		o(&opts)
	}

	// initiates go-routine pool instance
	GlobalTaskPool = makeTaskPool(0)

	s := &Server{
		opts:   opts,
		conns:  newConnPool(),
		wg:     &sync.WaitGroup{},
		lis:    make(map[net.Listener]bool),
		goAtom: utils.CreateAtomicInt64(0),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	for _, kv := range s.opts.contextkv {
		s.ctx = context.WithValue(s.ctx, kv.Key, kv.Value)
	}
	return s
}

func (s *Server) SetVolRecOptions(opt ...ServerVolRecOption) {
	var opts volRecOpts
	for _, o := range opt {
		o(&opts)
	}
	s.volRecOpts = opts
}

func (s *Server) ConnsSize() int {
	if s.conns == nil {
		return 0
	}
	return int(s.conns.Count())
}

func (s *Server) AddVolumeLogJob(logAll bool, logRead bool, logWrite bool, logInbound bool, logOutbound bool) {
	var (
		logFunc = func() {
			// utils.Log("connsSize:", s.ConnsSize(), "routine num:", s.goroutine, "allread:", fmt.Sprintf("%.4f", float64(s.readFlow)/1024/1024), "MB", "allwrite:", fmt.Sprintf("%.4f", float64(s.writeFlow)/1024/1024), "MB", "all:", fmt.Sprintf("%.4f", float64(s.allFlow)/1024/1024), "MB",
			// "read/s:", fmt.Sprintf("%.4f", float64(s.secondReadFlow)/1024/1024), "MB", "write/s:", fmt.Sprintf("%.4f", float64(s.secondWriteFlow)/1024/1024), "MB")
			if logAll {
				s.volRecOpts.allFlow = s.volRecOpts.allAtom.GetNewAndSetAtomic(0)
			}
			if logRead {
				s.volRecOpts.secondReadFlowB = s.volRecOpts.secondReadAtomB.GetNewAndSetAtomic(s.volRecOpts.secondReadFlowA)
				s.volRecOpts.secondReadFlowA = s.volRecOpts.secondReadAtomA.GetNewAndSetAtomic(0)
			}
			if logWrite {
				s.volRecOpts.secondWriteFlowB = s.volRecOpts.secondWriteAtomB.GetAndSetAtomic(s.volRecOpts.secondWriteFlowA)
				s.volRecOpts.secondWriteFlowA = s.volRecOpts.secondWriteAtomA.GetNewAndSetAtomic(0)
			}
			if logInbound {
				s.volRecOpts.inbound = s.volRecOpts.inboundAtomic.GetNewAndSetAtomic(0)
			}
			if logOutbound {
				s.volRecOpts.outbound = s.volRecOpts.outboundAtomic.GetNewAndSetAtomic(0)
			}
		}
	)
	//Assign the value of secondRead/WriteFlowA to secondRead/WriteFlowB for monitor use, then reset secondRead/WriteFlowA to 0
	if logAll || logRead || logWrite || logInbound || logOutbound {
		var myClock = clock.NewClock()
		myClock.AddJobRepeat(time.Second*1, 0, logFunc)
	}
}

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

	Mylog(s.opts.logOpen, LOG_MODULE_SERVER, fmt.Sprintf("server start, net %v addr %v ", l.Addr().Network(), l.Addr().String()))

	onStartLog := s.volRecOpts.onStartLog
	if onStartLog != nil {
		onStartLog(s)
	}

	var tempDelay time.Duration
	for {
		spbConn, err := l.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					continue
				} else if strings.Contains(netErr.Error(), syscall.EMFILE.Error()) {
					if tempDelay == 0 {
						tempDelay = 5 * time.Millisecond
					} else {
						tempDelay *= 2
					}
					if max := 1 * time.Second; tempDelay >= max {
						tempDelay = max
					}
					utils.ErrorLogf("accept error %v, retrying in %d\n", err, tempDelay)
					select {
					case <-time.After(tempDelay):
					case <-s.ctx.Done():
					}
					continue
				}
			}
			utils.ErrorLog("accept err:", err)
			return err
		}
		tcpConn, ok := spbConn.(*net.TCPConn)
		if ok {
			_ = tcpConn.SetNoDelay(false)
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
		metrics.ConnReconnection.WithLabelValues(strings.Split(sc.GetName(), ":")[0]).Inc()
		metrics.ConnNumbers.WithLabelValues("server").Inc()

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

		Mylog(s.opts.logOpen, LOG_MODULE_SERVER, fmt.Sprintf("accepted client %v id: %v total: %v", sc.GetName(), netid, s.ConnsSize()))
		// s.conns.Range(func(k, v interface{}) bool {
		// 	i := k.(int64)
		// 	c := v.(*ServerConn)
		// 	Mylog(s.opts.logOpen,"client(%d) %s", i, c.GetName())
		// 	return true
		// })
	}
}

func (s *Server) Stop() {
	s.mu.Lock()
	listeners := s.lis
	s.lis = nil
	s.mu.Unlock()

	for l := range listeners {
		l.Close()
		Mylog(s.opts.logOpen, LOG_MODULE_SERVER, fmt.Sprintf("stop accepting at address %v", l.Addr().String()))
	}

	// close all connections
	conns := map[int64]*ServerConn{}

	s.conns.Range(func(id int64, conn *ServerConn) bool {
		conns[id] = conn
		return true
	})
	// let GC do the cleanings
	s.conns = nil

	for _, c := range conns {
		c.spbConn.Close()
	}
	Mylog(s.opts.logOpen, LOG_MODULE_SERVER, fmt.Sprintf("closed connection cnt: %v", len(conns)))

	s.mu.Lock()
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()
}

func OnConnectOption(cb func(WriteCloser) bool) ServerOption {
	return func(o *options) {
		o.onConnect = cb
	}
}

func OnMessageOption(cb func(msg.RelayMsgBuf, WriteCloser)) ServerOption {
	return func(o *options) {
		o.onMessage = cb
	}
}

func OnCloseOption(cb func(WriteCloser)) ServerOption {
	return func(o *options) {
		o.onClose = cb
	}
}

func OnErrorOption(cb func(WriteCloser)) ServerOption {
	return func(o *options) {
		o.onError = cb
	}
}

func OnBadAppVerOption(cb func(uint16, uint8, uint16) []byte) ServerOption {
	return func(o *options) {
		o.onBadAppVer = cb
	}
}

func BufferSizeOption(indicator int) ServerOption {
	return func(o *options) {
		o.bufferSize = indicator
	}
}

func LogOpenOption(b bool) ServerOption {
	return func(o *options) {
		o.logOpen = b
	}
}

func MaxConnectionsOption(indicator int) ServerOption {
	return func(o *options) {
		o.maxConnections = indicator
	}
}

func ContextKVOption(kv []ContextKV) ServerOption {
	return func(o *options) {
		o.contextkv = kv
	}
}

func MaxFlowOption(indicator int) ServerOption {
	return func(o *options) {
		o.maxflow = indicator
	}
}

func MinAppVersionOption(minAppVersion uint16) ServerOption {
	return func(o *options) {
		o.minAppVersion = minAppVersion
	}
}

func OnStartLogOption(cb func(*Server)) ServerVolRecOption {
	return func(o *volRecOpts) {
		o.onStartLog = cb
	}
}

func P2pAddressOption(p2pAddress string) ServerOption {
	return func(o *options) {
		o.p2pAddress = p2pAddress
	}
}

func ReadDeadlineOption(timeout int64) ServerOption {
	return func(o *options) {
		o.readTimeout = timeout
	}
}
func (s *Server) Unicast(ctx context.Context, netid int64, msg *msg.RelayMsgBuf) error {
	v, ok := s.conns.Load(netid)
	if ok {
		return v.Write(msg, ctx)
	}
	Mylog(s.opts.logOpen, LOG_MODULE_WRITELOOP, fmt.Sprintf("conn id not found: %v", msg))
	return nil
}

func (s *Server) Broadcast(msg *msg.RelayMsgBuf) {
	s.conns.Range(func(id int64, conn *ServerConn) bool {
		if err := conn.Write(msg, context.Background()); err != nil {
			Mylog(s.opts.logOpen, LOG_MODULE_WRITELOOP, fmt.Sprintf("broadcast error: %v conn id:%v", err.Error(), id))
			return false
		}
		return true
	})
}

//nolint:unused
func (s *Server) GetWriteFlow() int64 {
	return s.volRecOpts.writeFlow
}

//nolint:unused
func (s *Server) GetReadFlow() int64 {
	return s.volRecOpts.readFlow
}

func (s *Server) GetSecondReadFlow() int64 {
	return s.volRecOpts.secondReadFlowB
}

func (s *Server) GetSecondWriteFlow() int64 {
	return s.volRecOpts.secondWriteFlowB
}

func (s *Server) GetInboundAndReset() int64 {
	ret := s.volRecOpts.inbound
	s.volRecOpts.inbound = s.volRecOpts.inboundAtomic.GetNewAndSetAtomic(0)
	return ret
}

func (s *Server) GetOutboundAndReset() int64 {
	ret := s.volRecOpts.outbound
	s.volRecOpts.outbound = s.volRecOpts.outboundAtomic.GetNewAndSetAtomic(0)
	return ret
}
