package core

// server readloop writeloop handleloop
import (
	"context"
	"io"
	"net"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/golang/protobuf/proto"
	message "github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/cmem"
)

// MsgHandler
type MsgHandler struct {
	message message.RelayMsgBuf
	handler HandlerFunc
}

// WriteCloser
type WriteCloser interface {
	Write(*message.RelayMsgBuf) error
	Close()
}

var (
	GoroutineMap = &sync.Map{}
)

// ServerConn
type ServerConn struct {
	netid   int64
	belong  *Server
	spbConn net.Conn

	once      *sync.Once
	wg        *sync.WaitGroup
	sendCh    chan *message.RelayMsgBuf
	handlerCh chan MsgHandler

	mu        sync.Mutex // guards following
	name      string
	heart     int64
	minAppVer uint16
	ctx       context.Context
	cancel    context.CancelFunc
}

// CreateServerConn
func CreateServerConn(id int64, s *Server, c net.Conn) *ServerConn {
	sc := &ServerConn{
		netid:     id,
		belong:    s,
		spbConn:   c,
		once:      &sync.Once{},
		wg:        &sync.WaitGroup{},
		sendCh:    make(chan *message.RelayMsgBuf, s.opts.bufferSize),
		handlerCh: make(chan MsgHandler, s.opts.bufferSize),
		heart:     time.Now().UnixNano(),
	}
	// context.WithValue get key-value context
	sc.ctx, sc.cancel = context.WithCancel(context.WithValue(s.ctx, serverCtxKey, s))
	sc.name = c.RemoteAddr().String()
	sc.minAppVer = s.opts.minAppVersion
	return sc
}

// ServerFromCtx
func ServerFromCtx(ctx context.Context) (*Server, bool) {
	server, ok := ctx.Value(serverCtxKey).(*Server)
	return server, ok
}

// GetNetID
func (sc *ServerConn) GetNetID() int64 {
	return sc.netid
}

// SetConnName
func (sc *ServerConn) SetConnName(name string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.name = name
}

// GetName
func (sc *ServerConn) GetName() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	name := sc.name
	return name
}

// GetIP
func (sc *ServerConn) GetIP() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	host, _, _ := net.SplitHostPort(sc.name)
	return host
}

// GetPort
func (sc *ServerConn) GetPort() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	_, port, _ := net.SplitHostPort(sc.name)
	return port
}

func (sc *ServerConn) GetLocalAddr() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.spbConn.LocalAddr().String()
}

func (sc *ServerConn) GetRemoteAddr() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.spbConn.RemoteAddr().String()
}

// Start
func (sc *ServerConn) Start() {
	Mylog(sc.belong.opts.logOpen, "server conn start", sc.spbConn.LocalAddr(), "->", sc.spbConn.RemoteAddr(), "\n")
	onConnect := sc.belong.opts.onConnect
	if onConnect != nil {
		onConnect(sc)
	}
	loopers := []func(WriteCloser, *sync.WaitGroup){readLoop, writeLoop, handleLoop}
	strArr := []string{"read", "write", "handle"}
	for i, l := range loopers {
		looper := l
		sc.wg.Add(1)
		sc.belong.goroutine = sc.belong.goAtom.IncrementAndGetNew()
		name := sc.GetName() + strArr[i]
		GoroutineMap.Store(name, strArr[i])
		go looper(sc, sc.wg)
	}
}

// Write
/**
error is caught at application layer, if it's utils.ErrWouldBlock，sleep and then continue write
*/
func (sc *ServerConn) Write(message *message.RelayMsgBuf) error {
	return asyncWrite(sc, message)
}

