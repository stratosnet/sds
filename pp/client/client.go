package client

import (
	"math"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// Offline Offline
type Offline struct {
	IsSp           bool
	NetworkAddress string
}

const (
	LAST_RECONNECT_KEY               = "last_reconnect"
	MIN_RECONNECT_INTERVAL_THRESHOLD = 60  // seconds
	MAX_RECONNECT_INTERVAL_THRESHOLD = 600 // seconds
	RECONNECT_INTERVAL_MULTIPLIER    = 2
)

type LastReconnectRecord struct {
	SpP2PAddress                string
	Time                        time.Time
	NextAllowableReconnectInSec int64
}

// OfflineChan OfflineChan
var OfflineChan = make(chan *Offline, 2)

// SPConn super node connection
var SPConn *cf.ClientConn

// SPMaintenanceMap stores records of SpUnderMaintenance, K - SpP2pAddress, V - list of MaintenanceRecord
var SPMaintenanceMap *utils.AutoCleanMap

// PPConn current connected pp node
var PPConn *cf.ClientConn

// UpConnMap upload connection
var UpConnMap = &sync.Map{}

// DownloadConnMap  download connection between  P-PP
var DownloadConnMap = &sync.Map{}

// ConnMap client connection map
var ConnMap = make(map[string]*cf.ClientConn)

// NewClient
func NewClient(server string, heartbeat bool) (*cf.ClientConn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)
	if err != nil {
		return nil, errors.Wrap(err, "resolve TCP address error")
	}
	c, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	onConnect := cf.OnConnectOption(func(c core.WriteCloser) bool {
		utils.DebugLog("on connect")
		return true
	})
	onError := cf.OnErrorOption(func(c core.WriteCloser) {
		utils.Log("on error")
	})
	onClose := cf.OnCloseOption(func(c core.WriteCloser) {
		utils.Log("on close", c.(*cf.ClientConn).GetName())
		delete(ConnMap, c.(*cf.ClientConn).GetName())

		if PPConn != nil {
			if PPConn == c.(*cf.ClientConn) {
				utils.DebugLog("lost gateway PP conn, delete and change to new PP")
				select {
				case OfflineChan <- &Offline{
					IsSp:           false,
					NetworkAddress: PPConn.GetRemoteAddr(),
				}:
				default:
					break
				}
			}
		}
		if SPConn != nil {
			if SPConn.GetName() == c.(*cf.ClientConn).GetName() {
				utils.DebugLog("lost SP conn, name: ", SPConn.GetName(), " netId is ", SPConn.GetNetID())
				SPConn = nil
				select {
				case OfflineChan <- &Offline{
					IsSp: true,
				}:
				default:
					break
				}
			}
		}

	})

	serverPort, err := strconv.ParseUint(setting.Config.Port, 10, 16)
	if err != nil {
		return nil, errors.Wrapf(err, "Invalid port number in config [%v]", setting.Config.Port)
	}
	serverPortOpt := cf.ServerPortOption(uint16(serverPort))

	options := []cf.ClientOption{
		onConnect,
		onError,
		onClose,
		cf.OnMessageOption(func(msg msg.RelayMsgBuf, c core.WriteCloser) {}),
		cf.BufferSizeOption(100),
		cf.HeartCloseOption(!heartbeat),
		cf.LogOpenOption(true),
		cf.MinAppVersionOption(setting.Config.Version.MinAppVer),
		cf.P2pAddressOption(setting.P2PAddress),
		serverPortOpt,
	}
	utils.Logf("attempting to connect to %v", server)
	conn := cf.CreateClientConn(0, c, options...)
	conn.Start()
	ConnMap[server] = conn

	return conn, nil
}

func GetConnectionName(conn core.WriteCloser) string {
	if conn == nil {
		return ""
	}
	switch conn.(type) {
	case *core.ServerConn:
		return conn.(*core.ServerConn).GetName()
	case *cf.ClientConn:
		return conn.(*cf.ClientConn).GetName()
	}
	return ""
}

// RecordSpMaintenance, return boolean flag of switching to new SP
func RecordSpMaintenance(spP2pAddress string, recordTime time.Time) bool {
	if SPMaintenanceMap == nil {
		resetSPMaintenanceMap(spP2pAddress, recordTime, MIN_RECONNECT_INTERVAL_THRESHOLD)
		return true
	}
	if value, ok := SPMaintenanceMap.Load(LAST_RECONNECT_KEY); ok {
		lastRecord := value.(*LastReconnectRecord)
		if time.Now().Before(lastRecord.Time.Add(time.Duration(lastRecord.NextAllowableReconnectInSec) * time.Second)) {
			// if new maintenance rsp incoming in between the interval, extend the KV by storing it again (not changing value)
			SPMaintenanceMap.Store(LAST_RECONNECT_KEY, lastRecord)
			return false
		}
		// if new maintenance rsp incoming beyond the interval, reset the map and modify the NextAllowableReconnectInSec
		nextReconnectInterval := int64(math.Min(MAX_RECONNECT_INTERVAL_THRESHOLD,
			float64(lastRecord.NextAllowableReconnectInSec*RECONNECT_INTERVAL_MULTIPLIER)))
		resetSPMaintenanceMap(spP2pAddress, recordTime, nextReconnectInterval)
		return true
	}
	resetSPMaintenanceMap(spP2pAddress, recordTime, MIN_RECONNECT_INTERVAL_THRESHOLD)
	return true
}

func resetSPMaintenanceMap(spP2pAddress string, recordTime time.Time, nextReconnectInterval int64) {
	// reset the interval to 60s
	SPMaintenanceMap = nil
	SPMaintenanceMap = utils.NewAutoCleanMap(time.Duration(nextReconnectInterval) * time.Second)
	SPMaintenanceMap.Store(LAST_RECONNECT_KEY, &LastReconnectRecord{
		SpP2PAddress:                spP2pAddress,
		Time:                        recordTime,
		NextAllowableReconnectInSec: nextReconnectInterval,
	})
}
