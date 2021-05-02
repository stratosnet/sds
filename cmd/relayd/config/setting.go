package setting

import "github.com/stratosnet/sds/utils"

type sds struct {
	ClientPort     string
	NetworkAddress string
	WebsocketPort  string
}

type stratoschain struct {
	ClientPort     string
	NetworkAddress string
	WebsocketPort  string
}

type config struct {
	SDS          sds
	StratosChain stratoschain
}

var Config *config

func LoadConfig(path string) {
	Config = new(config)
	utils.LoadYamlConfig(Config, path)
}
