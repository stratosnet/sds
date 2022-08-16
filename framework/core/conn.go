package core

// server readloop writeloop handleloop
import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/metrics"
	message "github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/cmem"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/encryption"
	"github.com/stratosnet/sds/utils/types"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
)

// MsgHandler
type MsgHandler struct {
	message message.RelayMsgBuf
	handler HandlerFunc
}

// WriteCloser
type WriteCloser interface {
	Write(*message.RelayMsgBuf, context.Context) error
	Close()
}

var (
	GoroutineMap     = &sync.Map{}
	HandshakeChanMap = &sync.Map{} // map[string]chan []byte    Map that stores channels used during handshake process
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

	mu    sync.Mutex // guards following
	name  string
	heart int64

	minAppVer        uint16
	sharedKey        []byte // ECDH shared key derived during handshake
	remoteP2pAddress string

	ctx    context.Context
	cancel context.CancelFunc
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

func (sc *ServerConn) GetLocalP2pAddress() string {
	return sc.belong.opts.p2pAddress
}

func (sc *ServerConn) GetRemoteP2pAddress() string {
	return sc.remoteP2pAddress
}

func (sc *ServerConn) handshake() (error, bool) {
	// Set handshake timeout
	if err := sc.spbConn.SetDeadline(time.Now().Add(time.Duration(utils.HandshakeTimeOut) * time.Second)); err != nil {
		return err, false
	}

	// This is either a client creating a connection, or a temporary connection made for a handshake
	// Read the first message from the connection. It should indicate what kind of connection it is
	buffer := make([]byte, ConnFirstMsgSize)
	if _, err := io.ReadFull(sc.spbConn, buffer); err != nil {
		return err, false
	}
	connType, serverPort, channelId, err := ParseFirstMessage(buffer)
	if err != nil {
		return err, false
	}

	switch connType {
	case ConnTypeClient:
		host, _, err := net.SplitHostPort(sc.GetRemoteAddr())
		if err != nil {
			return err, false
		}
		remoteServer := host + ":" + strconv.FormatUint(uint64(serverPort), 10)

		// Open a new tcp connection to the remote addr from current conn
		handshakeAddr, err := net.ResolveTCPAddr("tcp4", remoteServer)
		if err != nil {
			utils.ErrorLog("Couldn't resolve TCP address", err)
			return err, false
		}
		handshakeConn, err := net.DialTCP("tcp", nil, handshakeAddr)
		if err != nil {
			utils.ErrorLog("DialTCP failed for new connection handshake", err)
			return err, false
		}
		if err = handshakeConn.SetDeadline(time.Now().Add(time.Duration(utils.HandshakeTimeOut) * time.Second)); err != nil {
			return err, false
		}

		// Write the connection type as first message
		firstMessage := CreateFirstMessage(ConnTypeHandshake, 0, channelId)
		if err = WriteFull(handshakeConn, firstMessage); err != nil {
			return err, false
		}

		// Create tmp key
		tmpPrivKeyBytes := ed25519.NewKey()
		tmpPrivKey := ed25519.PrivKeyBytesToPrivKey(tmpPrivKeyBytes)
		tmpPubKeyBytes := ed25519.PrivKeyBytesToPubKeyBytes(tmpPrivKeyBytes)

		// Write tmp key to handshake conn
		handshakeSignature, err := tmpPrivKey.Sign([]byte(HandshakeMessage))
		if err != nil {
			return err, false
		}
		if err = WriteFull(handshakeConn, append(tmpPubKeyBytes, handshakeSignature...)); err != nil {
			return err, false
		}

		// Read tmp key from original conn and verify
		buffer = make([]byte, tmed25519.PubKeySize+tmed25519.SignatureSize)
		if _, err = io.ReadFull(sc.spbConn, buffer); err != nil {
			return err, false
		}
		peerPubKeyBytes := buffer[:tmed25519.PubKeySize]
		peerPubKey := ed25519.PubKeyBytesToPubKey(peerPubKeyBytes)
		peerSignature := buffer[tmed25519.PubKeySize:]
		if !peerPubKey.VerifySignature([]byte(HandshakeMessage), peerSignature) {
			return errors.New("Invalid signature in tmp key from peer"), false
		}

		// ECDH. Store encryption info in conn
		sharedPrivKeyBytes, err := encryption.ECDH(tmpPrivKeyBytes, peerPubKeyBytes)
		if err != nil {
			return err, false
		}
		sc.sharedKey = sharedPrivKeyBytes

		// Send local p2p address
		encryptedMsg, err := EncryptAndPack(sharedPrivKeyBytes, []byte(sc.GetLocalP2pAddress()))
		if err != nil {
			return err, false
		}
		if err = WriteFull(sc.spbConn, encryptedMsg); err != nil {
			return err, false
		}

		// Read remote p2p address
		p2pAddressBytes, _, err := ReadEncryptedHeaderAndBody(sc.spbConn, sharedPrivKeyBytes, utils.MessageBeatLen)
		if err != nil {
			return err, false
		}
		sc.remoteP2pAddress = string(p2pAddressBytes)
		if _, err = types.P2pAddressFromBech(sc.remoteP2pAddress); err != nil {
			return errors.Wrap(err, "incorrect P2pAddress"), false
		}

		_ = handshakeConn.Close()
	case ConnTypeHandshake:
		// Read tmp key from conn
		buffer = make([]byte, tmed25519.PubKeySize+tmed25519.SignatureSize)
		if _, err := io.ReadFull(sc.spbConn, buffer); err != nil {
			return err, false
		}

		// Write tmp key to channel for the corresponding client conn
		value, ok := HandshakeChanMap.Load(strconv.FormatUint(uint64(channelId), 10))
		if !ok {
			return errors.Errorf("No corresponding client conn was found for %v", sc.GetLocalAddr()), false
		}
		clientChan := value.(chan []byte)
		select {
		case clientChan <- buffer:
			return nil, true
		case <-time.After(utils.HandshakeTimeOut * time.Second):
			return errors.New("Timed out when writing to client channel"), false
		}
	default:
		return errors.Errorf("Invalid connection type [%v]", string(buffer)), false
	}

	return sc.spbConn.SetDeadline(time.Time{}), false // Remove handshake timeout
}

// Start server starts readLoop, writeLoop, handleLoop
func (sc *ServerConn) Start() {
	err, isHandshakeConn := sc.handshake()
	if err != nil {
		utils.ErrorLog("server conn handshake error", sc.spbConn.LocalAddr(), "->", sc.spbConn.RemoteAddr(), err)
		sc.Close()
		return
	}
	if isHandshakeConn {
		sc.Close()
		return
	}

	Mylog(sc.belong.opts.logOpen, fmt.Sprintf("server conn start %v -> %v (%v)", sc.spbConn.LocalAddr(), sc.spbConn.RemoteAddr(), sc.remoteP2pAddress))
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
func (sc *ServerConn) Write(message *message.RelayMsgBuf, ctx context.Context) error {
	return asyncWrite(sc, message, ctx)
}

func asyncWrite(c interface{}, m *message.RelayMsgBuf, ctx context.Context) (err error) {
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
	reqId := GetReqIdFromContext(ctx)
	header.GetMessageHeader(m.MSGHead.Tag, m.MSGHead.Version, m.MSGHead.Len, string(m.MSGHead.Cmd), reqId, msgH)
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
		Mylog(sc.belong.opts.logOpen, fmt.Sprintf("conn close gracefully %v -> %v (%v)", sc.spbConn.LocalAddr(), sc.spbConn.RemoteAddr(), sc.remoteP2pAddress))

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
		metrics.ConnNumbers.WithLabelValues("server").Dec()
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
		MSGHead: header.MakeMessageHeader(1, sc.minAppVer, uint32(len(data)), header.RspBadVersion),
		MSGData: data,
	}, context.Background())
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
			_ = spbConn.SetDeadline(time.Now().Add(time.Duration(utils.ReadTimeOut) * time.Second))
			Mylog(sc.belong.opts.logOpen, "server read ok", msgH.Len)
			if msgH.Len == 0 {
				Mylog(sc.belong.opts.logOpen, "server receive msgHeader")
				headerBytes, n, err := ReadEncryptedHeaderAndBody(spbConn, sc.sharedKey, utils.MessageBeatLen)
				sc.increaseReadFlow(n)
				if err != nil {
					if err == io.EOF {
						return
					}
					utils.ErrorLog("server header err", err)
					return
				}
				header.DecodeHeader(headerBytes, &msgH)
				headerBytes = nil

				if msgH.Version < sc.minAppVer {
					sc.SendBadVersionMsg(msgH.Version, utils.ByteToString(msgH.Cmd))
					return
				}

				//when header shows msg length = 0, directly handle msg
				if msgH.Len == 0 {
					// if utils.ByteToString(msgH.Cmd) == header.ReqHeart {
					// 	sc.spbConn.SetDeadline(time.Now().Add(time.Duration(utils.HeartTimeOut) * time.Second))
					// 	handler := GetHandlerFunc(header.RspHeart)
					// 	if handler != nil {
					// 		sc.handlerCh <- MsgHandler{message.RelayMsgBuf{}, handler}
					// 	}
					// } else {
					handler := GetHandlerFunc(utils.ByteToString(msgH.Cmd))
					if handler != nil {
						metrics.Events.WithLabelValues(utils.ByteToString(msgH.Cmd)).Inc()
						sc.handlerCh <- MsgHandler{message.RelayMsgBuf{}, handler}
					}
					// }
				}

			} else {
				// start to process the msg if there is more than just the header to read
				nonce, dataLen, n, err := ReadEncryptedHeader(spbConn)
				sc.increaseReadFlow(n)
				if err != nil {
					utils.ErrorLog("error when reading encrypted header for msg body", err)
					return
				}
				if dataLen > utils.MessageBeatLen {
					utils.ErrorLogf("encrypted message length over sized [%v], for cmd [%v]", dataLen, utils.ByteToString(msgH.Cmd))
					return
				}

				Mylog(sc.belong.opts.logOpen, "start", time.Now().Unix())
				var onereadlen = 1024
				msgBuf := make([]byte, utils.MessageBeatLen)
				for ; i < int(dataLen); i = i + n {
					if int(dataLen)-i < 1024 {
						onereadlen = int(dataLen) - i
					}
					spbConn.SetDeadline(time.Now().Add(time.Duration(utils.ReadTimeOut) * time.Second))
					n, err = io.ReadFull(spbConn, msgBuf[i:i+onereadlen])
					// Mylog(s.opts.logOpen,"server n = ", msgBuf[0:msgH.Len])
					// Mylog(s.opts.logOpen,"len(msgBuf[0:msgH.Len]):", i)
					sc.increaseReadFlow(n)
					if err != nil {
						utils.ErrorLog("server body err", err)
						return
					}
				}
				Mylog(sc.belong.opts.logOpen, "end", time.Now().Unix())
				if uint32(i) == dataLen {
					decryptedBody, err := encryption.DecryptAES(sc.sharedKey, msgBuf[:dataLen], nonce)
					if err != nil {
						utils.ErrorLog("server body decryption err", err)
						return
					}

					msg = &message.RelayMsgBuf{
						MSGHead: msgH,
						MSGData: decryptedBody,
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
					metrics.Events.WithLabelValues(utils.ByteToString(msgH.Cmd)).Inc()
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
		sendCh chan *message.RelayMsgBuf
		cDone  <-chan struct{}
		sDone  <-chan struct{}
		packet *message.RelayMsgBuf
		sc     *ServerConn
	)

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
					if err := sc.writePacket(packet); err != nil {
						utils.ErrorLog(err)
						return
					}
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
				if err := sc.writePacket(packet); err != nil {
					utils.ErrorLog(err)
					return
				}
				packet = nil
			}
		}
	}
}

