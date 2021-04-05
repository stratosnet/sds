package table

import (
	"github.com/qsnetwork/sds/utils/database"
	"time"
)

const (
	// STATE_DELETE
	STATE_DELETE = 0
	// STATE_OK
	STATE_OK = 1

	// IS_COVER
	IS_COVER = 1
)

// File
type File struct {
	Id       uint32
	Name     string
	Hash     string
	Size     uint64
	SliceNum uint64
	Download uint32
	State    byte
	Time     int64
	IsCover  byte
	UserHasFile
}

// TableName
func (f *File) TableName() string {
	return "file"
}

// PrimaryKey
func (f *File) PrimaryKey() []string {
	return []string{"id"}
}

// SetData
func (f *File) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(f, data)
}

// GetCacheKey
func (f *File) GetCacheKey() string {
	if f.WalletAddress != "" {
		return "file#" + f.Hash + "-" + f.WalletAddress
	}
	return "file#" + f.Hash
}

// Where
func (f *File) Where() map[string]interface{} {
	params := map[string]interface{}{
		"where": map[string]interface{}{
			"hash = ?": f.Hash,
		},
	}
	if f.WalletAddress != "" {
		params = map[string]interface{}{
			"alias":   "e",
			"columns": "e.*, uhf.wallet_address",
			"join":    []string{"user_has_file", "e.hash = uhf.file_hash", "uhf"},
			"where":   map[string]interface{}{"e.hash = ? AND uhf.wallet_address = ?": []interface{}{f.Hash, f.WalletAddress}},
		}
	}
	return params
}

// GetTimeOut
func (f *File) GetTimeOut() time.Duration {
	return time.Second * 60 * 60
}

// Event
func (f *File) Event(event int, dt *database.DataTable) {
	switch event {
	case database.AFTER_INSERT:
		if f.WalletAddress != "" {
			dt.StoreTable(&UserHasFile{WalletAddress: f.WalletAddress, FileHash: f.Hash})
		}
	case database.BEFORE_DELETE:
		if f.WalletAddress != "" {
			dt.DeleteTable(&UserHasFile{WalletAddress: f.WalletAddress, FileHash: f.Hash})
		}
	}
}
