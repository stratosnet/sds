package core

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stratosnet/sds/utils"
)

var RemoteReqIdMap = utils.NewAutoCleanMap(24 * time.Hour)

func RegisterRemoteReqId(ctx context.Context, remoteReqId string) context.Context {
	reqId := GetReqIdFromContext(ctx)

	if reqId == 0 {
		reqId, _ = utils.NextSnowFlakeId()
	}

	if remoteReqId == "" {
		remoteReqId = uuid.New().String()
	}
	StoreRemoteReqId(reqId, remoteReqId)
	return CreateContextWithReqId(ctx, reqId)
}

func GetRemoteReqId(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if parentReqId := GetParentReqIdFromContext(ctx); parentReqId > 0 {
		if rootId, ok := RemoteReqIdMap.Load(parentReqId); ok {
			return rootId.(string)
		}
	}

	if reqId := GetReqIdFromContext(ctx); reqId > 0 {
		if rootId, ok := RemoteReqIdMap.Load(reqId); ok {
			return rootId.(string)
		}
	}

	return ""
}

func InheritRemoteReqIdFromParentReqId(ctx context.Context, reqId int64) {
	if ctx == nil || ctx.Value(parentReqIDCtxKey) == nil {
		return
	}
	if parentReqId := GetParentReqIdFromContext(ctx); parentReqId > 0 {
		if rootId, ok := RemoteReqIdMap.Load(parentReqId); ok {
			StoreRemoteReqId(reqId, rootId.(string))
		}
	}
}

func RegisterReqId(ctx context.Context, rootReqId string) {
	reqId := GetReqIdFromContext(ctx)
	if reqId > 0 {
		StoreRemoteReqId(reqId, rootReqId)
	}
}

func StoreRemoteReqId(reqId int64, rootReqId string) {
	RemoteReqIdMap.Store(reqId, rootReqId)
}
