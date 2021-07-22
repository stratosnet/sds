package table

import (
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils/database"
	"path/filepath"
	"strings"
	"time"
)

// User
type User struct {
	Id             uint64
	IsPp           byte
	Belong         string
	P2pAddress     string
	WalletAddress  string
	NetworkAddress string
	FreeDisk       uint64
	DiskSize       uint64
	Name           string
	Puk            string
	LastLoginTime  int64
	LoginTimes     uint64
	RegisterTime   int64
	InvitationCode string
	Capacity       uint64
	UsedCapacity   uint64
	IsUpgrade      byte
	BeInvited      byte
}

// TableName
func (u *User) TableName() string {
	return "user"
}

// PrimaryKey
func (u *User) PrimaryKey() []string {
	return []string{"id"}
}

// SetData
func (u *User) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(u, data)
}

// GetCacheKey
func (u *User) GetCacheKey() string {
	return "user#" + u.P2pAddress
}

// GetTimeOut
func (u *User) GetTimeOut() time.Duration {
	return 0
}

// Where
func (u *User) Where() map[string]interface{} {
	return map[string]interface{}{
		"where": map[string]interface{}{
			"p2p_address = ?": u.P2pAddress,
		},
	}
}

// Event
func (u *User) Event(event int, dt *database.DataTable) {}

// GetCapacity
func (u *User) GetCapacity() uint64 {
	return u.Capacity / 1048576
}

// GetFreeCapacity
func (u *User) GetFreeCapacity() uint64 {
	var freeCapacity uint64 = 0
	if u.Capacity > u.UsedCapacity {
		freeCapacity = (u.Capacity - u.UsedCapacity) / 1048576
	}
	return freeCapacity
}

// GetShareFiles
func (u *User) GetShareFiles(ct *database.CacheTable) []*protos.FileInfo {

	type ShareFile struct {
		File
		Path     string
		ShareId  string
		RandCode string
		Time     int64
	}

	fileRes, err := ct.FetchTables([]ShareFile{}, map[string]interface{}{
		"alias":   "f",
		"columns": "f.*, ud.path, us.time, us.rand_code, us.share_id",
		"join": [][]string{
			{"user_share", "us.hash = f.hash AND us.share_type = ? AND us.open_type = ?", "us"},
			{"user_directory_map_file", "f.hash = udmf.file_hash", "udmf", "left"},
			{"user_directory", "udmf.dir_hash = ud.dir_hash AND ud.wallet_address = us.wallet_address", "ud", "left"},
		},
		"where": map[string]interface{}{"us.wallet_address = ?": []interface{}{SHARE_TYPE_FILE, OPEN_TYPE_PUBLIC, u.WalletAddress}},
	})

	if err != nil {
		return []*protos.FileInfo{}
	}
	files := fileRes.([]ShareFile)
	if len(files) <= 0 {
		return []*protos.FileInfo{}
	}

	fileInfos := make([]*protos.FileInfo, len(files))
	for idx, shareFile := range files {
		fileInfos[idx] = &protos.FileInfo{
			FileHash:           shareFile.Hash,
			FileName:           shareFile.Name,
			FileSize:           shareFile.Size,
			IsPrivate:          false,
			IsDirectory:        false,
			StoragePath:        shareFile.Path,
			OwnerWalletAddress: u.WalletAddress,
			ShareLink:          (&UserShare{}).GenerateShareLink(shareFile.ShareId, shareFile.RandCode),
			CreateTime:         uint64(shareFile.Time),
		}
	}

	return fileInfos
}

// GetShareDirs
func (u *User) GetShareDirs(ct *database.CacheTable) []*protos.FileInfo {

	type ShareDir struct {
		UserDirectory
		WalletAddress string
		ShareId       string
		RandCode      string
		Time          int64
	}

	dirRes, err := ct.FetchTables([]ShareDir{}, map[string]interface{}{
		"alias":   "ud",
		"columns": "ud.*, us.time, us.rand_code, us.share_id",
		"join":    []string{"user_share", "us.hash = ud.dir_hash AND us.share_type = ?", "us"},
		"where": map[string]interface{}{
			"ud.wallet_address = ?": []interface{}{SHARE_TYPE_DIR, u.WalletAddress},
		},
	})

	if err != nil {
		return []*protos.FileInfo{}
	}

	dirs := dirRes.([]ShareDir)
	if len(dirs) <= 0 {
		return []*protos.FileInfo{}
	}

	fileInfos := make([]*protos.FileInfo, len(dirs))
	for idx, shareDir := range dirs {
		sPath := ""
		if strings.ContainsRune(shareDir.Path, '/') {
			sPath = filepath.Dir(shareDir.Path)
		}
		fileInfos[idx] = &protos.FileInfo{
			FileHash:           shareDir.DirHash,
			FileName:           filepath.Base(shareDir.Path),
			FileSize:           0,
			IsPrivate:          false,
			IsDirectory:        true,
			StoragePath:        sPath,
			OwnerWalletAddress: u.WalletAddress,
			ShareLink:          new(UserShare).GenerateShareLink(shareDir.ShareId, shareDir.RandCode),
			CreateTime:         uint64(shareDir.Time),
		}
	}
	return fileInfos
}
