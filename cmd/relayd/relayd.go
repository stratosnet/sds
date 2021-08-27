package main

import (
	"fmt"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/relay/client"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Not enough arguments. Please specify the config file to use")
		return
	}
	err := setting.LoadConfig(os.Args[1])
	if err != nil {
		fmt.Println("Error loading the config file: " + err.Error())
		return
	}

	multiClient := client.NewClient()
	defer multiClient.Stop()

	err = multiClient.Start()
	if err != nil {
		fmt.Println("Shutting down. Could not start relay client: " + err.Error())
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
