package main

import (
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/relay/client"
	"github.com/stratosnet/sds/utils"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		utils.Log("Not enough arguments. Please specify the config file to use")
		return
	}
	err := setting.LoadConfig(os.Args[1])
	if err != nil {
		utils.ErrorLog("Error loading the config file", err)
		return
	}

	multiClient := client.NewClient()
	defer multiClient.Stop()

	err = multiClient.Start()
	if err != nil {
		utils.ErrorLog("Shutting down. Could not start relay client", err)
		return
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
			return
		case <-multiClient.Ctx.Done():
			return
		}
	}
}
