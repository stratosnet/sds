package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/alex023/clock"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/cmd/common"
	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/console"
)

const (
	verboseFlag = "verbose"
)

func run(cmd *cobra.Command, args []string, isExec bool) {
	c, err := rpc.Dial(setting.IpcEndpoint)

	if err != nil {
		utils.ErrorLog(err)
		return
	}
	defer c.Close()

	helpStr := "\n" +
		"help                                                           show all the commands\n" +
		"wallets                                                        acquire all wallet wallets' address\n" +
		"newwallet                                                      create new wallet, input password in prompt\n" +
		"login <walletAddress> ->password                               unlock and log in wallet, input password in prompt\n" +
		"registerpeer                                                   register peer to index node\n" +
		"rp                                                             register peer to index node\n" +
		"activate <amount> <fee> optional<gas>                          send transaction to stchain to become an active PP node\n" +
		"updateDeposit <depositDelta> <fee> optional<gas>               send transaction to stchain to update active pp's deposit\n" +
		"deactivate <fee> optional<gas>                                 send transaction to stchain to stop being an active PP node\n" +
		"startmining                                                    start mining\n" +
		"prepay <amount> <fee> optional<beneficiary> <gas>              prepay stos to get ozone\n" +
		"put <filepath> optional<isEncrypted> optional<nodeTier> optional<allowHigherTier>\n" +
		"                                                               upload file, need to consume ozone\n" +
		"putstream <filepath> optional<isEncrypted> optional<nodeTier> optional<allowHigherTier>\n" +
		"                                                               upload video file for streaming, need to consume ozone. (alpha version, encode format config impossible)\n" +
		"list <filename>                                                query uploaded file by self\n" +
		"list <page id>                                                 query all files owned by the wallet, paginated\n" +
		"delete <filehash>                                              delete file\n" +
		"get <sdm://account/filehash> <saveAs>                          download file, need to consume ozone\n" +
		"    e.g: get sdm://st1jn9skjsnxv26mekd8eu8a8aquh34v0m4mwgahg/e2ba7fd2390aad9213f2c60854e2b7728c6217309fcc421de5aacc7d4019a4fe\n" +
		"sharefile <filehash> <duration> <is_private>                   share an uploaded file\n" +
		"allshare                                                       list all shared files\n" +
		"getsharefile <sharelink> <password>                            download a shared file, need to consume ozone\n" +
		"cancelshare <shareID>                                          cancel a shared file\n" +
		"ver                                                            version\n" +
		"monitor                                                        show monitor\n" +
		"stopmonitor                                                    stop monitor\n" +
		"monitortoken                                                   show token for pp monitor service\n" +
		"config  <key> <value>                                          set config key value\n" +
		"getoz <walletAddress> ->password                               get current ozone balance\n" +
		"status                                                         get current resource node status\n" +
		"maintenance start <duration>                                   put the node in maintenance mode for the requested duration (in seconds)\n" +
		"maintenance stop                                               stop the current maintenance\n" +
		"downgradeinfo                                                  get information of last downgrade happened on this pp node\n" +
		"performancemeasure                                             turn on performance measurement log for 60 seconds\n"

	help := func(line string, param []string) bool {
		fmt.Println(helpStr)
		return true
	}

	wallets := func(line string, param []string) bool {
		return callRpc(c, "wallets", param)
	}

	getoz := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("missing wallet address")
			return false
		}
		password := console.MyGetPassword("input password", false)
		return callRpc(c, "getoz", []string{param[0], password})
	}

	newwallet := func(line string, param []string) bool {
		err := utils.SetupWallet(setting.Config.Home.AccountsPath, setting.HDPath, updateWalletConfig)
		if err != nil {
			fmt.Println(err)
			return false
		}
		return true
	}

	login := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("input wallet address")
			return false
		}
		if len(param[0]) != 41 {
			fmt.Println("input correct wallet address")
			return false
		}
		password := console.MyGetPassword("input password", false)

		return callRpc(c, "login", []string{param[0], password})
	}

	start := func(line string, param []string) bool {
		return callRpc(c, "start", param)
	}

	registerPP := func(line string, param []string) bool {
		return callRpc(c, "registerPP", param)
	}

	activate := func(line string, param []string) bool {
		return callRpc(c, "activate", param)
	}

	updateDeposit := func(line string, param []string) bool {
		return callRpc(c, "updateDeposit", param)
	}

	status := func(line string, param []string) bool {
		return callRpc(c, "status", param)
	}

	deactivate := func(line string, param []string) bool {
		return callRpc(c, "deactivate", param)
	}

	prepay := func(line string, param []string) bool {
		return callRpc(c, "prepay", param)
	}

	upload := func(line string, param []string) bool {
		return callRpc(c, "upload", param)
	}

	uploadStream := func(line string, param []string) bool {
		return callRpc(c, "uploadStream", param)
	}

	backupStatus := func(line string, param []string) bool {
		return callRpc(c, "backupStatus", param)
	}

	list := func(line string, param []string) bool {
		return callRpc(c, "list", param)
	}

	download := func(line string, param []string) bool {
		return callRpc(c, "download", param)
	}

	deleteFn := func(line string, param []string) bool {
		return callRpc(c, "deleteFn", param)
	}

	ver := func(line string, param []string) bool {
		return callRpc(c, "ver", param)
	}

	monitor := func(line string, param []string) bool {
		return callRpc(c, "monitor", param)
	}

	stopmonitor := func(line string, param []string) bool {
		return callRpc(c, "stopMonitor", param)
	}

	config := func(line string, param []string) bool {
		return callRpc(c, "config", param)
	}

	sharepath := func(line string, param []string) bool {
		return callRpc(c, "sharePath", param)
	}

	sharefile := func(line string, param []string) bool {
		return callRpc(c, "shareFile", param)
	}

	allshare := func(line string, param []string) bool {
		return callRpc(c, "allShare", param)
	}

	cancelshare := func(line string, param []string) bool {
		return callRpc(c, "cancelShare", param)
	}

	getsharefile := func(line string, param []string) bool {
		return callRpc(c, "getShareFile", param)
	}

	pauseget := func(line string, param []string) bool {
		return callRpc(c, "pauseGet", param)
	}

	pauseput := func(line string, param []string) bool {
		return callRpc(c, "pausePut", param)
	}

	cancelget := func(line string, param []string) bool {
		return callRpc(c, "cancelGet", param)
	}
	monitortoken := func(line string, param []string) bool {
		return callRpc(c, "monitorToken", param)
	}
	maintenance := func(line string, param []string) bool {
		return callRpc(c, "maintenance", param)
	}
	downgradeInfo := func(line string, param []string) bool {
		return callRpc(c, "downgradeInfo", param)
	}
	performanceMeasure := func(line string, param []string) bool {
		return callRpc(c, "performanceMeasure", param)
	}
	checkReplica := func(line string, param []string) bool {
		return callRpc(c, "checkReplica", param)
	}

	nc := make(chan serv.LogMsg)
	sub, err := c.Subscribe(context.Background(), "sdslog", nc, "logSubscription")
	if err != nil {
		utils.ErrorLog("can't subscribe:", err)
		return
	}
	defer destroySub(c, sub)

	go printLogNotification(nc)

	console.Mystdin.RegisterProcessFunc("help", help, true)
	console.Mystdin.RegisterProcessFunc("h", help, true)
	console.Mystdin.RegisterProcessFunc("wallets", wallets, false)
	console.Mystdin.RegisterProcessFunc("getoz", getoz, true)
	console.Mystdin.RegisterProcessFunc("newwallet", newwallet, false)
	console.Mystdin.RegisterProcessFunc("login", login, false)
	console.Mystdin.RegisterProcessFunc("startmining", start, true)
	console.Mystdin.RegisterProcessFunc("rp", registerPP, true)
	console.Mystdin.RegisterProcessFunc("registerpeer", registerPP, true)
	console.Mystdin.RegisterProcessFunc("activate", activate, true)
	console.Mystdin.RegisterProcessFunc("updateDeposit", updateDeposit, true)
	console.Mystdin.RegisterProcessFunc("status", status, true)
	console.Mystdin.RegisterProcessFunc("deactivate", deactivate, true)
	console.Mystdin.RegisterProcessFunc("prepay", prepay, true)
	console.Mystdin.RegisterProcessFunc("u", upload, true)
	console.Mystdin.RegisterProcessFunc("put", upload, true)
	console.Mystdin.RegisterProcessFunc("putstream", uploadStream, true)
	console.Mystdin.RegisterProcessFunc("backupStatus", backupStatus, true)
	console.Mystdin.RegisterProcessFunc("d", download, true)
	console.Mystdin.RegisterProcessFunc("get", download, true)
	console.Mystdin.RegisterProcessFunc("list", list, true)
	console.Mystdin.RegisterProcessFunc("ls", list, true)
	console.Mystdin.RegisterProcessFunc("delete", deleteFn, true)
	console.Mystdin.RegisterProcessFunc("rm", deleteFn, true)
	console.Mystdin.RegisterProcessFunc("ver", ver, false)
	console.Mystdin.RegisterProcessFunc("monitor", monitor, true)
	console.Mystdin.RegisterProcessFunc("stopmonitor", stopmonitor, true)

	console.Mystdin.RegisterProcessFunc("config", config, true)
	console.Mystdin.RegisterProcessFunc("sharefile", sharefile, true)
	console.Mystdin.RegisterProcessFunc("sharepath", sharepath, true)
	console.Mystdin.RegisterProcessFunc("allshare", allshare, false)
	console.Mystdin.RegisterProcessFunc("cancelshare", cancelshare, true)
	console.Mystdin.RegisterProcessFunc("getsharefile", getsharefile, true)

	console.Mystdin.RegisterProcessFunc("pauseget", pauseget, true)
	console.Mystdin.RegisterProcessFunc("pauseput", pauseput, true)
	console.Mystdin.RegisterProcessFunc("cancelget", cancelget, true)
	console.Mystdin.RegisterProcessFunc("monitortoken", monitortoken, true)
	console.Mystdin.RegisterProcessFunc("maintenance", maintenance, true)
	console.Mystdin.RegisterProcessFunc("downgradeinfo", downgradeInfo, true)
	console.Mystdin.RegisterProcessFunc("performancemeasure", performanceMeasure, true)
	console.Mystdin.RegisterProcessFunc("CheckReplica", checkReplica, true)

	if isExec {
		exit := false
		if len(args) > 0 {
			exit = console.Mystdin.RunCmd(args[0], args[1:], true)
		}

		if exit {
			return
		}

		verbose, err := cmd.Flags().GetBool(verboseFlag)
		if err != nil || !verbose {
			return
		}

		printExitMsg()
		clock.NewClock().AddJobRepeat(10*time.Second, 0, printExitMsg)

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
		return
	}

	fmt.Println(helpStr)
	console.Mystdin.Run()
}

func execute(cmd *cobra.Command, args []string) {
	run(cmd, args, true)
}

func terminal(cmd *cobra.Command, args []string) {
	run(cmd, args, false)
}

func terminalPreRunE(cmd *cobra.Command, args []string) error {
	return common.LoadConfig(cmd)
}

func callRpc(c *rpc.Client, line string, param []string) bool {
	var result serv.CmdResult
	err := c.Call(&result, "sds_"+line, param)
	if err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println(result.Msg)
	return true
}

func printLogNotification(nc <-chan serv.LogMsg) {
	for n := range nc {
		fmt.Print(n.Msg)
	}
}

func destroySub(c *rpc.Client, sub *rpc.ClientSubscription) {
	var cleanResult interface{}
	sub.Unsubscribe()
	_ = c.Call(&cleanResult, "sdslog_cleanUp")
}

func printExitMsg() {
	fmt.Println("Press the right bracket ']' to exit")
}
