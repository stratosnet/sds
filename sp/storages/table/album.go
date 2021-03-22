package table

import (
	"errors"
	"path/filepath"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/tools"
	"github.com/qsnetwork/qsds/utils/database"
	"strings"
	"time"
)

const (
	STATE_EXP    = 0
	STATE_NORMAL = 1

	ALBUM_IS_PRIVATE = 0
	ALBUM_IS_PUBLIC  = 1
)

// Album
type Album struct {
	AlbumId       string
	Name          string
	Introduction  string
	Cover         string
	Type          byte
	WalletAddress string
	VisitCount    uint32
	Time          int64
	State         byte
	IsPrivate     byte
}

// TableName
func (a *Album) TableName() string {
	return "album"
}

// PrimaryKey
func (a *Album) PrimaryKey() []string {
	return []string{"album_id"}
}

// SetData
func (a *Album) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(a, data)
}

// GetCacheKey
func (a *Album) GetCacheKey() string {
	return "album#" + a.AlbumId
}

// Where
func (a *Album) Where() map[string]interface{} {
	return map[string]interface{}{
		"where": map[string]interface{}{
			"album_id = ?": a.AlbumId,
		},
	}
}

// GetTimeOut
func (a *Album) GetTimeOut() time.Duration {
	return time.Second * 3600
}

// Event
func (a *Album) Event(event int, dt *database.DataTable) {}

// AddFile
func (a *Album) AddFile(ct *database.CacheTable, file *protos.FileInfo) error {
	if file.FileHash == "" || a.AlbumId == "" {
		return errors.New("file hash or album ID can't be empty")
	}
	ahf := new(AlbumHasFile)
	ahf.FileHash = file.FileHash
	ahf.AlbumId = a.AlbumId
	ahf.Sort = file.SortId
	ahf.Time = time.Now().Unix()
	ct.StoreTable(ahf)
	return nil
}

// RemoveFile
func (a *Album) RemoveFile(ct *database.CacheTable, fileHash string) error {
	if fileHash == "" || a.AlbumId == "" {
		return errors.New("file hash or album ID can't be empty")
	}
	ahf := new(AlbumHasFile)
	ahf.FileHash = fileHash
	ahf.AlbumId = a.AlbumId
	ct.DeleteTable(ahf)
	return nil
}

// GetFiles
func (a *Album) GetFiles(ct *database.CacheTable) []*protos.FileInfo {
	if a.AlbumId != "" {

		type AlbumFile struct {
			File
			AddTime       int64
			Path          string
			WalletAddress string
		}

		res, err := ct.FetchTables([]AlbumFile{}, map[string]interface{}{
			"alias":   "e",
			"columns": "e.*, ahf.time as add_time, ud.path, a.wallet_address",
			"join": [][]string{
				{"album_has_file", "e.hash = ahf.file_hash", "ahf"},
				{"album", "ahf.album_id = a.album_id", "a"},
				{"user_directory_map_file", "e.hash = udmf.file_hash", "udmf", "left"},
				{"user_directory", "udmf.dir_hash = ud.dir_hash", "ud", "left"},
			},
			"where": map[string]interface{}{
				"ahf.album_id = ?": a.AlbumId,
			},
			"orderBy": "ahf.sort ASC",
		})

		if err == nil {
			files := res.([]AlbumFile)
			if len(files) > 0 {
				fileInfos := make([]*protos.FileInfo, len(files))
				for idx, file := range files {
					sPath := ""
					if file.Path != "" && strings.ContainsRune(file.Path, '/') {
						sPath = filepath.Dir(file.Path)
					}
					fileInfos[idx] = &protos.FileInfo{
						FileName:           file.Name,
						FileSize:           file.Size,
						FileHash:           file.Hash,
						CreateTime:         uint64(file.AddTime),
						IsDirectory:        false,
						StoragePath:        sPath,
						OwnerWalletAddress: file.WalletAddress,
					}
				}

				return fileInfos
			}
		}
	}

	return nil
}

// GetCoverLink
func (a *Album) GetCoverLink(walletAddress string) string {
	if a.Cover != "" {
		return tools.GenerateDownloadLink(walletAddress, a.Cover)
	}
	return ""
}
