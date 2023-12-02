package core

// server readloop writeloop handleloop
import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/stratosnet/sds/sds-msg/header"
	"github.com/stratosnet/sds/sds-msg/protos"

	fwed25519 "github.com/stratosnet/sds/framework/crypto/ed25519"
	"github.com/stratosnet/sds/framework/crypto/encryption"
	"github.com/stratosnet/sds/framework/metrics"
	fwmsg "github.com/stratosnet/sds/framework/msg"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
)

type MsgHandler struct {
	message   fwmsg.RelayMsgBuf
	handler   HandlerFunc
	recvStart int64
}

type WriteCloser interface {
	Write(*fwmsg.RelayMsgBuf, context.Context) error
	Close()
}

type WriteHook struct {
	MessageId uint8
	Fn        WriteHookFunc
}

var (
	GoroutineMap     = &sync.Map{}
	HandshakeChanMap = &sync.Map{} // map[string]chan []byte    Map that stores channels used during handshake process
	TimeRcv          int64
)

type ServerConn struct {
	netid   int64
	belong  *Server
	spbConn net.Conn

	once      *sync.Once
	wg        *sync.WaitGroup
	sendCh    chan *fwmsg.RelayMsgBuf
	handlerCh chan MsgHandler

	mu    sync.Mutex // guards following
	name  string
	heart int64

	minAppVer            uint16
	sharedKey            []byte // ECDH shared key derived during handshake
	remoteP2pAddress     string
	remoteNetworkAddress string // Actual network address of the remote node

	ctx    context.Context
	cancel context.CancelFunc

	writeHook []WriteHook

	encryptMessage bool
}

func CreateServerConn(id int64, s *Server, c net.Conn) *ServerConn {
	sc := &ServerConn{
		netid:     id,
		belong:    s,
		spbConn:   c,
		once:      &sync.Once{},
		wg:        &sync.WaitGroup{},
		sendCh:    make(chan *fwmsg.RelayMsgBuf, s.opts.bufferSize),
		handlerCh: make(chan MsgHandler, s.opts.bufferSize),
		heart:     time.Now().UnixNano(),
	}
	// context.WithValue get key-value context
	sc.ctx, sc.cancel = context.WithCancel(context.WithValue(s.ctx, serverCtxKey, s))
	sc.name = c.RemoteAddr().String()
	sc.minAppVer = s.opts.minAppVersion
	return sc
}

func ServerFromCtx(ctx context.Context) (*Server, bool) {
	server, ok := ctx.Value(serverCtxKey).(*Server)
	return server, ok
}

func (sc *ServerConn) GetNetID() int64 {
	return sc.netid
}

func (sc *ServerConn) SetConnName(name string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.name = name
}

func (sc *ServerConn) GetName() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	name := sc.name
	return name
}

func (sc *ServerConn) GetIP() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	host, _, _ := net.SplitHostPort(sc.name)
	return host
}

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

// GetRemoteAddr returns the address from which the connection is directly coming from. In a VM with port forwarding, this might be the address of the host machine
func (sc *ServerConn) GetRemoteAddr() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.spbConn.RemoteAddr().String()
}

// GetRemoteNetworkAddress returns the actual remote network address, as advertised by the remote node itself
func (sc *ServerConn) GetRemoteNetworkAddress() string {
	return sc.remoteNetworkAddress
}

func (sc *ServerConn) SetRemoteNetworkAddress(networkAddress string) {
	sc.remoteNetworkAddress = networkAddress
}

func (sc *ServerConn) GetLocalP2pAddress() string {
	return sc.belong.opts.p2pAddress
}

func (sc *ServerConn) GetRemoteP2pAddress() string {
	return sc.remoteP2pAddress
}

func (sc *ServerConn) SetWriteHook(h []WriteHook) {
	sc.mu.Lock()
	sc.writeHook = h
	sc.mu.Unlock()
}

