package types

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const GET_SHARE_FILE_PREFIX = "sds://"

const SHARE_LINK_LENGTH = 23

type GetShareFile struct {
	ShareLink string
	Password  string
}

func (s GetShareFile) String() string {
	return fmt.Sprintf("%s%s/%s", GET_SHARE_FILE_PREFIX, s.ShareLink, s.Password)
}

func ParseShareLink(getShareString string) (*GetShareFile, error) {
	if getShareString[:len(GET_SHARE_FILE_PREFIX)] != GET_SHARE_FILE_PREFIX {
		return nil, errors.New("invalid get shared file link prefix, expected " + GET_SHARE_FILE_PREFIX)
	}

	parts := strings.Split(getShareString[len(GET_SHARE_FILE_PREFIX):], "/")

	if len(parts) == 0 || len(parts[0]) != SHARE_LINK_LENGTH {
		return nil, errors.New("share link length is not correct")
	}

	if len(parts) == 1 {
		return &GetShareFile{
			ShareLink: parts[0],
			Password:  "",
		}, nil
	}

	return &GetShareFile{
		ShareLink: parts[0],
		Password:  parts[1],
	}, nil
}
