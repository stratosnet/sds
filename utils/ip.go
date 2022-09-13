package utils

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
)

func SplitHostPort(addr string) (net.IP, string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, "", errors.Wrap(err, fmt.Sprintf("cannot split address [%v] into host and port", addr))
	}

	ip := net.ParseIP(host)
	if len(ip) == 0 {
		return nil, "", errors.Errorf("couldn't parse host [%v] into an IP address", host)
	}
	return ip, port, nil
}