func asyncWrite(c interface{}, m *message.RelayMsgBuf) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = utils.ErrServerClosed
		}
	}()

	var (
		sendCh chan *message.RelayMsgBuf
	)
	sendCh = c.(*ServerConn).sendCh
	msgH := make([]byte, utils.MsgHeaderLen)
	header.GetMessageHeader(m.MSGHead.Tag, m.MSGHead.Version, m.MSGHead.Len, string(m.MSGHead.Cmd), m.MSGHead.ReqId, msgH)
	// msgData := make([]byte, utils.MessageBeatLen)
	// copy(msgData[0:], msgH)
	// copy(msgData[utils.MsgHeaderLen:], m.MSGData)
	// memory := &message.RelayMsgBuf{
	// 	MSGHead: m.MSGHead,
	// 	MSGData: msgData[0 : m.MSGHead.Len+utils.MsgHeaderLen],
	// }
	memory := &message.RelayMsgBuf{
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

// Close
func (sc *ServerConn) Close() {
	sc.belong.goroutine = sc.belong.goAtom.DecrementAndGetNew()
	sc.once.Do(func() {
		Mylog(sc.belong.opts.logOpen, "conn close gracefully", sc.spbConn.LocalAddr(), " ->", sc.spbConn.RemoteAddr(), "\n")

		// close
		onClose := sc.belong.opts.onClose
		if onClose != nil {
			onClose(sc)
		}

		// close conns
		sc.belong.conns.Delete(sc.netid)
		Mylog(sc.belong.opts.logOpen, "closed", sc.belong.ConnsSize())
		// close net.Conn, any blocked read or write operation will be unblocked and
		// return errors.
		if tc, ok := sc.spbConn.(*net.TCPConn); ok {
			// set connection closer behavior when there are data to be sent or confirmed in the connection:
			// if sec < 0 (default), the data sending will be finished before close.
			// if sec = 0, the data will be dropped
			// if sec > 0, the data sending will continue for <sec> second and then remaining data will be dropped
			tc.SetLinger(0)
		}
		sc.spbConn.Close()
		// cancel readLoop, writeLoop and handleLoop go-routines.
		sc.mu.Lock()
		sc.cancel()
		Mylog(sc.belong.opts.logOpen, "enter close")
		sc.mu.Unlock()

		sc.wg.Wait()

		close(sc.sendCh)
		close(sc.handlerCh)

		sc.belong.wg.Done()
		sc.belong.goroutine = sc.belong.goAtom.DecrementAndGetNew()
	})
}

func (sc *ServerConn) SendBadVersionMsg(version uint16, cmd string) {
	req := &protos.RspBadVersion{
		Version:        int32(version),
		MinimumVersion: int32(sc.minAppVer),
		Command:        cmd,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		utils.ErrorLog(err)
		return
	}

	err = sc.Write(&message.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, sc.minAppVer, uint32(len(data)), header.RspBadVersion, utils.ZeroId()),
		MSGData: data,
	})
	if err != nil {
		utils.ErrorLog(err)
	}
	time.Sleep(500 * time.Millisecond)
	return
}

