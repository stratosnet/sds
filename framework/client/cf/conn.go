package cf

// client connect management, readloop writeloop handleloop

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/alex023/clock"
	"github.com/pkg/errors"

	"github.com/stratosnet/sds/framework/core"
	fwed25519 "github.com/stratosnet/sds/framework/crypto/ed25519"
	"github.com/stratosnet/sds/framework/crypto/encryption"
	"github.com/stratosnet/sds/framework/metrics"
	"github.com/stratosnet/sds/framework/msg"
	"github.com/stratosnet/sds/framework/msg/header"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
)

var (
	maxDownloadRate uint64
	maxUploadRate   uint64
)

const (
	LOG_MODULE_START      = "start: "
	LOG_MODULE_WRITELOOP  = "writeLoop: "
	LOG_MODULE_READLOOP   = "readLoop: "
	LOG_MODULE_HANDLELOOP = "handleLoop: "
	LOG_MODULE_CLOSE      = "close: "
)

type MsgHandler struct {
	message   msg.RelayMsgBuf
	handler   core.HandlerFunc
	recvStart int64
}

type onConnectFunc func(core.WriteCloser) bool
type onMessageFunc func(msg.RelayMsgBuf, core.WriteCloser)
type onCloseFunc func(core.WriteCloser)
type onErrorFunc func(core.WriteCloser)
type ContextKV struct {
	Key   interface{}
	Value interface{}
}
type options struct {
	onConnect  onConnectFunc
	onMessage  onMessageFunc
	onClose    onCloseFunc
	onError    onErrorFunc
	bufferSize int
	reconnect  bool // only ClientConn
	heartClose bool
	logOpen    bool
	minAppVer  uint16
	p2pAddress string
	serverIp   net.IP
	serverPort uint16
	contextkv  []ContextKV
}

// ClientOption client configuration
type ClientOption func(*options)

type WriteHook struct {
	MessageId uint8
	Fn        core.WriteHookFunc
}

type ClientConn struct {
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
	sharedKey        []byte // ECDH shared key derived during handshake
	remoteP2pAddress string
	writeHook        []WriteHook
	encryptMessage   bool
}

func ReconnectOption(rec bool) ClientOption {
	return func(o *options) {
		o.reconnect = rec
	}
}

func CreateClientConn(netid int64, addr string, opt ...ClientOption) *ClientConn {
	var opts options
	for _, o := range opt {
		o(&opts)
	}
	return newClientConnWithOptions(netid, addr, opts)
}

func MinAppVersionOption(b uint16) ClientOption {
	return func(o *options) {
		o.minAppVer = b
	}
}

func BufferSizeOption(indicator int) ClientOption {
	return func(o *options) {
		o.bufferSize = indicator
	}
}

func HeartCloseOption(b bool) ClientOption {
	return func(o *options) {
		o.heartClose = b
	}
}

func LogOpenOption(b bool) ClientOption {
	return func(o *options) {
		o.logOpen = b
	}
}

// P2pAddressOption sets the local P2P address for this conn
func P2pAddressOption(p2pAddress string) ClientOption {
	return func(o *options) {
		o.p2pAddress = p2pAddress
	}
}

// ServerIpOption sets the IP used by the server conn when establishing the handshake
func ServerIpOption(serverIp net.IP) ClientOption {
	return func(o *options) {
		o.serverIp = serverIp
	}
}

// ServerPortOption sets the port used by the local p2p server
func ServerPortOption(serverPort uint16) ClientOption {
	return func(o *options) {
		o.serverPort = serverPort
	}
}

func ContextKVOption(kv []ContextKV) ClientOption {
	return func(o *options) {
		o.contextkv = kv
	}
}

func Mylog(b bool, module string, v ...interface{}) {
	if b {
		utils.DebugLogfWithCalldepth(5, "Client Conn: "+module+"%v", v...)
	}
}

