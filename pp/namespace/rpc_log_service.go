package namespace

import (
	"context"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

func RpcLogService() *rpcLogService {
	return &rpcLogService{}
}

type rpcLogService struct{}

//func (s *rpcLogService) CleanUp() {
//	utils.ClearRpcLogger()
//}

func (s *rpcLogService) LogSubscription(ctx context.Context, terminalId string) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	subscription := notifier.CreateSubscription()

	go func() {
		for {
			select {
			case reqIdLogMap := <-utils.RpcReqIdLogCh:
				for reqId, log := range reqIdLogMap {
					if tid, ok := utils.ReqTerminalIdMap.Load(reqId); ok {
						if tid == terminalId {
							notifier.Notify(subscription.ID, &log)
							break
						}
					}
				}
			case <-subscription.Err(): // client send an unsubscribe request
				// ReqTerminalIdMap & RpcLoggerMap are AutoCleanMap, don't support iteration and doesn't need to clean up manually
				return
			case <-notifier.Closed(): // connection dropped
				// ReqTerminalIdMap & RpcLoggerMap are AutoCleanMap, don't support iteration and doesn't need to clean up manually
				return
			}
		}
	}()

	return subscription, nil
}
