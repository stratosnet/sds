package main

import (
	"fmt"
	"os"
	"path/filepath"

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
		Use:               "ppd",
		Short:             "resource node",
		PersistentPreRunE: rootPreRunE,
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

	execCmd := &cobra.Command{
		Use:     "exec",
		Short:   "execute the command to node demon",
		PreRunE: terminalPreRunE,
		Run:     execute,
	}
	cmd.AddCommand(execCmd)
	return cmd
}

func getGenConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "create default configuration file",
		RunE:  genConfig,
	}
	cmd.AddCommand(getAccountCmd())
	cmd.Flags().BoolP(createP2pKeyFlag, "p", false, "create p2p key with config file, need interactive input")
	cmd.Flags().BoolP(createWalletFlag, "w", false, "create wallet with config file, need interactive input")
	return cmd
}

func getAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "accounts",
		Short:   "create accounts for the node",
		PreRunE: terminalPreRunE,
		RunE:    createAccounts,
	}
	cmd.Flags().StringP(mnemonicFlag, "m", "", "bip39 mnemonic phrase, will generate one if not provide")
	cmd.Flags().String(hdPathFlag, setting.HD_PATH, "hd-path for the wallet created")
	cmd.Flags().StringP(passwordFlag, "p", "", "wallet password, if not provided, will need to input in prompt")
	cmd.Flags().StringP(nicknameFlag, "n", "wallet", "name of wallet")
	cmd.Flags().BoolP(savePassFlag, "s", false, "save wallet password to configuration file")
	cmd.Flags().String(p2pPassFlag, "aaa", "p2p password, optional")
	cmd.Flags().Bool(newP2pKeyFlag, false, "create a new p2p key even there exist one already")
	return cmd
}

func getVersionCmd() *cobra.Command {

	version := setting.Version
	cmd := &cobra.Command{
		Use:   "version",
		Short: "get version of the build",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
	return cmd
}

func rootPreRunE(cmd *cobra.Command, args []string) error {
	homePath, err := cmd.Flags().GetString(HOME)
	if err != nil {
		utils.ErrorLog("failed to get 'home' path for the node")
		return err
	}
	homePath, err = utils.Absolute(homePath)
	if err != nil {
		return err
	}
	setting.SetupRoot(homePath)
	utils.NewDefaultLogger(filepath.Join(setting.GetRootPath(), "./tmp/logs/stdout.log"), true, true)
	return nil
}
