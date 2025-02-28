package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/cmd/common"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/setting"
)

// time: Feb 3 2025 16:56 UTC
const DEFAULT_UNIX_TIME = 1738601804

func cleanStorage(cmd *cobra.Command, _ []string) error {
	var s string

	// double check whether the configuration of storage_path is from the configuration file or not. If not, return an error.
	_, configPath, err := common.GetPaths(cmd, false)
	if err != nil {
		return err
	}

	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("The config at location", configPath, "does not exist")
		return err
	}
	setting.Config.Home.StoragePath = ""
	err = utils.LoadTomlConfig(setting.Config, configPath)
	if err != nil {
		fmt.Println("Failed load configurations from the config file")
		return err
	}
	if setting.Config.Home.StoragePath == "" {
		fmt.Println("Storage_path is not configured in the configuration file")
		return errors.New("storage path is not configured")
	}

	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Please confirm to clean the old data in storage %s: [Y/n]", setting.Config.Home.StoragePath)
		s, _ = r.ReadString('\n')
		ss := strings.Split(s, "\n")
		if ss[0] == "N" || ss[0] == "n" {
			fmt.Println("Data in storage is NOT cleaned.")
			os.Exit(0)
		}
		if ss[0] == "Y" || ss[0] == "y" || ss[0] == "" {
			break
		}
	}

	cmdString := fmt.Sprintf(`find %s -type f -not -newermt "$(date -d @%d)" -exec sudo rm -rf {} \;`, setting.Config.Home.StoragePath, uint64(DEFAULT_UNIX_TIME))
	c := exec.Command("sh", "-c", cmdString)
	out, err := c.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	// Print output (if any)
	fmt.Println(string(out))
	return nil
}
