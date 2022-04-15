package setting

import (
	"github.com/stratosnet/sds/utils"
)

type connectionRetries struct {
	Max           int `yaml:"max"`
	SleepDuration int `yaml:"sleepDuration"`
}

type sds struct {
	ApiPort           string            `yaml:"apiPort"`
	ClientPort        string            `yaml:"clientPort"`
	NetworkAddress    string            `yaml:"networkAddress"`
	WebsocketPort     string            `yaml:"websocketPort"`
	ConnectionRetries connectionRetries `yaml:"connectionRetries"`
}

type broadcast struct {
	ChannelSize int `yaml:"channelSize"`
	MaxMsgPerTx int `yaml:"maxMsgPerTx"`
}

type stratoschain struct {
	RestServer        string            `yaml:"restServer"`
	WebsocketServer   string            `yaml:"websocketServer"`
	ConnectionRetries connectionRetries `yaml:"connectionRetries"`
	Broadcast         broadcast         `yaml:"broadcast"`
}

type transactionsConfig struct {
	Fee int64 `yaml:"fee"`
	Gas int64 `yaml:"gas"`
}

type blockchainInfoConfig struct {
	ChainId      string             `yaml:"chainId"`
	Token        string             `yaml:"token"`
	Transactions transactionsConfig `yaml:"transactions"`
}

type Version struct {
	AppVer    uint16 `yaml:"appVer"`
	MinAppVer uint16 `yaml:"minAppVer"`
}

type config struct {
	BlockchainInfo blockchainInfoConfig `yaml:"blockchainInfo"`
	SDS            sds                  `yaml:"sds"`
	StratosChain   stratoschain         `yaml:"stratosChain"`
	Version        Version              `yaml:"version"`
}

var Config *config

func LoadConfig(path string) error {
	Config = new(config)
	err := utils.LoadYamlConfig(Config, path)
	if err != nil {
		return err
	}

	return nil
}
