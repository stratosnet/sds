package net

import (
	"github.com/stratosnet/sds/utils/cache"
	"github.com/stratosnet/sds/utils/database/config"
)

// Config server config struct
type Config struct {
	Version     uint16            `yaml:"Version"`
	Net         NetworkConfig     `yaml:"Net"`
	Peers       PeersConfig       `yaml:"Peers"`
	HashRing    HashRingConfig    `yaml:"HashRing"`
	FileStorage FileStorageConfig `yaml:"FileStorage"`
	Cache       cache.Config      `yaml:"Cache"`
	Database    config.Connect    `yaml:"Database"`
	BpList      BpListConfig      `yaml:"BpList"`
	Ecdsa       EcdsaConfig       `yaml:"Ecdsa"`
	User        UserConfig        `yaml:"User"`
	Token       string            `yaml:"Token"`
	Log         Log               `yaml:"Log"`
}

type NetworkConfig struct {
	Host          string `yaml:"Host"`
	Port          string `yaml:"Port"`
	WebsocketPort string `yaml:"WebsocketPort"`
}

type PeersConfig struct {
	List             int     `yaml:"List"`
	RegisterSwitch   bool    `yaml:"RegisterSwitch"`
	ProvideDiskScale float32 `yaml:"ProvideDiskScale"`
}

type HashRingConfig struct {
	VirtualNodeNum uint32 `yaml:"VirtualNodeNum"`
}

type FileStorageConfig struct {
	SliceBlockSize    uint64 `yaml:"SliceBlockSize"`
	PictureLibAddress string `yaml:"PictureLibAddress"`
}

type BpConfig struct {
	NetworkAddress string `yaml:"NetworkAddress"`
	WalletAddress  string `yaml:"WalletAddress"`
}

type BpListConfig []BpConfig

type EcdsaConfig struct {
	PrivateKeyPath string `yaml:"PrivateKeyPath"`
	PrivateKeyPass string `yaml:"PrivateKeyPass"`
}

type UserConfig struct {
	UpgradeReward      uint64 `yaml:"UpgradeReward"`
	InviteReward       uint64 `yaml:"InviteReward"`
	InitializeCapacity uint64 `yaml:"InitializeCapacity"`
}

/*
Log level:
const (
	Detail LogLevel = 1
	Debug  LogLevel = 10
	Info   LogLevel = 20
	Warn   LogLevel = 30
	Error  LogLevel = 40
	Fatal  LogLevel = 50
)
*/
type Log struct {
	Path       string `yaml:"Path"`
	Level      int    `yaml:"Level"`
	OutputFile bool   `yaml:"OutputFile"`
	OutputStd  bool   `yaml:"OutputStd"`
}
