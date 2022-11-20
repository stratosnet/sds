package p2pserver

import (
	"context"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

// offline offline
type offline struct {
	IsSp           bool
	NetworkAddress string
}

// initClient
func (p *P2pServer) initClient() {
	p.offlineChan = make(chan *offline, 2)
	p.uploadConnMap = &sync.Map{}
	p.downloadConnMap = &sync.Map{}
	p.connMap = make(map[string]*cf.ClientConn)
}

// NewClient
func (p *P2pServer) NewClient(ctx context.Context, server string, heartbeat bool) (*cf.ClientConn, error) {
	utils.DebugLog("NewClient:", server)
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
		//p.ClientMutex.Lock()
		delete(p.connMap, c.(*cf.ClientConn).GetName())
		//p.ClientMutex.Unlock()

		if p.ppConn != nil {
			if p.ppConn == c.(*cf.ClientConn) {
				utils.DebugLog("lost gateway PP conn, delete and change to new PP")
				select {
				case p.offlineChan <- &offline{
					IsSp:           false,
					NetworkAddress: p.ppConn.GetRemoteAddr(),
				}:
				default:
					break
				}
			}
		}
		if p.spConn != nil {
			if p.spConn.GetName() == c.(*cf.ClientConn).GetName() {
				utils.DebugLog("lost SP conn, name: ", p.spConn.GetName(), " netId is ", p.spConn.GetNetID())
				p.spConn = nil
				select {
				case p.offlineChan <- &offline{
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

	var ckv []cf.ContextKV
	for _, key := range p.connContextKey {
		ckv = append(ckv, cf.ContextKV{Key: key, Value: ctx.Value(key)})
	}

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
		cf.ContextKVOption(ckv),
	}
	utils.Logf("attempting to connect to %v", server)
	conn := cf.CreateClientConn(0, c, options...)
	conn.Start()
	//p.ClientMutex.Lock()
	p.connMap[server] = conn
	//p.ClientMutex.Unlock()

	return conn, nil
}

// GetConnectionName
func (p *P2pServer) GetConnectionName(conn core.WriteCloser) string {
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

// GetClientConn
func (p *P2pServer) GetClientConn(networkAddr string) (*cf.ClientConn, bool) {
	//p.ClientMutex.Lock()
	//defer p.ClientMutex.Unlock()
	if cc, ok := p.connMap[networkAddr]; ok {
		return cc, true
	} else {
		return nil, false
	}
}

// CleanUpConnMap
func (p *P2pServer) CleanUpConnMap(fileHash string) {
	p.uploadConnMap.Range(func(k, v interface{}) bool {
		if strings.HasPrefix(k.(string), fileHash) {
			p.uploadConnMap.Delete(k.(string))
		}
		return true
	})
}

// SetPpClientConn
func (p *P2pServer) SetPpClientConn(ppConn *cf.ClientConn) {
	p.ppConn = ppConn
}

func (p *P2pServer) ReadOfflineChan() chan *offline {
	return p.offlineChan
}

func (p *P2pServer) SpConnValid() bool {
	return p.spConn != nil
}

func (p *P2pServer) GetSpName() string {
	if p.spConn == nil {
		return "{NA} Invalid SpConn"
	}
	return p.spConn.GetName()
}

// StoreDownloadConn access function for member downloadConnMap
func (p *P2pServer) StoreDownloadConn(key string, conn *cf.ClientConn) {
	p.downloadConnMap.Store(key, conn)
}

// LoadDownloadConn access function for member downloadConnMap
func (p *P2pServer) LoadDownloadConn(key string) (*cf.ClientConn, bool) {
	if c, ok := p.downloadConnMap.Load(key); ok {
		return c.(*cf.ClientConn), true
	} else {
		return nil, false
	}
}

// DeleteDownloadConn access function for member downloadConnMap
func (p *P2pServer) DeleteDownloadConn(key string) {
	p.downloadConnMap.Delete(key)
}

// RangeDownloadConn access function for member downloadConnMap
func (p *P2pServer) RangeDownloadConn(rf func(k, v interface{}) bool) {
	p.downloadConnMap.Range(rf)
}

// StoreUploadConn access function for member downloadConnMap
func (p *P2pServer) StoreUploadConn(key string, conn *cf.ClientConn) {
	p.uploadConnMap.Store(key, conn)
}

// LoadUploadConn access function for member downloadConnMap
func (p *P2pServer) LoadUploadConn(key string) (*cf.ClientConn, bool) {
	if c, ok := p.uploadConnMap.Load(key); ok {
		return c.(*cf.ClientConn), true
	} else {
		return nil, false
	}
}

// DeleteUploadConn access function for member downloadConnMap
func (p *P2pServer) DeleteUploadConn(key string) {
	p.uploadConnMap.Delete(key)
}

// RangeUploadConn access function for member downloadConnMap
func (p *P2pServer) RangeUploadConn(rf func(k, v interface{}) bool) {
	p.uploadConnMap.Range(rf)
}

// GetSpConn
func (p *P2pServer) GetSpConn() *cf.ClientConn {
	return p.spConn
}

// GetPpConn
func (p *P2pServer) GetPpConn() *cf.ClientConn {
	return p.ppConn
}

// RecordSpMaintenance, return boolean flag of switching to new SP
func (p *P2pServer) RecordSpMaintenance(spP2pAddress string, recordTime time.Time) bool {
	if p.SPMaintenanceMap == nil {
		p.resetSPMaintenanceMap(spP2pAddress, recordTime, MIN_RECONNECT_INTERVAL_THRESHOLD)
		return true
	}
	if value, ok := p.SPMaintenanceMap.Load(LAST_RECONNECT_KEY); ok {
		lastRecord := value.(*LastReconnectRecord)
		if time.Now().Before(lastRecord.Time.Add(time.Duration(lastRecord.NextAllowableReconnectInSec) * time.Second)) {
			// if new maintenance rsp incoming in between the interval, extend the KV by storing it again (not changing value)
			p.SPMaintenanceMap.Store(LAST_RECONNECT_KEY, lastRecord)
			return false
		}
		// if new maintenance rsp incoming beyond the interval, reset the map and modify the NextAllowableReconnectInSec
		nextReconnectInterval := int64(math.Min(MAX_RECONNECT_INTERVAL_THRESHOLD,
			float64(lastRecord.NextAllowableReconnectInSec*RECONNECT_INTERVAL_MULTIPLIER)))
		p.resetSPMaintenanceMap(spP2pAddress, recordTime, nextReconnectInterval)
		return true
	}
	p.resetSPMaintenanceMap(spP2pAddress, recordTime, MIN_RECONNECT_INTERVAL_THRESHOLD)
	return true
}

func (p *P2pServer) resetSPMaintenanceMap(spP2pAddress string, recordTime time.Time, nextReconnectInterval int64) {
	// reset the interval to 60s
	p.SPMaintenanceMap = nil
	p.SPMaintenanceMap = utils.NewAutoCleanMap(time.Duration(nextReconnectInterval) * time.Second)
	p.SPMaintenanceMap.Store(LAST_RECONNECT_KEY, &LastReconnectRecord{
		SpP2PAddress:                spP2pAddress,
		Time:                        recordTime,
		NextAllowableReconnectInSec: nextReconnectInterval,
	})
}
