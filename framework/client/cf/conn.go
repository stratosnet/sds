package cf

// client connect management, readloop writeloop handleloop

import (
	"context"
	"errors"
	"io"
	"net"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/utils/cmem"

	"github.com/stratosnet/sds/utils"

	"github.com/alex023/clock"
)

var (
	limitDownloadSpeed   uint64
	limitUploadSpeed     uint64
	isLimitDownloadSpeed bool
	isLimitUploadSpeed   bool
)

// MsgHandler
type MsgHandler struct {
	message msg.RelayMsgBuf
	handler core.HandlerFunc
}

type onConnectFunc func(core.WriteCloser) bool
type onMessageFunc func(msg.RelayMsgBuf, core.WriteCloser)
type onCloseFunc func(core.WriteCloser)
type onErrorFunc func(core.WriteCloser)

type options struct {
	onConnect  onConnectFunc
	onMessage  onMessageFunc
	onClose    onCloseFunc
	onError    onErrorFunc
	bufferSize int
	reconnect  bool // only ClientConn
	heartClose bool
	logOpen    bool
}

// ClientOption client configuration
type ClientOption func(*options)

// ClientConn
type ClientConn struct {
	// TODO to add p2p key usage (handshake)
	addr      string
	opts      options
	netid     int64
	spbConn   net.Conn
	once      *sync.Once
	wg        *sync.WaitGroup
	sendCh    chan *msg.RelayMsgBuf
	handlerCh chan MsgHandler
	// timing    *TimingWheel
	mu   sync.Mutex // guards following
	name string
	// heart   int64
	pending          []int64
	ctx              context.Context
	cancel           context.CancelFunc
	jobs             []clock.Job
	secondReadFlowA  int64
	secondReadFlowB  int64
	secondReadAtomA  *utils.AtomicInt64
	secondReadAtomB  *utils.AtomicInt64
	secondWriteFlowA int64
	secondWriteAtomA *utils.AtomicInt64
	secondWriteFlowB int64
	secondWriteAtomB *utils.AtomicInt64
	inbound          int64              // for traffic log
	inboundAtomic    *utils.AtomicInt64 // for traffic log
	outbound         int64              // for traffic log
	outboundAtomic   *utils.AtomicInt64 // for traffic log
	is_active        bool
}

// ReconnectOption
func ReconnectOption() ClientOption {
	return func(o *options) {
		o.reconnect = true
	}
}

// CreateClientConn
func CreateClientConn(netid int64, c net.Conn, opt ...ClientOption) *ClientConn {
	var opts options
	for _, o := range opt {
		o(&opts)
	}
	return newClientConnWithOptions(netid, c, opts)
}

// BufferSizeOption
func BufferSizeOption(indicator int) ClientOption {
	return func(o *options) {
		o.bufferSize = indicator
	}
}

// HeartCloseOption
func HeartCloseOption(b bool) ClientOption {
	return func(o *options) {
		o.heartClose = b
	}
}

// LogOpenOption
func LogOpenOption(b bool) ClientOption {
	return func(o *options) {
		o.logOpen = b
	}
}

// Mylog my
func Mylog(b bool, v ...interface{}) {
	if b {
		utils.DebugLog(v...)
	}
}

// client
func newClientConnWithOptions(netid int64, c net.Conn, opts options) *ClientConn {
	if opts.bufferSize == 0 {
		opts.bufferSize = 100
	}
	cc := &ClientConn{
		addr:             c.RemoteAddr().String(),
		opts:             opts,
		netid:            netid,
		spbConn:          c,
		once:             &sync.Once{},
		wg:               &sync.WaitGroup{},
		sendCh:           make(chan *msg.RelayMsgBuf, opts.bufferSize),
		handlerCh:        make(chan MsgHandler, opts.bufferSize),
		secondReadAtomA:  utils.CreateAtomicInt64(0),
		secondReadAtomB:  utils.CreateAtomicInt64(0),
		secondWriteAtomA: utils.CreateAtomicInt64(0),
		secondWriteAtomB: utils.CreateAtomicInt64(0),
		inboundAtomic:    utils.CreateAtomicInt64(0),
		outboundAtomic:   utils.CreateAtomicInt64(0),
		is_active:        false,
	}
	cc.ctx, cc.cancel = context.WithCancel(context.Background())
	cc.name = c.RemoteAddr().String()
	cc.pending = []int64{}
	return cc
}

