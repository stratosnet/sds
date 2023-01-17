package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/cmd/relayd/setting"
	"github.com/stratosnet/sds/utils"
)

const (
	HOME                string = "home"
	SP_HOME             string = "sp-home"
	CONFIG              string = "config"
	DEFAULT_CONFIG_PATH string = "./config/config.toml"
)

func main() {
	rootCmd := getRootCmd()
	startCmd := getStartCmd()

	rootCmd.AddCommand(startCmd)

	err := rootCmd.Execute()
	if err != nil {
		utils.ErrorLog(err)
	}
}

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "relayd",
		Short:             "relayd",
		PersistentPreRunE: rootPreRunE,
	}

	dir, err := os.Getwd()
	if err != nil {
		utils.ErrorLog("failed to get working directory")
		panic(err)
	}
	rootCmd.PersistentFlags().StringP(HOME, "r", dir, "home path for the relayd process")
	rootCmd.PersistentFlags().StringP(CONFIG, "c", DEFAULT_CONFIG_PATH, "configuration file path ")
	return rootCmd
}

func getStartCmd() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "start relayd",
		RunE:  startRunE,
	}

	dir, err := os.Getwd()
	if err != nil {
		utils.ErrorLog("failed to get working directory")
		panic(err)
	}
	startCmd.Flags().StringP(SP_HOME, "s", dir, "home path for the associated SP node")
	return startCmd
}

func rootPreRunE(cmd *cobra.Command, _ []string) error {
	homePath, err := cmd.Flags().GetString(HOME)
	if err != nil {
		utils.ErrorLog("failed to get 'home' path for the relayd process")
		return err
	}
	homePath, err = utils.Absolute(homePath)
	if err != nil {
		utils.ErrorLog("cannot convert home path to absolute path")
		return err
	}
	setting.HomePath = homePath
	_ = utils.NewDefaultLogger(filepath.Join(homePath, "tmp/logs/stdout.log"), true, true)

	configPath, err := cmd.Flags().GetString(CONFIG)
	if err != nil {
		utils.ErrorLog("failed to get 'config' path for the relayd process")
		return err
	}
	configPath = filepath.Join(homePath, configPath)

	err = setting.LoadConfig(configPath)
	if err != nil {
		utils.ErrorLog("Error loading the setting file", err)
		return err
	}
	return nil
}
