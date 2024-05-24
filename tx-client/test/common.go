package test

import (
	"github.com/stratosnet/sds/tx-client/grpc"
)

const (
	grpcServerTest   = "127.0.0.1:9090"
	grpcInsecureTest = true
)

func initGrpcTestSettings() {
	grpc.SERVER = grpcServerTest
	grpc.INSECURE = grpcInsecureTest
}