// GetNetID
func (cc *ClientConn) GetNetID() int64 {
	return cc.netid
}

// SetConnName
func (cc *ClientConn) SetConnName(name string) {
	cc.mu.Lock()
	cc.name = name
	cc.mu.Unlock()
}

// GetName
func (cc *ClientConn) GetName() string {
	cc.mu.Lock()
	name := cc.name
	cc.mu.Unlock()
	return name
}

// SetLimitDownloadSpeed
func SetLimitDownloadSpeed(down uint64, isLimitDown bool) {
	limitDownloadSpeed = down
	isLimitDownloadSpeed = isLimitDown
}

// SetLimitUploadSpeed
func SetLimitUploadSpeed(up uint64, isLimitUpload bool) {
	limitUploadSpeed = up
	isLimitUploadSpeed = isLimitUpload
}

// GetIP get connection ip
func (cc *ClientConn) GetIP() string {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	host, _, _ := net.SplitHostPort(cc.name)
	return host
}

// GetPort
func (cc *ClientConn) GetPort() string {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	_, port, _ := net.SplitHostPort(cc.name)
	return port
}

func (cc *ClientConn) GetLocalAddr() string {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.spbConn.LocalAddr().String()
}

func (cc *ClientConn) GetRemoteAddr() string {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.spbConn.RemoteAddr().String()
}

// SetContextValue
func (cc *ClientConn) SetContextValue(k, v interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.ctx = context.WithValue(cc.ctx, k, v)
}

// ContextValue
func (cc *ClientConn) ContextValue(k interface{}) interface{} {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.ctx.Value(k)
}

// Start client start readLoop, writeLoop, handleLoop
func (cc *ClientConn) Start() {
	Mylog(cc.opts.logOpen, "client conn start", cc.spbConn.LocalAddr(), "->", cc.spbConn.RemoteAddr(), "\n")
	onConnect := cc.opts.onConnect
	if onConnect != nil {
		onConnect(cc)
	}
	loopers := []func(core.WriteCloser, *sync.WaitGroup){readLoop, writeLoop, handleLoop}
	for _, l := range loopers {
		looper := l
		cc.wg.Add(1)
		go looper(cc, cc.wg)
	}
	var (
		myClock               = clock.NewClock()
		handler               = core.GetHandlerFunc(header.ReqHeart)
		spLatencyCheckHandler = core.GetHandlerFunc(header.ReqSpLatencyCheck)

		spLatencyCheckJobFunc = func() {
			if spLatencyCheckHandler != nil {
				cc.handlerCh <- MsgHandler{msg.RelayMsgBuf{}, spLatencyCheckHandler}
			}
		}

		jobFunc = func() {
			if handler != nil {
				cc.handlerCh <- MsgHandler{msg.RelayMsgBuf{}, handler}
			}
		}
		logFunc = func() {
			cc.inbound = cc.inboundAtomic.AddAndGetNew(cc.secondReadFlowA)
			cc.outbound = cc.outboundAtomic.AddAndGetNew(cc.secondWriteFlowA)
			cc.secondReadFlowB = cc.secondReadAtomB.GetNewAndSetAtomic(cc.secondReadFlowA)
			cc.secondWriteFlowB = cc.secondWriteAtomB.GetNewAndSetAtomic(cc.secondWriteFlowA)
			cc.secondReadFlowA = cc.secondReadAtomA.GetNewAndSetAtomic(0)
			cc.secondWriteFlowA = cc.secondWriteAtomA.GetNewAndSetAtomic(0)
		}
	)
	if !cc.opts.heartClose {
		hbJob, _ := myClock.AddJobRepeat(time.Second*utils.ClientSendHeartTime, 0, jobFunc)
		cc.jobs = append(cc.jobs, hbJob)
	}
	latencyJob, _ := myClock.AddJobRepeat(time.Second*utils.LatencyCheckSpListInterval, 0, spLatencyCheckJobFunc)
	cc.jobs = append(cc.jobs, latencyJob)
	logJob, _ := myClock.AddJobRepeat(time.Second*1, 0, logFunc)
	cc.jobs = append(cc.jobs, logJob)
}

