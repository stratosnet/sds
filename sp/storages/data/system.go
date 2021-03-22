package data

// System
type System struct {
	Version                 uint16
	ForceUpdate             bool
	Connected               uint64
	User                    uint64
	OnlinePPCount           uint64
	MissingBackupWalletAddr []string
	UpgradeReward           uint64
	InviteReward            uint64
	InitializeCapacity      uint64
}

// GetCacheKey
func (s *System) GetCacheKey() string {
	return "system"
}
