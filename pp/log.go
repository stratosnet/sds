package pp

import (
	"context"
	"fmt"

	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/utils"
)

func logDepthWithContext(context context.Context, level utils.LogLevel, calldepth int, v ...interface{}) {
	utils.MyLogger.LogDepth(level, calldepth+1, v...)

	if context == nil {
		return
	}

	loggerKey := getLoggerMapKeyFromContext(context)

	if loggerKey == 0 {
		return
	}

	if logger := utils.GetRpcLoggerByReqId(loggerKey); logger != nil {
		logger.LogDepth(level, calldepth, v...)
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

func CreateReqIdAndRegisterRpcLogger(ctx context.Context, terminalId string) context.Context {
	rpcLoggerReqId, _ := utils.NextSnowFlakeId()
	utils.RegisterReqToLogger(rpcLoggerReqId, terminalId)
	return core.CreateContextWithReqId(ctx, rpcLoggerReqId)
}

func Log(ctx context.Context, v ...interface{}) {
	logDepthWithContext(ctx, utils.Info, 4, v...)
}

func Logf(ctx context.Context, template string, v ...interface{}) {
	logDepthWithContext(ctx, utils.Info, 4, fmt.Sprintf(template, v...))
}

func ErrorLog(ctx context.Context, v ...interface{}) {
	logDepthWithContext(ctx, utils.Error, 4, v...)
}

func ErrorLogf(ctx context.Context, template string, v ...interface{}) {
	logDepthWithContext(ctx, utils.Error, 4, fmt.Errorf(template, v...))
}

func DebugLog(ctx context.Context, v ...interface{}) {
	logDepthWithContext(ctx, utils.Debug, 4, v...)
}

func DebugLogf(ctx context.Context, template string, v ...interface{}) {
	logDepthWithContext(ctx, utils.Debug, 4, fmt.Sprintf(template, v...))
}
