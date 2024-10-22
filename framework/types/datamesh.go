package types

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/stratosnet/sds/framework/crypto"
)

const (
	DATA_MESH_PROTOCOL = "sdm://"
	FileHandleLength   = 88

	SHARED_DATA_MESH_PROTOCOL = "sds://"
	NormalShareLinkLength     = 23
	IpfsShareLinkLength       = 46
	IpfsCidPrefix             = "Qm"
)

type DataMeshId struct {
	Owner string
	Hash  string
}

func DataMeshIdFromString(idString string) (*DataMeshId, error) {
	if idString[:len(DATA_MESH_PROTOCOL)] != DATA_MESH_PROTOCOL {
		return nil, errors.New("invalid data mesh id prefix, expected " + DATA_MESH_PROTOCOL)
	}
	if idString[47:48] != "/" {
		return nil, errors.New("invalid data mesh id")
	}

	parts := strings.Split(idString[len(DATA_MESH_PROTOCOL):], "/")
	if len(parts) != 2 {
		return nil, errors.New("invalid data mesh id, no owner or no hash")
	}

	_, err := WalletAddressFromBech32(parts[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode owner")
	}
	ok := crypto.ValidateHash(parts[1])
	if !ok {
		return nil, errors.New("failed to decode hash")
	}
	return &DataMeshId{
		Owner: parts[0],
		Hash:  parts[1],
	}, nil
}

// ParseFileHandle
func ParseFileHandle(handle string) (protocol, walletAddress, fileHash, fileName string, err error) {
	handleInBytes := []byte(handle)

	if handle == "" || len(handle) < FileHandleLength {
		err = errors.New("handle is null or length is not correct")
		return
	}

	if string(handleInBytes[3:6]) != "://" || string(handleInBytes[47:48]) != "/" {
		err = errors.New("format error")
		return
	}

	protocol = string(handleInBytes[:3])
	walletAddress = string(handleInBytes[6:47])
	fileHash = string(handleInBytes[48:88])

	if len(handle) > FileHandleLength+1 {
		fileName = string(handleInBytes[89:])
	}

	if protocol != "sdm" ||
		walletAddress == "" || len(walletAddress) < 41 ||
		fileHash == "" || len(fileHash) < 40 {
		err = errors.New("file handle parse fail")
	}

	return
}

func (d DataMeshId) String() string {
	return fmt.Sprintf("%s%s/%s", DATA_MESH_PROTOCOL, d.Owner, d.Hash)
}

type ShareDataMeshId struct {
	Link     string
	Password string
}

func SetShareLink(shareId, randCode string) *ShareDataMeshId {
	if len(shareId) == IpfsShareLinkLength {
		return &ShareDataMeshId{Link: shareId}
	}
	return &ShareDataMeshId{Link: fmt.Sprintf("%s_%s", shareId, randCode)}
}

func CheckIpfsCid(cid string) bool {
	if len(cid) == IpfsShareLinkLength && strings.HasPrefix(cid, IpfsCidPrefix) {
		return true
	}
	return false
}

func ParseShareLink(getShareString string) (*ShareDataMeshId, error) {
	if getShareString[:len(SHARED_DATA_MESH_PROTOCOL)] != SHARED_DATA_MESH_PROTOCOL {
		return nil, errors.New("invalid get shared file link prefix, expected " + SHARED_DATA_MESH_PROTOCOL)
	}

	parts := strings.Split(getShareString[len(SHARED_DATA_MESH_PROTOCOL):], "/")

	if len(parts) == 0 || (len(parts[0]) != NormalShareLinkLength && len(parts[0]) != IpfsShareLinkLength) {
		return nil, errors.New("share link length is not correct")
	}

	if len(parts) == 1 {
		return &ShareDataMeshId{
			Link:     parts[0],
			Password: "",
		}, nil
	}

	return &ShareDataMeshId{
		Link:     parts[0],
		Password: parts[1],
	}, nil
}

func (s ShareDataMeshId) Parse() (shareId, randCode string) {
	if s.Link == "" {
		return "", ""
	}
	if len(s.Link) == IpfsShareLinkLength {
		return s.Link, ""
	}
	if len(s.Link) == NormalShareLinkLength {
		args := strings.Split(s.Link, "_")
		if len(args) >= 2 {
			return args[0], args[1]
		}
	}
	return "", ""
}

func (s ShareDataMeshId) String() string {
	return fmt.Sprintf("%s%s", SHARED_DATA_MESH_PROTOCOL, s.Link)
}
