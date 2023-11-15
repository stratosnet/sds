package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/stratosnet/sds/relay/cmd/relayd/setting"
)

const (
	Home              string = "home"
	SpHome            string = "sp-home"
	Config            string = "config"
	DefaultConfigPath string = "./config/config.toml"
)

func genConfig(cmd *cobra.Command, args []string) error {

	path, err := cmd.Flags().GetString(Config)
	if err != nil {
		return errors.Wrap(err, "failed to get the configuration file path")
	}
	if path == DefaultConfigPath {
		home, err := cmd.Flags().GetString(Home)
		if err != nil {
			return err
		}
		path = filepath.Join(home, path)
	}

	if _, err = os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(path), 0700)
	}
	if err != nil {
		return err
	}

	err = setting.LoadConfig(path)
	if err != nil {
		fmt.Println("generating default config file")
		err = setting.GenDefaultConfig(path)
		if err != nil {
			return errors.Wrap(err, "failed to generate config file at given path")
		}

	}

	return setting.LoadConfig(path)
}
