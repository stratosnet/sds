package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	txclientutils "github.com/stratosnet/tx-client/utils"

	"github.com/stratosnet/framework/utils"

	"github.com/stratosnet/relay/cmd/relayd/setting"
)

func main() {
	rootCmd := getRootCmd()
	startCmd := getStartCmd()
	configCmd := getGenConfigCmd()
	syncCmd := getSyncCmd()

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(syncCmd)

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
	rootCmd.PersistentFlags().StringP(Home, "r", dir, "home path for the relayd process")
	rootCmd.PersistentFlags().StringP(Config, "c", DefaultConfigPath, "configuration file path ")
	return rootCmd
}

func getStartCmd() *cobra.Command {
	startCmd := &cobra.Command{
		Use:     "start",
		Short:   "start relayd",
		RunE:    startRunE,
		PreRunE: startPreRunE,
	}

	dir, err := os.Getwd()
	if err != nil {
		utils.ErrorLog("failed to get working directory")
		panic(err)
	}
	startCmd.Flags().StringP(SpHome, "s", dir, "home path for the associated SP node")
	return startCmd
}

func getGenConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "create default configuration file",
		RunE:  genConfig,
	}
	return cmd
}

func rootPreRunE(cmd *cobra.Command, _ []string) error {
	homePath, err := cmd.Flags().GetString(Home)
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
	_ = txclientutils.NewDefaultLogger(filepath.Join(homePath, "tmp/logs/tx-client-stdout.log"), true, true)
	return nil
}

func getSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sync",
		Short:   "sync stchain tx to sp",
		RunE:    sync,
		PreRunE: syncPreRunE,
	}
	dir, err := os.Getwd()
	if err != nil {
		utils.ErrorLog("failed to get working directory")
		panic(err)
	}

	cmd.PersistentFlags().StringP(Home, "r", dir, "home path for the relayd process")
	return cmd
}
