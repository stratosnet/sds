package grpc

import (
	"crypto/tls"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	SERVER   string
	INSECURE bool
)

func CreateGrpcConn() (*grpc.ClientConn, error) {
	if SERVER == "" {
		return nil, errors.New("the stratos-chain GRPC server URL is not set")
	}
	dialOptions, err := getDialOptions()
	if err != nil {
		return nil, err
	}
	return grpc.Dial(SERVER, dialOptions...)
}

func getDialOptions() (options []grpc.DialOption, err error) {
	options = make([]grpc.DialOption, 0)

	var tpCredentials credentials.TransportCredentials

	if INSECURE {
		tpCredentials = insecure.NewCredentials()
	} else {
		tpCredentials = credentials.NewTLS(&tls.Config{})
	}

	securityOpt := grpc.WithTransportCredentials(tpCredentials)
	options = append(options, securityOpt)

	return
}
