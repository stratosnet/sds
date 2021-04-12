package table

import "github.com/stratosnet/sds/utils/database"

// UserCollectAlbum
type UserCollectAlbum struct {
	WalletAddress string
	AlbumId       string
	Time          int64
}

// TableName
func (uca *UserCollectAlbum) TableName() string {
	return "user_collect_album"
}

// PrimaryKey
func (uca *UserCollectAlbum) PrimaryKey() []string {
	return []string{"wallet_address", "album_id"}
}

// SetData
func (uca *UserCollectAlbum) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(uca, data)
}

// Event
func (uca *UserCollectAlbum) Event(event int, dt *database.DataTable) {}
