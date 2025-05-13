package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
	cid2 "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/crypto"
)

const (
	DATA_MESH_PROTOCOL = "sdm://"
	FileHandleLength   = 88

	SHARED_DATA_MESH_PROTOCOL = "sds://"

	// xxxxxxxxxxxxxxxx_xxxxxxxxx_xxxxxx
	NormalShareLinkV2Length = 16 + 1*(1+10) + 1 + 6

	// xxxxxx_xxxxxxxxxxxxxxxx or xxxxxxxxxxxxxxxx_xxxxxx
	NormalShareLinkV1Length = 16 + 0*(1*10) + 1 + 6

	MAX_META_INFO_LENGTH = 40
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

func GenerateNormalShareLinkV2() string {
	// uuid is a 32 byte []byte. Here it is restructured to xxxxxxxxxxxxxxxx_xxxxxxxxx_xxxxxx.
	uuid := uuid.New()
	dst := make([]byte, 32)
	hex.Encode(dst, []byte(uuid[:]))
	return fmt.Sprintf("%s_%s_%s", dst[:16], dst[16:26], dst[26:32])
}
func SetShareLink(shareId, randCode string) *ShareDataMeshId {
	if randCode == "" {
		return &ShareDataMeshId{Link: shareId}
	}
	return &ShareDataMeshId{Link: fmt.Sprintf("%s_%s", randCode, shareId)}
}

func CheckIpfsCid(cid string) error {
	_, err := cid2.Decode(cid)
	return err
}

func DecodeCid(cid string) (cid2.Cid, error) {
	return cid2.Decode(cid)
}

func ParseShareLink(getShareString string) (*ShareDataMeshId, error) {
	if getShareString[:len(SHARED_DATA_MESH_PROTOCOL)] != SHARED_DATA_MESH_PROTOCOL {
		return nil, errors.New("invalid get shared file link prefix, expected " + SHARED_DATA_MESH_PROTOCOL)
	}

	parts := strings.Split(getShareString[len(SHARED_DATA_MESH_PROTOCOL):], "/")

	if len(parts) == 0 {
		return nil, errors.New("wrong share link: empty")
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
	// Ipfs sharelink
	_, err := cid2.Decode(s.Link)
	if err == nil {
		return s.Link, ""
	}

	if len(s.Link) == NormalShareLinkV2Length {
		return s.Link, ""
	}

	if len(s.Link) == NormalShareLinkV1Length {
		args := strings.Split(s.Link, "_")
		if len(args) >= 2 {
			return args[1], args[0]
		}
	}

	return "", ""
}

func (s ShareDataMeshId) String() string {
	return fmt.Sprintf("%s%s", SHARED_DATA_MESH_PROTOCOL, s.Link)
}
