package event

// client pp event handler
import (
	"context"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"reflect"

	"github.com/golang/protobuf/proto"
)

// RegisterEventHandle
func RegisterEventHandle() {
	spbf.Register(header.RspGetPPList, RspGetPPList)
	spbf.Register(header.RspRegister, RspRegisterChain)
	spbf.Register(header.ReqRegister, ReqRegisterChain)
	spbf.Register(header.RspMining, RspMining)
	spbf.Register(header.RspActivate, RspActivate)
	spbf.Register(header.RspActivated, RspActivated)
	spbf.Register(header.RspFindMyFileList, RspFindMyFileList)
	spbf.Register(header.ReqFindMyFileList, ReqFindMyFileList)
	spbf.Register(header.ReqUploadFileSlice, ReqUploadFileSlice)
	spbf.Register(header.RspUploadFile, RspUploadFile)
	spbf.Register(header.RspUploadFileSlice, RspUploadFileSlice)
	spbf.Register(header.RspReportUploadSliceResult, RspReportUploadSliceResult)
	spbf.Register(header.ReqFileStorageInfo, ReqFileStorageInfo)
	spbf.Register(header.ReqDownloadSlice, ReqDownloadSlice)
	spbf.Register(header.RspDownloadSlice, RspDownloadSlice)
	spbf.Register(header.RspReportDownloadResult, RspReportDownloadResult)
	spbf.Register(header.RspRegisterNewPP, RspRegisterNewPP)
	spbf.Register(header.ReqTransferNotice, ReqTransferNotice)
	spbf.Register(header.RspValidateTransferCer, RspValidateTransferCer)
	spbf.Register(header.ReqTransferDownload, ReqTransferDownload)
	spbf.Register(header.RspTransferDownload, RspTransferDownload)
	spbf.Register(header.RspTransferDownloadResult, RspTransferDownloadResult)
	spbf.Register(header.RspReportTransferResult, RspReportTransferResult)
	spbf.Register(header.RspDownloadSliceWrong, RspDownloadSliceWrong)
	spbf.Register(header.RspFileStorageInfo, RspFileStorageInfo)
	spbf.Register(header.ReqGetHDInfo, ReqGetHDInfo)
	spbf.Register(header.RspGetHDInfo, RspGetHDInfo)
	spbf.Register(header.ReqDeleteSlice, ReqDeleteSlice)
	spbf.Register(header.RspDeleteSlice, RspDeleteSlice)
	spbf.Register(header.ReqMakeDirectory, ReqMakeDirectory)
	spbf.Register(header.RspMakeDirectory, RspMakeDirectory)
	spbf.Register(header.ReqRemoveDirectory, ReqRemoveDirectory)
	spbf.Register(header.RspRemoveDirectory, RspRemoveDirectory)
	spbf.Register(header.ReqMoveFileDirectory, ReqMoveFileDirectory)
	spbf.Register(header.RspMoveFileDirectory, RspMoveFileDirectory)
	spbf.Register(header.ReqDownloadSlicePause, ReqDownloadSlicePause)
	spbf.Register(header.RspDownloadSlicePause, RspDownloadSlicePause)
	spbf.Register(header.ReqCreateAlbum, ReqCreateAlbum)
	spbf.Register(header.RspCreateAlbum, RspCreateAlbum)
	spbf.Register(header.ReqFindMyAlbum, ReqFindMyAlbum)
	spbf.Register(header.RspFindMyAlbum, RspFindMyAlbum)
	spbf.Register(header.ReqEditAlbum, ReqEditAlbum)
	spbf.Register(header.RspEditAlbum, RspEditAlbum)
	spbf.Register(header.ReqAlbumContent, ReqAlbumContent)
	spbf.Register(header.RspAlbumContent, RspAlbumContent)
	spbf.Register(header.ReqSearchAlbum, ReqSearchAlbum)
	spbf.Register(header.RspSearchAlbum, RspSearchAlbum)
	spbf.Register(header.ReqCollectionAlbum, ReqCollectionAlbum)
	spbf.Register(header.RspCollectionAlbum, RspCollectionAlbum)
	spbf.Register(header.ReqAbstractAlbum, ReqAbstractAlbum)
	spbf.Register(header.RspAbstractAlbum, RspAbstractAlbum)
	spbf.Register(header.ReqMyCollectionAlbum, ReqMyCollectionAlbum)
	spbf.Register(header.RspMyCollectionAlbum, RspMyCollectionAlbum)
	spbf.Register(header.ReqDeleteAlbum, ReqDeleteAlbum)
	spbf.Register(header.RspDeleteAlbum, RspDeleteAlbum)
	spbf.Register(header.ReqSaveFolder, ReqSaveFolder)
	spbf.Register(header.RspSaveFolder, RspSaveFolder)
	spbf.Register(header.UploadSpeedOfProgress, UploadSpeedOfProgress)
	spbf.Register(header.ReqGetCapacity, ReqGetCapacity)
	spbf.Register(header.RspGetCapacity, RspGetCapacity)

	spbf.Register(header.ReqShareLink, ReqShareLink)
	spbf.Register(header.RspShareLink, RspShareLink)
	spbf.Register(header.ReqShareFile, ReqShareFile)
	spbf.Register(header.RspShareFile, RspShareFile)
	spbf.Register(header.ReqDeleteShare, ReqDeleteShare)
	spbf.Register(header.RspDeleteShare, RspDeleteShare)
	spbf.Register(header.ReqGetShareFile, ReqGetShareFile)
	spbf.Register(header.RspGetShareFile, RspGetShareFile)

	spbf.Register(header.ReqSaveFile, ReqSaveFile)
	spbf.Register(header.RspSaveFile, RspSaveFile)

	spbf.Register(header.ReqHeart, SendHeartBeat)
	spbf.Register(header.RspHeart, RspHeartBeat)
	spbf.Register(header.RspGetBPList, RspGetBPList)
	spbf.Register(header.ReqDeleteFile, ReqDeleteFile)
	spbf.Register(header.RspDeleteFile, RspDeleteFile)
	spbf.Register(header.ReqConfig, ReqGetMyConfig)
	spbf.Register(header.RspConfig, RspGetMyConfig)

	spbf.Register(header.ReqInvite, ReqInvite)
	spbf.Register(header.RspInvite, RspInvite)
	spbf.Register(header.ReqGetReward, ReqGetReward)
	spbf.Register(header.RspGetReward, RspGetReward)

	spbf.Register(header.ReqFindDirectoryTree, ReqFindDirectoryTree)
	spbf.Register(header.RspFindDirectoryTree, RspFindDirectoryTree)

	spbf.Register(header.ReqFileSort, ReqFileSort)
	spbf.Register(header.RspFileSort, RspFileSort)

	spbf.Register(header.ReqFindDirectory, ReqFindDirectory)
	spbf.Register(header.RspFindDirectory, RspFindDirectory)
}

