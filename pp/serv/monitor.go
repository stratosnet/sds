package serv

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

type TrafficDataResult struct {
	Return    string        `json:"return"`
	TraffInfo []TrafficInfo `json:"trafficinfo"`
}

type DiskUsageResult struct {
	Return   string `json:"return"`
	DataHost int64  `json:"datahost"`
}

type PeerInfo struct {
	NetworkAddress string `json:"networkaddress"`
	P2pAddress     string `json:"p2paddress"`
	Status         int    `json:"status"`
	Latency        int64  `json:"latency"`
}

type PeerListResult struct {
	Return   string     `json:"return"`
	Total    int64      `json:"total"`
	PeerList []PeerInfo `json:"peerlist"`
}

type OnlineStateResult struct {
	Return string `json:"return"`
	Online bool   `json:"online"`
	Since  int64  `json:"since"`
}

type ParamTrafficInfo struct {
	SubId string `json:"subid"`
	Lines uint64 `json:"lines"`
}
type ParamGetDiskUsage struct {
	SubId string `json:"subid"`
}

type ParamGetPeerList struct {
	SubId string `json:"subid"`
}

type ParamGetOnLineState struct {
	SubId string `json:"subid"`
}

type monitorApi struct {
}

func MonitorApi() *monitorApi {
	return &monitorApi{}
}

func monitorAPI() []rpc.API {
	return []rpc.API{
		{
			Namespace: "monitor",
			Version:   "1.0",
			Service:   MonitorApi(),
			Public:    true,
		},
	}
}

// key: rpc.Subscription, value: chan TranfficInfo
var subscriptions = &sync.Map{}

// key: rpc.Subscription.ID (string), value: struct{} (nothing)
var subscribedIds = &sync.Map{}

// subscribeTrafficInfo subscribe the channel to receive the traffic info from the generator
func subscribeTrafficInfo(s rpc.Subscription, c chan TrafficInfo) {
	var e struct{}
	subscribedIds.Store(string(s.ID), e)
	subscriptions.Store(s, c)
}

// unsubscribeTrafficInfo unsubscribe the channel listening to the traffic info
func unsubscribeTrafficInfo(s rpc.Subscription) {
	subscribedIds.Delete(s.ID)
	subscriptions.Delete(s)
}

// TrafficInfoToMonitorClient traffic info generator calls this to feed the notifiers to the subscribed clients
func TrafficInfoToMonitorClient(t TrafficInfo) {
	subscriptions.Range(func(k, v interface{}) bool {
		v.(chan TrafficInfo) <- t
		return true
	})
}

// CreateInitialToken the initial token is used to generate all following tokens
func CreateInitialToken() string {
	epoch := strconv.FormatInt(time.Now().Unix(), 10)
	return utils.CalcHash([]byte(epoch + setting.P2PAddress))
}

// calculateToken
func calculateToken(time int64) string {
	t := strconv.FormatInt(time, 10)
	return utils.CalcHash([]byte(t + setting.MonitorInitialToken))
}

// GetCurrentToken
func GetCurrentToken() string {
	return calculateToken(time.Now().Unix() / 3600)
}

// verifyToken verify if the input token matches the generated token
func verifyToken(token string) bool {
	t := time.Now().Unix() / 3600
	if calculateToken(t) == token {
		return true
	} else if calculateToken(t-1) == token {
		return true
	}
	return false
}

// GetTrafficData fetch the traffic data from the file
func (api *monitorApi) GetTrafficData(param ParamTrafficInfo) (*TrafficDataResult, error) {
	if _, found := subscribedIds.Load(param.SubId); !found {
		return nil, errors.New("client hasn't subscribed to the service")
	}
	lines := utils.GetLastLinesFromTrafficLog(setting.TrafficLogPath, param.Lines)

	var ts []TrafficInfo
	var i uint64
	var line string
	for i = 0; i < param.Lines; i++ {
		line = lines[i]
		date := line[17:36]

		content := strings.SplitN(line, "{", 2)
		if len(content) < 2 {
			return &TrafficDataResult{Return: "-1"}, nil
		}

		c := "{" + content[1]

		var t TrafficDumpInfo
		if err := json.Unmarshal([]byte(c), &t); err != nil {
			return &TrafficDataResult{Return: "-1"}, nil
		}

		t.TrafficInfo.TimeStamp = date
		ts = append(ts, t.TrafficInfo)
	}

	return &TrafficDataResult{Return: "0", TraffInfo: ts}, nil
}

// GetDiskUsage the size of files in setting.Config.StorehousePath, not the disk usage of the computer
func (api *monitorApi) GetDiskUsage(param ParamGetDiskUsage) (*DiskUsageResult, error) {
	if _, found := subscribedIds.Load(param.SubId); !found {
		return nil, errors.New("client hasn't subscribed to the service")
	}

	var size int64
	filepath.Walk(setting.Config.StorehousePath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})

	return &DiskUsageResult{Return: "0", DataHost: size}, nil
}

// GetPeerList the peer pp list
func (api *monitorApi) GetPeerList(param ParamGetPeerList) (*PeerListResult, error) {
	if _, found := subscribedIds.Load(param.SubId); !found {
		return nil, errors.New("client hasn't subscribed to the service")
	}

	pl, t := peers.GetPPList()
	var peer PeerInfo
	var peers []PeerInfo
	var i int64
	for i = 0; i < t; i++ {
		peer = PeerInfo{
			NetworkAddress: pl[i].NetworkAddress,
			P2pAddress:     pl[i].P2pAddress,
			Status:         pl[i].Status,
			Latency:        pl[i].Latency,
		}
		peers = append(peers, peer)
	}

	return &PeerListResult{
		Return:   "0",
		Total:    t,
		PeerList: peers,
	}, nil
}

// GetOnlineState if the pp node is online
func (api *monitorApi) GetOnlineState(param ParamGetOnLineState) *OnlineStateResult {
	if setting.OnlineTime == 0 {
	}
	return &OnlineStateResult{
		Return: "0",
		Online: setting.OnlineTime != 0,
		Since:  setting.OnlineTime,
	}
}

// Subscription client calls the method monitor_subscribe with this function as the parameter
func (api *monitorApi) Subscription(ctx context.Context, token string) (*rpc.Subscription, error) {
	if !verifyToken(token) {
		return nil, errors.New("failed token check")
	}
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	if notifier == nil {
		return &rpc.Subscription{}, errors.New("invalid notifier")
	}
	var (
		rpcSub      = notifier.CreateSubscription()
		trafficInfo = make(chan TrafficInfo)
	)
	if rpcSub == nil {
		return nil, errors.New("invalid subscriber")
	}

	subscribeTrafficInfo(*rpcSub, trafficInfo)

	go func() {
		for {
			select {
			case ti := <-trafficInfo:
				notifier.Notify(rpcSub.ID, &ti)
			case <-rpcSub.Err(): // client send an unsubscribe request
				unsubscribeTrafficInfo(*rpcSub)
				return
			case <-notifier.Closed(): // connection dropped
				unsubscribeTrafficInfo(*rpcSub)
				return
			}
		}
	}()

	return rpcSub, nil
}
