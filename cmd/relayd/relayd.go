package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/relay/client"
	"github.com/stratosnet/sds/utils"
)

func startRunE(cmd *cobra.Command, _ []string) error {
	spHomePath, err := cmd.Flags().GetString(SP_HOME)
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGKILL,
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