// ClientClose Actively closes the client connection
func (cc *ClientConn) ClientClose() {
	cc.is_active = true
	cc.once.Do(func() {
		Mylog(cc.opts.logOpen, "client close connection", cc.spbConn.LocalAddr(), "->", cc.spbConn.RemoteAddr(), "\n")

		// callback on close
		onClose := cc.opts.onClose
		if onClose != nil {
			onClose(cc)
			cc.is_active = false
		}

		// close net.Conn
		cc.spbConn.Close()

		// cancel readLoop, writeLoop and handleLoop go-routines.
		cc.mu.Lock()
		cc.cancel()
		cc.pending = nil
		cc.mu.Unlock()

		// wait until all go-routines exited.
		cc.wg.Wait()

		utils.DebugLog("cc.wg.Wait() finished")

		// close all channels.
		close(cc.sendCh)
		close(cc.handlerCh)
		if len(cc.jobs) > 0 {
			utils.DebugLogf("cancel %v jobs, %v", len(cc.jobs), cc.GetName())
			for _, job := range cc.jobs {
				job.Cancel()
			}
		}
	})
}

// Close
func (cc *ClientConn) Close() {
	cc.once.Do(func() {
		Mylog(cc.opts.logOpen, "conn close gracefully", cc.spbConn.LocalAddr(), "->", cc.spbConn.RemoteAddr(), "\n")

		// callback on close
		onClose := cc.opts.onClose
		if onClose != nil {
			onClose(cc)
		}

		// close net.Conn
		cc.spbConn.Close()

		// cancel readLoop, writeLoop and handleLoop go-routines.
		cc.mu.Lock()
		cc.cancel()
		cc.pending = nil
		cc.mu.Unlock()

		// wait until all go-routines exited.
		cc.wg.Wait()

		// close all channels.
		close(cc.sendCh)
		close(cc.handlerCh)
		if len(cc.jobs) > 0 {
			utils.DebugLogf("cancel %v jobs, %v", len(cc.jobs), cc.GetName())
			for _, job := range cc.jobs {
				job.Cancel()
			}
		}
		if cc.opts.reconnect {
			cc.reconnect()
		}
	})
}

// reconnect
func (cc *ClientConn) reconnect() {
	var c net.Conn
	var err error
	c, err = net.Dial("tcp", cc.addr)
	if err != nil {
		if p := recover(); p != nil {
			Mylog(cc.opts.logOpen, "net dial error", err)
		}
		return
	}
	*cc = *newClientConnWithOptions(cc.netid, c, cc.opts)
	cc.Start()
}

// Write ,
// func (cc *ClientConn) Write(message msg.RelayMsgBuf) error {
// 	return asyncWrite(cc, message)
// }

// func asyncWrite(c interface{}, m msg.RelayMsgBuf) (err error) {
// 	defer func() {
// 		if p := recover(); p != nil {
// 			err = utils.ErrServerClosed
// 		}
// 	}()

// 	var (
// 		sendCh chan []byte
// 	)
// 	sendCh = c.(*ClientConn).sendCh
// 	msgH := header.GetMessageHeader(m.MSGHead.Tag, m.MSGHead.Version, m.MSGHead.Len, string(m.MSGHead.Cmd))
// 	m.MSGData = append(msgH, m.MSGData...)

// 	select {
// 	case sendCh <- m.MSGData:
// 		err = nil
// 	default:
// 		err = utils.ErrWouldBlock
// 	}

// 	if err != nil {
// 		Mylog(cc.opts.logOpen,"asyncWrite error ", err)
// 		return
// 	}

// 	return
// }

// Write
func (cc *ClientConn) Write(message *msg.RelayMsgBuf) error {
	return asyncWrite(cc, message)
}

