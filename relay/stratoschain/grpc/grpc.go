package grpc

import (
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	URL string

	insecureOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
	options     = []grpc.DialOption{
		insecureOpt,
	}
)

func CreateGrpcConn() (*grpc.ClientConn, error) {
	if URL == "" {
		return nil, errors.New("the stratos-chain GRPC server URL is not set")
	}
	return grpc.Dial(URL, options...)
}
