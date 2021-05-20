package main

import (
	"fmt"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/relay/client"
	"github.com/stratosnet/sds/utils"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	utils.NewLogger("stdout.log", true, true)
	setting.LoadConfig("./config/config.yaml")

	multiClient := client.NewClient()
	defer multiClient.Stop()

	err := multiClient.Start()
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
