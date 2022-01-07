package serv

import (
	"context"

	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

func RpcLogService() *rpcLogService {
	return &rpcLogService{}
}

type rpcLogService struct{}

type rpcWriter struct {
	notifier     *rpc.Notifier
	subscription *rpc.Subscription
}

type LogMsg struct {
	Msg string `json:"msg"`
}

func (l rpcWriter) Write(p []byte) (n int, err error) {
	err = l.notifier.Notify(l.subscription.ID, LogMsg{Msg: string(p)})
	if err != nil {
		return 0, err
	} else {
		return len(p), nil
	}
}

func (s *rpcLogService) CleanUp() {
	utils.MyLogger.ClearRpcLogger()
}

func (s *rpcLogService) LogSubscription(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	subscription := notifier.CreateSubscription()
	utils.MyLogger.SetRpcLogger(rpcWriter{
		notifier:     notifier,
		subscription: subscription,
	})

	return subscription, nil
}