func asyncWrite(c *ClientConn, m *msg.RelayMsgBuf) (err error) {
	if c == nil {
		return errors.New("nil client connection")
	}
	defer func() {
		if p := recover(); p != nil {
			err = utils.ErrServerClosed
		}
	}()

	var (
		sendCh chan *msg.RelayMsgBuf
	)
	sendCh = c.sendCh
	msgH := make([]byte, 16)
	header.GetMessageHeader(m.MSGHead.Tag, m.MSGHead.Version, m.MSGHead.Len, string(m.MSGHead.Cmd), msgH)
	// msgData := make([]byte, utils.MessageBeatLen)
	// copy((*msgData)[0:], msgH)
	// copy((*msgData)[utils.MsgHeaderLen:], m.MSGData)
	// memory := &msg.RelayMsgBuf{
	// 	MSGHead: m.MSGHead,
	// 	MSGData: (*msgData)[0 : m.MSGHead.Len+utils.MsgHeaderLen],
	// }
	memory := &msg.RelayMsgBuf{
		MSGHead: m.MSGHead,
	}
	memory.Alloc = cmem.Alloc(uintptr(m.MSGHead.Len + utils.MsgHeaderLen))
	memory.MSGData = (*[1 << 30]byte)(unsafe.Pointer(memory.Alloc))[:m.MSGHead.Len+utils.MsgHeaderLen]
	(*reflect.SliceHeader)(unsafe.Pointer(&memory.MSGData)).Cap = int(m.MSGHead.Len + utils.MsgHeaderLen)
	copy(memory.MSGData[0:], msgH)
	copy(memory.MSGData[utils.MsgHeaderLen:], m.MSGData)
	select {
	case sendCh <- memory:
		err = nil
		// default:
		// 	err = utils.ErrWouldBlock
	}

	if err != nil {
		utils.ErrorLog("asyncWrite error ", err)
		memory = nil
		return
	}

	return
}

// readLoop
func readLoop(c core.WriteCloser, wg *sync.WaitGroup) {
	var (
		spbConn net.Conn
		cDone   <-chan struct{}
		// setHeartBeatFunc func(int64)
		onMessage onMessageFunc
		handlerCh chan MsgHandler
		message   = new(msg.RelayMsgBuf)
		err       error
		cc        *ClientConn
	)
	cc = c.(*ClientConn)
	spbConn = c.(*ClientConn).spbConn
	cDone = c.(*ClientConn).ctx.Done()
	onMessage = c.(*ClientConn).opts.onMessage
	handlerCh = c.(*ClientConn).handlerCh
	Mylog(cc.opts.logOpen, "read start")
	defer func() {
		if p := recover(); p != nil {
			Mylog(cc.opts.logOpen, "panics:", p, "\n")
		}
		wg.Done()
		Mylog(cc.opts.logOpen, "client readLoop go-routine exited")
		if !cc.is_active {
			c.Close()
		}
	}()

	// var msgBuf []byte
	var msgH header.MessageHead
	i := 0
	var lr utils.LimitRate

	for {
		select {
		case <-cDone: // connection closed
			Mylog(cc.opts.logOpen, "receiving cancel signal from conn")
			return
		default:
			// Mylog(cc.opts.logOpen,"client read ok", msgH.Len)
			if msgH.Len == 0 {
				buffer := make([]byte, utils.MsgHeaderLen)
				n, err := io.ReadFull(spbConn, buffer)
				cc.secondReadFlowA = cc.secondReadAtomA.AddAndGetNew(int64(n))
				if err != nil {
					utils.ErrorLog("client heart err", err)
					return
				}
				header.NewDecodeHeader(buffer, &msgH)
				buffer = nil
				// Mylog(cc.opts.logOpen,"client msg size", msgH.Cmd)
				if msgH.Len == 0 {
					handler := core.GetHandlerFunc(utils.ByteToString(msgH.Cmd))
					if handler != nil {
						handlerCh <- MsgHandler{msg.RelayMsgBuf{}, handler}
					}
				}
			} else {

				var onereadlen = 1024
				var n int
				msgBuf := make([]byte, 0, utils.MessageBeatLen)
				cmd := utils.ByteToString(msgH.Cmd)
				for ; i < int(msgH.Len); i = i + n {
					if int(msgH.Len)-i < 1024 {
						onereadlen = int(msgH.Len) - i
					}
					n, err = io.ReadFull(spbConn, msgBuf[i:i+onereadlen])
					cc.secondReadFlowA = cc.secondReadAtomA.AddAndGetNew(int64(n))
					if err != nil {
						utils.ErrorLog("client body err", err)
						return
					}
					if cmd == header.RspDownloadSlice {
						if isLimitDownloadSpeed {
							if limitDownloadSpeed > 0 {
								lr.SetRate(limitDownloadSpeed)
								lr.Limit()
							}
						}
					}
				}
				if uint32(i) == msgH.Len {
					message = &msg.RelayMsgBuf{
						MSGHead: msgH,
						MSGData: msgBuf[0:msgH.Len],
					}
					handler := core.GetHandlerFunc(utils.ByteToString(msgH.Cmd))
					//Mylog(cc.opts.logOpen, "read handler:", handler, utils.ByteToString(msgH.Cmd))
					if handler == nil {
						if onMessage != nil {
							Mylog(cc.opts.logOpen, "client message", message, " call onMessage()\n")
							onMessage(*message, c.(core.WriteCloser))
						} else {
							// Mylog(cc.opts.logOpen,"client no handler or onMessage() found for message\n")
						}
						msgH.Len = 0
						i = 0
						msgBuf = nil
						continue
					}
					handlerCh <- MsgHandler{*message, handler}
					msgH.Len = 0
					msgBuf = nil
					i = 0

				} else {
					utils.ErrorLog("msgHeader length not match")
					msgH.Len = 0
					msgBuf = nil
					return
				}
			}
			// setHeartBeatFunc(time.Now().UnixNano())
		}
	}
}

