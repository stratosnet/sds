package table

import (
	"github.com/qsnetwork/sds/utils/database"
)

// AlbumHasFile
type AlbumHasFile struct {
	AlbumId  string
	FileHash string
	Time     int64
	Sort     uint64
}

// TableName
func (ahf *AlbumHasFile) TableName() string {
	return "album_has_file"
}

// PrimaryKey
func (ahf *AlbumHasFile) PrimaryKey() []string {
	return []string{"album_id", "file_hash"}
}

// SetData
func (ahf *AlbumHasFile) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(ahf, data)
}

// Event
func (ahf *AlbumHasFile) Event(event int, dt *database.DataTable) {}