func newClientConnWithOptions(netid int64, addr string, opts options) *ClientConn {
	if opts.bufferSize == 0 {
		opts.bufferSize = 200
	}

	cc := &ClientConn{
		addr:             addr,
		opts:             opts,
		netid:            netid,
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
	for _, kv := range cc.opts.contextkv {
		cc.ctx = context.WithValue(cc.ctx, kv.Key, kv.Value)
	}

	cc.name = cc.addr
	cc.pending = []int64{}
	return cc
}

func (cc *ClientConn) GetNetID() int64 {
	return cc.netid
}

func (cc *ClientConn) SetConnName(name string) {
	cc.mu.Lock()
	cc.name = name
	cc.mu.Unlock()
}

func (cc *ClientConn) GetName() string {
	cc.mu.Lock()
	name := cc.name
	cc.mu.Unlock()
	return name
}

func SetMaxDownloadRate(rate uint64) {
	maxDownloadRate = rate
}

func SetMaxUploadRate(rate uint64) {
	maxUploadRate = rate
}

// GetIP get connection ip
func (cc *ClientConn) GetIP() string {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	host, _, _ := net.SplitHostPort(cc.name)
	return host
}

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

func (cc *ClientConn) GetLocalP2pAddress() string {
	return cc.opts.p2pAddress
}

func (cc *ClientConn) GetRemoteP2pAddress() string {
	return cc.remoteP2pAddress
}

func (cc *ClientConn) SetContextValue(k, v interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.ctx = context.WithValue(cc.ctx, k, v)
}

func (cc *ClientConn) ContextValue(k interface{}) interface{} {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.ctx.Value(k)
}

func (cc *ClientConn) SetWriteHook(h []WriteHook) {
	cc.mu.Lock()
	cc.writeHook = h
	cc.mu.Unlock()
}

func (cc *ClientConn) handshake() error {
	// Set handshake timeout
	if err := cc.spbConn.SetDeadline(time.Now().Add(time.Duration(utils.HandshakeTimeOut) * time.Second)); err != nil {
		return err
	}

	// Create a channel to receive tmp key from handshake connection
	handshakeChan := make(chan []byte)
	channelId := rand.Uint32()
	channelIdString := strconv.FormatUint(uint64(channelId), 10)
	core.HandshakeChanMap.Store(channelIdString, handshakeChan)
	defer func() {
		core.HandshakeChanMap.Delete(channelIdString)
	}()

	// Write the connection type as first message
	firstMessage := core.CreateFirstMessage(core.ConnTypeClient, cc.opts.serverIp, cc.opts.serverPort, channelId)
	if err := core.WriteFull(cc.spbConn, firstMessage); err != nil {
		return err
	}

	// Create tmp key
	tmpPrivKey := fwed25519.GenPrivKey()
	tmpPrivKeyBytes := tmpPrivKey.Bytes()
	tmpPubKeyBytes := tmpPrivKey.PubKey().Bytes()

	// Write tmp key to conn
	handshakeSignature, err := tmpPrivKey.Sign([]byte(core.HandshakeMessage))
	if err != nil {
		return err
	}
	if err = core.WriteFull(cc.spbConn, append(tmpPubKeyBytes, handshakeSignature...)); err != nil {
		return err
	}

	// Receive tmp key from channel:
	var tmpKeyMsg []byte
	select {
	case tmpKeyMsg = <-handshakeChan:
		if len(tmpKeyMsg) < fwed25519.PubKeySize+fwed25519.SignatureSize {
			return errors.Errorf("Handshake message too small (%v bytes)", len(tmpKeyMsg))
		}
	case <-time.After(utils.HandshakeTimeOut * time.Second):
		return errors.New("Timed out when reading from server channel")
	}

	peerPubKeyBytes := tmpKeyMsg[:fwed25519.PubKeySize]

	peerPubKey := fwed25519.PubKeyFromBytes(peerPubKeyBytes)
	peerSignature := tmpKeyMsg[fwed25519.PubKeySize:]
	if !peerPubKey.VerifySignature([]byte(core.HandshakeMessage), peerSignature) {
		return errors.New("Invalid signature in tmp key from peer")
	}

	// ECDH. Store encryption info in conn
	sharedPrivKeyBytes, err := encryption.ECDH(tmpPrivKeyBytes, peerPubKeyBytes)
	if err != nil {
		return err
	}
	cc.sharedKey = sharedPrivKeyBytes

	// Send local p2p address
	encryptedMsg, err := core.Pack(sharedPrivKeyBytes, []byte(cc.GetLocalP2pAddress()))
	if err != nil {
		return err
	}
	if err = core.WriteFull(cc.spbConn, encryptedMsg); err != nil {
		return err
	}

	// Read remote p2p address
	p2pAddressBytes, _, err := core.Unpack(cc.spbConn, sharedPrivKeyBytes, utils.MessageBeatLen)
	if err != nil {
		return err
	}
	cc.remoteP2pAddress = string(p2pAddressBytes)
	if _, err = fwtypes.P2PAddressFromBech32(cc.remoteP2pAddress); err != nil {
		return errors.Wrap(err, "incorrect P2pAddress")
	}

	return cc.spbConn.SetDeadline(time.Time{}) // Remove handshake timeout
}

// Start client starts readLoop, writeLoop, handleLoop
func (cc *ClientConn) Start() {
	cc.encryptMessage = true

	tcpAddr, err := net.ResolveTCPAddr("tcp4", cc.addr)
	if err != nil {
		Mylog(cc.opts.logOpen, LOG_MODULE_START, fmt.Sprintf("bad server address: %v, %v", cc.addr, err.Error()))
		cc.ClientClose(false)
		return
	}
	cc.spbConn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		utils.DebugLogf("cc.spbConn:%p", cc.spbConn)
		Mylog(cc.opts.logOpen, LOG_MODULE_START, fmt.Sprintf("failed to dial tcp: %v, %v", tcpAddr.String(), err.Error()))
		cc.ClientClose(false)
		return
	}
	metrics.ConnNumbers.WithLabelValues("client").Inc()

	err = cc.handshake()
	if err != nil {
		Mylog(cc.opts.logOpen, LOG_MODULE_START, fmt.Sprintf("handshake error %v -> %v, %v", cc.spbConn.LocalAddr(), cc.spbConn.RemoteAddr(), err.Error()))
		cc.ClientClose(true)
		return
	}

	Mylog(cc.opts.logOpen, LOG_MODULE_START, fmt.Sprintf("start conn %v -> %v (%v)", cc.spbConn.LocalAddr(), cc.spbConn.RemoteAddr(), cc.remoteP2pAddress))
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

	myClock := clock.NewClock()
	//handler               = core.GetHandlerFunc(header.ReqHeart)

	//jobFunc = func() {
	//	if handler != nil {
	//		cc.handlerCh <- MsgHandler{msg.RelayMsgBuf{}, handler}
	//	}
	//}
	logFunc := func() {
		cc.inbound = cc.inboundAtomic.AddAndGetNew(cc.secondReadFlowA)
		cc.outbound = cc.outboundAtomic.AddAndGetNew(cc.secondWriteFlowA)
		cc.secondReadFlowB = cc.secondReadAtomB.GetNewAndSetAtomic(cc.secondReadFlowA)
		cc.secondWriteFlowB = cc.secondWriteAtomB.GetNewAndSetAtomic(cc.secondWriteFlowA)
		cc.secondReadFlowA = cc.secondReadAtomA.GetNewAndSetAtomic(0)
		cc.secondWriteFlowA = cc.secondWriteAtomA.GetNewAndSetAtomic(0)
	}

	//if !cc.opts.heartClose {
	//	hbJob, _ := myClock.AddJobRepeat(time.Second*utils.ClientSendHeartTime, 0, jobFunc)
	//	cc.jobs = append(cc.jobs, hbJob)
	//}

	logJob, _ := myClock.AddJobRepeat(time.Second*1, 0, logFunc)
	cc.jobs = append(cc.jobs, logJob)
}

