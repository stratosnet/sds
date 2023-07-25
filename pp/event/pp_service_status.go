package event

import (
	"context"
	"fmt"

	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/network"
)

// RspGetPPStatus
func GetPPServiceStatus(ctx context.Context) rpc.ServiceStatusResult {
	pp.DebugLogf(ctx, "get GetPPServiceStatus request")
	rpcResult := &rpc.ServiceStatusResult{Return: rpc.SUCCESS}
	rpcResult.Message = formatPPServiceStatus(ctx)
	return *rpcResult
}

func formatPPServiceStatus(ctx context.Context) string {
	regStatStr, onlineStatStr := "", ""
	regStat := network.GetPeer(ctx).GetStateFromFsm()
	switch regStat.Id {
	case network.STATE_NOT_REGISTERED:
		regStatStr = "Not registered"
		onlineStatStr = "OFFLINE"
	case network.STATE_REGISTERING:
		regStatStr = "Registering"
		onlineStatStr = "OFFLINE"
	case network.STATE_REGISTERED:
		regStatStr = "Registered"
		onlineStatStr = "ONLINE"
	default:
		regStatStr = "Unknown"
		onlineStatStr = "Unknown"
	}

	msgStr := fmt.Sprintf("*** current service status of pp node ***\n"+
		"Registration Status: %v | Mining: %v ", regStatStr, onlineStatStr)
	pp.Log(ctx, msgStr)
	return msgStr
}
