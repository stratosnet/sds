package main

import (
	"fmt"
	"github.com/stratosnet/sds/cmd/relayd/client"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	setting.LoadConfig("./config/config.yaml")

	multiClient := client.NewClient()
	err := multiClient.Start()
	if err != nil {
		fmt.Println("Shutting down. Could not start relay client: " + err.Error())
		return
	}
	fmt.Println("Successfully subscribed to events from SDS and stratos-chain, and started client to send messages back")

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
			multiClient.Stop()
			os.Exit(0)
		}
	}
}
