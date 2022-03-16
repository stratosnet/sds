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

type stratoschain struct {
	RestServer        string            `yaml:"restServer"`
	WebsocketServer   string            `yaml:"websocketServer"`
	ConnectionRetries connectionRetries `yaml:"connectionRetries"`
}

type config struct {
	BlockchainInfo blockchainInfoConfig `yaml:"blockchainInfo"`
	SDS            sds                  `yaml:"sds"`
	StratosChain   stratoschain         `yaml:"stratosChain"`
}

type blockchainInfoConfig struct {
	ChainId string `yaml:"chainId"`
	Token   string `yaml:"token"`
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
