package main

import (
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	sdkmath "cosmossdk.io/math"
	"github.com/spf13/cobra"

	txclienttype "github.com/stratosnet/tx-client/types"

	"github.com/stratosnet/framework/utils"

	"github.com/stratosnet/relay/client"
	"github.com/stratosnet/relay/cmd/relayd/setting"
	"github.com/stratosnet/relay/server"
)

func startRunE(cmd *cobra.Command, _ []string) error {
	err := registerDenoms()
	if err != nil {
		return errors.New(utils.FormatError(err))
	}

	spHomePath, err := cmd.Flags().GetString(server.SpHome)
	if err != nil {
		utils.ErrorLog("failed to get 'sp-home' path for the relayd process")
		return err
	}
	spHomePath, err = utils.Absolute(spHomePath)
	if err != nil {
		utils.ErrorLog("cannot convert sp-home path to absolute path")
		return err
	}

	multiClient, err := client.NewClient(spHomePath)
	if err != nil {
		utils.ErrorLog("cannot create new relay client")
		return err
	}

	defer multiClient.Stop()
	defer os.Exit(1)

	err = multiClient.Start()
	if err != nil {
		utils.ErrorLog("Shutting down. Could not start relay client", err)
		return err
	}

	err = server.BaseServer.Start()
	defer server.BaseServer.Stop()
	if err != nil {
		return errors.New(utils.FormatError(err))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)

	for {
		select {
		case <-quit:
			utils.Log("Quit signal detected. Shutting down...")
			return nil
		case <-multiClient.Ctx.Done():
			return nil
		}
	}
}

func startPreRunE(cmd *cobra.Command, _ []string) error {
	homePath, err := cmd.Flags().GetString(server.Home)
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

	configPath, err := cmd.Flags().GetString(server.Config)
	if err != nil {
		utils.ErrorLog("failed to get 'config' path for the relayd process")
		return err
	}
	configPath = filepath.Join(homePath, configPath)
	setting.SetIPCEndpoint(homePath)
	err = setting.LoadConfig(configPath)
	if err != nil {
		utils.ErrorLog("Error loading the setting file", err)
		return err
	}
	return nil
}

// RegisterDenoms registers the denominations to the PP.
func registerDenoms() error {
	if err := txclienttype.RegisterDenom(txclienttype.Stos, sdkmath.LegacyOneDec()); err != nil {
		return err
	}
	if err := txclienttype.RegisterDenom(txclienttype.Gwei, sdkmath.LegacyNewDecWithPrec(1, txclienttype.GweiDenomUnit)); err != nil {
		return err
	}
	if err := txclienttype.RegisterDenom(txclienttype.Wei, sdkmath.LegacyNewDecWithPrec(1, txclienttype.WeiDenomUnit)); err != nil {
		return err
	}

	return nil
}
