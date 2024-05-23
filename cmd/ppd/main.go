package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/cmd/common"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/setting"
)

func main() {

	rootCmd := getRootCmd()
	nodeCmd := getNodeCmd()
	terminalCmd := getTerminalCmd()
	configCmd := getGenConfigCmd()
	verCmd := getVersionCmd()
	exportCmd := getExportCmd()

	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(terminalCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(verCmd)
	rootCmd.AddCommand(exportCmd)

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "ppd",
		Short:             "resource node",
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

func getNodeCmd() *cobra.Command {
	nodeCmd := &cobra.Command{
		Use:     "start",
		Short:   "start the node",
		PreRunE: common.NodePreRunE,
		RunE:    common.NodePP,
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
	execCmd.Flags().BoolP(verboseFlag, "v", false, "output logs")
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
	cmd.AddCommand(getUpdateConfigCmd())
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
	cmd.Flags().String(hdPathFlag, setting.HDPath, "hd-path for the new wallet")
	cmd.Flags().StringP(passwordFlag, "p", "", "wallet password, if not provided, will need to input in prompt")
	cmd.Flags().StringP(nicknameFlag, "n", "wallet", "name of wallet")
	cmd.Flags().BoolP(savePassFlag, "s", false, "save wallet password to configuration file")
	cmd.Flags().String(p2pPassFlag, "aaa", "p2p password, optional")
	cmd.Flags().Bool(newP2pKeyFlag, false, "create a new p2p key even if one already exists")
	cmd.Flags().String(hdPathP2pFlag, setting.HDPathP2p, "hd-path for the new p2p key")
	cmd.Flags().String(p2pPrivKeyFlag, "", "Hex-encoded p2p private key, optional")
	return cmd
}

func getUpdateConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update the config file to the latest version",
		RunE:  updateConfigVersion,
	}

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

func getExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "export a wallet or p2p key info",
	}
	cmd.AddCommand(getExportWalletCmd())
	cmd.AddCommand(getExportP2pCmd())

	cmd.PersistentFlags().StringP(addressFlag, "a", "", "address of the key to export")
	cmd.PersistentFlags().StringP(passwordFlag, "p", "aaa", "password of the key to export")
	return cmd
}

func getExportWalletCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wallet",
		Short: "export a wallet info",
		RunE:  exportWallet,
	}
	return cmd
}

func getExportP2pCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "p2p",
		Short: "export a p2p key info",
		RunE:  exportP2pKey,
	}
	return cmd
}