// PPMsgHeader
func PPMsgHeader(data []byte, head string) header.MessageHead {
	return header.MakeMessageHeader(1, uint16(setting.Config.Version), uint32(len(data)), head)

}

// SendMessage
func sendMessage(conn spbf.WriteCloser, pb proto.Message, cmd string) {
	data, err := proto.Marshal(pb)

	if utils.CheckError(err) {
		utils.ErrorLog("error decoding")
		return
	}
	msg := &msg.RelayMsgBuf{
		MSGHead: PPMsgHeader(data, cmd),
		MSGData: data,
	}
	switch conn.(type) {
	case *spbf.ServerConn:
		conn.(*spbf.ServerConn).Write(msg)
	case *cf.ClientConn:
		conn.(*cf.ClientConn).Write(msg)
	}
}

// SendMessageToSPServer SendMessageToSPServer
func SendMessageToSPServer(pb proto.Message, cmd string) {
	if client.SPConn == nil {
		utils.DebugLog("client.SPConn = client.NewClient(setting.Config.SPNetAddress, setting.IsPP)")
		client.SPConn = client.NewClient(setting.Config.SPNetAddress, setting.IsPP)
	}
	sendMessage(client.SPConn, pb, cmd)
}

// TransferSendMessageToPPServ
func transferSendMessageToPPServ(addr string, msgBuf *msg.RelayMsgBuf) {
	if client.ConnMap[addr] != nil {

		client.ConnMap[addr].Write(msgBuf)
		utils.DebugLog("conn exist, transfer")
	} else {
		utils.DebugLog("new conn, connect and transfer")
		client.NewClient(addr, false).Write(msgBuf)
	}
}

//todo:
// transferSendMessageToPPServ
func sendMessageToBPServ(addr string, msgBuf *msg.RelayMsgBuf) {
	if client.ConnMap[addr] != nil {

		utils.DebugLog("exist BP connection, transfer")
		err := client.ConnMap[addr].Write(msgBuf)
		if utils.CheckError(err) {
			GetBPList()
			utils.DebugLog("(1)error report to BP， get BPList again")
		}
	} else {
		utils.DebugLog("new BP connection, connect and transfer")
		cf := client.NewClient(addr, false)
		if cf != nil {
			err := cf.Write(msgBuf)
			if utils.CheckError(err) {
				utils.DebugLog("(2)error report to BP， get BPList again")
				GetBPList()
			}
		} else {
			utils.DebugLog("(3)error report to BP， get BPList again")
			GetBPList()
		}
	}
}

// transferSendMessageToSPServer
func transferSendMessageToSPServer(msg *msg.RelayMsgBuf) {
	if client.SPConn == nil {
		client.SPConn = client.NewClient(setting.Config.SPNetAddress, setting.IsPP)
	}
	client.SPConn.Write(msg)
}

// transferSendMessageToClient
func transferSendMessageToClient(waller string, msgBuf *msg.RelayMsgBuf) {
	if netid, ok := serv.RegisterPeerMap.Load(waller); ok {
		utils.Log("transfer to netid = ", netid)
		serv.GetPPServer().Unicast(netid.(int64), msgBuf)
	} else {
		utils.DebugLog("waller ===== ", waller)
	}
}

// 跟所有BP上报消息
func sendBPMessage(msg chan *msg.RelayMsgBuf) {
	utils.DebugLog("sendBPMessagesendBPMessagesendBPMessagesendBPMessage")
	select {
	case m := <-msg:
		go sendAllBP(m)
	default:
		return
	}
}
func sendAllBP(m *msg.RelayMsgBuf) {
	utils.DebugLog("sendAllBPsendAllBPsendAllBP")
	for _, bp := range setting.BPList {
		sendMessageToBPServ(bp, m)
	}
}

func unmarshalData(ctx context.Context, target interface{}) bool {
	msgBuf := spbf.MessageFromContext(ctx)
	utils.DebugLog("msgBuf len = ", len(msgBuf.MSGData))
	if utils.CheckError(proto.Unmarshal(msgBuf.MSGData, target.(proto.Message))) {
		utils.ErrorLog("protobuf Unmarshal error,target =", reflect.TypeOf(target))
		return false
	}
	if _, ok := reflect.TypeOf(target).Elem().FieldByName("Data"); !ok {
		utils.DebugLog("target = ", target)
	} else {
		utils.DebugLog("analyse target")
	}
	return true
}

// ReqTransferSendSP
func ReqTransferSendSP(ctx context.Context, conn spbf.WriteCloser) {
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}
