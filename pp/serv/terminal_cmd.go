package serv

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
)

const (
	DefaultMsg = "Request Accepted"
)

type CmdResult struct {
	Msg string
}

type terminalCmd struct {
}

func TerminalAPI() *terminalCmd {
	return &terminalCmd{}
}

func (api *terminalCmd) Wallets(param []string) (CmdResult, error) {
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	Wallets(ctx)
	return CmdResult{Msg: ""}, nil
}

func (api *terminalCmd) Getoz(param []string) (CmdResult, error) {
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	files, err := GetWallets(ctx, param[0], param[1])

	if err != nil {
		fmt.Println(err)
		return CmdResult{Msg: ""}, err
	}
	fileName := param[0] + ".json"
	for _, info := range files {
		if info.Name() == ".placeholder" || info.Name() != fileName {
			continue
		}
		pp.Log(ctx, "find file: "+filepath.Join(setting.Config.AccountDir, fileName))
		keyjson, err := ioutil.ReadFile(filepath.Join(setting.Config.AccountDir, fileName))
		if utils.CheckError(err) {
			pp.ErrorLog(ctx, "getPublicKey ioutil.ReadFile", err)
			fmt.Println(err)
			return CmdResult{Msg: ""}, err
		}
		_, err = utils.DecryptKey(keyjson, param[1])

		if utils.CheckError(err) {
			pp.ErrorLog(ctx, "getPublicKey DecryptKey", err)
			return CmdResult{Msg: ""}, err
		}
		if err := event.GetWalletOz(ctx, param[0], task.LOCAL_REQID); err != nil {
			return CmdResult{Msg: ""}, err
		}
		return CmdResult{Msg: DefaultMsg}, nil
	}

	pp.ErrorLogf(ctx, "Wallet %v does not exists", param[0])
	return CmdResult{Msg: ""}, err
}

func (api *terminalCmd) NewWallet(param []string) (CmdResult, error) {
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	CreateWallet(ctx, param[0], param[1], param[2], param[3])
	return CmdResult{Msg: ""}, nil
}

func (api *terminalCmd) Login(param []string) (CmdResult, error) {
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	err := Login(ctx, param[0], param[1])
	return CmdResult{Msg: ""}, err
}

