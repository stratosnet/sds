package table

import (
	"strconv"

	"github.com/stratosnet/sds/utils/database"
)

const (
	// TRAFFIC_TASK_TYPE_UPLOAD
	TRAFFIC_TASK_TYPE_UPLOAD = 0
	// TRAFFIC_TASK_TYPE_DOWNLOAD
	TRAFFIC_TASK_TYPE_DOWNLOAD = 1
	// TRAFFIC_TASK_TYPE_TRANSFER
	TRAFFIC_TASK_TYPE_TRANSFER = 2
)

// Traffic
type Traffic struct {
	Id                    uint32
	ProviderP2pAddress    string
	ProviderWalletAddress string
	ConsumerWalletAddress string
	TaskId                string
	TaskType              byte
	Volume                uint64
	DeliveryTime          int64
	ResponseTime          int64
}

// TableName
func (t *Traffic) TableName() string {
	return "traffic"
}

// PrimaryKey
func (t *Traffic) PrimaryKey() []string {
	return []string{"id"}
}

// SetData
func (t *Traffic) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(t, data)
}

// Event
func (t *Traffic) Event(event int, dt *database.DataTable) {}

// GetCacheKey
func (t *Traffic) GetCacheKey() string {
	return "traffic#" + t.TaskId
}

func (t *Traffic) GetHeaders() []string {
	return []string{
		"Id",
		"ProviderP2pAddress",
		"ProviderWalletAddress",
		"ConsumerWalletAddress",
		"TaskId",
		"TaskType",
		"Volume",
		"DeliveryTime",
		"ResponseTime",
	}
}

func (t *Traffic) ToSlice() []string {
	return []string{
		strconv.FormatUint(uint64(t.Id), 10),
		t.ProviderP2pAddress,
		t.ProviderWalletAddress,
		t.ConsumerWalletAddress,
		t.TaskId,
		strconv.FormatUint(t.Volume, 10),
		strconv.FormatInt(t.DeliveryTime, 10),
		strconv.FormatInt(t.ResponseTime, 10),
	}
}
