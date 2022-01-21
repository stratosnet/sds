package event

import (
	"context"
	"crypto/ed25519"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/utils"
)

// ReqFileSliceBackupNotice An SP node wants this PP node to fetch the specified slice from the PP node who stores it
func ReqFileSliceBackupNotice(ctx context.Context, conn core.WriteCloser) {
	utils.DebugLog("get ReqFileSliceBackupNotice")
	var target protos.ReqFileSliceBackupNotice
	if !requests.UnmarshalData(ctx, &target) {
		return
	}
	utils.DebugLog("target = ", target)

	if target.PpInfo.P2PAddress == setting.P2PAddress {
		utils.DebugLog("Ignoring slice backup notice because this node already owns the file")
		return
	}

	signMessage := target.FileHash + "#" + target.SliceStorageInfo.SliceHash + "#" + target.SpP2PAddress
	if !ed25519.Verify(target.Pubkey, []byte(signMessage), target.Sign) {
		utils.ErrorLog("Invalid slice backup notice signature")
		return
	}

	if !task.CheckTransfer(&target) {
		utils.DebugLog("CheckTransfer failed")
		return
	}

	task.TransferTaskMap[target.TaskId] = task.TransferTask{
		FromSp:           true,
		PpInfo:           target.PpInfo,
		SliceStorageInfo: target.SliceStorageInfo,
	}

	peers.TransferSendMessageToPPServ(target.PpInfo.NetworkAddress, requests.ReqTransferDownloadData(&target))
}