func (sc *ServerConn) handshake() (error, bool) {
	// Set handshake timeout
	if err := sc.spbConn.SetDeadline(time.Now().Add(time.Duration(utils.HandshakeTimeOut) * time.Second)); err != nil {
		return err, false
	}

	// This is either a client creating a connection, or a temporary connection made for a handshake
	// Read the first fwmsg from the connection. It should indicate what kind of connection it is
	buffer := make([]byte, ConnFirstMsgSize)
	if _, err := io.ReadFull(sc.spbConn, buffer); err != nil {
		return err, false
	}
	connType, serverIP, serverPort, channelId, err := ParseFirstMessage(buffer)
	if err != nil {
		return err, false
	}

	switch connType {
	case ConnTypeClient:
		remoteServer := serverIP.String() + ":" + strconv.FormatUint(uint64(serverPort), 10)
		sc.remoteNetworkAddress = remoteServer

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

		// Write the connection type as first fwmsg
		firstMessage := CreateFirstMessage(ConnTypeHandshake, nil, 0, channelId)
		if err = WriteFull(handshakeConn, firstMessage); err != nil {
			return err, false
		}

		// Create tmp key
		tmpPrivKey := fwed25519.GenPrivKey()
		tmpPrivKeyBytes := tmpPrivKey.Bytes()
		tmpPubKeyBytes := tmpPrivKey.PubKey().Bytes()

		// Write tmp key to handshake conn
		handshakeSignature, err := tmpPrivKey.Sign([]byte(HandshakeMessage))
		if err != nil {
			return err, false
		}
		if err = WriteFull(handshakeConn, append(tmpPubKeyBytes, handshakeSignature...)); err != nil {
			return err, false
		}

		// Read tmp key from original conn and verify
		buffer = make([]byte, fwed25519.PubKeySize+fwed25519.SignatureSize)
		if _, err = io.ReadFull(sc.spbConn, buffer); err != nil {
			return err, false
		}
		peerPubKeyBytes := buffer[:fwed25519.PubKeySize]
		peerPubKey := fwed25519.PubKeyFromBytes(peerPubKeyBytes)
		peerSignature := buffer[fwed25519.PubKeySize:]
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
		encryptedMsg, err := Pack(sharedPrivKeyBytes, []byte(sc.GetLocalP2pAddress()))
		if err != nil {
			return err, false
		}
		if err = WriteFull(sc.spbConn, encryptedMsg); err != nil {
			return err, false
		}

		// Read remote p2p address
		p2pAddressBytes, _, err := Unpack(sc.spbConn, sharedPrivKeyBytes, utils.MessageBeatLen)
		if err != nil {
			return err, false
		}
		sc.remoteP2pAddress = string(p2pAddressBytes)
		if _, err = fwtypes.P2PAddressFromBech32(sc.remoteP2pAddress); err != nil {
			return errors.Wrap(err, "incorrect P2pAddress"), false
		}

		_ = handshakeConn.Close()
	case ConnTypeHandshake:
		// Read tmp key from conn
		buffer = make([]byte, fwed25519.PubKeySize+fwed25519.SignatureSize)
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
	sc.encryptMessage = true

	err, isHandshakeConn := sc.handshake()
	if err != nil {
		Mylog(sc.belong.opts.logOpen, LOG_MODULE_START, fmt.Sprintf("handshake error %v -> %v, %v", sc.spbConn.LocalAddr(), sc.spbConn.RemoteAddr(), err.Error()))
		sc.Close()
		return
	}
	if isHandshakeConn {
		sc.Close()
		return
	}

	Mylog(sc.belong.opts.logOpen, LOG_MODULE_START, fmt.Sprintf("start %v -> %v (%v)", sc.spbConn.LocalAddr(), sc.spbConn.RemoteAddr(), sc.remoteP2pAddress))
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

/*
error is caught at application layer, if it's utils.ErrWouldBlock，sleep and then continue write
*/
func (sc *ServerConn) Write(message *fwmsg.RelayMsgBuf, ctx context.Context) error {
	if message.MSGSign.P2pAddress == "" || message.MSGSign.P2pPubKey == nil {
		return errors.New("missing sign related information")
	}
	return asyncWrite(sc, message, ctx)
}

func asyncWrite(c interface{}, m *fwmsg.RelayMsgBuf, ctx context.Context) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = errors.Wrapf(utils.ErrServerClosed, "recover occurred during asyncWrite, err: %v", p)
		}
	}()

	sendCh := c.(*ServerConn).sendCh
	if m.MSGHead.ReqId == 0 {
		reqId := GetReqIdFromContext(ctx)
		if reqId == 0 {
			reqId = GenerateNewReqId(m.MSGHead.Cmd)
			InheritRpcLoggerFromParentReqId(ctx, reqId)
			InheritRemoteReqIdFromParentReqId(ctx, reqId)
		}
		m.MSGHead.ReqId = reqId
	}
	m.PacketId = GetPacketIdFromContext(ctx)
	sendCh <- m
	return
}

