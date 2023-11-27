package namespace

import (
	"context"

	"github.com/stratosnet/framework/utils"
	"github.com/stratosnet/sds/rpc"
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
	writer := newRpcWriter(terminalId)
	logCh := writer.getLogCh()
	utils.AddRpcLogger(writer, terminalId)
	go func() {
		for {
			select {
			case log := <-logCh:
				err := notifier.Notify(subscription.ID, log)
				if err != nil {
					break
				}

			case <-subscription.Err(): // client send an unsubscribe request
				utils.DisableRpcLogger(terminalId)
				return
			}
		}
	}()

	return subscription, nil
}

func newRpcWriter(id string) *rpcWriter {
	return &rpcWriter{
		terminalId: id,
		logCh:      make(chan utils.LogMsg),
	}
}

type rpcWriter struct {
	terminalId string
	logCh      chan utils.LogMsg
}

func (l *rpcWriter) getLogCh() chan utils.LogMsg {
	return l.logCh
}

func (l *rpcWriter) Write(p []byte) (n int, err error) {
	l.logCh <- utils.LogMsg{Msg: string(p)}
	return len(p), nil
}
