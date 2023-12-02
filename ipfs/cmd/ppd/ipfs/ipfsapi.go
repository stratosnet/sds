package ipfs

import (
	"context"
	nethttp "net/http"
	"os"

	cmds "github.com/ipfs/go-ipfs-cmds"
	"github.com/ipfs/go-ipfs-cmds/cli"
	"github.com/ipfs/go-ipfs-cmds/http"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/ipfs/pp/ipfs"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/rpc"
)

type ipfsenv struct {
	rpcClient  *rpc.Client
	httpRpcUrl string
	requester  requester
}

const (
	IpcNamespace             = "remoterpc"
	HttpRpcNamespace         = "user"
	HttpRpcUrl               = "httpRpcUrl"
	RpcModeFlag              = "rpcMode"
	RpcModeHttpRpc           = "httpRpc"
	RpcModeIpc               = "ipc"
	IpcEndpoint              = "ipcEndpoint"
	IpfsPortFlag             = "port"
	HttpRpcDefaultUrl        = "http://127.0.0.1:9301"
	HOME              string = "home"
	PasswordFlag             = "password"
)

var (
	WalletPrivateKey fwcryptotypes.PrivKey
	WalletPublicKey  fwcryptotypes.PubKey
	WalletAddress    string
	WalletPassword   string
)

func IpfsapiPreRunE(cmd *cobra.Command, args []string) error {
	homePath, err := cmd.Flags().GetString(HOME)
	if err != nil {
		utils.ErrorLog("failed to get 'home' path for the client")
		return err
	}
	setting.SetIPCEndpoint(homePath)
	return nil
}

func Ipfsapi(cmd *cobra.Command, args []string) {
	portParam, _ := cmd.Flags().GetString(IpfsPortFlag)
	env := getCmdEnv(cmd)

	password, err := cmd.Flags().GetString("password")
	if err != nil {
		panic(errors.New("failed to get password from the parameters"))
	}
	WalletPassword = password

	config := http.NewServerConfig()
	config.APIPath = "/api/v0"
	h := http.NewHandler(env, RootCmd, config)

	// create http rpc server
	err = nethttp.ListenAndServe(":"+portParam, h)
	if err != nil {
		panic(err)
	}
}

func Ipfsmigrate(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		panic("missing file cid")
	}
	fileName := ""
	if len(args) > 1 && args[1] != "" {
		fileName = args[1]
	}

	ctx := context.Background()
	env := getCmdEnv(cmd)

	// get file from ips
	filePath, err := ipfs.GetFile(ctx, args[0], fileName)
	if err != nil {
		panic(err)
	}

	req, err := cli.Parse(ctx, []string{CMD_ADD, filePath}, os.Stdin, RootCmd)
	if err != nil {
		panic(err)
	}

	cliRe, err := cli.NewResponseEmitter(os.Stdout, os.Stderr, req)
	if err != nil {
		panic(err)
	}

	wait := make(chan struct{})
	var re cmds.ResponseEmitter = cliRe
	if pr, ok := req.Command.PostRun[cmds.CLI]; ok {
		var (
			res   cmds.Response
			lower = re
		)

		re, res = cmds.NewChanResponsePair(req)

		go func() {
			defer close(wait)
			err := pr(res, lower)
			if err != nil {
				utils.ErrorLog(err)
			}
		}()
	} else {
		close(wait)
	}

	RootCmd.Call(req, re, env)
	<-wait

	os.Exit(cliRe.Status())
}

func getCmdEnv(cmd *cobra.Command) ipfsenv {
	rpcModeParam, _ := cmd.Flags().GetString(RpcModeFlag)
	ipcEndpointParam, _ := cmd.Flags().GetString(IpcEndpoint)
	httpRpcUrl, _ := cmd.Flags().GetString(HttpRpcUrl)
	env := ipfsenv{}
	if rpcModeParam == RpcModeIpc {
		ipcEndpoint := setting.IpcEndpoint
		if ipcEndpointParam != "" {
			ipcEndpoint = ipcEndpointParam
		}
		c, err := rpc.Dial(ipcEndpoint)
		if err != nil {
			panic("failed to dial ipc endpoint")
		}
		env.rpcClient = c
		env.requester = ipcRequester{}
	} else if rpcModeParam == RpcModeHttpRpc {
		env.requester = httpRequester{}
		env.httpRpcUrl = httpRpcUrl
	} else {
		panic("unsupported rpc mode")
	}
	return env
}
