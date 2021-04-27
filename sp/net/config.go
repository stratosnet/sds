package net

import (
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/database/config"
)

// Config server config struct
type Config struct {
	Version uint16 `yaml:"Version"`
	Net     struct {
		Host          string `yaml:"Host"`
		Port          string `yaml:"Port"`
		WebsocketPort string `yaml:"WebsocketPort"`
	} `yaml:"Net"`
	Peers struct {
		List             int     `yaml:"List"`
		RegisterSwitch   bool    `yaml:"RegisterSwitch"`
		ProvideDiskScale float32 `yaml:"ProvideDiskScale"`
	} `yaml:"Peers"`
	HashRing struct {
		VirtualNodeNum uint32 `yaml:"VirtualNodeNum"`
	} `yaml:"HashRing"`
	FileStorage struct {
		SliceBlockSize    uint64 `yaml:"SliceBlockSize"`
		PictureLibAddress string `yaml:"PictureLibAddress"`
	} `yaml:"FileStorage"`
	Cache    cache.Config   `yaml:"Cache"`
	Database config.Connect `yaml:"Database"`
	BpList   []struct {
		NetworkAddress string `yaml:"NetworkAddress"`
		WalletAddress  string `yaml:"WalletAddress"`
	} `yaml:"BpList"`
	Ecdsa struct {
		PrivateKeyPath string `yaml:"PrivateKeyPath"`
		PrivateKeyPass string `yaml:"PrivateKeyPass"`
	} `yaml:"Ecdsa"`
	User struct {
		UpgradeReward      uint64 `yaml:"UpgradeReward"`
		InviteReward       uint64 `yaml:"InviteReward"`
		InitializeCapacity uint64 `yaml:"InitializeCapacity"`
	} `yaml:"User"`
}
