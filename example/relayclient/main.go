package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/cmd/common"
	"github.com/stratosnet/sds/utils"
)

const (
	DefaultUrl = "http://127.0.0.1:9881"
	//DefaultPassword = "aaa"
)

var (
	//TODO wallet related features to be activated in the future (currently commented out)
	//WalletPrivateKey types.AccPrivKey
	//WalletPublicKey  types.AccPubKey
	//WalletAddress    string

	Url string
)

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      int             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func isWalletFile(fileName string) bool {
	match, _ := filepath.Match("st1*", fileName)
	return match
}

func isWalletKeyPath(filePath string) bool {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	return !fileInfo.IsDir() && filepath.Ext(filePath) == ".json" && isWalletFile(fileInfo.Name())
}

func findWalletFile(folder string) string {
	var walletPath string

	_ = filepath.Walk(folder, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return nil
		}

		if isWalletKeyPath(path) {
			// only catch the first wallet file
			walletPath = path
			return nil
		}
		return nil
	})
	return walletPath
}

// rootPreRunE
func rootPreRunE(cmd *cobra.Command, _ []string) error {
	var err error
	Url, err = cmd.Flags().GetString("url")
	if err != nil {
		return errors.New(utils.FormatError(err))
	}

	_, err = cmd.Flags().GetString(common.Home) // homePath
	if err != nil {
		return errors.New(utils.FormatError(err))
	}

	//walletArg, err := cmd.Flags().GetString("wallet")
	//if err != nil {
	//	return errors.New(utils.FormatError(err))
	//}
	//
	//password, err := cmd.Flags().GetString("password")
	//if err != nil {
	//	return errors.New(utils.FormatError(err))
	//}

	//walletFolder := filepath.Join(homePath, "accounts")
	//walletPath := walletArg
	//if isWalletFile(walletArg) {
	//	walletPath = filepath.Join(walletFolder, walletArg)
	//	if filepath.Ext(walletPath) == "" {
	//		walletPath += ".json"
	//	}
	//} else if walletArg == "" {
	//	walletPath = findWalletFile(walletFolder)
	//}
	//
	//if walletPath == "" {
	//	return errors.New("couldn't locate wallet file")
	//}
	//
	//keyjson, err := os.ReadFile(walletPath)
	//if err != nil {
	//	return errors.New(utils.FormatError(err))
	//}
	//
	//key, err := utils.DecryptKey(keyjson, password)
	//if err != nil {
	//	return errors.New(utils.FormatError(err))
	//}
	//
	//WalletAddress, err = key.Address.WalletAddressToBech()
	//if err != nil {
	//	return errors.New(utils.FormatError(err))
	//}
	//WalletPrivateKey = types.BytesToAccPriveKey(key.PrivateKey)
	//WalletPublicKey = WalletPrivateKey.PubKeyFromPrivKey()

	return nil
}

// main
func main() {
	rootCmd := &cobra.Command{
		Use:               "relay_client",
		Short:             "relay client for test/maintenance purpose",
		PersistentPreRunE: rootPreRunE,
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		utils.ErrorLog("failed to get working directory")
		panic(err)
	}

	rootCmd.PersistentFlags().StringP("url", "u", DefaultUrl, "url to the RPC server, e.g. http://3.24.59.6:8235")
	rootCmd.PersistentFlags().StringP(common.Home, "r", workingDirectory, "path for the node")
	//rootCmd.PersistentFlags().StringP("wallet", "w", "", "wallet address to be used, or path to the wallet key file (default: the first wallet in folder ./accounts/)")
	//rootCmd.PersistentFlags().StringP("password", "p", DefaultPassword, "the password of the wallet file")

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "sync missed stchain tx by its hash",
		RunE:  sync,
	}

	rootCmd.AddCommand(syncCmd)

	combineLogger := utils.NewDefaultLogger("./logs/stdout.log", true, true)
	combineLogger.SetLogLevel(utils.Debug)

	err = rootCmd.Execute()
	if err != nil {
		utils.ErrorLog(err)
	}
}
