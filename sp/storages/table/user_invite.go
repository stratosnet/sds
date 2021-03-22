package table

import (
	"github.com/qsnetwork/qsds/utils/database"
	"time"
)

type UserInvite struct {
	InvitationCode string
	WalletAddress  string // invitor
	Times          byte   // invite times
}

// TableName
func (uir *UserInvite) TableName() string {
	return "user_invite"
}

// PrimaryKey
func (uir *UserInvite) PrimaryKey() []string {
	return []string{"invitation_code"}
}

// SetData
func (uir *UserInvite) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(uir, data)
}

// Event
func (uir *UserInvite) Event(event int, dt *database.DataTable) {}

// GetCacheKey
func (uir *UserInvite) GetCacheKey() string {
	return "user_invite#" + uir.InvitationCode
}

// GetTimeOut
func (uir *UserInvite) GetTimeOut() time.Duration {
	return 3600 * time.Second
}

// Where
func (uir *UserInvite) Where() map[string]interface{} {
	return map[string]interface{}{
		"where": map[string]interface{}{
			"invitation_code = ?": uir.InvitationCode,
		},
	}
}