// ClientClose Actively closes the client connection
func (cc *ClientConn) ClientClose(closeLowLayerConn bool) {
	cc.is_active = true
	cc.once.Do(func() {
		Mylog(cc.opts.logOpen, LOG_MODULE_CLOSE, "forced close conn")

		// callback on close
		onClose := cc.opts.onClose
		if onClose != nil {
			onClose(cc)
			cc.is_active = false
		}

		// close net.Conn
		if closeLowLayerConn {
			cc.spbConn.Close()
		}
		metrics.ConnNumbers.WithLabelValues("client").Dec()

		// cancel readLoop, writeLoop and handleLoop go-routines.
		cc.mu.Lock()
		cc.cancel()
		cc.pending = nil
		cc.mu.Unlock()

		// wait until all go-routines exited.
		cc.wg.Wait()

		utils.DetailLog("cc.wg.Wait() finished")

		// close all channels.
		close(cc.sendCh)
		close(cc.handlerCh)
		if len(cc.jobs) > 0 {
			utils.DetailLogf("cancel %v jobs, %v", len(cc.jobs), cc.GetName())
			for _, job := range cc.jobs {
				job.Cancel()
			}
		}
	})
}

func (cc *ClientConn) Close() {
	cc.once.Do(func() {
		Mylog(cc.opts.logOpen, LOG_MODULE_CLOSE, fmt.Sprintf("close conn gracefully %v -> %v (%v)", cc.spbConn.LocalAddr(), cc.spbConn.RemoteAddr(), cc.remoteP2pAddress))
		// callback on close
		onClose := cc.opts.onClose
		if onClose != nil {
			onClose(cc)
		}

		// close net.Conn
		if cc.spbConn != nil {
			cc.spbConn.Close()
		}

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
			utils.DetailLogf("cancel %v jobs, %v", len(cc.jobs), cc.GetName())
			for _, job := range cc.jobs {
				job.Cancel()
			}
		}
		if cc.opts.reconnect {
			cc.reconnect()
		}
	})
}

