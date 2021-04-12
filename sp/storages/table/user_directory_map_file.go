package table

import (
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database"
	"time"
)

// UserDirectoryMapFile
type UserDirectoryMapFile struct {
	DirHash  string
	FileHash string
	Owner    string
	UserDirectory
}

// TableName
func (udmf *UserDirectoryMapFile) TableName() string {
	return "user_directory_map_file"
}

// PrimaryKey
func (udmf *UserDirectoryMapFile) PrimaryKey() []string {
	return []string{"dir_hash", "file_hash"}
}

// SetData
func (udmf *UserDirectoryMapFile) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(udmf, data)
}

// Event
func (udmf *UserDirectoryMapFile) Event(event int, dt *database.DataTable) {
	switch event {
	case database.BEFORE_INSERT:
		if udmf.WalletAddress != "" && udmf.Path != "" {
			dir := &UserDirectory{WalletAddress: udmf.WalletAddress, Path: udmf.Path}
			err := dt.FetchTable(dir, map[string]interface{}{
				"where": map[string]interface{}{"dir_hash = ?": dir.GenericHash()},
			})
			if err != nil {
				newDir := &UserDirectory{
					DirHash:       udmf.DirHash,
					Path:          udmf.Path,
					WalletAddress: udmf.WalletAddress,
					Time:          time.Now().Unix(),
				}
				if pathOk, err := newDir.OptPath(udmf.Path); err == nil {
					newDir.Path = pathOk
					newDir.DirHash = dir.GenericHash()
					dt.StoreTable(newDir)
				} else {
					utils.ErrorLog("UserDirectoryMapFile: event ", err.Error())
				}
			}
		}
	}
}