// writeLoop
func writeLoop(c core.WriteCloser, wg *sync.WaitGroup) {
	var (
		spbConn net.Conn
		sendCh  chan *msg.RelayMsgBuf
		cDone   <-chan struct{}
		packet  *msg.RelayMsgBuf
		err     error
		cc      *ClientConn
	)
	cc = c.(*ClientConn)
	spbConn = c.(*ClientConn).spbConn
	sendCh = c.(*ClientConn).sendCh
	cDone = c.(*ClientConn).ctx.Done()
	defer func() {
		if p := recover(); p != nil {
			Mylog(cc.opts.logOpen, "panics:", p, "\n")
		}
		// OuterFor:
		// 	for {
		// 		select {
		// 		case packet = <-sendCh:
		// 			if packet != nil {
		// 				var onereadlen = 1024
		// 				var n int
		// 				// Mylog(cc.opts.logOpen,"msgLen", len(packet.MSGData))
		// 				for i := 0; i < len(packet.MSGData); i = i + n {
		// 					// Mylog(cc.opts.logOpen,"len(msgBuf[0:msgH.Len]):", i)
		// 					if len(packet.MSGData)-i < 1024 {
		// 						onereadlen = len(packet.MSGData) - i
		// 						// Mylog(cc.opts.logOpen,"onereadlen:", onereadlen)
		// 					}
		// 					n, err = spbConn.Write(packet.MSGData[i : i+onereadlen])
		// 					// Mylog(cc.opts.logOpen,"server n = ", msgBuf[0:msgH.Len])
		// 					// Mylog(cc.opts.logOpen,"i+onereadlen:", i+onereadlen)
		// 					if err != nil {
		// 						utils.ErrorLog("client body err", err)
		// 						return
		// 					}
		// 				}
		// 				runtime.SetFinalizer(packet, func(item *msg.RelayMsgBuf) {
		// 					if item != nil {
		// 						cmem.Free(packet.Alloc)
		// 						item = nil
		// 					}
		// 				})
		// 			}
		// 		default:
		// 			break OuterFor
		// 		}
		// 	}
		wg.Done()
		Mylog(cc.opts.logOpen, "writeLoop go-routine exited")
		if !cc.is_active {
			c.Close()
		}
	}()

	var lr utils.LimitRate
	for {
		select {
		case <-cDone: // connection closed
			Mylog(cc.opts.logOpen, "receiving cancel signal from conn")
			return
		case packet = <-sendCh:
			if packet != nil {
				// _, err = spbConn.Write(packet.MSGData)
				// spbConn.SetWriteDeadline(time.Now().Add(time.Duration(utils.WriteTimeOut) * time.Second))
				// if err != nil {
				// 	utils.ErrorLog("error writing data", err, "\n")
				// 	return
				// }
				var onereadlen = 1024
				var n int
				// Mylog(cc.opts.logOpen, "write header", packet.MSGData[:16])
				// Mylog(cc.opts.logOpen, "write body", packet.MSGData[16:])
				cmd := utils.ByteToString(packet.MSGHead.Cmd)
				for i := 0; i < len(packet.MSGData); i = i + n {
					// Mylog(cc.opts.logOpen,"len(msgBuf[0:msgH.Len]):", i)
					if len(packet.MSGData)-i < 1024 {
						onereadlen = len(packet.MSGData) - i
						// Mylog(cc.opts.logOpen,"onereadlen:", onereadlen)
					}
					n, err = spbConn.Write(packet.MSGData[i : i+onereadlen])
					cc.secondWriteFlowA = cc.secondWriteAtomA.AddAndGetNew(int64(n))
					// Mylog(cc.opts.logOpen,"server n = ", msgBuf[0:msgH.Len])
					// Mylog(cc.opts.logOpen,"i+onereadlen:", i+onereadlen)
					if err != nil {
						utils.ErrorLog("client body err", err)
						return
					}
					if cmd == header.ReqUploadFileSlice {
						if isLimitUploadSpeed {
							if limitUploadSpeed > 0 {
								lr.SetRate(limitUploadSpeed)
								lr.Limit()
							}
						}
					}
				}
				// runtime.SetFinalizer(packet, func(item *msg.RelayMsgBuf) {
				cmem.Free(packet.Alloc)
				packet = nil
				// })
			}
		}
	}
}

