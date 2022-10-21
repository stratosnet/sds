package event

import (
	"context"
	"crypto/ed25519"
	"strconv"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	utiled25519 "github.com/stratosnet/sds/utils/crypto/ed25519"
)

func RspSpUnderMaintenance(ctx context.Context, conn core.WriteCloser) {
	var target protos.RspSpUnderMaintenance
	if !requests.UnmarshalData(ctx, &target) {
		return
	}

	switch conn.(type) {
	case *core.ServerConn:
		utils.DebugLog("Ignore RspSpUnderMaintenance in SeverConn")
		return
	case *cf.ClientConn:
		if conn.(*cf.ClientConn).GetName() != client.SPConn.GetName() {
			utils.DebugLog("Ignore RspSpUnderMaintenance from non SP node")
			return
		}

		if !verifySignature(target) {
			utils.DebugLog("signature not verified in RspSpUnderMaintenance")
			return
		}

		if target.MaintenanceType == int32(protos.SpMaintenanceType_CONSENSUS) {
			utils.Logf("SP[%v] is currently under maintenance, maintenance_type: %v",
				target.SpP2PAddress, protos.SpMaintenanceType_CONSENSUS.String())

			// record SpMaintenance
			triggerSpSwitch := client.RecordSpMaintenance(target.SpP2PAddress, target.Time)
			if triggerSpSwitch {
				ReqHBLatencyCheckSpList(ctx, conn)
			}
		}
	}
}

func verifySignature(target protos.RspSpUnderMaintenance) bool {
	sign := target.Sign
	if len(sign) < 1 {
		return false
	}

	val, ok := setting.SPMap.Load(target.SpP2PAddress)
	if !ok {
		utils.ErrorLog("cannot find sp info by given the SP address ", target.SpP2PAddress)
		return false
	}

	spInfo, ok := val.(setting.SPBaseInfo)
	if !ok {
		utils.ErrorLog("Fail to parse SP info ", target.SpP2PAddress)
		return false
	}

	_, pubKeyRaw, err := bech32.DecodeAndConvert(spInfo.P2PPublicKey)
	if err != nil {
		utils.ErrorLog("Error when trying to decode P2P pubKey bech32", err)
		return false
	}

	p2pPubKey := utiled25519.PubKeyBytesToPubKey(pubKeyRaw)

	timeStr := strconv.FormatInt(target.Time, 10)
	if !ed25519.Verify(p2pPubKey.Bytes(), []byte(target.SpP2PAddress+timeStr+header.RspSpUnderMaintenance), sign) {
		return false
	}

	return true
}