func (sc *ServerConn) Close() {
	sc.belong.goroutine = sc.belong.goAtom.DecrementAndGetNew()
	sc.once.Do(func() {
		Mylog(sc.belong.opts.logOpen, LOG_MODULE_CLOSE, fmt.Sprintf("close conn gracefully %v -> %v (%v)", sc.spbConn.LocalAddr(), sc.spbConn.RemoteAddr(), sc.remoteP2pAddress))

		// close
		onClose := sc.belong.opts.onClose
		if onClose != nil {
			onClose(sc)
		}

		// close conns
		if sc.belong.conns != nil {
			sc.belong.conns.Delete(sc.netid) // If the server is closing, conns might be already nil, so no need to call delete
		}
		Mylog(sc.belong.opts.logOpen, LOG_MODULE_CLOSE, sc.belong.ConnsSize())
		// close net.Conn, any blocked read or write operation will be unblocked and
		// return errors.
		if tc, ok := sc.spbConn.(*net.TCPConn); ok {
			// set connection closer behavior when there are data to be sent or confirmed in the connection:
			// if sec < 0 (default), the data sending will be finished before close.
			// if sec = 0, the data will be dropped
			// if sec > 0, the data sending will continue for <sec> second and then remaining data will be dropped
			_ = tc.SetLinger(0)
		}
		_ = sc.spbConn.Close()
		// cancel readLoop, writeLoop and handleLoop go-routines.
		sc.mu.Lock()
		sc.cancel()
		sc.mu.Unlock()

		sc.wg.Wait()

		close(sc.sendCh)
		close(sc.handlerCh)
		metrics.ConnNumbers.WithLabelValues("server").Dec()
		sc.belong.wg.Done()
		sc.belong.goroutine = sc.belong.goAtom.DecrementAndGetNew()
	})
}

func (sc *ServerConn) SendBadVersionMsg(version uint16, cmd uint8) {
	req := &protos.RspBadVersion{
		Version:        int32(version),
		MinimumVersion: int32(sc.minAppVer),
		Command:        uint32(cmd),
	}
	data, err := proto.Marshal(req)
	if err != nil {
		utils.ErrorLog(err)
		return
	}

	err = sc.Write(&fwmsg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, sc.minAppVer, uint32(len(data)), header.RspBadVersion),
		MSGBody: data,
	}, context.Background())
	if err != nil {
		utils.ErrorLog(err)
	}
	time.Sleep(500 * time.Millisecond)
}

