package table

import (
	"github.com/stratosnet/sds/utils/database"
)

const (
	TRANSFER_RECORD_STATUS_SUCCESS = 0

	TRANSFER_RECORD_STATUS_CHECK = 1

	TRANSFER_RECORD_STATUS_CONFIRM = 2

	TRANSFER_RECORD_STATUS_EXCEPTION = 3
)

// TransferRecord
type TransferRecord struct {
	Id                 uint32
	SliceHash          string
	SliceSize          uint64
	TransferCer        string
	FromP2pAddress     string
	FromWalletAddress  string
	FromNetworkAddress string
	ToP2pAddress       string
	ToWalletAddress    string
	ToNetworkAddress   string
	Status             byte
	Time               int64
}

// TableName
func (t *TransferRecord) TableName() string {
	return "transfer_record"
}

// PrimaryKey
func (t *TransferRecord) PrimaryKey() []string {
	return []string{"id"}
}

// SetData
func (t *TransferRecord) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(t, data)
}

// GetCacheKey
func (t *TransferRecord) GetCacheKey() string {
	return "transfer_record#" + t.TransferCer
}

// Event
func (t *TransferRecord) Event(event int, dt *database.DataTable) {}
