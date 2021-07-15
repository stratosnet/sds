package table

import (
	"github.com/stratosnet/sds/utils/database"
)

// FileSliceStorage map for file slice and storage pp
type FileSliceStorage struct {
	SliceHash      string
	P2pAddress     string
	NetworkAddress string
}

// TableName
func (fss *FileSliceStorage) TableName() string {
	return "file_slice_storage"
}

// PrimaryKey
func (fss *FileSliceStorage) PrimaryKey() []string {
	return []string{"slice_hash", "p2p_address"}
}

// SetData
func (fss *FileSliceStorage) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(fss, data)
}

// Event
func (fss *FileSliceStorage) Event(event int, dt *database.DataTable) {}