// readLoop
func readLoop(c WriteCloser, wg *sync.WaitGroup) {
	var (
		spbConn   net.Conn
		cDone     <-chan struct{}
		sDone     <-chan struct{}
		onMessage onMessageFunc
		handlerCh chan MsgHandler
		msg       = new(message.RelayMsgBuf)
		sc        *ServerConn
		err       error
	)

	spbConn = c.(*ServerConn).spbConn
	cDone = c.(*ServerConn).ctx.Done()
	sDone = c.(*ServerConn).belong.ctx.Done()
	onMessage = c.(*ServerConn).belong.opts.onMessage
	handlerCh = c.(*ServerConn).handlerCh
	sc = c.(*ServerConn)
	Mylog(sc.belong.opts.logOpen, "read start")
	defer func() {
		if p := recover(); p != nil {
			Mylog(sc.belong.opts.logOpen, "panics:", p, "\n")
		}
		wg.Done()
		Mylog(sc.belong.opts.logOpen, "server readLoop go-routine exited")
		GoroutineMap.Delete(sc.GetName() + "read")
		c.Close()
	}()

	var msgH header.MessageHead
	i := 0
	for {
		select {
		case <-cDone: // connection closed
			Mylog(sc.belong.opts.logOpen, "read receiving cancel signal from conn")
			return
		case <-sDone: // server closed
			Mylog(sc.belong.opts.logOpen, "read receiving cancel signal from server")
			return
		default:
			spbConn.SetDeadline(time.Now().Add(time.Duration(utils.ReadTimeOut) * time.Second))
			Mylog(sc.belong.opts.logOpen, "server read ok", msgH.Len)
			if msgH.Len == 0 {
				Mylog(sc.belong.opts.logOpen, "server receive msgHeader")
				buffer := make([]byte, utils.MsgHeaderLen)
				n, err := io.ReadFull(spbConn, buffer)
				// Mylog(sc.belong.opts.logOpen, "server header==>", buffer)
				sc.belong.readFlow = sc.belong.readAtom.AddAndGetNew(int64(n))
				sc.belong.secondReadFlowA = sc.belong.secondReadAtomA.AddAndGetNew(int64(n))
				sc.belong.allFlow = sc.belong.allAtom.AddAndGetNew(int64(n))
				if err != nil {
					if err == io.EOF {
						return
					}
					utils.ErrorLog("server header err", err)
					return
				}
				header.NewDecodeHeader(buffer, &msgH)
				buffer = nil

				//when header shows msg length = 0, directly handle msg
				if msgH.Len == 0 {
					// if utils.ByteToString(msgH.Cmd) == header.ReqHeart {
					// 	sc.spbConn.SetDeadline(time.Now().Add(time.Duration(utils.HeartTimeOut) * time.Second))
					// 	handler := GetHandlerFunc(header.RspHeart)
					// 	if handler != nil {
					// 		sc.handlerCh <- MsgHandler{message.RelayMsgBuf{}, handler}
					// 	}
					// } else {
					if msgH.Version < sc.minAppVer {
						sc.SendBadVersionMsg(msgH.Version, utils.ByteToString(msgH.Cmd))
						return
					}
					handler := GetHandlerFunc(utils.ByteToString(msgH.Cmd))
					if handler != nil {
						sc.handlerCh <- MsgHandler{message.RelayMsgBuf{}, handler}
					}
					// }
				}

			} else {
				// start to process msg if there are more than header to read
				if msgH.Len > utils.MessageBeatLen {
					utils.ErrorLog("msgHeader over sized")
					return
				}
				Mylog(sc.belong.opts.logOpen, "start", time.Now().Unix())
				var onereadlen = 1024
				var n int
				msgBuf := make([]byte, utils.MessageBeatLen)
				for ; i < int(msgH.Len); i = i + n {
					if int(msgH.Len)-i < 1024 {
						onereadlen = int(msgH.Len) - i
					}
					spbConn.SetDeadline(time.Now().Add(time.Duration(utils.ReadTimeOut) * time.Second))
					n, err = io.ReadFull(spbConn, msgBuf[i:i+onereadlen])
					// Mylog(s.opts.logOpen,"server n = ", msgBuf[0:msgH.Len])

					// Mylog(s.opts.logOpen,"len(msgBuf[0:msgH.Len]):", i)
					sc.belong.readFlow = sc.belong.readAtom.AddAndGetNew(int64(n))
					sc.belong.secondReadFlowA = sc.belong.secondReadAtomA.AddAndGetNew(int64(n))
					sc.belong.allFlow = sc.belong.allAtom.AddAndGetNew(int64(n))
					if err != nil {
						utils.ErrorLog("server body err", err)
						return
					}
				}
				Mylog(sc.belong.opts.logOpen, "end", time.Now().Unix())
				if uint32(i) == msgH.Len {
					if msgH.Version < sc.minAppVer {
						sc.SendBadVersionMsg(msgH.Version, utils.ByteToString(msgH.Cmd))
						return
					}

					msg = &message.RelayMsgBuf{
						MSGHead: msgH,
						MSGData: msgBuf[0:msgH.Len],
					}

					handler := GetHandlerFunc(utils.ByteToString(msgH.Cmd))
					if handler == nil {
						if onMessage != nil {
							onMessage(*msg, c.(WriteCloser))
						} else {
							Mylog(sc.belong.opts.logOpen, "server no handler or onMessage() found for message", "\n")
						}
						msgH.Len = 0
						i = 0
						msgBuf = nil
						continue
					}
					handlerCh <- MsgHandler{*msg, handler}
					msgH.Len = 0
					i = 0
					msgBuf = nil
					Mylog(sc.belong.opts.logOpen, "server msg receive complete, reported to application layer, msgHeader set to empty")
				} else {
					utils.ErrorLog("msgH.Len size not match")
					msgH.Len = 0
					i = 0
					msgBuf = nil
					return
				}
			}
			// setHeartBeatFunc(time.Now().UnixNano())
		}
	}
}

