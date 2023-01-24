package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/cmd/relayd/setting"
)

func genConfig(cmd *cobra.Command, args []string) error {

	path, err := cmd.Flags().GetString(CONFIG)
	if err != nil {
		return errors.Wrap(err, "failed to get the configuration file path")
	}
	if path == DEFAULT_CONFIG_PATH {
		home, err := cmd.Flags().GetString(HOME)
		if err != nil {
			return err
		}
		path = filepath.Join(home, path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(path), 0700)
	}
	err = setting.LoadConfig(path)
	if err != nil {
		fmt.Println("generating default config file")
		err = setting.GenDefaultConfig(path)
		if err != nil {
			return errors.Wrap(err, "failed to generate config file at given path")
		}

	}

	setting.LoadConfig(path)
	return nil
}
