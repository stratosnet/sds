package table

import (
	"errors"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils/database"
	"path/filepath"
	"strings"
	"time"
)

const (
	SHARE_TYPE_FILE = 0
	SHARE_TYPE_DIR  = 1

	OPEN_TYPE_PUBLIC  = 0
	OPEN_TYPE_PRIVATE = 1
)

// UserShare
type UserShare struct {
	ShareId       string
	RandCode      string
	OpenType      byte
	Deadline      int64
	ShareType     byte
	Password      string
	Hash          string
	WalletAddress string
	Time          int64
}

// TableName
func (us *UserShare) TableName() string {
	return "user_share"
}

// PrimaryKey
func (us *UserShare) PrimaryKey() []string {
	return []string{"share_id"}
}

// SetData
func (us *UserShare) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(us, data)
}

// GetCacheKey
func (us *UserShare) GetCacheKey() string {
	return "user_share#" + us.ShareId
}

// GetTimeOut
func (us *UserShare) GetTimeOut() time.Duration {
	return time.Second * time.Duration(us.Deadline-time.Now().Unix())
}

// Where
func (us *UserShare) Where() map[string]interface{} {
	return map[string]interface{}{"where": map[string]interface{}{"share_id = ? ": us.ShareId}}
}

// ParseShareLink
func (us *UserShare) ParseShareLink(link string) (randCode, shareId string) {
	if link == "" {
		return "", ""
	}
	args := strings.Split(link, "_")
	if len(args) >= 2 {
		randCode = args[0]
		shareId = args[1]
	}
	return
}

// Event
func (us *UserShare) Event(event int, dt *database.DataTable) {}

// GetShareContent
func (us *UserShare) GetShareContent(ct *database.CacheTable) (*protos.FileInfo, error) {

	if us.ShareId == "" {
		return nil, errors.New("share doesn't exist")
	}

	fileInfo := new(protos.FileInfo)
	fileInfo.CreateTime = uint64(us.Time)
	fileInfo.OwnerWalletAddress = us.WalletAddress
	if us.ShareType == SHARE_TYPE_FILE {
		if us.Hash == "" {
			return nil, errors.New("file hash is null")
		}

		file := new(File)
		file.Hash = us.Hash
		file.WalletAddress = us.WalletAddress
		if ct.Fetch(file) != nil {
			return nil, errors.New("shared file not exist or deleted")
		}

		directory := new(UserDirectory)
		err := ct.FetchTable(directory, map[string]interface{}{
			"alias": "ud",
			"join":  []string{"user_directory_map_file", "udmf.file_hash = ? AND ud.dir_hash = udmf.dir_hash", "udmf"},
			"where": map[string]interface{}{"": us.Hash},
		})

		sPath := ""
		if err == nil {
			sPath = directory.Path
		}

		fileInfo.FileHash = us.Hash
		fileInfo.FileName = file.Name
		fileInfo.FileSize = file.Size
		fileInfo.IsDirectory = false
		fileInfo.StoragePath = sPath

	} else {

		if us.Hash == "" {
			return nil, errors.New("dir hash is null")
		}

		directory := new(UserDirectory)
		directory.DirHash = us.Hash
		if ct.Fetch(directory) != nil {
			return nil, errors.New("shared directory not exist or deleted")
		}

		sPath := ""
		if strings.ContainsRune(directory.Path, '/') {
			sPath = filepath.Dir(directory.Path)
		}
		fileInfo.FileName = filepath.Base(directory.Path)
		fileInfo.FileHash = directory.DirHash
		fileInfo.FileSize = 0
		fileInfo.IsDirectory = true
		fileInfo.StoragePath = sPath
	}
	return fileInfo, nil
}

// GetShareAllContent
func (us *UserShare) GetShareAllContent(ct *database.CacheTable) ([]*protos.FileInfo, error) {

	if us.ShareId == "" {
		return nil, errors.New("share doesn't exist")
	}

	fileInfos := make([]*protos.FileInfo, 0)
	userDirectory := new(UserDirectory)
	if us.ShareType == SHARE_TYPE_FILE {
		if us.Hash == "" {
			return nil, errors.New("file hash is null")
		}
		files := userDirectory.FindFiles(ct, us.WalletAddress, "", "", us.Hash, "", protos.FileSortType_DEF, false)
		if len(files) > 0 {
			fileInfos = append(fileInfos, files...)
		}

	} else {
		if us.Hash == "" {
			return nil, errors.New("dir hash is null")
		}
		directory := new(UserDirectory)
		directory.DirHash = us.Hash
		if ct.Fetch(directory) != nil {
			return nil, errors.New("shared file not exist or deleted")
		}

		files := userDirectory.FindDirs(ct, us.WalletAddress, directory.Path)
		if len(files) > 0 {
			fileInfos = append(fileInfos, files...)
		}
	}
	return fileInfos, nil
}

// GenerateShareLink
func (us *UserShare) GenerateShareLink(shareId, randCode string) string {
	return strings.Join([]string{randCode, shareId}, "_")
}
