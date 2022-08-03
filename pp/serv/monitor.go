package serv

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	Lines uint64 `json:"lines"`
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

var subscriptions = &sync.Map{}

// subscribeTrafficInfo subscribe the channel to receive the traffic info from the generator
func subscribeTrafficInfo(s rpc.Subscription, c chan TrafficInfo) {
	utils.DebugLog("Subscribed TI")
	subscriptions.Store(s, c)
}

// unsubscribeTrafficInfo unsubscribe the channel listening to the traffic info
func unsubscribeTrafficInfo(s rpc.Subscription) {
	utils.DebugLog("UN-subscribed TI")
	subscriptions.Delete(s)
}

// TrafficInfoToMonitorClient traffic info generator calls this to feed the notifiers to the subscribed clients
func TrafficInfoToMonitorClient(t TrafficInfo) {
	utils.DebugLog("Sending TI to clients:")
	subscriptions.Range(func(k, v interface{}) bool {
		utils.DebugLog("-->")
		v.(chan TrafficInfo) <- t
		return true
	})
}

// GetTrafficData fetch the traffic data from the file
func (api *monitorApi) GetTrafficData(param ParamTrafficInfo) *TrafficDataResult {
	lines := utils.GetLastLinesFromTrafficLog(setting.TrafficLogPath, param.Lines)

	var ts []TrafficInfo
	var i uint64
	var line string
	for i = 0; i < param.Lines; i++ {
		line = lines[i]
		date := line[17:36]
		content := strings.SplitN(line, "{", 2)
		if len(content) < 2 {
			return &TrafficDataResult{Return: "-1"}
		}

		c := "{" + content[1]

		var t TrafficDumpInfo
		if err := json.Unmarshal([]byte(c), &t); err != nil {
			return &TrafficDataResult{Return: "-1"}
		}

		t.TrafficInfo.TimeStamp = date
		ts = append(ts, t.TrafficInfo)
	}

	return &TrafficDataResult{Return: "0", TraffInfo: ts}
}

// GetDiskUsage the size of files in setting.Config.StorehousePath, not the disk usage of the computer
func (api *monitorApi) GetDiskUsage() *DiskUsageResult {
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

	return &DiskUsageResult{Return: "0", DataHost: size}
}

// GetPeerList the peer pp list
func (api *monitorApi) GetPeerList() *PeerListResult {
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
	}
}

// GetOnlineState if the pp node is online
func (api *monitorApi) GetOnlineState() *OnlineStateResult {
	if setting.OnlineTime == 0 {
	}
	return &OnlineStateResult{
		Return: "0",
		Online: setting.OnlineTime != 0,
		Since:  setting.OnlineTime,
	}
}

// Subscription client calls the method monitor_subscribe with this function as the parameter
func (api *monitorApi) Subscription(ctx context.Context) (*rpc.Subscription, error) {
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
