package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

func main() {

	rootCmd := getRootCmd()
	nodeCmd := getNodeCmd()
	terminalCmd := getTerminalCmd()
	configCmd := getGenConfigCmd()
	verCmd := getVersionCmd()

	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(terminalCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(verCmd)

	err := rootCmd.Execute()
	if err != nil {
		utils.ErrorLog(err)
	}
	return
}

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ppd",
		Short: "meta(indexing) node",
	}

	dir, err := os.Getwd()
	if err != nil {
		utils.ErrorLog("failed to get working directory")
		panic(err)
	}

	rootCmd.PersistentFlags().StringP(HOME, "r", dir, "path for the node")
	rootCmd.PersistentFlags().StringP(CONFIG, "c", defaultConfigPath, "configuration file path ")
	return rootCmd
}

func getNodeCmd() *cobra.Command {
	nodeCmd := &cobra.Command{
		Use:     "start",
		Short:   "start the node",
		PreRunE: nodePreRunE,
		RunE:    nodePP,
	}
	return nodeCmd
}

func getTerminalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "terminal",
		Short:   "open terminal attached to node demon",
		PreRunE: terminalPreRunE,
		Run:     terminal,
	}

	return cmd
}

func getGenConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "create default configuration file",
		RunE:  genConfig,
	}
	return cmd
}



func getVersionCmd() *cobra.Command{

	version := setting.Version
	cmd := &cobra.Command{
		Use:    "version",
		Short:  "get version of the build",
		Run: 	func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
	return cmd
}