package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alex023/clock"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/stratosnet/sds/cmd/common"
	"github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/framework/utils/console"
	"github.com/stratosnet/sds/pp/serv"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/rpc"
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
		"registerpeer                                                   register peer to index node\n" +
		"rp                                                             register peer to index node\n" +
		"activate <amount> <fee> optional<gas>                          send transaction to stchain to become an active PP node\n" +
		"updateDeposit <depositDelta> <fee> optional<gas>               send transaction to stchain to update active pp's deposit\n" +
		"deactivate <fee> optional<gas>                                 send transaction to stchain to stop being an active PP node\n" +
		"startmining                                                    start mining\n" +
		"prepay <amount> <fee> optional<beneficiary> <gas>              prepay stos to get ozone\n" +
		"put <filepath> optional<isEncrypted> optional<nodeTier>        \n" +
		"               optional<allowHigherTier>                       upload file, need to consume ozone\n" +
		"putstream <filepath> optional<isEncrypted> optional<nodeTier>  \n" +
		"                     optional<allowHigherTier>                 upload video file for streaming, need to consume ozone. (alpha version, encode format config impossible)\n" +
		"list <filename>                                                query uploaded file by self\n" +
		"list <page id>                                                 query all files owned by the wallet, paginated\n" +
		"delete <filehash>                                              delete file\n" +
		"get <sdm://account/filehash> <saveAs>                          download file, need to consume ozone\n" +
		"                                                               e.g: get sdm://st1jn9skjsnxv26mekd8eu8a8aquh34v0m4mwgahg/v05ahm50ugfjrgd3ga8mqi6bqka32ks3dooe1p9g\n" +
		"sharefile <filehash> <duration> <is_private>                   share an uploaded file\n" +
		"allshare                                                       list all shared files\n" +
		"getsharefile <sharelink> <password>                            download a shared file, need to consume ozone\n" +
		"cancelshare <shareID>                                          cancel a shared file\n" +
		"clearexpshare                                                  clear all expired share links\n" +
		"ver                                                            version\n" +
		"monitor                                                        show monitor\n" +
		"stopmonitor                                                    stop monitor\n" +
		"monitortoken                                                   show token for pp monitor service\n" +
		"config  <key> <value>                                          set config key value\n" +
		"getoz <walletAddress>                                          get current ozone balance\n" +
		"status                                                         get current resource node status\n" +
		"filestatus <filehash>                                          get current state of an uploaded file\n" +
		"maintenance start <duration>                                   put the node in maintenance mode for the requested duration (in seconds)\n" +
		"maintenance stop                                               stop the current maintenance\n" +
		"downgradeinfo                                                  get information of last downgrade happened on this pp node\n" +
		"performancemeasure                                             turn on performance measurement log for 60 seconds\n" +
		"withdraw <amount> <fee> optional<targetAddr> optional<gas>     withdraw matured reward (from address is the configured node wallet)\n" +
		"send <toAddress> <amount> <fee> optional<gas>                  sending coins to another account (from address is the configured node wallet)\n"

	terminalId := uuid.New().String()

	help := func(line string, param []string) bool {
		fmt.Println(helpStr)
		return true
	}

	wallets := func(line string, param []string) bool {
		return callRpc(c, terminalId, "wallets", param)
	}

	getoz := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("missing wallet address")
			return false
		}
		return callRpc(c, terminalId, "getoz", param)
	}

	newwallet := func(line string, param []string) bool {
		err := types.SetupWallet(setting.Config.Home.AccountsPath, setting.HDPath, updateWalletConfig)
		if err != nil {
			fmt.Println(err)
			return false
		}
		return true
	}

	start := func(line string, param []string) bool {
		return callRpc(c, terminalId, "start", param)
	}

	registerPP := func(line string, param []string) bool {
		return callRpc(c, terminalId, "registerPP", param)
	}

	activate := func(line string, param []string) bool {
		return callRpc(c, terminalId, "activate", param)
	}

	updateDeposit := func(line string, param []string) bool {
		return callRpc(c, terminalId, "updateDeposit", param)
	}

	status := func(line string, param []string) bool {
		return callRpc(c, terminalId, "status", param)
	}

	fileStatus := func(line string, param []string) bool {
		return callRpc(c, terminalId, "fileStatus", param)
	}

	deactivate := func(line string, param []string) bool {
		return callRpc(c, terminalId, "deactivate", param)
	}

	prepay := func(line string, param []string) bool {
		return callRpc(c, terminalId, "prepay", param)
	}

	upload := func(line string, param []string) bool {
		return callRpc(c, terminalId, "upload", param)
	}

	uploadStream := func(line string, param []string) bool {
		return callRpc(c, terminalId, "uploadStream", param)
	}

	backupStatus := func(line string, param []string) bool {
		return callRpc(c, terminalId, "backupStatus", param)
	}

	list := func(line string, param []string) bool {
		return callRpc(c, terminalId, "list", param)
	}

	download := func(line string, param []string) bool {
		return callRpc(c, terminalId, "download", param)
	}

	deleteFn := func(line string, param []string) bool {
		return callRpc(c, terminalId, "deleteFn", param)
	}

	ver := func(line string, param []string) bool {
		return callRpc(c, terminalId, "ver", param)
	}

	monitor := func(line string, param []string) bool {
		return callRpc(c, terminalId, "monitor", param)
	}

	stopmonitor := func(line string, param []string) bool {
		return callRpc(c, terminalId, "stopMonitor", param)
	}

	config := func(line string, param []string) bool {
		return callRpc(c, terminalId, "config", param)
	}

	sharepath := func(line string, param []string) bool {
		return callRpc(c, terminalId, "sharePath", param)
	}

	sharefile := func(line string, param []string) bool {
		return callRpc(c, terminalId, "shareFile", param)
	}

	allshare := func(line string, param []string) bool {
		return callRpc(c, terminalId, "allShare", param)
	}

	cancelshare := func(line string, param []string) bool {
		return callRpc(c, terminalId, "cancelShare", param)
	}

	clearexpshare := func(line string, param []string) bool {
		return callRpc(c, terminalId, "clearExpShare", param)
	}

	getsharefile := func(line string, param []string) bool {
		return callRpc(c, terminalId, "getShareFile", param)
	}

	pauseget := func(line string, param []string) bool {
		return callRpc(c, terminalId, "pauseGet", param)
	}

	pauseput := func(line string, param []string) bool {
		return callRpc(c, terminalId, "pausePut", param)
	}

	cancelget := func(line string, param []string) bool {
		return callRpc(c, terminalId, "cancelGet", param)
	}
	monitortoken := func(line string, param []string) bool {
		return callRpc(c, terminalId, "monitorToken", param)
	}
	maintenance := func(line string, param []string) bool {
		return callRpc(c, terminalId, "maintenance", param)
	}
	downgradeInfo := func(line string, param []string) bool {
		return callRpc(c, terminalId, "downgradeInfo", param)
	}
	performanceMeasure := func(line string, param []string) bool {
		return callRpc(c, terminalId, "performanceMeasure", param)
	}
	checkReplica := func(line string, param []string) bool {
		return callRpc(c, terminalId, "checkReplica", param)
	}
	withdraw := func(line string, param []string) bool {
		return callRpc(c, terminalId, "withdraw", param)
	}
	send := func(line string, param []string) bool {
		return callRpc(c, terminalId, "send", param)
	}

	nc := make(chan utils.LogMsg)
	sub, err := c.Subscribe(context.Background(), "sdslog", nc, "logSubscription", terminalId)
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
	console.Mystdin.RegisterProcessFunc("startmining", start, true)
	console.Mystdin.RegisterProcessFunc("rp", registerPP, true)
	console.Mystdin.RegisterProcessFunc("registerpeer", registerPP, true)
	console.Mystdin.RegisterProcessFunc("activate", activate, true)
	console.Mystdin.RegisterProcessFunc("updateDeposit", updateDeposit, true)
	console.Mystdin.RegisterProcessFunc("status", status, true)
	console.Mystdin.RegisterProcessFunc("filestatus", fileStatus, true)
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
	console.Mystdin.RegisterProcessFunc("clearexpshare", clearexpshare, true)

	console.Mystdin.RegisterProcessFunc("pauseget", pauseget, true)
	console.Mystdin.RegisterProcessFunc("pauseput", pauseput, true)
	console.Mystdin.RegisterProcessFunc("cancelget", cancelget, true)
	console.Mystdin.RegisterProcessFunc("monitortoken", monitortoken, true)
	console.Mystdin.RegisterProcessFunc("maintenance", maintenance, true)
	console.Mystdin.RegisterProcessFunc("downgradeinfo", downgradeInfo, true)
	console.Mystdin.RegisterProcessFunc("performancemeasure", performanceMeasure, true)
	console.Mystdin.RegisterProcessFunc("CheckReplica", checkReplica, true)
	console.Mystdin.RegisterProcessFunc("withdraw", withdraw, true)
	console.Mystdin.RegisterProcessFunc("send", send, true)

	if isExec {
		exit := false
		if len(args) > 0 {
			strKey := strings.ToLower(args[0])
			exit = console.Mystdin.RunCmd(strKey, args[1:], true)
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

func callRpc(c *rpc.Client, terminalId string, line string, param []string) bool {
	var result serv.CmdResult

	paramWithTid := []string{terminalId}
	if len(param) > 0 {
		paramWithTid = append(paramWithTid, param...)
	}
	err := c.Call(&result, "sds_"+line, paramWithTid)
	if err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println(result.Msg)
	return true
}

func printLogNotification(nc <-chan utils.LogMsg) {
	for n := range nc {
		fmt.Print(n.Msg)
	}
}

func destroySub(c *rpc.Client, sub *rpc.ClientSubscription) {
	//var cleanResult interface{}
	sub.Unsubscribe()
	//_ = c.Call(&cleanResult, "sdslog_cleanUp")
}

func printExitMsg() {
	fmt.Println("Press the right bracket ']' to exit")
}
