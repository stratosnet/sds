package serv

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	cmds "github.com/ipfs/go-ipfs-cmds"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/pp/api/ipfsrpc"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
)

type CtxEnv struct {
	Ctx context.Context
}

const (
	IPFS_WAIT_TIMEOUT_ADD  = 15 * time.Second
	IPFS_WAIT_TIMEOUT_GET  = 15 * time.Second
	IPFS_WAIT_TIMEOUT_LIST = 15 * time.Second
	TIMEOUT_MESSAGE        = "time out"
)

// Define the root of the commands
var RootCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"add": {
			Arguments: []cmds.Argument{
				cmds.StringArg("filePath", true, true, "filePath"),
				cmds.StringArg("encrypted", false, true, "encrypted"),
			},
			Run: add,
		},
		"get": {
			Arguments: []cmds.Argument{
				cmds.StringArg("filePath", true, true, "filePath"),
				cmds.StringArg("saveAs", false, true, "walletAddress"),
			},
			Run: get,
		},
		"ls": {
			Arguments: []cmds.Argument{},
			Run:       list,
		},
	},
}

func add(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
	ctxEnv, _ := env.(CtxEnv)
	args := req.Arguments
	encrypted := ""
	if len(args) > 1 {
		encrypted = args[1]
	}

	reqId := uuid.New().String()
	ctx := core.RegisterRemoteReqId(ctxEnv.Ctx, reqId)
	resultCh := file.SubscribeIpfsUpload(reqId)
	defer file.UnsubscribeIpfsUpload(reqId)

	isEncrypted := false
	if encrypted == "encrypt" {
		isEncrypted = true
	}
	pathStr := file.EscapePath(args[0:1])

	go event.RequestUploadFile(ctx, pathStr, isEncrypted, nil)

	timeout := time.After(IPFS_WAIT_TIMEOUT_ADD)
	var result *ipfsrpc.UploadResult
	for {
		select {
		case <-timeout:
			return re.CloseWithError(errors.New(TIMEOUT_MESSAGE))
		case result = <-resultCh:
			if result.Return == ipfsrpc.SUCCESS {
				//TODO alter message
				return re.Emit(*result)
			} else if result.Return == ipfsrpc.FAILED {
				return re.CloseWithError(errors.New(result.Message))
			} else if result.Return == ipfsrpc.UPLOAD_DATA {
				timeout = time.After(IPFS_WAIT_TIMEOUT_ADD)
				continue
			}
		}
	}
}

func get(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
	ctxEnv, _ := env.(CtxEnv)
	filePath := req.Arguments[0]

	reqId := uuid.New().String()
	ctx := core.RegisterRemoteReqId(ctxEnv.Ctx, reqId)
	resultCh := file.SubscribeIpfsDownload(reqId)
	defer file.UnsubscribeIpfsDownload(reqId)

	saveAs := ""
	if len(req.Arguments) == 2 {
		saveAs = req.Arguments[1]
	}
	go event.GetFileStorageInfo(ctx, filePath, "", saveAs, false, nil)

	timeout := time.After(IPFS_WAIT_TIMEOUT_GET)
	var result *ipfsrpc.DownloadResult
	for {
		select {
		case <-timeout:
			return re.CloseWithError(errors.New(TIMEOUT_MESSAGE))
		case result = <-resultCh:
			if result.Return == ipfsrpc.SUCCESS {
				return re.Emit(nil)
			} else if result.Return == ipfsrpc.FAILED {
				return re.CloseWithError(errors.New(result.Message))
			} else if result.Return == ipfsrpc.DOWNLOAD_DATA {
				timeout = time.After(IPFS_WAIT_TIMEOUT_GET)
			}
		}
	}
}

func list(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
	ctxEnv, _ := env.(CtxEnv)
	reqId := uuid.New().String()
	ctx, _ := context.WithTimeout(ctxEnv.Ctx, IPFS_WAIT_TIMEOUT_LIST)
	ctx = core.RegisterRemoteReqId(ctx, reqId)

	resultCh := file.SubscribeIpfsFileList(reqId)
	defer file.UnsubscribeIpfsFileList(reqId)

	go event.FindFileList(ctx, "", setting.WalletAddress, 0, "", 0, true)

	var result *ipfsrpc.FileListResult
	for {
		select {
		case <-ctx.Done():
			return re.CloseWithError(errors.New(TIMEOUT_MESSAGE))
		case result = <-resultCh:
			return re.Emit(result)
		}
	}
}