func (api *terminalCmd) Start(param []string) (CmdResult, error) {
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	peers.StartMining(ctx)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) RegisterPP(param []string) (CmdResult, error) {
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	event.RegisterNewPP(ctx)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Activate(param []string) (CmdResult, error) {
	if len(param) != 3 {
		return CmdResult{Msg: ""}, errors.New("expecting 3 params. Input amount of tokens, fee amount and gas amount")
	}

	amount, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be an integer")
	}
	fee, err := strconv.ParseInt(param[1], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be an integer")
	}
	gas, err := strconv.ParseInt(param[2], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be an integer")
	}

	if setting.State != types.PP_INACTIVE {
		return CmdResult{Msg: "the pp is already active"}, nil
	}

	if !setting.IsPP {
		return CmdResult{Msg: "register as a PP node first"}, nil
	}

	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	if err := event.Activate(ctx, amount, fee, gas); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) UpdateStake(param []string) (CmdResult, error) {
	if len(param) != 4 {
		return CmdResult{Msg: ""}, errors.New("expecting 4 params. Input amount of stakeDelta, fee amount, " +
			"gas amount and flag of incrStake(0 for desc, 1 for incr)")
	}

	stakeDelta, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be an integer")
	}
	if stakeDelta < int64(setting.DEFAULT_MIN_UNSUSPEND_STAKE) {
		return CmdResult{Msg: ""}, errors.New("the minimum value to update stake is " + strconv.FormatInt(setting.DEFAULT_MIN_UNSUSPEND_STAKE, 10))
	}

	fee, err := strconv.ParseInt(param[1], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be an integer")
	}
	gas, err := strconv.ParseInt(param[2], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be an integer")
	}
	incrStake, err := strconv.ParseBool(param[3])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid flag for stake change. 0 for desc, 1 for incr")
	}

	if setting.State != types.PP_ACTIVE {
		//	//fmt.Println("PP node not activated yet")
		//	return CmdResult{Msg: "PP node not activated yet"}, nil
	}

	if !setting.IsPP {
		return CmdResult{Msg: "register as a PP node first"}, nil
	}

	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	if err := event.UpdateStake(ctx, stakeDelta, fee, gas, incrStake); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Status(param []string) (CmdResult, error) {
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	peers.GetPPStatusFromSP(ctx)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Deactivate(param []string) (CmdResult, error) {
	if len(param) != 2 {
		return CmdResult{Msg: ""}, errors.New("expecting 2 params. Input fee amount and gas amount")
	}

	fee, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be an integert")
	}
	gas, err := strconv.ParseInt(param[1], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be an integer")
	}

	if setting.State == types.PP_INACTIVE {
		return CmdResult{Msg: "The node is already inactive"}, nil
	}

	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	if err := event.Deactivate(ctx, fee, gas); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Prepay(param []string) (CmdResult, error) {
	if len(param) != 3 {
		return CmdResult{Msg: ""}, errors.New("expecting 3 params. Input amount of tokens, fee amount and gas amount")
	}

	amount, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be an integer")
	}
	fee, err := strconv.ParseInt(param[1], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be an integer")
	}
	gas, err := strconv.ParseInt(param[2], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be an integerr")
	}

	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	if err := event.Prepay(ctx, amount, fee, gas); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Upload(param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input upload file path")
	}
	isEncrypted := false
	if len(param) > 1 && param[1] == "encrypt" {
		isEncrypted = true
	}
	pathStr := file.EscapePath(param[0:1])

	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	event.RequestUploadFile(ctx, pathStr, "", isEncrypted, nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) UploadStream(param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input upload file path")
	}
	pathStr := file.EscapePath(param)
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	event.RequestUploadStream(ctx, pathStr, "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) BackupStatus(param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input file hash")
	}
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	event.ReqBackupStatus(ctx, param[0])
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) List(param []string) (CmdResult, error) {
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	if len(param) == 0 {
		event.FindFileList(ctx, "", setting.WalletAddress, 0, "", "", 0, true, nil)
	} else {
		pageId, err := strconv.ParseUint(param[0], 10, 64)
		if err == nil {
			event.FindFileList(ctx, "", setting.WalletAddress, pageId, "", "", 0, true, nil)
		} else {
			event.FindFileList(ctx, param[0], setting.WalletAddress, 0, "", "", 0, true, nil)
		}
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Download(param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input download path, e.g: sdm://account_address/file_hash|filename(optional)")
	}
	saveAs := ""
	if len(param) == 2 {
		saveAs = param[1]
	}
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	event.GetFileStorageInfo(ctx, param[0], "", task.LOCAL_REQID, saveAs, false, nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) DeleteFn(param []string) (CmdResult, error) {
	if len(param) == 0 {
		fmt.Println("input file hash")
		return CmdResult{}, errors.New("input file hash")
	}
	if !utils.VerifyHash(param[0]) {
		return CmdResult{}, errors.New("input correct file hash")
	}
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	event.DeleteFile(ctx, param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Ver(param []string) (CmdResult, error) {
	return CmdResult{Msg: fmt.Sprintf("version: %v", setting.Config.Version.Show)}, nil
}

func (api *terminalCmd) Monitor(param []string) (CmdResult, error) {
	ShowMonitor()
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) StopMonitor(param []string) (CmdResult, error) {
	StopMonitor()
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Config(param []string) (CmdResult, error) {
	if len(param) < 2 {
		return CmdResult{Msg: ""}, errors.New("input parameter name and value, 'name value' with space separator")
	}
	setting.SetConfig(param[0], param[1])

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) SharePath(param []string) (CmdResult, error) {
	if len(param) < 3 {
		return CmdResult{Msg: ""}, errors.New("input directory hash, share duration(in seconds, 0 for default value), is_private (0:public,1:private)")
	}
	time, timeErr := strconv.Atoi(param[1])
	if timeErr != nil {
		return CmdResult{Msg: ""}, errors.New("input share duration(in seconds, 0 for default value)")
	}
	private, err := strconv.Atoi(param[2])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("input is_private (0:public,1:private)")
	}
	isPrivate := false
	if private == 1 {
		isPrivate = true
	}
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	// if len(str1) == setting.FILEHASHLEN { //
	// 	event.GetReqShareFile("", str1, "", int64(time), isPrivate, nil)
	// } else {
	event.GetReqShareFile(ctx, "", "", param[0], setting.WalletAddress, int64(time), isPrivate, nil)
	// }
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) ShareFile(param []string) (CmdResult, error) {
	if len(param) < 3 {
		return CmdResult{Msg: ""}, errors.New("input file hash or directory path, share duration(in seconds, 0 for default value), is_private (0:public,1:private)")
	}
	time, timeErr := strconv.Atoi(param[1])
	if timeErr != nil {
		fmt.Println("input share duration(in seconds, 0 for default value)")
		return CmdResult{Msg: ""}, errors.New("input share duration(in seconds, 0 for default value)")
	}
	private, err := strconv.Atoi(param[2])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("input is private (0:public,1:private)")
	}
	isPrivate := false
	if private == 1 {
		isPrivate = true
	}
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	event.GetReqShareFile(ctx, "", param[0], "", setting.WalletAddress, int64(time), isPrivate, nil)
	// if len(str1) == setting.FILEHASHLEN { //
	// 	event.GetReqShareFile("", str1, "", int64(time), isPrivate, nil)
	// } else {
	// }
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) AllShare(param []string) (CmdResult, error) {
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	if len(param) < 1 {
		event.GetAllShareLink(ctx, "", setting.WalletAddress, 0, nil)
	} else {
		page, err := strconv.ParseUint(param[0], 10, 64)
		if err != nil {
			return CmdResult{Msg: ""}, errors.New("invalid page id.")
		}
		event.GetAllShareLink(ctx, "", setting.WalletAddress, page, nil)
	}

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) CancelShare(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input share id")
	}
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	event.DeleteShare(ctx, param[0], "", setting.WalletAddress, nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) GetShareFile(param []string) (CmdResult, error) {
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input share link and retrieval secret key(if any)")
	}
	if len(param) < 2 {
		event.GetShareFile(ctx, param[0], "", "", task.LOCAL_REQID, setting.WalletAddress, setting.WalletPublicKey, nil, nil)
	} else {
		event.GetShareFile(ctx, param[0], param[1], "", task.LOCAL_REQID, setting.WalletAddress, setting.WalletPublicKey, nil, nil)
	}

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) PauseGet(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the pause")
	}
	event.DownloadSlicePause(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) PausePut(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the pause")
	}
	ctx := pp.CreateReqIdAndRegisterRpcLogger(context.Background())
	event.UploadPause(ctx, param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) CancelGet(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the cancel")
	}
	event.DownloadSliceCancel(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) MonitorToken(param []string) (CmdResult, error) {
	utils.Log("Monitor token is:", GetCurrentToken())
	return CmdResult{Msg: DefaultMsg}, nil
}
