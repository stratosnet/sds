package table

import (
	"github.com/qsnetwork/qsds/utils/database"
)

// FileDownload
type FileDownload struct {
	Id              uint32
	FileHash        string
	ToWalletAddress string
	TaskId          string
	Time            int64
}

// TableName
func (fd *FileDownload) TableName() string {
	return "file_download"
}

// PrimaryKey
func (fd *FileDownload) PrimaryKey() []string {
	return []string{"id"}
}

// SetData
func (fd *FileDownload) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(fd, data)
}

// GetCacheKey
func (fd *FileDownload) GetCacheKey() string {
	return "file_download#" + fd.FileHash
}

func (fd *FileDownload) Event(event int, dt *database.DataTable) {}