func (sc *ServerConn) writePacket(packet *message.RelayMsgBuf) error {
	var onereadlen = 1024
	var n int
	// Mylog(s.opts.logOpen,"msgLen", len(packet.MSGData))

	// Encrypt and write message header
	encryptedHeader, err := EncryptAndPack(sc.sharedKey, packet.MSGData[:utils.MsgHeaderLen])
	if err != nil {
		return errors.Wrap(err, "server cannot encrypt header")
	}
	_ = sc.spbConn.SetDeadline(time.Now().Add(time.Duration(utils.WriteTimeOut) * time.Second))
	if err = WriteFull(sc.spbConn, encryptedHeader); err != nil {
		return errors.Wrap(err, "server write err")
	}
	sc.increaseWriteFlow(len(encryptedHeader))

	// Encrypt and write message data
	encryptedData, err := EncryptAndPack(sc.sharedKey, packet.MSGData[utils.MsgHeaderLen:])
	if err != nil {
		return errors.Wrap(err, "server cannot encrypt msg")
	}
	for i := 0; i < len(encryptedData); i = i + n {
		// Mylog(s.opts.logOpen,"len(msgBuf[0:msgH.Len]):", i)
		if len(encryptedData)-i < 1024 {
			onereadlen = len(encryptedData) - i
			// Mylog(s.opts.logOpen,"onereadlen:", onereadlen)
		}
		_ = sc.spbConn.SetDeadline(time.Now().Add(time.Duration(utils.WriteTimeOut) * time.Second))
		n, err = sc.spbConn.Write(encryptedData[i : i+onereadlen])
		// Mylog(s.opts.logOpen,"server n = ", msgBuf[0:msgH.Len])
		// Mylog(s.opts.logOpen,"i+onereadlen:", i+onereadlen)
		sc.increaseWriteFlow(n)
		if err != nil {
			return errors.Wrap(err, "server write err")
		} else {
			// Mylog(s.opts.logOpen,"i", i)
		}
	}
	cmem.Free(packet.Alloc)
	return nil
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
					ctxWithReqId := CreateContextWithReqId(ctx, msg.MSGHead.ReqId)
					// Mylog(s.opts.logOpen,"handler(CreateContextWithNetID(CreateContextWithMessage(ctxWithReqId, msg), netID), c )", netID)
					handler(CreateContextWithNetID(CreateContextWithMessage(ctxWithReqId, &msg), netID), c)
				})
				if err != nil {
					utils.ErrorLog(err)
				}
			}
		}
	}
}