// writeLoop
func writeLoop(c WriteCloser, wg *sync.WaitGroup) {
	var (
		spbConn net.Conn
		sendCh  chan *message.RelayMsgBuf
		cDone   <-chan struct{}
		sDone   <-chan struct{}
		packet  *message.RelayMsgBuf
		sc      *ServerConn
		err     error
	)

	spbConn = c.(*ServerConn).spbConn
	sendCh = c.(*ServerConn).sendCh
	cDone = c.(*ServerConn).ctx.Done()
	sDone = c.(*ServerConn).belong.ctx.Done()
	sc = c.(*ServerConn)
	defer func() {
		if p := recover(); p != nil {
			Mylog(sc.belong.opts.logOpen, "panics:", p, "\n")
		}
		// drain all pending messages before exit
	OuterFor:
		for {
			select {
			case packet = <-sendCh:
				if packet != nil {
					var onereadlen = 1024
					var n int
					// Mylog(s.opts.logOpen,"msgLen", len(packet.MSGData))
					for i := 0; i < len(packet.MSGData); i = i + n {
						// Mylog(s.opts.logOpen,"len(msgBuf[0:msgH.Len]):", i)
						if len(packet.MSGData)-i < 1024 {
							onereadlen = len(packet.MSGData) - i
							// Mylog(s.opts.logOpen,"onereadlen:", onereadlen)
						}
						n, err = spbConn.Write(packet.MSGData[i : i+onereadlen])
						// Mylog(s.opts.logOpen,"server n = ", msgBuf[0:msgH.Len])
						// Mylog(s.opts.logOpen,"i+onereadlen:", i+onereadlen)
						sc.belong.writeFlow = sc.belong.writeAtom.AddAndGetNew(int64(n))
						sc.belong.secondWriteFlowA = sc.belong.secondWriteAtomA.AddAndGetNew(int64(n))
						sc.belong.allFlow = sc.belong.allAtom.AddAndGetNew(int64(n))
						if err != nil {
							utils.ErrorLog("server write err", err)
							return
						}

						if err != nil {
							utils.ErrorLog("error writing data", err, "\n")
							return
						} else {
							// Mylog(s.opts.logOpen,"i", i)
						}
					}
					cmem.Free(packet.Alloc)
					packet = nil
				}
			default:
				break OuterFor
			}
		}
		wg.Done()
		Mylog(sc.belong.opts.logOpen, "writeLoop go-routine exited")
		GoroutineMap.Delete(sc.GetName() + "write")
		c.Close()
	}()

	for {
		select {
		case <-cDone: // connection closed
			Mylog(sc.belong.opts.logOpen, "write receiving cancel signal from conn")
			return
		case <-sDone: // server closed
			Mylog(sc.belong.opts.logOpen, "write receiving cancel signal from server")
			return
		case packet = <-sendCh:
			if packet != nil {
				var onereadlen = 1024
				var n int
				// Mylog(s.opts.logOpen,"msgLen", len(packet.MSGData))
				for i := 0; i < len(packet.MSGData); i = i + n {
					// Mylog(s.opts.logOpen,"len(msgBuf[0:msgH.Len]):", i)
					if len(packet.MSGData)-i < 1024 {
						onereadlen = len(packet.MSGData) - i
						// Mylog(s.opts.logOpen,"onereadlen:", onereadlen)
					}
					spbConn.SetDeadline(time.Now().Add(time.Duration(utils.WriteTimeOut) * time.Second))
					n, err = spbConn.Write(packet.MSGData[i : i+onereadlen])
					// Mylog(s.opts.logOpen,"server n = ", msgBuf[0:msgH.Len])
					// Mylog(s.opts.logOpen,"i+onereadlen:", i+onereadlen)
					sc.belong.writeFlow = sc.belong.writeAtom.AddAndGetNew(int64(n))
					sc.belong.secondWriteFlowA = sc.belong.secondWriteAtomA.AddAndGetNew(int64(n))
					sc.belong.allFlow = sc.belong.allAtom.AddAndGetNew(int64(n))
					if err != nil {
						utils.ErrorLog("server write err", err)
						return
					}

					if err != nil {
						utils.ErrorLog("error writing data", err, "\n")
						return
					} else {
						// Mylog(s.opts.logOpen,"i", i)
					}
				}
				cmem.Free(packet.Alloc)
				packet = nil
			}
		}
	}
}

// handleLoop
func handleLoop(c WriteCloser, wg *sync.WaitGroup) {
	var (
		cDone <-chan struct{}
		sDone <-chan struct{}
		// timerCh      chan *OnTimeOut
		handlerCh chan MsgHandler
		netID     int64
		ctx       context.Context
		err       error
		sc        *ServerConn
	)

	cDone = c.(*ServerConn).ctx.Done()
	sDone = c.(*ServerConn).belong.ctx.Done()
	// timerCh = c.(*ServerConn).timerCh
	handlerCh = c.(*ServerConn).handlerCh
	netID = c.(*ServerConn).netid
	ctx = c.(*ServerConn).ctx
	sc = c.(*ServerConn)
	defer func() {
		if p := recover(); p != nil {
			Mylog(sc.belong.opts.logOpen, "panics:", p, "\n")
		}
		wg.Done()
		Mylog(sc.belong.opts.logOpen, "handleLoop go-routine exited")
		GoroutineMap.Delete(sc.GetName() + "handle")
		c.Close()
	}()

	for {
		select {
		case <-cDone: // connection closed
			Mylog(sc.belong.opts.logOpen, "handle receiving cancel signal from conn")
			return
		case <-sDone: // server closed
			Mylog(sc.belong.opts.logOpen, "handle receiving cancel signal from server")
			return
		case msgHandler := <-handlerCh:
			msg, handler := msgHandler.message, msgHandler.handler
			if handler != nil {
				// if askForWorker {
				err = GlobalTaskPool.Job(netID, func() {
					// Mylog(s.opts.logOpen,"handler(CreateContextWithNetID(CreateContextWithMessage(ctx, msg), netID), c )", netID)
					handler(CreateContextWithNetID(CreateContextWithMessage(ctx, &msg), netID), c)
				})
				if err != nil {
					utils.ErrorLog(err)
				}
			}
		}
	}
}
