package cf

// client connect management, readloop writeloop handleloop

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/metrics"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/utils/cmem"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/encryption"
	"github.com/stratosnet/sds/utils/types"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/stratosnet/sds/utils"

	"github.com/alex023/clock"
)

var (
	limitDownloadSpeed   uint64
	limitUploadSpeed     uint64
	isLimitDownloadSpeed bool
	isLimitUploadSpeed   bool
	isSpLatencyChecked   bool
)

const (
	LOG_MODULE_START      = "start: "
	LOG_MODULE_WRITELOOP  = "writeLoop: "
	LOG_MODULE_READLOOP   = "readLoop: "
	LOG_MODULE_HANDLELOOP = "handleLoop: "
	LOG_MODULE_CLOSE      = "close: "
)

// MsgHandler
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
	serverPort uint16
	contextkv  []ContextKV
}

// ClientOption client configuration
type ClientOption func(*options)

type WriteHook struct {
	Message string
	Fn      func(packetId, costTime int64)
}

// ClientConn
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

// MinAppVersionOption
func MinAppVersionOption(b uint16) ClientOption {
	return func(o *options) {
		o.minAppVer = b
	}
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

// P2pAddressOption sets the local P2P address for this conn
func P2pAddressOption(p2pAddress string) ClientOption {
	return func(o *options) {
		o.p2pAddress = p2pAddress
	}
}

// ServerPortOption sets the port used by the local p2p server
func ServerPortOption(serverPort uint16) ClientOption {
	return func(o *options) {
		o.serverPort = serverPort
	}
}

// ContextKVOption
func ContextKVOption(kv []ContextKV) ClientOption {
	return func(o *options) {
		o.contextkv = kv
	}
}

// Mylog my
func Mylog(b bool, module string, v ...interface{}) {
	if b {
		utils.DebugLogfWithCalldepth(5, "Client Conn: "+module+"%v", v...)
	}
}

// client
func newClientConnWithOptions(netid int64, c net.Conn, opts options) *ClientConn {
	if opts.bufferSize == 0 {
		opts.bufferSize = 200
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
	for _, kv := range cc.opts.contextkv {
		cc.ctx = context.WithValue(cc.ctx, kv.Key, kv.Value)
	}

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

func (cc *ClientConn) GetLocalP2pAddress() string {
	return cc.opts.p2pAddress
}

func (cc *ClientConn) GetRemoteP2pAddress() string {
	return cc.remoteP2pAddress
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
	core.HandshakeChanMap.Store(strconv.FormatUint(uint64(channelId), 10), handshakeChan)
	defer func() {
		core.HandshakeChanMap.Delete(cc.GetRemoteAddr())
	}()

	// Write the connection type as first message
	firstMessage := core.CreateFirstMessage(core.ConnTypeClient, cc.opts.serverPort, channelId)
	if err := core.WriteFull(cc.spbConn, firstMessage); err != nil {
		return err
	}

	// Create tmp key
	tmpPrivKeyBytes := ed25519.NewKey()
	tmpPrivKey := ed25519.PrivKeyBytesToPrivKey(tmpPrivKeyBytes)
	tmpPubKeyBytes := ed25519.PrivKeyBytesToPubKeyBytes(tmpPrivKeyBytes)

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
		if len(tmpKeyMsg) < tmed25519.PubKeySize+tmed25519.SignatureSize {
			return errors.Errorf("Handshake message too small (%v bytes)", len(tmpKeyMsg))
		}
	case <-time.After(utils.HandshakeTimeOut * time.Second):
		return errors.New("Timed out when reading from server channel")
	}

	peerPubKeyBytes := tmpKeyMsg[:tmed25519.PubKeySize]
	peerPubKey := ed25519.PubKeyBytesToPubKey(peerPubKeyBytes)
	peerSignature := tmpKeyMsg[tmed25519.PubKeySize:]
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
	encryptedMsg, err := core.EncryptAndPack(sharedPrivKeyBytes, []byte(cc.GetLocalP2pAddress()))
	if err != nil {
		return err
	}
	if err = core.WriteFull(cc.spbConn, encryptedMsg); err != nil {
		return err
	}

	// Read remote p2p address
	p2pAddressBytes, _, err := core.ReadEncryptedHeaderAndBody(cc.spbConn, sharedPrivKeyBytes, utils.MessageBeatLen)
	if err != nil {
		return err
	}
	cc.remoteP2pAddress = string(p2pAddressBytes)
	if _, err = types.P2pAddressFromBech(cc.remoteP2pAddress); err != nil {
		return errors.Wrap(err, "incorrect P2pAddress")
	}

	return cc.spbConn.SetDeadline(time.Time{}) // Remove handshake timeout
}

// Start client starts readLoop, writeLoop, handleLoop
func (cc *ClientConn) Start() {
	metrics.ConnNumbers.WithLabelValues("client").Inc()
	err := cc.handshake()
	if err != nil {
		Mylog(cc.opts.logOpen, LOG_MODULE_START, fmt.Sprintf("handshake error %v -> %v, %v", cc.spbConn.LocalAddr(), cc.spbConn.RemoteAddr(), err.Error()))
		cc.ClientClose()
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
	var (
		myClock = clock.NewClock()
		//handler               = core.GetHandlerFunc(header.ReqHeart)
		spLatencyCheckHandler = core.GetHandlerFunc(header.ReqSpLatencyCheck)

		spLatencyCheckJobFunc = func() {
			if !isSpLatencyChecked && spLatencyCheckHandler != nil {
				cc.handlerCh <- MsgHandler{msg.RelayMsgBuf{}, spLatencyCheckHandler, time.Now().UnixMilli()}
				isSpLatencyChecked = true
			}
		}

		//jobFunc = func() {
		//	if handler != nil {
		//		cc.handlerCh <- MsgHandler{msg.RelayMsgBuf{}, handler}
		//	}
		//}
		logFunc = func() {
			cc.inbound = cc.inboundAtomic.AddAndGetNew(cc.secondReadFlowA)
			cc.outbound = cc.outboundAtomic.AddAndGetNew(cc.secondWriteFlowA)
			cc.secondReadFlowB = cc.secondReadAtomB.GetNewAndSetAtomic(cc.secondReadFlowA)
			cc.secondWriteFlowB = cc.secondWriteAtomB.GetNewAndSetAtomic(cc.secondWriteFlowA)
			cc.secondReadFlowA = cc.secondReadAtomA.GetNewAndSetAtomic(0)
			cc.secondWriteFlowA = cc.secondWriteAtomA.GetNewAndSetAtomic(0)
		}
	)
	//if !cc.opts.heartClose {
	//	hbJob, _ := myClock.AddJobRepeat(time.Second*utils.ClientSendHeartTime, 0, jobFunc)
	//	cc.jobs = append(cc.jobs, hbJob)
	//}
	latencyJob, _ := myClock.AddJobRepeat(time.Second*utils.LatencyCheckSpListInterval, 1, spLatencyCheckJobFunc)
	cc.jobs = append(cc.jobs, latencyJob)
	logJob, _ := myClock.AddJobRepeat(time.Second*1, 0, logFunc)
	cc.jobs = append(cc.jobs, logJob)
}

// ClientClose Actively closes the client connection
func (cc *ClientConn) ClientClose() {
	cc.is_active = true
	cc.once.Do(func() {
		Mylog(cc.opts.logOpen, LOG_MODULE_CLOSE, fmt.Sprintf("forced close conn %v -> %v (%v)", cc.spbConn.LocalAddr(), cc.spbConn.RemoteAddr(), cc.remoteP2pAddress))

		// callback on close
		onClose := cc.opts.onClose
		if onClose != nil {
			onClose(cc)
			cc.is_active = false
		}

		// close net.Conn
		cc.spbConn.Close()
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

// Close
func (cc *ClientConn) Close() {
	cc.once.Do(func() {
		Mylog(cc.opts.logOpen, LOG_MODULE_CLOSE, fmt.Sprintf("close conn gracefully %v -> %v (%v)", cc.spbConn.LocalAddr(), cc.spbConn.RemoteAddr(), cc.remoteP2pAddress))
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
func (cc *ClientConn) Write(message *msg.RelayMsgBuf, ctx context.Context) error {
	return asyncWrite(cc, message, ctx)
}

func asyncWrite(c *ClientConn, m *msg.RelayMsgBuf, ctx context.Context) (err error) {
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
	msgH := make([]byte, utils.MsgHeaderLen)
	reqId := core.GetReqIdFromContext(ctx)
	if reqId == 0 {
		reqId, _ = utils.NextSnowFlakeId()
		core.InheritRpcLoggerFromParentReqId(ctx, reqId)
		core.InheritRemoteReqIdFromParentReqId(ctx, reqId)
	}
	header.GetMessageHeader(m.MSGHead.Tag, m.MSGHead.Version, m.MSGHead.Len, string(m.MSGHead.Cmd), reqId, msgH)
	// msgData := make([]byte, utils.MessageBeatLen)
	// copy((*msgData)[0:], msgH)
	// copy((*msgData)[utils.MsgHeaderLen:], m.MSGData)
	// memory := &msg.RelayMsgBuf{
	// 	MSGHead: m.MSGHead,
	// 	MSGData: (*msgData)[0 : m.MSGHead.Len+utils.MsgHeaderLen],
	// }
	memory := &msg.RelayMsgBuf{
		MSGHead:  m.MSGHead,
		PacketId: core.GetPacketIdFromContext(ctx),
	}
	memory.MSGHead.ReqId = reqId
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

	m.MSGHead.ReqId = reqId
	core.TimoutMap.Store(ctx, reqId, m)

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

	// var msgBuf []byte
	var msgH header.MessageHead
	i := 0
	var lr utils.LimitRate

	for {
		select {
		case <-cDone: // connection closed
			Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "closes by conn")
			return
		default:
			recvStart := time.Now().UnixMilli()
			// Mylog(cc.opts.logOpen,"client read ok", msgH.Len)
			if msgH.Len == 0 {
				headerBytes, n, err := core.ReadEncryptedHeaderAndBody(spbConn, cc.sharedKey, utils.MessageBeatLen)
				cc.secondReadFlowA = cc.secondReadAtomA.AddAndGetNew(int64(n))
				if err != nil {
					Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "read header err: "+err.Error())
					return
				}

				header.DecodeHeader(headerBytes, &msgH)
				headerBytes = nil
				if msgH.Version < cc.opts.minAppVer {
					utils.DetailLogf("received a [%v] message with an outdated [%v] version (min version [%v])", utils.ByteToString(msgH.Cmd), msgH.Version, cc.opts.minAppVer)
					continue
				}

				// Mylog(cc.opts.logOpen,"client msg size", msgH.Cmd)
				if msgH.Len == 0 {
					handler := core.GetHandlerFunc(utils.ByteToString(msgH.Cmd))
					if handler != nil {
						handlerCh <- MsgHandler{msg.RelayMsgBuf{}, handler, recvStart}
					}
				}
			} else {
				// start to process the msg if there is more than just the header to read
				nonce, dataLen, n, err := core.ReadEncryptedHeader(spbConn)
				cc.secondReadFlowA = cc.secondReadAtomA.AddAndGetNew(int64(n))
				if err != nil {
					Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "read encrypted header err: "+err.Error())
					return
				}
				if dataLen > utils.MessageBeatLen {
					Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "read encrypted header err: over sized [%v], for cmd [%v]", dataLen, utils.ByteToString(msgH.Cmd))
					return
				}

				var onereadlen = 1024
				msgBuf := make([]byte, 0, utils.MessageBeatLen)
				cmd := utils.ByteToString(msgH.Cmd)
				for ; i < int(dataLen); i = i + n {
					if int(dataLen)-i < 1024 {
						onereadlen = int(dataLen) - i
					}
					n, err = io.ReadFull(spbConn, msgBuf[i:i+onereadlen])
					cc.secondReadFlowA = cc.secondReadAtomA.AddAndGetNew(int64(n))
					if err != nil {
						Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "read server body err: "+err.Error())
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

				if uint32(i) == dataLen {
					decryptedBody, err := encryption.DecryptAES(cc.sharedKey, msgBuf[:dataLen], nonce)
					if err != nil {
						utils.ErrorLog("client body decryption err", err)
						return
					}

					message = &msg.RelayMsgBuf{
						MSGHead: msgH,
						MSGData: decryptedBody,
					}
					handler := core.GetHandlerFunc(cmd)
					//Mylog(cc.opts.logOpen, "read handler:", handler, utils.ByteToString(msgH.Cmd))
					if handler == nil {
						if onMessage != nil {
							onMessage(*message, c.(core.WriteCloser))
						} else {
							Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "no handler or onMessage() found for message: "+utils.ByteToString(msgH.Cmd))
						}
						msgH.Len = 0
						i = 0
						msgBuf = nil
						continue
					}
					handlerCh <- MsgHandler{*message, handler, recvStart}
					msgH.Len = 0
					msgBuf = nil
					i = 0

				} else {
					Mylog(cc.opts.logOpen, LOG_MODULE_READLOOP, "msgH.Len doesn't match the size of data for message: "+utils.ByteToString(msgH.Cmd))
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
				packet = nil
			}
		}
	}
}

func (cc *ClientConn) writePacket(packet *msg.RelayMsgBuf) error {
	var lr utils.LimitRate
	var onereadlen = 1024
	var n int
	// Mylog(cc.opts.logOpen, "write header", packet.MSGData[:16])
	// Mylog(cc.opts.logOpen, "write body", packet.MSGData[16:])
	cmd := utils.ByteToString(packet.MSGHead.Cmd)

	// Encrypt and write message header
	encryptedHeader, err := core.EncryptAndPack(cc.sharedKey, packet.MSGData[:utils.MsgHeaderLen])
	if err != nil {
		return errors.Wrap(err, "client cannot encrypt header")
	}
	//_ = cc.spbConn.SetDeadline(time.Now().Add(time.Duration(utils.WriteTimeOut) * time.Second))
	if err = core.WriteFull(cc.spbConn, encryptedHeader); err != nil {
		return errors.Wrap(err, "client write err")
	}
	cc.secondWriteFlowA = cc.secondWriteAtomA.AddAndGetNew(int64(len(encryptedHeader)))

	// Encrypt and write message data
	encryptedData, err := core.EncryptAndPack(cc.sharedKey, packet.MSGData[utils.MsgHeaderLen:])
	if err != nil {
		return errors.Wrap(err, "server cannot encrypt msg")
	}
	writeStart := time.Now()
	for i := 0; i < len(encryptedData); i = i + n {
		// Mylog(cc.opts.logOpen,"len(msgBuf[0:msgH.Len]):", i)
		if len(encryptedData)-i < 1024 {
			onereadlen = len(encryptedData) - i
			// Mylog(cc.opts.logOpen,"onereadlen:", onereadlen)
		}
		//_ = cc.spbConn.SetDeadline(time.Now().Add(time.Duration(utils.WriteTimeOut) * time.Second))
		n, err = cc.spbConn.Write(encryptedData[i : i+onereadlen])
		cc.secondWriteFlowA = cc.secondWriteAtomA.AddAndGetNew(int64(n))
		// Mylog(cc.opts.logOpen,"server n = ", msgBuf[0:msgH.Len])
		// Mylog(cc.opts.logOpen,"i+onereadlen:", i+onereadlen)
		if err != nil {
			return errors.Wrap(err, "client write err")
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
	writeEnd := time.Now()
	costTime := writeEnd.Sub(writeStart).Milliseconds() + 1 // +1 in case of LT 1 ms

	for _, c := range cc.writeHook {
		if cmd == c.Message && c.Fn != nil {
			c.Fn(packet.PacketId, costTime)
		}
	}
	cmem.Free(packet.Alloc)
	return nil
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
	var log string = "handler start"
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
			log = utils.ByteToString(msgHandler.message.MSGHead.Cmd)
			handler(ctx, c)
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
