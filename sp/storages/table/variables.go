package table

import (
	"github.com/stratosnet/sds/utils/database"
	"time"
)

// Variable
type Variable struct {
	Name  string
	Value string
}

// TableName
func (v *Variable) TableName() string {
	return "variables"
}

// PrimaryKey
func (v *Variable) PrimaryKey() []string {
	return []string{"name"}
}

// SetData
func (v *Variable) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(v, data)
}

// GetCacheKey
func (v *Variable) GetCacheKey() string {
	return "variables#" + v.Name
}

// Event
func (v *Variable) Event(_ int, _ *database.DataTable) {}

// GetTimeOut
func (v *Variable) GetTimeOut() time.Duration {
	return 0
}

// Where
func (v *Variable) Where() map[string]interface{} {
	return map[string]interface{}{
		"where": map[string]interface{}{
			"name = ?": v.Name,
		},
	}
}
