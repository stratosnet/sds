package table

import (
	"github.com/stratosnet/sds/utils/database"
)

const (
	// DOWNLOAD_STATUS_SUCCESS
	DOWNLOAD_STATUS_SUCCESS = 0
	// DOWNLOAD_STATUS_CHECK
	DOWNLOAD_STATUS_CHECK = 1
)

// FileSliceDownload
type FileSliceDownload struct {
	Id                uint32
	SliceHash         string
	FromWalletAddress string
	ToWalletAddress   string
	TaskId            string
	Status            byte
	Time              int64
}

// TableName
func (fsd *FileSliceDownload) TableName() string {
	return "file_slice_download"
}

// PrimaryKey
func (fsd *FileSliceDownload) PrimaryKey() []string {
	return []string{"id"}
}

// GetCacheKey
func (fsd *FileSliceDownload) GetCacheKey() string {
	return "file_slice_download#" + fsd.TaskId
}

// SetData
func (fsd *FileSliceDownload) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(fsd, data)
}

// Event
func (fsd *FileSliceDownload) Event(event int, dt *database.DataTable) {}
