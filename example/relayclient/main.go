package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/cmd/common"
	"github.com/stratosnet/sds/utils"
)

const (
	DefaultUrl = "http://127.0.0.1:9881"
)

var (
	Url string
)

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      int             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
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
