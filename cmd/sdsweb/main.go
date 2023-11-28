package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/stratosnet/sds/framework/utils"

	"github.com/stratosnet/sds/cmd/common"
)

const (
	RunPpd = "run-ppd"
)

func main() {
	rootCmd := getRootCmd()
	startCmd := getStartCmd()
	rootCmd.AddCommand(startCmd)

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "sdsweb",
		Short:             "resource node web interface",
		PersistentPreRunE: common.RootPreRunE,
	}

	dir, err := os.Getwd()
	if err != nil {
		utils.ErrorLog("failed to get working directory")
		panic(err)
	}

	rootCmd.PersistentFlags().StringP(common.Home, "r", dir, "path for the node")
	rootCmd.PersistentFlags().StringP(common.Config, "c", common.DefaultConfigPath, "configuration file path ")
	return rootCmd
}

func getStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start",
		Short:   "start the sdsweb UI along with the node",
		PreRunE: common.NodePreRunE,
		RunE:    startRunE,
	}

	cmd.Flags().Bool(RunPpd, true, "run ppd as well")
	return cmd
}

func startRunE(cmd *cobra.Command, args []string) error {
	err := startWebServer()
	if err != nil {
		return errors.New(utils.FormatError(err))
	}

	runPpd, err := cmd.Flags().GetBool(RunPpd)
	if err != nil {
		return err
	}

	if runPpd {
		return common.NodePP(cmd, args)
	}

	// Run the UI server only
	quit := common.GetQuitChannel()
	sig := <-quit
	utils.Logf("Quit signal detected: [%s]. Shutting down...", sig.String())
	return nil
}