func (sc *ServerConn) increaseWriteFlow(n int) {
	logOpts := sc.belong.volRecOpts
	if logOpts.logAll || logOpts.logOutbound || logOpts.logWrite {
		sc.belong.volRecOpts.writeFlow = sc.belong.volRecOpts.writeAtom.AddAndGetNew(int64(n))
		sc.belong.volRecOpts.secondWriteFlowA = sc.belong.volRecOpts.secondWriteAtomA.AddAndGetNew(int64(n))
		sc.belong.volRecOpts.allFlow = sc.belong.volRecOpts.allAtom.AddAndGetNew(int64(n))
	}
}

func (sc *ServerConn) increaseReadFlow(n int) {
	logOpts := sc.belong.volRecOpts
	if logOpts.logAll {
		sc.belong.volRecOpts.allFlow = sc.belong.volRecOpts.allAtom.AddAndGetNew(int64(n))
	}
	if logOpts.logRead {
		sc.belong.volRecOpts.readFlow = sc.belong.volRecOpts.readAtom.AddAndGetNew(int64(n))
		sc.belong.volRecOpts.secondReadFlowA = sc.belong.volRecOpts.secondReadAtomA.AddAndGetNew(int64(n))
	}
	if logOpts.logInbound {
		sc.belong.volRecOpts.inbound = sc.belong.volRecOpts.inboundAtomic.AddAndGetNew(int64(n))
	}
}
