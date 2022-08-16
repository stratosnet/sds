package pp

import (
	"context"
	"fmt"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/utils"
)

func logDepthWithContext(context context.Context, level utils.LogLevel, calldepth int, v ...interface{}) {
	utils.MyLogger.LogDepth(level, 3, v...)

	if context == nil {
		return
	}

	loggerKey := getLoggerMapKeyFromContext(context)

	if loggerKey == 0 {
		return
	}

	if value, ok := utils.RpcLoggerMap.Load(loggerKey); ok {
		rpcLogger := value.(*utils.Logger)
		rpcLogger.LogDepth(level, calldepth, v...)
	}
}

func getLoggerMapKeyFromContext(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}

	if parentReqId := core.GetParentReqIdFromContext(ctx); parentReqId > 0 {
		return parentReqId
	}

	if reqId := core.GetReqIdFromContext(ctx); reqId > 0 {
		return reqId
	}

	return 0
}

func CreateReqIdAndRegisterRpcLogger(ctx context.Context) context.Context {
	reqId, _ := utils.NextSnowFakeId()
	utils.RpcLoggerMap.Store(reqId, utils.RpcLogger)
	return core.CreateContextWithReqId(ctx, reqId)
}

func Log(ctx context.Context, v ...interface{}) {
	logDepthWithContext(ctx, utils.Info, 3, v...)
}

func Logf(ctx context.Context, template string, v ...interface{}) {
	logDepthWithContext(ctx, utils.Info, 3, fmt.Sprintf(template, v...))
}

func ErrorLog(ctx context.Context, v ...interface{}) {
	logDepthWithContext(ctx, utils.Error, 3, v...)
}

func ErrorLogf(ctx context.Context, template string, v ...interface{}) {
	logDepthWithContext(ctx, utils.Error, 3, fmt.Errorf(template, v...))
}

func DebugLog(ctx context.Context, v ...interface{}) {
	logDepthWithContext(ctx, utils.Debug, 3, v...)
}

func DebugLogf(ctx context.Context, template string, v ...interface{}) {
	logDepthWithContext(ctx, utils.Debug, 3, fmt.Sprintf(template, v...))
}
