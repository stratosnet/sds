package table

import (
	"github.com/qsnetwork/sds/utils/database"
)

// UserHasFile
type UserHasFile struct {
	WalletAddress string
	FileHash      string
}

// TableName
func (uhf *UserHasFile) TableName() string {
	return "user_has_file"
}

// PrimaryKey
func (uhf *UserHasFile) PrimaryKey() []string {
	return []string{"wallet_address", "file_hash"}
}

// SetData
func (uhf *UserHasFile) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(uhf, data)
}

// Event
func (uhf *UserHasFile) Event(event int, dt *database.DataTable) {}
