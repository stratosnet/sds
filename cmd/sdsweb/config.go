package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
)

const webConfigName = "node_default_config.json"

type Config struct {
	WsUrl        string `json:"wsUrl"`
	MonitorToken string `json:"monitorToken"`
}

func UpdateWebConfig() error {
	config := &Config{
		WsUrl: fmt.Sprintf("ws://127.0.0.1:%v", setting.Config.Monitor.Port),
	}
	if setting.Config.WebServer.TokenOnStartup {
		config.MonitorToken = serv.GetCurrentToken()
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	configPath := filepath.Join(setting.Config.WebServer.Path, webConfigName)
	return os.WriteFile(configPath, configBytes, 0644)
}
