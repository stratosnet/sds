package test

import (
	"github.com/stratosnet/sds/tx-client/grpc"
	"github.com/stratosnet/sds/tx-client/utils"
)

const (
	grpcServerTest   = "127.0.0.1:9090"
	grpcInsecureTest = true
	logPath          = "./logs/relayer-tx-client-stdout.log"
)

func initGrpcTestSettings() {
	grpc.SERVER = grpcServerTest
	grpc.INSECURE = grpcInsecureTest
	_ = utils.NewDefaultLogger(logPath, true, true)
}
