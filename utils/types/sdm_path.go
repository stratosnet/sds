package types

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const DATA_MASH_PREFIX = "sdm://"

type DataMashId struct {
	Owner string
	Hash  string
}

func (d DataMashId) String() string {
	return fmt.Sprintf("%s%s/%s", DATA_MASH_PREFIX, d.Owner, d.Hash)
}

func DataMashIdFromString(idString string) (*DataMashId, error) {
	if idString[:len(DATA_MASH_PREFIX)] != DATA_MASH_PREFIX {
		return nil, errors.New("invalid network ID prefix. Expected " + DATA_MASH_PREFIX)
	}

	parts := strings.Split(idString[len(DATA_MASH_PREFIX):], "/")
	if len(parts) != 2 {
		return nil, errors.New("invalid network ID. No P2P address or network address detected")
	}
	return &DataMashId{
		Owner: parts[0],
		Hash:  parts[1],
	}, nil
}
