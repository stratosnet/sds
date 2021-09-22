package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/api"
	"github.com/stratosnet/sds/pp/api/rest"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/websocket"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/console"
)

func terminal(cmd *cobra.Command, args []string) {

	helpStr := "\n" +
		"help                                show all the commands\n" +
		"wallets                             acquire all wallet wallets' address\n" +
		"newwallet ->password                create new wallet\n" +
		"login walletAddress ->password      unlock and log in wallet \n" +
		"registerminer                       apply to be PP miner\n" +
		"activate                            send transaction to stratos-chain to become an active PP node\n" +
		"deactivate                          send transaction to stratos-chain to stop being a PP node\n" +
		"start                               start mining\n" +
		"put filepath                        upload file\n" +
		"list filename                       inquire uploaded file by self\n" +
		"list                                inquire all files\n" +
		"delete filehash                     delete file\n" +
		"get spb://account/filehash|filename download file\n" +
		"For example: get spb://0x96983DA5Aed28Ac0FF7646fE1C3260AACe9ECB7B/e2ba7fd2390aad9213f2c60854e2b7728c6217309fcc421de5aacc7d4019a4fe|test.mp4\n" +
		"ver                                 version\n" +
		"monitor                             show monitor\n" +
		"stopmonitor                         stop monitor\n" +
		"config                              config key value"
	fmt.Println(helpStr)

	help := func(line string, param []string) bool {
		fmt.Println(helpStr)
		return true
	}

	wallets := func(line string, param []string) bool {
		peers.Wallets()
		return true
	}

	newWallet := func(line string, param []string) bool {
		if len(param) < 2 {
			fmt.Println("Not enough arguments. Please provide the new wallet name and hdPath")
			return false
		}

		password := console.MyGetPassword("input password", true)
		if len(password) == 0 {
			fmt.Println("wrong password")
			return false
		}

		mnemonic := console.MyGetPassword("input bip39 mnemonic (leave blank to generate a new one)", false)
		passphrase := console.MyGetPassword("input bip39 passphrase", false)

		peers.CreateWallet(password, param[0], mnemonic, passphrase, param[1])
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
		if len(password) == 0 {
			fmt.Println("empty password")
			return false
		}
		peers.Login(param[0], password)

		return false
	}

	start := func(line string, param []string) bool {
		event.StartMining()
		return true
	}

	registerPP := func(line string, param []string) bool {
		event.RegisterNewPP()
		return true
	}

	activate := func(line string, param []string) bool {
		if len(param) < 3 {
			fmt.Println("Expecting 3 params. Input amount of tokens, fee amount and gas amount")
			return false
		}
		amount, err := strconv.ParseInt(param[0], 10, 64)
		if err != nil {
			fmt.Println("Invalid amount param. Should be an integer")
			return false
		}
		fee, err := strconv.ParseInt(param[1], 10, 64)
		if err != nil {
			fmt.Println("Invalid fee param. Should be an integer")
			return false
		}
		gas, err := strconv.ParseInt(param[2], 10, 64)
		if err != nil {
			fmt.Println("Invalid gas param. Should be an integer")
			return false
		}

		if setting.State != setting.PP_INACTIVE {
			return true
		}

		if !setting.IsPP {
			fmt.Println("register as a PP node first")
			return true
		}

		return event.Activate(amount, fee, gas) == nil
	}

	deactivate := func(line string, param []string) bool {
		if len(param) < 2 {
			fmt.Println("Expecting 2 params. Input fee amount and gas amount")
			return false
		}
		fee, err := strconv.ParseInt(param[0], 10, 64)
		if err != nil {
			fmt.Println("Invalid fee param. Should be an integer")
			return false
		}
		gas, err := strconv.ParseInt(param[1], 10, 64)
		if err != nil {
			fmt.Println("Invalid gas param. Should be an integer")
			return false
		}

		if setting.State == setting.PP_INACTIVE {
			fmt.Println("The node is already inactive")
			return true
		}

		return event.Deactivate(fee, gas) == nil
	}

	prepay := func(line string, param []string) bool {
		if len(param) < 3 {
			fmt.Println("Expecting 3 params. Input amount of tokens, fee amount and gas amount")
			return false
		}
		amount, err := strconv.ParseInt(param[0], 10, 64)
		if err != nil {
			fmt.Println("Invalid amount param. Should be an integer")
			return false
		}
		fee, err := strconv.ParseInt(param[1], 10, 64)
		if err != nil {
			fmt.Println("Invalid fee param. Should be an integer")
			return false
		}
		gas, err := strconv.ParseInt(param[2], 10, 64)
		if err != nil {
			fmt.Println("Invalid gas param. Should be an integer")
			return false
		}

		return event.Prepay(amount, fee, gas) == nil
	}

	upload := func(line string, param []string) bool {
		if len(param) == 0 {
			fmt.Println("input upload file path")
			return false
		}
		pathStr := file.EscapePath(param)
		event.RequestUploadFile(pathStr, "", nil)
		return true
	}

	uploadStream := func(line string, param []string) bool {
		if len(param) == 0 {
			fmt.Println("input upload file path")
			return false
		}
		pathStr := file.EscapePath(param)
		event.RequestUploadStream(pathStr, "", nil)
		return true
	}

	list := func(line string, param []string) bool {
		if len(param) == 0 {
			event.FindMyFileList("", event.NowDir, "", "", 0, true, nil)

		} else {
			event.FindMyFileList(param[0], event.NowDir, "", "", 0, true, nil)
		}
		return true
	}

	download := func(line string, param []string) bool {
		if len(param) == 0 {
			fmt.Println("input download path, e.g: spb://account_address/file_hash|filename(optional)")
			return false
		}
		event.GetFileStorageInfo(param[0], "", "", false, false, nil)
		return true
	}

	deleteFn := func(line string, param []string) bool {
		if len(param) == 0 {
			fmt.Println("input file hash")
			return false
		}
		if len(param[0]) != setting.FILEHASHLEN {
			fmt.Println("input correct file hash")
		}
		event.DeleteFile(param[0], "", nil)
		return true
	}

	ver := func(line string, param []string) bool {
		fmt.Println("version:", setting.Config.VersionShow)
		return true
	}

	monitor := func(line string, param []string) bool {
		setting.ShowMonitor()
		return true
	}

	stopmonitor := func(line string, param []string) bool {
		setting.StopMonitor()
		return true
	}

	config := func(line string, param []string) bool {
		if len(param) < 2 {
			fmt.Println("input parameter name and value, 'name value' with space separator ")
			return false
		}
		setting.SetConfig(param[0], param[1])

		return true
	}

	mkdir := func(line string, param []string) bool {
		if len(param) == 0 {
			fmt.Println("input directory name")
			return false
		}
		event.MakeDirectory(param[0], "", nil)
		return true
	}

	rmdir := func(line string, param []string) bool {
		if len(param) == 0 {
			fmt.Println("input directory name")
			return false
		}
		event.RemoveDirectory(param[0], "", nil)
		return true
	}

	mvdir := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("input file hash and target directory path")
			return false
		}
		if len(param) == 1 { // root path
			event.MoveFileDirectory(param[0], event.NowDir, "", "", nil)
			return true
		}
		if event.NowDir == "" {
			event.MoveFileDirectory(param[0], "", param[1], "", nil)
		} else {
			event.MoveFileDirectory(param[0], event.NowDir, param[1], "", nil)
		}
		return true
	}

	cd := func(line string, param []string) bool {
		if len(param) < 1 {
			return false
		}
		event.Goto(param[0])
		return true
	}

	savefile := func(line string, param []string) bool {
		if len(param) < 2 {
			fmt.Println("input file hash and wallet address")
			return false
		}
		event.SaveOthersFile(param[0], param[1], "", nil)
		return true
	}

	sharepath := func(line string, param []string) bool {
		if len(param) < 3 {
			fmt.Println("input file hash and directory path, expire time(0 for non-expire), is private (0:public,1:private)")
			return false
		}
		time, timeErr := strconv.Atoi(param[1])
		if timeErr != nil {
			fmt.Println("input expire time(0 means non-expire)")
			return false
		}
		private, err := strconv.Atoi(param[2])
		if err != nil {
			fmt.Println("input is_private (0:public,1:private)")
			return false
		}
		isPrivate := false
		if private == 1 {
			isPrivate = true
		}
		// if len(str1) == setting.FILEHASHLEN { //
		// 	event.GetReqShareFile("", str1, "", int64(time), isPrivate, nil)
		// } else {
		event.GetReqShareFile("", "", param[0], int64(time), isPrivate, nil)
		// }
		return true
	}

	sharefile := func(line string, param []string) bool {
		if len(param) < 3 {
			fmt.Println("input file hash or directory path, expire time(0 for non-expire), is private (0:public,1:private)")
			return false
		}
		time, timeErr := strconv.Atoi(param[1])
		if timeErr != nil {
			fmt.Println("input expire time(0 for non-expire)")
			return false
		}
		private, err := strconv.Atoi(param[2])
		if err != nil {
			fmt.Println("input is private (0:public,1:private)")
			return false
		}
		isPrivate := false
		if private == 1 {
			isPrivate = true
		}
		event.GetReqShareFile("", param[0], "", int64(time), isPrivate, nil)
		// if len(str1) == setting.FILEHASHLEN { //
		// 	event.GetReqShareFile("", str1, "", int64(time), isPrivate, nil)
		// } else {
		// }
		return true
	}

	allshare := func(line string, param []string) bool {
		event.GetAllShareLink("", nil)
		return true
	}

	cancelshare := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("input share id")
			return false
		}
		event.DeleteShare(param[0], "", nil)
		return true
	}

	getsharefile := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("input share link and retrieval secret key(if any)")
			return false
		}
		if len(param) < 2 {
			event.GetShareFile(param[0], "", "", nil)
		} else {
			event.GetShareFile(param[0], param[1], "", nil)
		}

		return true
	}

	pauseget := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("input file hash of the pause")
			return false
		}
		event.DownloadSlicePause(param[0], "", nil)
		return true
	}

	pauseput := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("input file hash of the pause")
			return false
		}
		event.UploadPause(param[0], "", nil)
		return true
	}

	cancelget := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("input file hash of the cancel")
			return false
		}
		event.DownloadSliceCancel(param[0], "", nil)
		return true
	}

	createalbum := func(line string, param []string) bool {
		if len(param) < 5 {
			fmt.Println("input album name, album abstract, album cover hash, album type(0:movie,1:music,2:other), file hash(if multiple file, separate by comma)")
			return false
		}
		files := make([]*protos.FileInfo, 0)
		strs := strings.Split(param[4], ",")
		for i, val := range strs {
			t := &protos.FileInfo{
				FileHash: val,
				SortId:   uint64(i),
			}
			files = append(files, t)
		}
		event.CreateAlbum(param[0], param[1], param[2], param[3], "", files, false, nil)
		return true
	}

	albumlist := func(line string, param []string) bool {
		event.FindMyAlbum("", 0, 0, "", "", nil)
		return true
	}

	albumedit := func(line string, param []string) bool {
		// if len(param) < 3 {
		// 	fmt.Println("input album id, action(0:add,1:delete,2:update), file hash(if multiple file, separated by comma)")
		// 	return false
		// }
		// if param[1] == "2" { // edit
		// 	event.EditAlbum(param[0], param[1], param[2], param[3], param[4], "", nil, nil)
		// } else { // add
		// 	strs := strings.Split(param[2], ",")
		// 	event.EditAlbum(param[0], param[1], "", "", "", "", strs, nil)
		// }
		return true
	}

	albumcontent := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("input album id")
			return false
		}
		event.AlbumContent(param[0], "", nil)
		return true
	}

	albumsearch := func(line string, param []string) bool {

		if len(param) < 3 {
			fmt.Println("input keyword, album type(0:movie,1:music,2:other), sort type(0:newest, 1:hottest)")
			return false
		}
		event.SearchAlbum(param[0], param[1], param[2], "", 0, 0, nil)
		return true
	}

	invite := func(line string, param []string) bool {
		if len(param) < 1 {
			fmt.Println("input invite code")
			return false
		}
		event.Invite(param[0], "", nil)
		return true
	}

	reward := func(line string, param []string) bool {
		event.GetReward("", nil)
		return true
	}

	if setting.Config.AutoRun {
		AutoStart(setting.Config.WalletAddress, setting.Config.WalletPassword)
	}

	if setting.WalletAddress != "" && setting.Config.InternalPort != "" {
		go api.StartHTTPServ()
		peers.Login(setting.Config.WalletAddress, setting.Config.WalletPassword)
		// setting.ShowMonitor()
		go func() {
			netListen, err := net.Listen("tcp4", ":1203")
			if err != nil {
				utils.ErrorLog("p err", err)
			}
			// overChan := make(chan bool, 0)
			for {
				utils.DebugLog("!!!!!!!!!!!!!!!!!!")
				conn, err := netListen.Accept()
				if err != nil {
					utils.ErrorLog("Accept err", err)
				}
				utils.DebugLog(">>>>>>>>>>>>>>>>")
				go websocket.SocketRead(conn)
				go func() {
					for {
						writeErr := websocket.SocketStart(conn, setting.UpMap, setting.DownMap, setting.ResultMap)
						if writeErr != nil {
							return
						}
						time.Sleep(666 * time.Millisecond)
					}
				}()
			}
		}()
	}

	if setting.Config.RestPort != "" {
		go rest.StartHTTPServ()
	}

	console.Mystdin.RegisterProcessFunc("help", help)
	console.Mystdin.RegisterProcessFunc("h", help)
	console.Mystdin.RegisterProcessFunc("wallets", wallets)
	console.Mystdin.RegisterProcessFunc("newwallet", newWallet)
	console.Mystdin.RegisterProcessFunc("login", login)
	console.Mystdin.RegisterProcessFunc("start", start)
	console.Mystdin.RegisterProcessFunc("rp", registerPP)
	console.Mystdin.RegisterProcessFunc("registerminer", registerPP)
	console.Mystdin.RegisterProcessFunc("activate", activate)
	console.Mystdin.RegisterProcessFunc("deactivate", deactivate)
	console.Mystdin.RegisterProcessFunc("prepay", prepay)
	console.Mystdin.RegisterProcessFunc("u", upload)
	console.Mystdin.RegisterProcessFunc("put", upload)
	console.Mystdin.RegisterProcessFunc("putstream", uploadStream)
	console.Mystdin.RegisterProcessFunc("d", download)
	console.Mystdin.RegisterProcessFunc("get", download)
	console.Mystdin.RegisterProcessFunc("list", list)
	console.Mystdin.RegisterProcessFunc("ls", list)
	console.Mystdin.RegisterProcessFunc("delete", deleteFn)
	console.Mystdin.RegisterProcessFunc("rm", deleteFn)
	console.Mystdin.RegisterProcessFunc("ver", ver)
	console.Mystdin.RegisterProcessFunc("monitor", monitor)
	console.Mystdin.RegisterProcessFunc("stopmonitor", stopmonitor)

	console.Mystdin.RegisterProcessFunc("config", config)
	console.Mystdin.RegisterProcessFunc("mkdir", mkdir)
	console.Mystdin.RegisterProcessFunc("rmdir", rmdir)
	console.Mystdin.RegisterProcessFunc("mvdir", mvdir)
	console.Mystdin.RegisterProcessFunc("savefile", savefile)
	console.Mystdin.RegisterProcessFunc("cd", cd)
	console.Mystdin.RegisterProcessFunc("sharefile", sharefile)
	console.Mystdin.RegisterProcessFunc("sharepath", sharepath)
	console.Mystdin.RegisterProcessFunc("allshare", allshare)
	console.Mystdin.RegisterProcessFunc("cancelshare", cancelshare)
	console.Mystdin.RegisterProcessFunc("getsharefile", getsharefile)

	console.Mystdin.RegisterProcessFunc("createalbum", createalbum)
	console.Mystdin.RegisterProcessFunc("albumlist", albumlist)
	console.Mystdin.RegisterProcessFunc("albumedit", albumedit)
	console.Mystdin.RegisterProcessFunc("albumcontent", albumcontent)
	console.Mystdin.RegisterProcessFunc("albumsearch", albumsearch)

	console.Mystdin.RegisterProcessFunc("pauseget", pauseget)
	console.Mystdin.RegisterProcessFunc("pauseput", pauseput)
	console.Mystdin.RegisterProcessFunc("cancelget", cancelget)

	console.Mystdin.RegisterProcessFunc("invite", invite)
	console.Mystdin.RegisterProcessFunc("reward", reward)

	console.Mystdin.Run()
}

func terminalPreRunE(cmd *cobra.Command, args []string) error {
	err := nodePreRunE(cmd, args)
	peers.GetNetworkAddress()
	return err
}

// AutoStart
func AutoStart(account, password string) {
	if account == "" || password == "" {
		fmt.Println("add account and password in config")
		return
	}
	setting.IsAuto = true
	peers.Login(account, password)
}

