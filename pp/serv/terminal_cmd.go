package serv

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/framework/crypto"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	msgtypes "github.com/stratosnet/sds/sds-msg/types"
	msgutils "github.com/stratosnet/sds/sds-msg/utils"
	txclienttypes "github.com/stratosnet/sds/tx-client/types"

	"github.com/stratosnet/sds/pp"
	"github.com/stratosnet/sds/pp/account"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/metrics"
	"github.com/stratosnet/sds/pp/namespace/stratoschain"
	"github.com/stratosnet/sds/pp/network"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/task"
	pptypes "github.com/stratosnet/sds/pp/types"
)

const (
	DefaultMsg               = "Request Accepted"
	DefaultDesiredUploadTier = 2
)

type CmdResult struct {
	Msg string
}

type terminalCmd struct {
}

func TerminalAPI() *terminalCmd {
	return &terminalCmd{}
}

func getTerminalIdFromParam(paramWithTerminalId []string) (terminalId string, param []string, err error) {
	if len(paramWithTerminalId) == 0 {
		err = errors.New("params is empty")
		return
	}
	terminalId = paramWithTerminalId[0]
	if len(paramWithTerminalId) > 1 {
		param = paramWithTerminalId[1:]
	}
	return
}

func (api *terminalCmd) Wallets(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, _, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	account.Wallets(ctx)
	return CmdResult{Msg: ""}, nil
}

func (api *terminalCmd) Getoz(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	if _, err := fwtypes.WalletAddressFromBech32(param[0]); err != nil {
		return CmdResult{Msg: ""}, err
	}

	if err := event.GetWalletOz(ctx, param[0], task.LOCAL_REQID); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) NewWallet(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	account.CreateWallet(ctx, param[0], param[1], param[2], param[3])
	return CmdResult{Msg: ""}, nil
}

