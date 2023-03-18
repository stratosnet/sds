package ipfs

import (
	"os"
	"os/exec"
	"time"

	"github.com/alex023/clock"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/cmd/ppd/cliutils"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
)

const (
	IpcNamespaceTerminal = "rpc"
)

func Ipfsmigrate(cmd *cobra.Command, args []string) {
	ipcEndpointParam, _ := cmd.Flags().GetString(IpcNamespaceTerminal)
	ipcEndpoint := setting.IpcEndpoint
	if ipcEndpointParam != "" {
		ipcEndpoint = ipcEndpointParam
	}

	c, err := rpc.Dial(ipcEndpoint)
	if err != nil {
		panic("failed to dial ipc endpoint")
	}
	defer c.Close()

	sub, nc, err := cliutils.SubscribeLog(c)
	if err != nil {
		utils.ErrorLog("can't subscribe:", err)
		return
	}
	defer cliutils.DestroySub(c, sub)
	go cliutils.PrintLogNotification(nc)

	if len(args) > 0 {
		cliutils.CallRpc(c, "migrateIpfsFile", args)
	}

	cliutils.PrintExitMsg()
	clock.NewClock().AddJobRepeat(10*time.Second, 0, cliutils.PrintExitMsg)

	// disable input buffering
	_ = exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	// do not display entered characters on the screen
	_ = exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	var b = make([]byte, 1)
	for {
		_, _ = os.Stdin.Read(b)
		if b[0] == ']' {
			break
		}
	}
}
