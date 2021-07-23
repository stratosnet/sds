package table

import (
	"github.com/stratosnet/sds/utils/database"
	"time"
)

// UserOzone
type UserOzone struct {
	WalletAddress string
	AvailableUoz  uint64
}

// TableName
func (uo *UserOzone) TableName() string {
	return "user_ozone"
}

// PrimaryKey
func (uo *UserOzone) PrimaryKey() []string {
	return []string{"wallet_address"}
}

// SetData
func (uo *UserOzone) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(uo, data)
}

// GetCacheKey get the key used for in memory cache
func (uo *UserOzone) GetCacheKey() string {
	return "user_ozone#" + uo.WalletAddress
}

// GetTimeOut get the timeout, this table has no timeout
func (uo *UserOzone) GetTimeOut() time.Duration {
	return 0
}

// Where get the where command for this table
func (uo *UserOzone) Where() map[string]interface{} {
	return map[string]interface{}{
		"where": map[string]interface{}{
			"wallet_address = ?": uo.WalletAddress,
		},
	}
}

// Event
func (uo *UserOzone) Event(event int, dt *database.DataTable) {}