func (api *terminalCmd) StartMining(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, _, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	switch state := network.GetPeer(ctx).GetStateFromFsm(); state.Id {
	case network.STATE_NOT_REGISTERED:
	case network.STATE_SUSPENDED:
	case network.STATE_OFFLINE:
	case network.STATE_NOT_ACTIVATED, network.STATE_INIT, network.STATE_NOT_CREATED:
		return CmdResult{Msg: ""}, errors.New("register and activate the node before start mining")
	default:
		return CmdResult{Msg: ""}, errors.New("mining already started")
	}

	network.GetPeer(ctx).RunFsm(ctx, network.EVENT_START_MINING)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) RegisterPP(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, _, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	nowSec := time.Now().Unix()
	// sign the wallet signature by wallet private key
	wsignMsg := msgutils.RegisterNewPPWalletSignMessage(setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("wallet failed to sign message")
	}
	event.RegisterNewPP(ctx, setting.WalletAddress, setting.WalletPublicKey.Bytes(), wsign, nowSec)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Activate(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 2 {
		return CmdResult{Msg: ""}, errors.New("expecting at least 2 params. Input amount of tokens, fee amount and (optionally) --gas")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	amount, err := txclienttypes.ParseCoinNormalized(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be a valid token")
	}

	fee, err := txclienttypes.ParseCoinNormalized(param[1])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := txclienttypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}

	var gas uint64

	if len(param) > 2 {
		for _, p := range param[2:] {
			if !strings.Contains(p, "=") {
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", p)
			}

			kv := strings.SplitN(p, "=", 2)
			switch kv[0] {
			case "--gas":
				gas, err = strconv.ParseUint(kv[1], 10, 64)
				if err != nil {
					return CmdResult{Msg: ""}, errors.New("invalid param --gas. Should be a positive integer")
				}
				txFee.Gas = gas
			default:
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", kv[0])
			}
		}
	}

	switch state := network.GetPeer(ctx).GetStateFromFsm(); state.Id {
	case network.STATE_NOT_ACTIVATED:
		break
	case network.STATE_NOT_CREATED:
		return CmdResult{Msg: "register as a PP node first"}, nil
	default:
		return CmdResult{Msg: "the pp is already active"}, nil
	}

	if err := event.Activate(ctx, amount, txFee); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) UpdateDeposit(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 2 {
		return CmdResult{Msg: ""}, errors.New("expecting at least 2 params. Input amount of depositDelta, fee amount, " +
			"(optional) --gas")
	}

	depositDelta, err := txclienttypes.ParseCoinNormalized(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be a valid token")
	}
	minUnsuspendDeposit, err := txclienttypes.ParseCoinNormalized(setting.DefaultMinUnsuspendDeposit)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if depositDelta.IsLT(minUnsuspendDeposit) {
		return CmdResult{Msg: ""}, errors.New("the minimum value to update deposit is " + minUnsuspendDeposit.String())
	}

	fee, err := txclienttypes.ParseCoinNormalized(param[1])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := txclienttypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}

	var gas uint64

	if len(param) > 2 {
		for _, p := range param[2:] {
			if !strings.Contains(p, "=") {
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", p)
			}

			kv := strings.SplitN(p, "=", 2)
			switch kv[0] {
			case "--gas":
				gas, err = strconv.ParseUint(kv[1], 10, 64)
				if err != nil {
					return CmdResult{Msg: ""}, errors.New("invalid param --gas. Should be a positive integer")
				}
				txFee.Gas = gas
			default:
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", kv[0])
			}
		}
	}

	if !setting.IsPP {
		return CmdResult{Msg: "register as a PP node first"}, nil
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	if err := event.UpdateDeposit(ctx, depositDelta, txFee); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Status(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, _, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	//get status from cache
	cachedStatus := event.GetPPStatusCache()
	if cachedStatus != nil {
		statusMsg := event.FormatPPStatusInfo(ctx, cachedStatus, true)
		return CmdResult{Msg: statusMsg}, nil
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	network.GetPeer(ctx).GetPPStatusFromSP(ctx)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) FileStatus(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("expecting at least 1 param. Input filehash")
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	fileHash := param[0]
	timestamp := time.Now().Unix()

	signature, err := setting.WalletPrivateKey.Sign([]byte(msgutils.GetFileStatusWalletSignMessage(fileHash, setting.WalletAddress, timestamp)))
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if rsp := event.GetFileStatus(ctx, fileHash, setting.WalletAddress, setting.WalletPublicKey.Bytes(), signature, timestamp); rsp != nil {
		// Result is available now. Otherwise, it will be logged when RspFileStatus event is received
		if bytes, err := json.Marshal(rsp); err == nil {
			pp.Logf(ctx, "File status result: %v", string(bytes))
		}
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Deactivate(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("expecting at least 1 param. Input fee amount and (optional) --gas")
	}

	fee, err := txclienttypes.ParseCoinNormalized(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := txclienttypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}
	var gas uint64

	if len(param) > 1 {
		for _, p := range param[1:] {
			if !strings.Contains(p, "=") {
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", p)
			}

			kv := strings.SplitN(p, "=", 2)
			switch kv[0] {
			case "--gas":
				gas, err = strconv.ParseUint(kv[1], 10, 64)
				if err != nil {
					return CmdResult{Msg: ""}, errors.New("invalid param --gas. Should be a positive integer")
				}
				txFee.Gas = gas
			default:
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", kv[0])
			}
		}
	}

	if setting.State == msgtypes.PP_INACTIVE {
		return CmdResult{Msg: "The node is already inactive"}, nil
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	if err := event.Deactivate(ctx, txFee); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Prepay(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 2 {
		return CmdResult{Msg: ""},
			errors.New("expecting at least 2 params. Input amount of tokens, fee amount, (optional) --beneficiary, and (optional) --gas")
	}

	amount, err := txclienttypes.ParseCoinNormalized(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be a valid token" + err.Error())
	}

	fee, err := txclienttypes.ParseCoinNormalized(param[1])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := txclienttypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}

	// use wallet address as default beneficiary address
	beneficiaryAddr, _ := fwtypes.WalletAddressFromBech32(setting.WalletAddress)
	var gas uint64

	if len(param) > 2 {
		for _, p := range param[2:] {
			if !strings.Contains(p, "=") {
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", p)
			}

			kv := strings.SplitN(p, "=", 2)
			switch kv[0] {
			case "--beneficiary":
				beneficiaryAddr, err = fwtypes.WalletAddressFromBech32(kv[1])
				if err != nil {
					return CmdResult{Msg: ""}, errors.New("invalid param --beneficiary. Should be a valid wallet address" + err.Error())
				}
			case "--gas":
				gas, err = strconv.ParseUint(kv[1], 10, 64)
				if err != nil {
					return CmdResult{Msg: ""}, errors.New("invalid param --gas. Should be a positive integer")
				}
				txFee.Gas = gas
			default:
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", kv[0])
			}
		}
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	nowSec := time.Now().Unix()
	// sign the wallet signature by wallet private key
	wsignMsg := msgutils.PrepayWalletSignMessage(setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("wallet failed to sign message")
	}
	if err := event.Prepay(ctx, beneficiaryAddr, amount, txFee,
		setting.WalletAddress, setting.WalletPublicKey.Bytes(), wsign, nowSec); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) validateUploadPath(pathStr string) error {
	if strings.HasSuffix(pathStr, "/.") {
		return errors.New("the input path is not allowed")
	}
	if strings.HasSuffix(pathStr, "/..") {
		return errors.New("the input path is not allowed")
	}
	if strings.HasPrefix(pathStr, "/etc") {
		return errors.New("files in system folders are not permitted to upload")
	}
	if strings.HasPrefix(pathStr, "/boot") {
		return errors.New("files in system folders are not permitted to upload")
	}
	return nil
}

func (api *terminalCmd) Upload(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) == 0 {
		return CmdResult{}, errors.New("input upload file path")
	}

	pathStr := file.EscapePath(param[0:1])
	if err = api.validateUploadPath(pathStr); err != nil {
		return CmdResult{}, err
	}

	isEncrypted := false
	desiredTier := uint32(DefaultDesiredUploadTier)
	allowHigherTier := true

	if len(param) > 1 {
		for _, p := range param[1:] {
			if !strings.Contains(p, "=") {
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", p)
			}

			kv := strings.SplitN(p, "=", 2)
			switch kv[0] {
			case "--isEncrypted":
				isEncrypted, err = strconv.ParseBool(kv[1])
				if err != nil {
					return CmdResult{Msg: ""}, errors.Errorf("invalid param --isEncrypted. Should be true or false: %v ", err.Error())
				}
			case "--nodeTier":
				tier, err := strconv.ParseUint(kv[1], 10, 32)
				if err != nil {
					return CmdResult{Msg: ""}, errors.Errorf("invalid param --nodeTier. Should be an integer: %v ", err.Error())
				}
				if tier <= utils.PpMinTier || tier > utils.PpMaxTier {
					return CmdResult{Msg: ""}, errors.New("invalid param --nodeTier. Should be between 1 and 3")
				}
				desiredTier = uint32(tier)
			case "--allowHigherTier":
				allowHigherTier, err = strconv.ParseBool(kv[1])
				if err != nil {
					return CmdResult{Msg: ""}, errors.Errorf("invalid param --allowHigherTier. Should be true or false: %v ", err.Error())
				}
			default:
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", kv[0])
			}
		}
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	event.RequestUploadFile(ctx, pathStr, isEncrypted, false, desiredTier, allowHigherTier,
		setting.WalletAddress, setting.WalletPublicKey.Bytes(), nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) UploadStream(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) == 0 {
		return CmdResult{}, errors.New("input upload file path")
	}
	pathStr := file.EscapePath(param)
	if err = api.validateUploadPath(pathStr); err != nil {
		return CmdResult{}, err
	}

	desiredTier := uint32(DefaultDesiredUploadTier)
	allowHigherTier := true

	if len(param) > 1 {
		for _, p := range param[1:] {
			if !strings.Contains(p, "=") {
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", p)
			}

			kv := strings.SplitN(p, "=", 2)
			switch kv[0] {
			case "--nodeTier":
				tier, err := strconv.ParseUint(kv[1], 10, 32)
				if err != nil {
					return CmdResult{Msg: ""}, errors.Errorf("invalid param --nodeTier. Should be an integer: %v ", err.Error())
				}
				if tier <= utils.PpMinTier || tier > utils.PpMaxTier {
					return CmdResult{Msg: ""}, errors.New("invalid param --nodeTier. Should be between 1 and 3")
				}
				desiredTier = uint32(tier)
			case "--allowHigherTier":
				allowHigherTier, err = strconv.ParseBool(kv[1])
				if err != nil {
					return CmdResult{Msg: ""}, errors.Errorf("invalid param --allowHigherTier. Should be true or false: %v ", err.Error())
				}
			default:
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", kv[0])
			}
		}
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	ctx = core.RegisterRemoteReqId(ctx, uuid.New().String())
	event.RequestUploadFile(ctx, pathStr, false, true, desiredTier, allowHigherTier,
		setting.WalletAddress, setting.WalletPublicKey.Bytes(), nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) BackupStatus(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	if len(param) == 0 {
		return CmdResult{}, errors.New("input file hash")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	event.ReqBackupStatus(ctx, param[0])
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) List(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	nowSec := time.Now().Unix()
	// sign the wallet signature by wallet private key
	wsignMsg := msgutils.FindMyFileListWalletSignMessage(setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("wallet failed to sign message")
	}

	if len(param) == 0 {
		event.FindFileList(ctx, "", setting.WalletAddress, 0, "", 0, true,
			setting.WalletPublicKey.Bytes(), wsign, nowSec)
	} else {
		pageId, err := strconv.ParseUint(param[0], 10, 64)
		if err == nil {
			event.FindFileList(ctx, "", setting.WalletAddress, pageId, "", 0, true,
				setting.WalletPublicKey.Bytes(), wsign, nowSec)
		} else {
			event.FindFileList(ctx, param[0], setting.WalletAddress, 0, "", 0, true,
				setting.WalletPublicKey.Bytes(), wsign, nowSec)
		}
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) ClearExpShare(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	if len(param) > 0 {
		return CmdResult{Msg: ""}, errors.New("invalid count for params")
	}
	nowSec := time.Now().Unix()
	// sign the wallet signature by wallet private key
	wsignMsg := msgutils.ClearExpiredShareLinksWalletSignMessage(setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("wallet failed to sign message")
	}
	event.ClearExpiredShareLinks(ctx, setting.WalletAddress, setting.WalletPublicKey.Bytes(), wsign, nowSec)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Download(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) == 0 {
		return CmdResult{}, errors.New("input download path, e.g: sdm://account_address/file_hash|filename(optional)")
	}
	saveAs := ""
	if len(param) == 2 {
		saveAs = param[1]
	}

	_, ownerWalletAddress, fileHash, _, err := fwtypes.ParseFileHandle(param[0])
	if err != nil {
		err = errors.New("wrong file path format, failed to parse")
		return CmdResult{Msg: ""}, err
	}
	if ownerWalletAddress != setting.WalletAddress {
		err = errors.New("only the file owner is allowed to download via sdm url")
		return CmdResult{Msg: ""}, err
	}
	if crypto.IsVideoStream(fileHash) {
		err = errors.New("video stream file cannot be downloaded by get cmd")
		return CmdResult{Msg: ""}, err
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	core.RegisterReqId(ctx, task.LOCAL_REQID)
	nowSec := time.Now().Unix()
	if task.CheckDownloadTask(fileHash, setting.WalletAddress, task.LOCAL_REQID) {
		return CmdResult{Msg: ""}, errors.New("* This file is being downloaded, please wait and try later")
	}

	req := requests.ReqFileStorageInfoData(ctx, param[0], "", saveAs, setting.WalletAddress, setting.WalletPublicKey.Bytes(), nil, nil, nowSec)
	if err := event.ReqGetWalletOzForDownload(ctx, setting.WalletAddress, task.LOCAL_REQID, req); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) DeleteFn(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) == 0 {
		fmt.Println("input file hash")
		return CmdResult{}, errors.New("input file hash")
	}
	if !crypto.ValidateHash(param[0]) {
		return CmdResult{}, errors.New("input correct file hash")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	nowSec := time.Now().Unix()
	fileHash := param[0]
	// sign the wallet signature by wallet private key
	wsignMsg := msgutils.DeleteFileWalletSignMessage(fileHash, setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("wallet failed to sign message")
	}
	event.DeleteFile(ctx, param[0], setting.WalletAddress, setting.WalletPublicKey.Bytes(), wsign, nowSec)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Ver(_ context.Context, _ []string) (CmdResult, error) {
	return CmdResult{Msg: fmt.Sprintf("version: %v", setting.Config.Version.Show)}, nil
}

func (api *terminalCmd) Monitor(ctx context.Context, _ []string) (CmdResult, error) {
	ShowMonitor(ctx)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) StopMonitor(_ context.Context, _ []string) (CmdResult, error) {
	StopMonitor()
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Config(ctx context.Context, param []string) (CmdResult, error) {
	_, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 2 {
		return CmdResult{}, errors.New("input parameter name and value, 'name value' with space separator")
	}

	value, err := utils.ParseTomlValue(param[1])
	if err != nil {
		return CmdResult{}, err
	}

	err = setting.SetConfig(param[0], value)
	if err != nil {
		return CmdResult{}, err
	}

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) SharePath(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 3 {
		return CmdResult{Msg: ""}, errors.New("input directory hash, share duration(in seconds, 0 for default value), is_private (0:public,1:private)")
	}
	shareDuration, timeErr := strconv.Atoi(param[1])
	if timeErr != nil || shareDuration < 0 {
		msg := fmt.Sprintf(
			"%v isn't a valid parameter for share duration in seconds, please specify a non-negative integer, "+
				"0 for default share duration",
			param[1])
		return CmdResult{Msg: ""}, errors.New(msg)
	}
	private, err := strconv.Atoi(param[2])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("input is_private (0:public,1:private)")
	}
	isPrivate := false
	if private == 1 {
		isPrivate = true
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	// if len(str1) == setting.FILEHASHLEN { //
	// 	event.GetReqShareFile("", str1, "", int64(time), isPrivate, nil)
	// } else {
	nowSec := time.Now().Unix()
	if !crypto.ValidateHash(param[0]) {
		return CmdResult{}, errors.New("input correct file hash")
	}
	fileHash := param[0]
	// sign the wallet signature by wallet private key
	wsignMsg := msgutils.ShareFileWalletSignMessage(fileHash, setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("wallet failed to sign message")
	}
	event.GetReqShareFile(ctx, "", param[0], setting.WalletAddress, int64(shareDuration), isPrivate,
		setting.WalletPublicKey.Bytes(), wsign, nowSec)
	// }
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) ShareFile(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 3 {
		return CmdResult{Msg: ""}, errors.New("input file hash or directory path, share duration(in seconds, 0 for default value), is_private (0:public,1:private)")
	}
	fileHash := param[0]
	if !crypto.ValidateHash(param[0]) {
		return CmdResult{}, errors.New("input correct file hash")
	}

	shareDuration, timeErr := strconv.Atoi(param[1])
	if timeErr != nil || shareDuration < 0 {
		msg := fmt.Sprintf(
			"%v isn't a valid parameter for share duration in seconds, please specify a non-negative integer, 0 for default share duration",
			param[1])
		fmt.Println(msg)
		return CmdResult{Msg: ""}, errors.New(msg)
	}
	private, err := strconv.Atoi(param[2])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("input is private (0:public,1:private)")
	}
	isPrivate := false
	if private == 1 {
		isPrivate = true
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	nowSec := time.Now().Unix()
	// sign the wallet signature by wallet private key
	wsignMsg := msgutils.ShareFileWalletSignMessage(fileHash, setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("wallet failed to sign message")
	}
	event.GetReqShareFile(ctx, param[0], "", setting.WalletAddress, int64(shareDuration), isPrivate,
		setting.WalletPublicKey.Bytes(), wsign, nowSec)
	// if len(str1) == setting.FILEHASHLEN { //
	// 	event.GetReqShareFile("", str1, "", int64(time), isPrivate, nil)
	// } else {
	// }
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) AllShare(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	// sign the wallet signature by wallet private key
	nowSec := time.Now().Unix()
	wsignMsg := msgutils.ShareLinkWalletSignMessage(setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("wallet failed to sign message")
	}
	if len(param) < 1 {
		event.GetAllShareLink(ctx, setting.WalletAddress, 0, setting.WalletPublicKey.Bytes(), wsign, nowSec)
	} else {
		page, err := strconv.ParseUint(param[0], 10, 64)
		if err != nil {
			return CmdResult{Msg: ""}, errors.New("invalid page id.")
		}
		event.GetAllShareLink(ctx, setting.WalletAddress, page, setting.WalletPublicKey.Bytes(), wsign, nowSec)
	}

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) CancelShare(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input share id")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	nowSec := time.Now().Unix()
	shareId := param[0]
	// sign the wallet signature by wallet private key
	wsignMsg := msgutils.DeleteShareWalletSignMessage(shareId, setting.WalletAddress, nowSec)
	wsign, err := setting.WalletPrivateKey.Sign([]byte(wsignMsg))
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("wallet failed to sign message")
	}
	event.DeleteShare(ctx, param[0], setting.WalletAddress, setting.WalletPublicKey.Bytes(), wsign, nowSec)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) GetShareFile(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	core.RegisterReqId(ctx, task.LOCAL_REQID)

	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input share link and retrieval secret key(if any)")
	}

	nowSec := time.Now().Unix()
	shareLink, err := pptypes.ParseShareLink(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	event.GetShareFile(ctx, shareLink.ShareLink, shareLink.Password, "", setting.WalletAddress, setting.WalletPublicKey.Bytes(), nil, nowSec)

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) PauseGet(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the pause")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	event.DownloadSlicePause(ctx, param[0], "")
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) PausePut(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the pause")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	event.UploadPause(ctx, param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) CancelGet(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the cancel")
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	event.DownloadSliceCancel(ctx, param[0], "")
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) MonitorToken(_ context.Context, _ []string) (CmdResult, error) {
	utils.Log("Monitor token is:", GetCurrentToken())
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Maintenance(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

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
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
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

func (api *terminalCmd) Replica(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) == 0 {
		return CmdResult{}, errors.New("input file path, e.g: sdm://account_address/file_hash|filename(optional)")
	}
	replicaIncreaseNum := uint32(0)
	if len(param) == 2 {
		ui64, err := strconv.ParseUint(param[1], 10, 64)
		if err != nil {
			return CmdResult{Msg: ""}, errors.New("failed to parse the increase number")
		}
		fmt.Println(ui64, reflect.TypeOf(ui64))

		replicaIncreaseNum = uint32(ui64)
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	core.RegisterReqId(ctx, task.LOCAL_REQID)
	event.GetFileReplicaInfo(ctx, param[0], replicaIncreaseNum)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) DowngradeInfo(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, _, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)
	// Parse params
	err = event.ReqGetPPDowngradeInfo(ctx)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) PerformanceMeasure(_ context.Context, _ []string) (CmdResult, error) {
	// Parse params
	metrics.StartLoggingPerformanceData()
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Withdraw(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 2 {
		return CmdResult{Msg: ""},
			errors.New("expecting at least 2 params. Input amount of tokens, fee amount, (optional) --targetAddr, and (optional) --gas")
	}

	amount, err := txclienttypes.ParseCoinNormalized(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be a valid token")
	}

	fee, err := txclienttypes.ParseCoinNormalized(param[1])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := txclienttypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}

	// use wallet address as default target address
	targetAddr, _ := fwtypes.WalletAddressFromBech32(setting.WalletAddress)
	var gas uint64

	if len(param) > 2 {
		for _, p := range param[2:] {
			if !strings.Contains(p, "=") {
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", p)
			}

			kv := strings.SplitN(p, "=", 2)
			switch kv[0] {
			case "--targetAddr":
				targetAddr, err = fwtypes.WalletAddressFromBech32(kv[1])
				if err != nil {
					return CmdResult{Msg: ""}, errors.New("invalid param --targetAddr. Should be a valid wallet address" + err.Error())
				}
			case "--gas":
				gas, err = strconv.ParseUint(kv[1], 10, 64)
				if err != nil {
					return CmdResult{Msg: ""}, errors.New("invalid param --gas. Should be a positive integer")
				}
				txFee.Gas = gas
			default:
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", kv[0])
			}
		}
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	if err = stratoschain.Withdraw(ctx, amount, targetAddr, txFee); err != nil {
		return CmdResult{Msg: ""}, err
	}

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Send(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 3 {
		return CmdResult{Msg: ""},
			errors.New("expecting at least 3 params. Input amount of tokens, to address, fee amount,and (optional) --gas")
	}

	toAddr, err := fwtypes.WalletAddressFromBech32(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid to param. Should be a valid bech32 wallet address" + err.Error())
	}

	amount, err := txclienttypes.ParseCoinNormalized(param[1])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be a valid token")
	}

	fee, err := txclienttypes.ParseCoinNormalized(param[2])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := txclienttypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}

	var gas uint64

	if len(param) > 3 {
		for _, p := range param[3:] {
			if !strings.Contains(p, "=") {
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", p)
			}

			kv := strings.SplitN(p, "=", 2)
			switch kv[0] {
			case "--gas":
				gas, err = strconv.ParseUint(kv[1], 10, 64)
				if err != nil {
					return CmdResult{Msg: ""}, errors.New("invalid param --gas. Should be a positive integer")
				}
				txFee.Gas = gas
			default:
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", kv[0])
			}
		}
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	if err = stratoschain.Send(ctx, amount, toAddr, txFee); err != nil {
		return CmdResult{Msg: ""}, err
	}

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) UpdateInfo(ctx context.Context, param []string) (CmdResult, error) {
	terminalId, param, err := getTerminalIdFromParam(param)
	if err != nil {
		return CmdResult{Msg: ""}, err
	}

	if len(param) < 1 {
		return CmdResult{Msg: ""},
			errors.New("expecting at least 1 param. Input fee amount and other optional params")
	}

	fee, err := txclienttypes.ParseCoinNormalized(param[0])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be a valid token")
	}
	txFee := txclienttypes.TxFee{
		Fee:      fee,
		Simulate: true,
	}

	moniker := ""
	identity := ""
	website := ""
	securityContact := ""
	details := ""
	var gas uint64

	if len(param) > 1 {
		for _, p := range param[1:] {
			if !strings.Contains(p, "=") {
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", p)
			}

			kv := strings.SplitN(p, "=", 2)
			switch kv[0] {
			case "--moniker":
				moniker = kv[1]
			case "--identity":
				identity = kv[1]
			case "--website":
				website = kv[1]
			case "--security_contact":
				securityContact = kv[1]
			case "--details":
				details = kv[1]
			case "--gas":
				gas, err = strconv.ParseUint(kv[1], 10, 64)
				if err != nil {
					return CmdResult{Msg: ""}, errors.New("invalid param --gas. Should be a positive integer")
				}
				txFee.Gas = gas
			default:
				return CmdResult{Msg: ""}, errors.Errorf("invalid param %v.", kv[0])
			}
		}
	}

	ctx = pp.CreateReqIdAndRegisterRpcLogger(ctx, terminalId)

	if err = stratoschain.UpdateResourceNode(ctx, moniker, identity, website, securityContact, details, txFee); err != nil {
		return CmdResult{Msg: ""}, err
	}

	return CmdResult{Msg: DefaultMsg}, nil
}
