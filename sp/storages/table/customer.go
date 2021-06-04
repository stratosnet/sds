package table

import (
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils/database"
	"path/filepath"
	"strings"
	"time"
)

// Customer is the person who used the system for file storage
type Customer struct {
	Id            uint64
	WalletAddress string
	NetworkAddress string
	TotalVolume   uint64
	UsedVolume    uint64
	Puk           string
	LastLoginTime int64
	LoginTimes    uint64
	RegisterTime  int64
}

// TableName get the name of mysql table
func (c *Customer) TableName() string {
	return "customer"
}

// PrimaryKey get the primary key column of this table
func (c *Customer) PrimaryKey() []string {
	return []string{"id"}
}

// SetData save to db
func (c *Customer) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(c, data)
}

// GetCacheKey get the key used for in memory cache
func (c *Customer) GetCacheKey() string {
	return "customer#" + c.WalletAddress
}

// GetTimeOut get the timeout, this table has no timeout
func (c *Customer) GetTimeOut() time.Duration {
	return 0
}

// Where get the where command for this table
func (c *Customer) Where() map[string]interface{} {
	return map[string]interface{}{
		"where": map[string]interface{}{
			"wallet_address = ?": c.WalletAddress,
		},
	}
}

// Event n/a
func (c *Customer) Event(_ int, _ *database.DataTable) {}

// GetTotalVolume get the total volume purchased by this customer
func (c *Customer) GetTotalVolume() uint64 {
	return c.TotalVolume / 1048576
}

// GetAvailableVolume get the available volume of this customer
func (c *Customer) GetAvailableVolume() uint64 {
	var freeVolume uint64 = 0
	if c.TotalVolume > c.UsedVolume {
		freeVolume = (c.TotalVolume - c.UsedVolume) / 1048576
	}
	return freeVolume
}

// GetShareFiles TODO needed?
func (c *Customer) GetShareFiles(ct *database.CacheTable) []*protos.FileInfo {

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
		"where": map[string]interface{}{"us.wallet_address = ?": []interface{}{SHARE_TYPE_FILE, OPEN_TYPE_PUBLIC, c.WalletAddress}},
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
			OwnerWalletAddress: c.WalletAddress,
			ShareLink:          new(UserShare).GenerateShareLink(shareFile.ShareId, shareFile.RandCode),
			CreateTime:         uint64(shareFile.Time),
		}
	}

	return fileInfos
}

// GetShareDirs TODO needed?
func (c *Customer) GetShareDirs(ct *database.CacheTable) []*protos.FileInfo {

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
			"ud.wallet_address = ?": []interface{}{SHARE_TYPE_DIR, c.WalletAddress},
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
			OwnerWalletAddress: c.WalletAddress,
			ShareLink:          new(UserShare).GenerateShareLink(shareDir.ShareId, shareDir.RandCode),
			CreateTime:         uint64(shareDir.Time),
		}
	}
	return fileInfos
}
