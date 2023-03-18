package cliutils

import (
	"context"
	"fmt"

	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/rpc"
)

func CallRpc(c *rpc.Client, line string, param []string) bool {
	var result serv.CmdResult
	err := c.Call(&result, "sds_"+line, param)
	if err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println(result.Msg)
	return true
}

func DestroySub(c *rpc.Client, sub *rpc.ClientSubscription) {
	var cleanResult interface{}
	sub.Unsubscribe()
	_ = c.Call(&cleanResult, "sdslog_cleanUp")
}

func PrintExitMsg() {
	fmt.Println("Press the right bracket ']' to exit")
}

func PrintLogNotification(nc <-chan serv.LogMsg) {
	for n := range nc {
		fmt.Print(n.Msg)
	}
}

func SubscribeLog(c *rpc.Client) (sub *rpc.ClientSubscription, nc chan serv.LogMsg, err error) {
	nc = make(chan serv.LogMsg)
	sub, err = c.Subscribe(context.Background(), "sdslog", nc, "logSubscription")
	return
}
