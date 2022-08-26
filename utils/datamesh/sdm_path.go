package datamesh

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/types"
)

const DATA_MESH_PREFIX = "sdm://"

type DataMeshId struct {
	Owner string
	Hash  string
}

func (d DataMeshId) String() string {
	return fmt.Sprintf("%s%s/%s", DATA_MESH_PREFIX, d.Owner, d.Hash)
}

func DataMeshIdFromString(idString string) (*DataMeshId, error) {
	if idString[:len(DATA_MESH_PREFIX)] != DATA_MESH_PREFIX {
		return nil, errors.New("invalid data mesh id prefix, expected " + DATA_MESH_PREFIX)
	}
	if idString[47:48] != "/" {
		return nil, errors.New("invalid data mesh id")
	}

	parts := strings.Split(idString[len(DATA_MESH_PREFIX):], "/")
	if len(parts) != 2 {
		return nil, errors.New("invalid data mesh id, no owner or no hash")
	}
	_, err := types.WalletAddressFromBech(parts[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode owner")
	}
	ok := utils.VerifyHash(parts[1])
	if !ok {
		return nil, errors.New("failed to decode hash")
	}
	return &DataMeshId{
		Owner: parts[0],
		Hash:  parts[1],
	}, nil
}