func readLoop(c WriteCloser, wg *sync.WaitGroup) {
	var (
		spbConn   net.Conn
		cDone     <-chan struct{}
		sDone     <-chan struct{}
		onMessage onMessageFunc
		handlerCh chan MsgHandler
		msg       *fwmsg.RelayMsgBuf
		sc        *ServerConn
	)

	spbConn = c.(*ServerConn).spbConn
	cDone = c.(*ServerConn).ctx.Done()
	sDone = c.(*ServerConn).belong.ctx.Done()
	onMessage = c.(*ServerConn).belong.opts.onMessage
	handlerCh = c.(*ServerConn).handlerCh
	sc = c.(*ServerConn)

	defer func() {
		if p := recover(); p != nil {
			Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "panic occurs:", p, "\n")
		}
		wg.Done()
		GoroutineMap.Delete(sc.GetName() + "read")
		c.Close()
	}()

	var msgH header.MessageHead
	var msgS fwmsg.MessageSign
	var headerBytes []byte
	var n int
	var err error
	var key []byte

	listenHeader := true
	i := 0
	pos := 0

	msgBuf := make([]byte, utils.MessageBeatLen)
	for {
		if sc.encryptMessage {
			key = sc.sharedKey
		} else {
			key = nil
		}
		select {
		case <-cDone: // connection closed
			Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "closes by conn")
			return
		case <-sDone: // server closed
			Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "closes by server")
			return
		default:
			recvStart := time.Now().UnixMilli()
			_ = spbConn.SetReadDeadline(time.Now().Add(time.Duration(utils.ReadTimeOut) * time.Second))
			if listenHeader {
				// listen to the header
				headerBytes, n, err = Unpack(spbConn, key, utils.MessageBeatLen)
				sc.increaseReadFlow(n)
				if err != nil {
					Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "read header err: "+err.Error())
					return
				}
				copy(msgBuf[:header.MsgHeaderLen], headerBytes[:header.MsgHeaderLen])
				msgH.Decode(msgBuf[:header.MsgHeaderLen])
				if msgH.Version < sc.minAppVer {
					sc.SendBadVersionMsg(msgH.Version, msgH.Cmd)
					Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "fwmsg versions don't match")
					return
				}
				// no matter the body is empty or not, fwmsg is always handled in the second part, after the signature verified.
				listenHeader = false
			} else {
				// listen to the second part: body + sign + data. They are concatenated to the header in msgBuf.
				nonce, secondPartLen, n, err := ReadEncryptionHeader(spbConn)
				sc.increaseReadFlow(n)
				if err != nil {
					Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "read encrypted header err: "+err.Error())
					return
				}
				if secondPartLen > utils.MessageBeatLen {
					Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, fmt.Sprintf("read encrypted header err: over sized [%v], for cmd [%v]", secondPartLen, msgH.Cmd))
					return
				}

				var onereadlen = 1024
				pos = header.MsgHeaderLen

				for ; i < int(secondPartLen); i = i + n {
					if int(secondPartLen)-i < 1024 {
						onereadlen = int(secondPartLen) - i
					}
					n, err = io.ReadFull(spbConn, msgBuf[pos:pos+onereadlen])
					pos += n
					sc.increaseReadFlow(n)
					if err != nil {
						Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "fwmsg body err: "+err.Error())
						return
					}
				}

				// handle the second part after all bytes are received
				if uint32(i) == secondPartLen {
					posBody := uint32(header.MsgHeaderLen)
					posSign := posBody + msgH.Len
					posData := posSign + fwmsg.MsgSignLen
					var posEnd uint32
					if sc.encryptMessage {
						secondPart, err := encryption.DecryptAES(sc.sharedKey, msgBuf[posBody:posBody+secondPartLen], nonce, true)
						if err != nil {
							Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "fwmsg body decryption err: "+err.Error())
							return
						}
						posEnd = posBody + uint32(len(secondPart))
					} else {
						posEnd = posBody + secondPartLen
					}

					// verify signature
					msgS.Decode(msgBuf[posSign : posSign+fwmsg.MsgSignLen])
					if err = msgS.Verify(msgBuf[:posSign], sc.remoteP2pAddress); err != nil {
						Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "read err: failed signature verification: "+err.Error())
						continue
					}
					// fwmsg body goes to field MSGBody, data goes to field MSGData if it exists
					msg = &fwmsg.RelayMsgBuf{
						MSGHead: header.CopyMessageHeader(msgH),
						MSGBody: make([]byte, posSign-posBody),
					}
					copy(msg.MSGBody, msgBuf[posBody:posSign])

					if posEnd > posData {
						msg.MSGData = utils.RequestBuffer()[:posEnd-posData]
						copy(msg.MSGData[:], msgBuf[posData:posEnd])
					}
					TimeRcv = time.Now().UnixMicro()
					handler := GetHandlerFunc(msgH.Cmd)
					if handler == nil {
						if onMessage != nil {
							onMessage(*msg, c)
						} else {
							Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "no handler or onMessage() found for fwmsg: "+strconv.FormatUint(uint64(msgH.Cmd), 10))
						}
						msgH.Len = 0
						i = 0
						listenHeader = true
						continue
					}
					if msgType := header.GetMsgTypeFromId(msgH.Cmd); msgType != nil {
						metrics.Events.WithLabelValues(msgType.Name).Inc()
					}
					handlerCh <- MsgHandler{*msg, handler, recvStart}
					i = 0
					listenHeader = true
				} else {
					if msgType := header.GetMsgTypeFromId(msgH.Cmd); msgType != nil {
						Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, "msgH.Len doesn't match the size of data for fwmsg: "+msgType.Name)
					} else {
						Mylog(sc.belong.opts.logOpen, LOG_MODULE_READLOOP, fmt.Sprintf("msgH.Len doesn't match the size of data for an invalid fwmsg: %d", msgH.Cmd))
					}
					return
				}
			}
		}
	}
}

