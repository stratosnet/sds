package table

import "github.com/stratosnet/sds/utils/database"

type UserInviteRecord struct {
	Id             uint32
	WalletAddress  string // invitation recipient
	InvitationCode string //
	Reward         uint64
	Time           int64 // accept time
}

// TableName
func (uir *UserInviteRecord) TableName() string {
	return "user_invite_record"
}

// PrimaryKey
func (uir *UserInviteRecord) PrimaryKey() []string {
	return []string{"id"}
}

// SetData
func (uir *UserInviteRecord) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(uir, data)
}

// Event
func (uir *UserInviteRecord) Event(event int, dt *database.DataTable) {}