func (cc *ClientConn) reconnect() {
	Mylog(cc.opts.logOpen, LOG_MODULE_START, fmt.Sprintf("reconnect to %v (%v)", cc.addr, cc.remoteP2pAddress))
	*cc = *newClientConnWithOptions(cc.netid, cc.addr, cc.opts)
	cc.Start()
}

func (cc *ClientConn) GetIsActive() bool {
	return cc.is_active
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

func (cc *ClientConn) Write(message *msg.RelayMsgBuf, ctx context.Context) error {
	if message.MSGSign.P2pAddress == "" || message.MSGSign.P2pPubKey == nil {
		return errors.New("missing sign related information")
	}
	return asyncWrite(cc, message, ctx)
}

func asyncWrite(c *ClientConn, m *msg.RelayMsgBuf, ctx context.Context) (err error) {
	if c == nil {
		return errors.New("nil client connection")
	}
	defer func() {
		if p := recover(); p != nil {
			err = errors.Wrapf(utils.ErrServerClosed, "recover occurred during asyncWrite, err: %v", p)
		}
	}()

	sendCh := c.sendCh

	if m.MSGHead.ReqId == 0 {
		reqId := core.GetReqIdFromContext(ctx)
		if reqId == 0 {
			reqId = core.GenerateNewReqId(m.MSGHead.Cmd)
			core.InheritRpcLoggerFromParentReqId(ctx, reqId)
			core.InheritRemoteReqIdFromParentReqId(ctx, reqId)
		}
		m.MSGHead.ReqId = reqId
	}

	m.PacketId = core.GetPacketIdFromContext(ctx)

	sendCh <- m
	core.TimoutMap.Store(ctx, m.MSGHead.ReqId, m)

	return
}

func readLoop(c core.WriteCloser, wg *sync.WaitGroup) {
	var (
		spbConn   net.Conn
		cDone     <-chan struct{}
		onMessage onMessageFunc
		handlerCh chan MsgHandler
		message   *msg.RelayMsgBuf
		cc        *ClientConn
	)
	cc = c.(*ClientConn)
	spbConn = c.(*ClientConn).spbConn
	cDone = c.(*ClientConn).ctx.Done()
	onMessage = c.(*ClientConn).opts.onMessage
	handlerCh = c.(*ClientConn).handlerCh

	defer func() {
		if p := recover(); p != nil {
			Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "panics:", p, "\n")
		}
		wg.Done()
		if !cc.is_active {
			c.Close()
		}
	}()

	var msgH header.MessageHead
	var msgS msg.MessageSign
	var lr utils.LimitRate
	var headerBytes []byte
	var n int
	var err error
	var key []byte

	listenHeader := true
	i := 0
	pos := 0

	// this buffer is only used in this loop. Messages need to be copied out of this buffer.
	msgBuf := make([]byte, utils.MessageBeatLen)
	for {
		if cc.encryptMessage {
			key = cc.sharedKey
		} else {
			key = nil
		}
		select {
		case <-cDone: // connection closed
			Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "closes by conn")
			return
		default:
			recvStart := time.Now().UnixMilli()
			_ = spbConn.SetReadDeadline(time.Now().Add(time.Duration(utils.ReadTimeOut) * time.Second))
			if listenHeader {
				// listen to the header
				headerBytes, n, err = core.Unpack(spbConn, key, utils.MessageBeatLen)
				cc.secondReadFlowA = cc.secondReadAtomA.AddAndGetNew(int64(n))
				if err != nil {
					Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "read header err: "+err.Error())
					return
				}
				copy(msgBuf[:header.MsgHeaderLen], headerBytes[:header.MsgHeaderLen])
				msgH.Decode(headerBytes[:header.MsgHeaderLen])
				if msgH.Version < cc.opts.minAppVer {
					msgType := header.GetMsgTypeFromId(msgH.Cmd)
					if msgType != nil {
						utils.DebugLogf("received a [%v] message with an outdated [%v] version (min version [%v])", msgType.Name, msgH.Version, cc.opts.minAppVer)
					} else {
						utils.DebugLogf("received a message with an invalid msg type %d", msgH.Cmd)
					}

					continue
				}
				// no matter the body is empty or not, message is always handled in the second part, after the signature verified.
				listenHeader = false
			} else {
				// listen to the second part: body + sign + data. They are concatenated to the header in msgBuf.
				nonce, secondPartLen, n, err := core.ReadEncryptionHeader(spbConn)
				cc.secondReadFlowA = cc.secondReadAtomA.AddAndGetNew(int64(n))
				if err != nil {
					Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "read encrypted header err: "+err.Error())
					return
				}
				if secondPartLen > utils.MessageBeatLen {
					msgType := header.GetMsgTypeFromId(msgH.Cmd)
					if msgType != nil {
						Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "read encrypted header err: over sized [%v], for cmd [%v]", secondPartLen, msgType.Name)
					} else {
						Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "read encrypted header err: over sized [%v], for an invalid cmd [%d]", secondPartLen, msgH.Cmd)
					}

					return
				}

				onereadlen := 1024
				pos = header.MsgHeaderLen
				cmd := msgH.Cmd

				for ; i < int(secondPartLen); i = i + n {
					if int(secondPartLen)-i < 1024 {
						onereadlen = int(secondPartLen) - i
					}
					n, err = io.ReadFull(spbConn, msgBuf[pos:pos+onereadlen])
					pos += n
					cc.secondReadFlowA = cc.secondReadAtomA.AddAndGetNew(int64(n))
					if err != nil {
						Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "read server body err: "+err.Error())
						return
					}
					if cmd == header.RspDownloadSlice.Id {
						if maxDownloadRate > 0 {
							lr.SetRate(maxDownloadRate)
							lr.Limit()
						}
					}
				}

				// handle the second part after all bytes are received
				if uint32(i) == secondPartLen {
					posBody := uint32(header.MsgHeaderLen)
					posSign := posBody + msgH.Len
					posData := posSign + msg.MsgSignLen
					var posEnd uint32
					if cc.encryptMessage {
						secondPart, err := encryption.DecryptAES(cc.sharedKey, msgBuf[posBody:posBody+secondPartLen], nonce, true)
						if err != nil {
							utils.ErrorLog("client body decryption err", err)
							return
						}
						posEnd = posBody + uint32(len(secondPart))
					} else {
						posEnd = posBody + secondPartLen
					}

					// verify signature
					msgS.Decode(msgBuf[posSign : posSign+msg.MsgSignLen])
					if err = msgS.Verify(msgBuf[:posSign], cc.remoteP2pAddress); err != nil {
						Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "read err: failed signature verification: "+err.Error())
						continue
					}
					// message body goes to field MSGBody, data goes to field MSGData if it exists
					message = &msg.RelayMsgBuf{
						MSGHead: header.CopyMessageHeader(msgH),
						MSGBody: make([]byte, posSign-posBody),
					}
					copy(message.MSGBody[:], msgBuf[posBody:posSign])

					if posEnd > posData {
						message.MSGData = utils.RequestBuffer()[:posEnd-posData]
						copy(message.MSGData[:], msgBuf[posData:posEnd])
					}

					handler := core.GetHandlerFunc(cmd)
					if handler == nil {
						if onMessage != nil {
							onMessage(*message, c)
						} else {
							if msgType := header.GetMsgTypeFromId(msgH.Cmd); msgType != nil {
								Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "no handler or onMessage() found for message: "+msgType.Name)
							} else {
								Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, fmt.Sprintf("no handler or onMessage() found for an invalid message: %d", cmd))
							}

						}
						msgH.Len = 0
						i = 0
						listenHeader = true
						continue
					}
					handlerCh <- MsgHandler{*message, handler, recvStart}

					i = 0
					listenHeader = true
				} else {
					if msgType := header.GetMsgTypeFromId(msgH.Cmd); msgType != nil {
						Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "msgH.Len doesn't match the size of data for message: "+msgType.Name)
					} else {
						Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, fmt.Sprintf("msgH.Len doesn't match the size of data for an invalid message: %d", msgH.Cmd))
					}
					return
				}
			}
		}
	}
}