func writeLoop(c WriteCloser, wg *sync.WaitGroup) {
	var (
		sendCh chan *fwmsg.RelayMsgBuf
		cDone  <-chan struct{}
		sDone  <-chan struct{}
		packet *fwmsg.RelayMsgBuf
		sc     *ServerConn
	)

	sendCh = c.(*ServerConn).sendCh
	cDone = c.(*ServerConn).ctx.Done()
	sDone = c.(*ServerConn).belong.ctx.Done()
	sc = c.(*ServerConn)
	defer func() {
		if p := recover(); p != nil {
			Mylog(sc.belong.opts.logOpen, LOG_MODULE_WRITELOOP, fmt.Sprintf("panics: %v", p))
		}
		// drain all pending messages before exit
	OuterFor:
		for {
			select {
			case packet, ok := <-sendCh:
				// selected, not received: break from the loop
				if !ok {
					break OuterFor
				}
				// drain pending messages
				if packet != nil {
					if err := sc.writePacket(packet); err != nil {
						utils.ErrorLog(err)
						break OuterFor
					}
				}
			default:
				break OuterFor
			}
		}
		wg.Done()
		GoroutineMap.Delete(sc.GetName() + "write")
		c.Close()
	}()

	for {
		select {
		case <-cDone: // connection closed
			Mylog(sc.belong.opts.logOpen, LOG_MODULE_WRITELOOP, "closes by conn")
			return
		case <-sDone: // server closed
			Mylog(sc.belong.opts.logOpen, LOG_MODULE_WRITELOOP, "closes by server")
			return
		case packet = <-sendCh:
			if packet != nil {
				if err := sc.writePacket(packet); err != nil {
					Mylog(sc.belong.opts.logOpen, LOG_MODULE_WRITELOOP, "write packet err", err.Error())
					return
				}
			}
		}
	}
}

