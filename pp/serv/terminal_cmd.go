package serv

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/account"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	"github.com/stratosnet/sds/pp/types"
	"github.com/stratosnet/sds/utils"
	utiltypes "github.com/stratosnet/sds/utils/types"
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

func (api *terminalCmd) Wallets(ctx context.Context, param []string) (CmdResult, error) {
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	account.Wallets(ctx)
	return CmdResult{Msg: ""}, nil
}

func (api *terminalCmd) Getoz(ctx context.Context, param []string) (CmdResult, error) {
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	files, err := account.GetWallets(ctx, param[0], param[1])

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

func (api *terminalCmd) NewWallet(ctx context.Context, param []string) (CmdResult, error) {
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	account.CreateWallet(ctx, param[0], param[1], param[2], param[3])
	return CmdResult{Msg: ""}, nil
}

func (api *terminalCmd) Start(ctx context.Context, param []string) (CmdResult, error) {
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	network.GetPeer(ctx).StartMining(ctx)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) RegisterPP(ctx context.Context, param []string) (CmdResult, error) {
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	event.RegisterNewPP(ctx)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Activate(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) < 2 {
		return CmdResult{Msg: ""}, errors.New("expecting at least 2 params. Input amount of tokens, fee amount and (optionally) gas amount")
	}

	amount, err := utiltypes.ParseCoinNormalized(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be a valid token")
	}

	fee, err := utiltypes.ParseCoinNormalized(param[1])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := utiltypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}
	if len(param) > 2 {
		gas, err := strconv.ParseUint(param[2], 10, 64)
		if err != nil {
			return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be a positive integer")
		}
		txFee.Gas = gas
		txFee.Simulate = false
	}

	if setting.State != types.PP_INACTIVE {
		return CmdResult{Msg: "the pp is already active"}, nil
	}

	if !setting.IsPP {
		return CmdResult{Msg: "register as a PP node first"}, nil
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	if err := event.Activate(ctx, amount, txFee); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) UpdateStake(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) < 3 {
		return CmdResult{Msg: ""}, errors.New("expecting at least 3 params. Input amount of stakeDelta, fee amount, " +
			"(optional) gas amount and flag of incrStake(0 for desc, 1 for incr)")
	}

	stakeDelta, err := utiltypes.ParseCoinNormalized(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be a valid token")
	}
	minUnsuspendStake, err := utiltypes.ParseCoinNormalized(setting.DEFAULT_MIN_UNSUSPEND_STAKE)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if stakeDelta.IsLT(minUnsuspendStake) {
		return CmdResult{Msg: ""}, errors.New("the minimum value to update stake is " + minUnsuspendStake.String())
	}

	fee, err := utiltypes.ParseCoinNormalized(param[1])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := utiltypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}
	lastParamIndex := 2
	if len(param) > 3 {
		lastParamIndex = 3
		gas, err := strconv.ParseUint(param[2], 10, 64)
		if err != nil {
			return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be a positive integer")
		}
		txFee.Gas = gas
		txFee.Simulate = false
	}

	incrStake, err := strconv.ParseBool(param[lastParamIndex])
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

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	if err := event.UpdateStake(ctx, stakeDelta, txFee, incrStake); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Status(ctx context.Context, param []string) (CmdResult, error) {
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	network.GetPeer(ctx).GetPPStatusFromSP(ctx)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Deactivate(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("expecting at least 1 param. Input fee amount and (optional) gas amount")
	}

	fee, err := utiltypes.ParseCoinNormalized(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := utiltypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}
	if len(param) > 1 {
		gas, err := strconv.ParseUint(param[1], 10, 64)
		if err != nil {
			return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be a positive integer")
		}
		txFee.Gas = gas
		txFee.Simulate = false
	}

	if setting.State == types.PP_INACTIVE {
		return CmdResult{Msg: "The node is already inactive"}, nil
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	if err := event.Deactivate(ctx, txFee); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Prepay(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) < 2 {
		return CmdResult{Msg: ""}, errors.New("expecting at least 2 params. Input amount of tokens, fee amount and (optional) gas amount")
	}

	amount, err := utiltypes.ParseCoinNormalized(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be a valid token" + err.Error())
	}
	fee, err := utiltypes.ParseCoinNormalized(param[1])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := utiltypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}
	if len(param) > 2 {
		gas, err := strconv.ParseUint(param[2], 10, 64)
		if err != nil {
			return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be a positive integer")
		}
		txFee.Gas = gas
		txFee.Simulate = false
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	if err := event.Prepay(ctx, amount, txFee); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Upload(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input upload file path")
	}
	isEncrypted := false
	if len(param) > 1 && param[1] == "encrypt" {
		isEncrypted = true
	}
	pathStr := file.EscapePath(param[0:1])

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	event.RequestUploadFile(ctx, pathStr, isEncrypted, nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) UploadStream(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input upload file path")
	}
	pathStr := file.EscapePath(param)
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	ctx = core.RegisterRemoteReqId(ctx, uuid.New().String())
	event.RequestUploadStream(ctx, pathStr, nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) BackupStatus(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input file hash")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	event.ReqBackupStatus(ctx, param[0])
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) List(ctx context.Context, param []string) (CmdResult, error) {
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	if len(param) == 0 {
		event.FindFileList(ctx, "", setting.WalletAddress, 0, "", 0, true)
	} else {
		pageId, err := strconv.ParseUint(param[0], 10, 64)
		if err == nil {
			event.FindFileList(ctx, "", setting.WalletAddress, pageId, "", 0, true)
		} else {
			event.FindFileList(ctx, param[0], setting.WalletAddress, 0, "", 0, true)
		}
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Download(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input download path, e.g: sdm://account_address/file_hash|filename(optional)")
	}
	saveAs := ""
	if len(param) == 2 {
		saveAs = param[1]
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	core.RegisterReqId(ctx, task.LOCAL_REQID)
	event.GetFileStorageInfo(ctx, param[0], "", saveAs, false, nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) DeleteFn(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) == 0 {
		fmt.Println("input file hash")
		return CmdResult{}, errors.New("input file hash")
	}
	if !utils.VerifyHash(param[0]) {
		return CmdResult{}, errors.New("input correct file hash")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	event.DeleteFile(ctx, param[0])
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Ver(ctx context.Context, param []string) (CmdResult, error) {
	return CmdResult{Msg: fmt.Sprintf("version: %v", setting.Config.Version.Show)}, nil
}

func (api *terminalCmd) Monitor(ctx context.Context, param []string) (CmdResult, error) {
	ShowMonitor(ctx)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) StopMonitor(ctx context.Context, param []string) (CmdResult, error) {
	StopMonitor()
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Config(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) < 2 {
		return CmdResult{Msg: ""}, errors.New("input parameter name and value, 'name value' with space separator")
	}
	setting.SetConfig(param[0], param[1])

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) SharePath(ctx context.Context, param []string) (CmdResult, error) {
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
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	// if len(str1) == setting.FILEHASHLEN { //
	// 	event.GetReqShareFile("", str1, "", int64(time), isPrivate, nil)
	// } else {
	event.GetReqShareFile(ctx, "", param[0], setting.WalletAddress, int64(time), isPrivate)
	// }
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) ShareFile(ctx context.Context, param []string) (CmdResult, error) {
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
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	event.GetReqShareFile(ctx, param[0], "", setting.WalletAddress, int64(time), isPrivate)
	// if len(str1) == setting.FILEHASHLEN { //
	// 	event.GetReqShareFile("", str1, "", int64(time), isPrivate, nil)
	// } else {
	// }
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) AllShare(ctx context.Context, param []string) (CmdResult, error) {
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	if len(param) < 1 {
		event.GetAllShareLink(ctx, setting.WalletAddress, 0)
	} else {
		page, err := strconv.ParseUint(param[0], 10, 64)
		if err != nil {
			return CmdResult{Msg: ""}, errors.New("invalid page id.")
		}
		event.GetAllShareLink(ctx, setting.WalletAddress, page)
	}

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) CancelShare(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input share id")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	event.DeleteShare(ctx, param[0], setting.WalletAddress)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) GetShareFile(ctx context.Context, param []string) (CmdResult, error) {
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	core.RegisterReqId(ctx, task.LOCAL_REQID)

	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input share link and retrieval secret key(if any)")
	}
	if len(param) < 2 {
		event.GetShareFile(ctx, param[0], "", "", setting.WalletAddress, setting.WalletPublicKey)
	} else {
		event.GetShareFile(ctx, param[0], param[1], "", setting.WalletAddress, setting.WalletPublicKey)
	}

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) PauseGet(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the pause")
	}
	event.DownloadSlicePause(ctx, param[0], "")
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) PausePut(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the pause")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	event.UploadPause(ctx, param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) CancelGet(ctx context.Context, param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the cancel")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	event.DownloadSliceCancel(ctx, param[0], "")
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) MonitorToken(ctx context.Context, param []string) (CmdResult, error) {
	utils.Log("Monitor token is:", GetCurrentToken())
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Maintenance(ctx context.Context, param []string) (CmdResult, error) {
	// Parse params
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("first parameter should be either 'start' or 'stop'")
	}
	start := true
	if param[0] == "stop" {
		start = false
	} else if param[0] != "start" {
		return CmdResult{Msg: ""}, errors.New("first parameter should be either 'start' or 'stop'")
	}

	duration := uint64(0)
	if start {
		if len(param) < 2 {
			return CmdResult{Msg: ""}, errors.New("second parameter should be the maintenance duration (in seconds)")
		}
		parsedDuration, err := strconv.ParseUint(param[1], 10, 64)
		if err != nil {
			return CmdResult{Msg: ""}, errors.New("second parameter should be the maintenance duration (in seconds)")
		}
		duration = parsedDuration
	}

	// Execute request
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx)
	if start {
		err := event.StartMaintenance(ctx, duration)
		if err != nil {
			return CmdResult{Msg: ""}, err
		}
	} else {
		err := event.StopMaintenance(ctx)
		if err != nil {
			return CmdResult{Msg: ""}, err
		}
	}
	return CmdResult{Msg: DefaultMsg}, nil
}
