package peers

import (
	"context"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

// PeerList is a list of the know PP node peers
var peerList types.PeerList

const (
	RELOAD_PP_LIST_INTERVAL_SHORT  = 5 * time.Second
	RELOAD_PP_LIST_INTERVAL_MEDIUM = 15 * time.Second
	RELOAD_PP_LIST_INTERVAL_LONG   = 30 * time.Second
)

// InitPPList
func InitPPList(ctx context.Context) {
	pplist, _, _ := peerList.GetPPList(ctx)
	if len(pplist) == 0 {
		GetPPListFromSP(ctx)
	} else {
		if success := ConnectToGatewayPP(ctx, pplist); !success {
			GetPPListFromSP(ctx)
			return
		}
		if setting.IsAuto && setting.State == types.PP_ACTIVE && !setting.IsLoginToSP {
			RegisterToSP(ctx, true)
		}
	}
}

func StartPpLatencyCheck(ctx context.Context) {
	ppPeerClock.AddJobRepeat(time.Second*setting.PpLatencyCheckInterval, 0, LatencyOfNextPp(ctx))
}

func StartStatusReportToSP(ctx context.Context) {
	utils.DebugLog("Status will be reported to SP while mining")
	// trigger first report at time-0 immediately
	ReportNodeStatus(ctx)()
	// trigger consecutive reports with interval
	ppPeerClock.AddJobRepeat(time.Second*setting.NodeReportIntervalSec, 0, ReportNodeStatus(ctx))
}

// GetPPListFromSP node get ppList from sp
func GetPPListFromSP(ctx context.Context) {
	pp.DebugLog(ctx, "SendMessage(client.SPConn, req, header.ReqGetPPList)")
	SendMessageToSPServer(ctx, requests.ReqGetPPlistData(), header.ReqGetPPList)
}

func ConnectToGatewayPP(ctx context.Context, pplist []*types.PeerInfo) bool {
	for _, ppInfo := range pplist {
		if ppInfo.NetworkAddress == setting.NetworkAddress {
			peerList.DeletePPByNetworkAddress(ctx, ppInfo.NetworkAddress)
			continue
		}
		ppConn, err := client.NewClient(ppInfo.NetworkAddress, true)
		if ppConn != nil {
			client.PPConn = ppConn
			return true
		}
		pp.DebugLogf(ctx, "failed to connect to PP %v: %v", ppInfo, utils.FormatError(err))
		peerList.DeletePPByNetworkAddress(ctx, ppInfo.NetworkAddress)
	}
	return false
}

//ScheduleReloadPPlist
//	Long: 	pp not activated
//	Medium: mining not yet started
//	Short: 	by default (mining)

//func ScheduleReloadPPlist() {
//	var future time.Duration
//	if setting.State != types.PP_ACTIVE {
//		future = RELOAD_PP_LIST_INTERVAL_LONG
//	} else if !setting.IsStartMining {
//		future = RELOAD_PP_LIST_INTERVAL_MEDIUM
//	} else {
//		future = RELOAD_PP_LIST_INTERVAL_SHORT
//	}
//	utils.DebugLog("scheduled to get pp-list after: ", future.Seconds(), "second")
//	ppPeerClock.AddJobWithInterval(future, GetPPListFromSP)
//}

//GetPPList will just get the list from
func GetPPList(ctx context.Context) (list []*types.PeerInfo, total int64) {
	list, total, _ = peerList.GetPPList(ctx)
	return
}

//SavePPList will save the target list to local list
func SavePPList(ctx context.Context, target *protos.RspGetPPList) error {
	return peerList.SavePPList(ctx, target)
}

//GetPPByP2pAddress
func GetPPByP2pAddress(ctx context.Context, p2pAddr string) *types.PeerInfo {
	return peerList.GetPPByP2pAddress(ctx, p2pAddr)
}

//UpdatePP will update one pp info to local list
func UpdatePP(ctx context.Context, pp *types.PeerInfo) {
	peerList.UpdatePP(ctx, pp)
}

//LantencyOfNextPp
func LatencyOfNextPp(ctx context.Context) func() {
	return func() {
		list, _, _ := peerList.GetPPList(ctx)
		for _, peer := range list {
			if peer.Latency == 0 {
				StartLatencyCheckToPp(ctx, peer.NetworkAddress)
			}
		}
	}
}

// StartLatencyCheckToPp
func StartLatencyCheckToPp(ctx context.Context, NetworkAddr string) error {
	start := time.Now().UnixNano()
	pb := &protos.ReqLatencyCheck{
		HbType:   protos.HeartbeatType_LATENCY_CHECK_PP,
		PingTime: strconv.FormatInt(start, 10),
	}
	data, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	msg := &msg.RelayMsgBuf{
		MSGHead: header.MakeMessageHeader(1, uint16(setting.Config.Version.AppVer), uint32(len(data)), header.ReqLatencyCheck),
		MSGData: data,
	}

	TransferSendMessageToPPServ(ctx, NetworkAddr, msg)
	return nil
}