func writeLoop(c core.WriteCloser, wg *sync.WaitGroup) {
	var (
		sendCh chan *msg.RelayMsgBuf
		cDone  <-chan struct{}
		packet *msg.RelayMsgBuf
		cc     *ClientConn
	)
	cc = c.(*ClientConn)
	sendCh = c.(*ClientConn).sendCh
	cDone = c.(*ClientConn).ctx.Done()
	defer func() {
		if p := recover(); p != nil {
			Mylog(cc.opts.logOpen, LOG_MODULE_WRITELOOP, fmt.Sprintf("panics: %v", p))
		}
		wg.Done()

		if !cc.is_active {
			c.Close()
		}
	}()

	for {
		select {
		case <-cDone: // connection closed
			Mylog(cc.opts.logOpen, LOG_MODULE_WRITELOOP, "closes by conn")
			return
		case packet = <-sendCh:
			if packet != nil {
				if err := cc.writePacket(packet); err != nil {
					Mylog(cc.opts.logOpen, LOG_MODULE_WRITELOOP, "write packet err: "+err.Error())
					return
				}
			}
		}
	}
}

func (cc *ClientConn) writePacket(m *msg.RelayMsgBuf) error {
	var lr utils.LimitRate
	var encodedHeader []byte
	var encodedData []byte
	var err error
	var onereadlen = 1024
	var n int

	packet := &msg.RelayMsgBuf{
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

	var key []byte
	// pack the header
	if cc.encryptMessage {
		key = cc.sharedKey
	} else {
		key = nil
	}

	// pack the header and send it out
	encodedHeader, err = core.Pack(key, packet.GetHeader())
	if err != nil {
		return errors.Wrap(err, "server cannot encrypt header")
	}
	_ = cc.spbConn.SetWriteDeadline(time.Now().Add(time.Duration(utils.WriteTimeOut) * time.Second))
	if err = core.WriteFull(cc.spbConn, encodedHeader); err != nil {
		return errors.Wrap(err, "client write err")
	}
	cc.secondWriteFlowA = cc.secondWriteAtomA.AddAndGetNew(int64(len(encodedHeader)))

	// pack the second part and send it out
	encodedData, err = core.Pack(key, packet.GetBytesAfterHeader())
	if err != nil {
		return errors.Wrap(err, "server cannot encrypt msg")
	}
	writeStart := time.Now()
	for i := 0; i < len(encodedData); i = i + n {
		if len(encodedData)-i < 1024 {
			onereadlen = len(encodedData) - i
		}
		n, err = cc.spbConn.Write(encodedData[i : i+onereadlen])
		if err != nil {
			break
		}
		cc.secondWriteFlowA = cc.secondWriteAtomA.AddAndGetNew(int64(n))
		if cmd == header.ReqUploadFileSlice.Id {
			if maxUploadRate > 0 {
				lr.SetRate(maxUploadRate)
				lr.Limit()
			}
		}
	}
	writeEnd := time.Now()
	costTime := writeEnd.Sub(writeStart).Milliseconds() + 1 // +1 in case of LT 1 ms
	for _, c := range cc.writeHook {
		if cmd == c.MessageId && c.Fn != nil {
			c.Fn(cc.ctx, packet.PacketId, costTime, cc)
		}
	}

	return nil
}

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
	log := "handler start"
	defer func() {
		if p := recover(); p != nil {
			Mylog(cc.opts.logOpen, LOG_MODULE_HANDLELOOP, "panic when handle message ("+log+") panic info: ", p)
		}
		wg.Done()
		if !cc.is_active {
			c.Close()
		}
	}()

	for {
		select {
		case <-cDone: // connection closed
			Mylog(cc.opts.logOpen, LOG_MODULE_HANDLELOOP, "closes by conn")
			return
		case msgHandler := <-handlerCh:
			msg, handler, recvStart := msgHandler.message, msgHandler.handler, msgHandler.recvStart
			core.TimoutMap.DeleteByRspMsg(&msg)
			ctxWithParentReqId := core.CreateContextWithParentReqId(ctx, msg.MSGHead.ReqId)
			ctxWithRecvStart := core.CreateContextWithRecvStartTime(ctxWithParentReqId, recvStart)
			ctx = core.CreateContextWithMessage(ctxWithRecvStart, &msg)
			ctx = core.CreateContextWithNetID(ctx, netID)
			ctx = core.CreateContextWithSrcP2pAddr(ctx, c.(*ClientConn).remoteP2pAddress)
			if msgType := header.GetMsgTypeFromId(msgHandler.message.MSGHead.Cmd); msgType != nil {
				log = msgType.Name
			}
			if handler != nil {
				handler(ctx, c)
			}
		}
	}
}

func OnConnectOption(cb func(core.WriteCloser) bool) ClientOption {
	return func(o *options) {
		o.onConnect = cb
	}
}

func OnMessageOption(cb func(msg.RelayMsgBuf, core.WriteCloser)) ClientOption {
	return func(o *options) {
		o.onMessage = cb
	}
}

func OnCloseOption(cb func(core.WriteCloser)) ClientOption {
	return func(o *options) {
		o.onClose = cb
	}
}

func OnErrorOption(cb func(core.WriteCloser)) ClientOption {
	return func(o *options) {
		o.onError = cb
	}
}

func (cc *ClientConn) GetSecondReadFlow() int64 {
	return cc.secondReadFlowB
}

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
