package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stratosnet/sds/cmd/common"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/ipfs/cmd/ppd/ipfs"
	"github.com/stratosnet/sds/pp/setting"
)

func main() {

	rootCmd := getRootCmd()
	nodeCmd := getNodeCmd()
	terminalCmd := getTerminalCmd()
	ipfsapiCmd := getIpfsCmd()
	configCmd := getGenConfigCmd()
	verCmd := getVersionCmd()

	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(terminalCmd)
	rootCmd.AddCommand(ipfsapiCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(verCmd)

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

func getIpfsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ipfs",
		Short:   "ipfs api server attached to node demon",
		PreRunE: ipfs.IpfsapiPreRunE,
		Run:     ipfs.Ipfsapi,
	}

	migrateCmd := &cobra.Command{
		Use:     "migrate",
		Short:   "migrate ipfs file to sds",
		PreRunE: ipfs.IpfsapiPreRunE,
		Run:     ipfs.Ipfsmigrate,
	}

	cmd.PersistentFlags().StringP(ipfs.RpcModeFlag, "m", "ipc", "use http rpc or ipc")
	cmd.PersistentFlags().String(ipfs.PasswordFlag, "", "wallet password")
	cmd.PersistentFlags().StringP(ipfs.IpfsPortFlag, "p", "6798", "port")
	cmd.PersistentFlags().StringP(ipfs.IpcEndpoint, "", "", "ipc endpoint path")
	cmd.PersistentFlags().StringP(ipfs.HttpRpcUrl, "", ipfs.HttpRpcDefaultUrl, "http rpc url")
	cmd.AddCommand(migrateCmd)
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
	cmd.Flags().String(hdPathFlag, setting.HDPath, "hd-path for the wallet created")
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