// handleLoop
func handleLoop(c core.WriteCloser, wg *sync.WaitGroup) {
	var (
		cDone     <-chan struct{}
		handlerCh chan MsgHandler
		netID     int64
		ctx       context.Context
		cc        *ClientConn
	)
	cc = c.(*ClientConn)
	cDone = c.(*ClientConn).ctx.Done()
	handlerCh = c.(*ClientConn).handlerCh
	netID = c.(*ClientConn).netid
	ctx = c.(*ClientConn).ctx

	defer func() {
		if p := recover(); p != nil {
			Mylog(cc.opts.logOpen, "panics:", p, "\n")
		}
		wg.Done()
		Mylog(cc.opts.logOpen, "handleLoop go-routine exited")
		if !cc.is_active {
			c.Close()
		}
	}()

	for {
		select {
		case <-cDone: // connection closed
			Mylog(cc.opts.logOpen, "receiving cancel signal from conn")
			return
		case msgHandler := <-handlerCh:
			msg, handler := msgHandler.message, msgHandler.handler
			handler(core.CreateContextWithNetID(core.CreateContextWithMessage(ctx, &msg), netID), c)
		}
	}
}

// OnConnectOption
func OnConnectOption(cb func(core.WriteCloser) bool) ClientOption {
	return func(o *options) {
		o.onConnect = cb
	}
}

// OnMessageOption
func OnMessageOption(cb func(msg.RelayMsgBuf, core.WriteCloser)) ClientOption {
	return func(o *options) {
		o.onMessage = cb
	}
}

// OnCloseOption
func OnCloseOption(cb func(core.WriteCloser)) ClientOption {
	return func(o *options) {
		o.onClose = cb
	}
}

// OnErrorOption
func OnErrorOption(cb func(core.WriteCloser)) ClientOption {
	return func(o *options) {
		o.onError = cb
	}
}

// GetSecondReadFlow
func (cc *ClientConn) GetSecondReadFlow() int64 {
	return cc.secondReadFlowB
}

// GetSecondWriteFlow
func (cc *ClientConn) GetSecondWriteFlow() int64 {
	return cc.secondWriteFlowB
}

func (cc *ClientConn) GetInboundAndReset() int64 {
	ret := cc.inbound
	cc.inbound = cc.inboundAtomic.GetNewAndSetAtomic(0)
	return ret
}

func (cc *ClientConn) GetOutboundAndReset() int64 {
	ret := cc.outbound
	cc.outbound = cc.outboundAtomic.GetNewAndSetAtomic(0)
	return ret
}