func (sc *ServerConn) writePacket(m *fwmsg.RelayMsgBuf) error {
	var encodedHeader []byte
	var encodedData []byte
	var onereadlen = 1024
	var n int
	var err error
	var key []byte
	packet := &fwmsg.RelayMsgBuf{
		MSGHead:  m.MSGHead,
		MSGSign:  m.MSGSign,
		PacketId: m.PacketId,
	}
	packet.PutIntoBuffer(m)
	defer packet.ReleaseAlloc()

	if len(m.MSGData) > 0 {
		defer utils.ReleaseBuffer(m.MSGData)
	}

	cmd := packet.MSGHead.Cmd
	// pack the header
	if sc.encryptMessage {
		key = sc.sharedKey
	} else {
		key = nil
	}
	encodedHeader, err = Pack(key, packet.GetHeader())
	if err != nil {
		return errors.Wrap(err, "server cannot encrypt header")
	}

	_ = sc.spbConn.SetWriteDeadline(time.Now().Add(time.Duration(utils.WriteTimeOut) * time.Second))
	if err = WriteFull(sc.spbConn, encodedHeader); err != nil {
		return errors.Wrap(err, "server write err")
	}
	sc.increaseWriteFlow(len(encodedHeader))

	// pack the fwmsg data
	encodedData, err = Pack(key, packet.GetBytesAfterHeader())
	if err != nil {
		return errors.Wrap(err, "server cannot encrypt msg")
	}

	writeStart := time.Now()
	for i := 0; i < len(encodedData); i = i + n {
		if len(encodedData)-i < 1024 {
			onereadlen = len(encodedData) - i
		}
		n, err = sc.spbConn.Write(encodedData[i : i+onereadlen])
		if err != nil {
			break
		}
		sc.increaseWriteFlow(n)
	}
	writeEnd := time.Now()
	costTime := writeEnd.Sub(writeStart).Milliseconds() + 1 // +1 in case of LT 1 ms

	for _, c := range sc.writeHook {
		if cmd == c.MessageId && c.Fn != nil {
			c.Fn(sc.ctx, packet.PacketId, costTime, sc)
		}
	}
	return nil
}

func handleLoop(c WriteCloser, wg *sync.WaitGroup) {
	var (
		cDone     <-chan struct{}
		sDone     <-chan struct{}
		handlerCh chan MsgHandler
		netID     int64
		ctx       context.Context
		err       error
		sc        *ServerConn
	)

	cDone = c.(*ServerConn).ctx.Done()
	sDone = c.(*ServerConn).belong.ctx.Done()
	handlerCh = c.(*ServerConn).handlerCh
	netID = c.(*ServerConn).netid
	ctx = c.(*ServerConn).ctx
	sc = c.(*ServerConn)
	var log string
	defer func() {
		if p := recover(); p != nil {
			Mylog(sc.belong.opts.logOpen, LOG_MODULE_HANDLELOOP, fmt.Sprintf("panic when handle fwmsg (%v), %v", log, p))
		}
		wg.Done()
		GoroutineMap.Delete(sc.GetName() + "handle")
		c.Close()
	}()

	for {
		select {
		case <-cDone: // connection closed
			Mylog(sc.belong.opts.logOpen, LOG_MODULE_HANDLELOOP, "closes by conn")
			return
		case <-sDone: // server closed
			Mylog(sc.belong.opts.logOpen, LOG_MODULE_HANDLELOOP, "closes by server")
			return
		case msgHandler := <-handlerCh:
			msg, handler, recvStart := msgHandler.message, msgHandler.handler, msgHandler.recvStart
			if handler != nil {
				// if askForWorker {
				err = GlobalTaskPool.Job(netID, func() {
					ctxWithReqId := CreateContextWithReqId(ctx, msg.MSGHead.ReqId)
					ctxWithRecvStart := CreateContextWithRecvStartTime(ctxWithReqId, recvStart)
					ctx := CreateContextWithMessage(ctxWithRecvStart, &msg)
					ctx = CreateContextWithNetID(ctx, netID)
					ctx = CreateContextWithSrcP2pAddr(ctx, sc.remoteP2pAddress)
					if msgType := header.GetMsgTypeFromId(msgHandler.message.MSGHead.Cmd); msgType != nil {
						log = msgType.Name
					}
					handler(ctx, c)
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
