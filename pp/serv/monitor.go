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

	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

const (
	MSG_GET_TRAFFIC_DATA_RESPONSE = "monitor_getTrafficData"
	MSG_GET_DIST_USAGE_RESPONSE   = "monitor_getDiskUsage"
	MSG_GET_PEER_LIST             = "monitor_getPeerList"
	MSG_GET_ONLINE_STATE          = "monitor_getOnlineState"
	MSG_GET_NODE_DETAILS          = "monitor_getNodeDetails"
)

type DiskUsage struct {
	DataHost int64 `json:"data_host"`
}

type PeerInfo struct {
	NetworkAddress string `json:"network_address"`
	P2pAddress     string `json:"p2p_address"`
	Status         int    `json:"status"`
	Latency        int64  `json:"latency"`
	Connection     string `json:"connection"`
}

type PeerList struct {
	Total    int64      `json:"total"`
	PeerList []PeerInfo `json:"peerlist"`
}

type OnlineState struct {
	Online bool  `json:"online"`
	Since  int64 `json:"since"`
}

type NodeDetails struct {
	Id              string `json:"id"`
	Address         string `json:"address"`
	AdvancedDetails *[]struct {
		Id      string `json:"id"`
		Title   string `json:"title"`
		Details string `json:"details"`
	} `json:"advanced_details,omitempty"`
}

type MonitorResult struct {
	Return      string         `json:"return"`
	MessageType string         `json:"message_type"`
	TrafficInfo *[]TrafficInfo `json:"traffic_info,omitempty"`
	OnlineState *OnlineState   `json:"online_state,omitempty"`
	PeerList    *PeerList      `json:"peer_list,omitempty"`
	DiskUsage   *DiskUsage     `json:"disk_usage,omitempty"`
	NodeDetails *NodeDetails   `json:"node_details,omitempty"`
}

type MonitorNotificationResult struct {
	TrafficInfo *TrafficInfo `json:"traffic_info"`
	OnlineState *OnlineState `json:"online_state,omitempty"`
	PeerList    *PeerList    `json:"peer_list,omitempty"`
	DiskUsage   *DiskUsage   `json:"disk_usage,omitempty"`
}

type ParamTrafficInfo struct {
	SubId string `json:"subid"`
	Lines uint64 `json:"lines"`
}
type ParamMonitor struct {
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
func (api *monitorApi) GetTrafficData(ctx context.Context, param ParamTrafficInfo) (*MonitorResult, error) {
	if _, found := subscribedIds.Load(param.SubId); !found {
		return nil, errors.New("client hasn't subscribed to the service")
	}
	lines := utils.GetLastLinesFromTrafficLog(setting.TrafficLogPath, param.Lines)

	var ts []TrafficInfo
	var line string
	for _, line = range lines {
		if len(line) <= 26 {
			return &MonitorResult{Return: "-1", MessageType: MSG_GET_TRAFFIC_DATA_RESPONSE}, nil
		}

		date := line[7:26]

		content := strings.SplitN(line, "{", 2)
		if len(content) < 2 {
			return &MonitorResult{Return: "-1", MessageType: MSG_GET_TRAFFIC_DATA_RESPONSE}, nil
		}

		c := "{" + content[1]
		var trafficDumpInfo TrafficDumpInfo
		err := json.Unmarshal([]byte(c), &trafficDumpInfo)
		if err != nil {
			return &MonitorResult{Return: "-1", MessageType: MSG_GET_TRAFFIC_DATA_RESPONSE}, nil
		}

		trafficDumpInfo.TrafficInfo.TimeStamp = date
		ts = append(ts, trafficDumpInfo.TrafficInfo)
	}
	if ts == nil {
		return &MonitorResult{Return: "-1", MessageType: MSG_GET_TRAFFIC_DATA_RESPONSE}, nil
	}

	return &MonitorResult{Return: "0", MessageType: MSG_GET_TRAFFIC_DATA_RESPONSE, TrafficInfo: &ts}, nil
}

func getDiskUsage() int64 {
	var size int64
	_ = filepath.Walk(setting.Config.StorehousePath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size
}

// GetDiskUsage the size of files in setting.Config.StorehousePath, not the disk usage of the computer
func (api *monitorApi) GetDiskUsage(ctx context.Context, param ParamMonitor) (*MonitorResult, error) {
	if _, found := subscribedIds.Load(param.SubId); !found {
		return nil, errors.New("client hasn't subscribed to the service")
	}

	return &MonitorResult{Return: "0", MessageType: MSG_GET_DIST_USAGE_RESPONSE, DiskUsage: &DiskUsage{DataHost: getDiskUsage()}}, nil
}
func getPeerList(ctx context.Context) ([]PeerInfo, int64) {
	if ps := p2pserver.GetP2pServer(ctx); ps != nil {
		pl, t, _ := ps.GetPPList(ctx)
		var peer PeerInfo
		var peers []PeerInfo
		var i int64
		for i = 0; i < t; i++ {
			peer = PeerInfo{
				NetworkAddress: pl[i].NetworkAddress,
				P2pAddress:     pl[i].P2pAddress,
				Status:         pl[i].Status,
				Latency:        pl[i].Latency,
				Connection:     "tcp4",
			}
			peers = append(peers, peer)
		}
		return peers, t
	}
	return nil, 0

}

// GetPeerList the peer pp list
func (api *monitorApi) GetPeerList(ctx context.Context, param ParamMonitor) (*MonitorResult, error) {
	if _, found := subscribedIds.Load(param.SubId); !found {
		return nil, errors.New("client hasn't subscribed to the service")
	}
	peers, t := getPeerList(ctx)
	return &MonitorResult{
		Return:      "0",
		MessageType: MSG_GET_PEER_LIST,
		PeerList:    &PeerList{Total: t, PeerList: peers},
	}, nil
}

// GetOnlineState if the pp node is online
func (api *monitorApi) GetOnlineState(ctx context.Context, param ParamMonitor) (*MonitorResult, error) {
	if _, found := subscribedIds.Load(param.SubId); !found {
		return nil, errors.New("client hasn't subscribed to the service")
	}
	return &MonitorResult{
		Return:      "0",
		MessageType: MSG_GET_ONLINE_STATE,
		OnlineState: &OnlineState{Online: setting.OnlineTime != 0, Since: setting.OnlineTime},
	}, nil
}

// GetNodeDetail the deatils of the node
func (api *monitorApi) GetNodeDetails(ctx context.Context, param ParamMonitor) (*MonitorResult, error) {
	if _, found := subscribedIds.Load(param.SubId); !found {
		return nil, errors.New("client hasn't subscribed to the service")
	}
	return &MonitorResult{
		Return:      "0",
		MessageType: MSG_GET_NODE_DETAILS,
		NodeDetails: &NodeDetails{Id: "1", Address: setting.P2PAddress},
	}, nil
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
				peers, t := getPeerList(ctx)
				result := &MonitorNotificationResult{
					TrafficInfo: &ti,
					OnlineState: &OnlineState{Online: setting.OnlineTime != 0, Since: setting.OnlineTime},
					PeerList:    &PeerList{Total: t, PeerList: peers},
					DiskUsage:   &DiskUsage{DataHost: getDiskUsage()},
				}
				_ = notifier.Notify(rpcSub.ID, result)
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
