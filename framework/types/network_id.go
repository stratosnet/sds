package types

import (
	"errors"
	"fmt"
	"strings"
)

const NetworkIDPrefix = "snode:"

type NetworkID struct {
	P2pAddress     string
	NetworkAddress string
}

func (n NetworkID) String() string {
	return fmt.Sprintf("%s%s@%s", NetworkIDPrefix, n.P2pAddress, n.NetworkAddress)
}

func IDFromString(idString string) (*NetworkID, error) {
	if idString[:len(NetworkIDPrefix)] != NetworkIDPrefix {
		return nil, errors.New("invalid network ID prefix. Expected " + NetworkIDPrefix)
	}

	parts := strings.Split(idString[len(NetworkIDPrefix):], "@")
	if len(parts) < 2 {
		return nil, errors.New("invalid network ID. No P2P address or network address detected")
	}
	return &NetworkID{
		P2pAddress:     parts[0],
		NetworkAddress: parts[1],
	}, nil
}
