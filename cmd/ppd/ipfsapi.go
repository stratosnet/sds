package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	nethttp "net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	cmds "github.com/ipfs/go-ipfs-cmds"
	"github.com/ipfs/go-ipfs-cmds/http"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/msg/protos"
	rpc_api "github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/datamesh"
	"github.com/stratosnet/sds/utils/types"
)

type ipfsenv struct {
	rpcClient  *rpc.Client
	httpRpcUrl string
	requester  requester
}

const (
	IPFS_WAIT_TIMEOUT_ADD  = 15 * time.Second
	IPFS_WAIT_TIMEOUT_GET  = 15 * time.Second
	IPFS_WAIT_TIMEOUT_LIST = 15 * time.Second
	TIMEOUT_MESSAGE        = "time out"
)

const (
	ipcNamespace      = "remoterpc"
	httpRpcNamespace  = "user"
	httpRpcUrl        = "httpRpcUrl"
	rpcModeFlag       = "rpcMode"
	rpcModeHttpRpc    = "httpRpc"
	rpcModeIpc        = "ipc"
	ipcEndpoint       = "ipcEndpoint"
	ipfsPortFlag      = "port"
	httpRpcDefaultUrl = "http://127.0.0.1:8335"
)

var (
	WalletPrivateKey types.AccPrivKey
	WalletPublicKey  types.AccPubKey
	WalletAddress    string
)

func ipfsapi(cmd *cobra.Command, args []string) {
	portParam, _ := cmd.Flags().GetString(ipfsPortFlag)
	rpcModeParam, _ := cmd.Flags().GetString(rpcModeFlag)
	ipcEndpointParam, _ := cmd.Flags().GetString(ipcEndpoint)
	httpRpcUrl, _ := cmd.Flags().GetString(httpRpcUrl)
	env := ipfsenv{}
	if rpcModeParam == rpcModeIpc {
		ipcEndpoint := setting.IpcEndpoint
		if ipcEndpointParam != "" {
			ipcEndpoint = ipcEndpointParam
		}
		c, _ := rpc.Dial(ipcEndpoint)
		env.rpcClient = c
		env.requester = ipcRequester{}
	} else if rpcModeParam == rpcModeHttpRpc {
		env.requester = httpRequester{}
		env.httpRpcUrl = httpRpcUrl
	} else {
		panic("unsupported rpc mode")
	}

	config := http.NewServerConfig()
	config.APIPath = "/api/v0"
	h := http.NewHandler(env, RootCmd, config)

	// create http rpc server
	err := nethttp.ListenAndServe(":"+portParam, h)
	if err != nil {
		panic(err)
	}
}

func ipfsapiPreRunE(cmd *cobra.Command, args []string) error {
	homePath, err := cmd.Flags().GetString(HOME)
	if err != nil {
		utils.ErrorLog("failed to get 'home' path for the client")
		return err
	}
	setting.SetIPCEndpoint(homePath)
	return nil
}

type requester interface {
	sendRequest(param interface{}, res any, rpcCmd string, env cmds.Environment) error
}

type httpRequester struct{}

type ipcRequester struct{}

func (requester httpRequester) sendRequest(param interface{}, res any, rpcCmd string, env cmds.Environment) error {
	ipfsenv, _ := env.(ipfsenv)
	var params []interface{}
	params = append(params, param)
	pm, err := json.Marshal(params)
	if err != nil {
		utils.ErrorLog("failed marshal param for " + rpcCmd)
		return nil
	}

	// wrap to the json-rpc message
	method := httpRpcNamespace + "_" + rpcCmd
	request := wrapJsonRpc(method, pm)

	if len(request) < 300 {
		utils.DebugLog("--> ", string(request))
	} else {
		utils.DebugLog("--> ", string(request[:230]), "... \"}]}")
	}

	// http post
	req, err := nethttp.NewRequest("POST", ipfsenv.httpRpcUrl, bytes.NewBuffer(request))
	if err != nil {
		return err
	}
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &nethttp.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if len(body) < 300 {
		utils.DebugLog("<-- ", string(body))
	} else {
		utils.DebugLog("<-- ", string(body[:230]), "... \"}]}")
	}

	resp.Body.Close()

	if body == nil {
		utils.ErrorLog("json marshal error")
		return err
	}

	// handle rsp
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return err
	}
	err = json.Unmarshal(rsp.Result, res)
	if err != nil {
		utils.ErrorLog("unmarshal failed")
		return err
	}
	return nil
}

func (requester ipcRequester) sendRequest(params interface{}, res any, ipcCmd string, env cmds.Environment) error {
	ipfsenv, _ := env.(ipfsenv)
	method := ipcNamespace + "_" + ipcCmd
	err := ipfsenv.rpcClient.Call(res, method, params)
	if err != nil {
		return err
	}
	return nil
}

// Define the root of the commands
var RootCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"add": {
			Arguments: []cmds.Argument{
				cmds.StringArg("fileName", true, true, "fileName"),
			},
			Run: add,
		},
		"get": {
			Arguments: []cmds.Argument{
				cmds.StringArg("sdmPath", true, true, "sdmPath"),
			},
			Run: get,
		},
		"ls": {
			Options: []cmds.Option{
				cmds.Uint64Option("page"),
			},
			Run: list,
		},
	},
}

func add(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
	ipfsenv, _ := env.(ipfsenv)
	requester := ipfsenv.requester

	// args[0] is the first param, instead of the subcommand "put"
	fileName := req.Arguments[0]
	hash := file.GetFileHash(fileName, "")
	utils.Log("- start uploading the file:", fileName)

	// compose request file upload params
	paramsFile, err := reqUploadMsg(fileName, hash)
	if err != nil {
		return re.CloseWithError(err)
	}

	utils.Log("- request uploading file (method: user_requestUpload)")

	var res rpc_api.Result
	err = requester.sendRequest(paramsFile, &res, "requestUpload", env)

	// Handle result:1 sending the content
	for res.Return == rpc_api.UPLOAD_DATA {
		utils.Log("- received response (return: UPLOAD_DATA)")
		// get the data from the file
		so := &protos.SliceOffset{
			SliceOffsetStart: *res.OffsetStart,
			SliceOffsetEnd:   *res.OffsetEnd,
		}
		rawData := file.GetFileData(fileName, so)
		encoded := base64.StdEncoding.EncodeToString(rawData)
		paramsData := uploadDataMsg(hash, encoded)
		utils.Log("- request upload date (method: user_uploadData)")

		err = requester.sendRequest(paramsData, &res, "uploadData", env)
		if err != nil {
			return re.CloseWithError(err)
		}
	}
	utils.Log("- uploading is done")
	return re.Emit("uploading is done")
}

func get(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
	ipfsenv, _ := env.(ipfsenv)
	requester := ipfsenv.requester

	sdmPath := req.Arguments[0]
	utils.Log("- start downloading the file: ", sdmPath)
	_, _, fileHash, _, err := datamesh.ParseFileHandle(sdmPath)
	if err != nil {
		return re.CloseWithError(errors.New("sdm format error"))
	}

	// compose "request file download" request
	r, err := reqDownloadMsg(fileHash, sdmPath)
	if err != nil {
		return re.CloseWithError(err)
	}

	utils.Log("- request downloading the file (method: user_requestDownload)")
	// http request-respond
	var res rpc_api.Result
	err = requester.sendRequest(r, &res, "requestDownload", env)
	if err != nil {
		return re.CloseWithError(err)
	}

	var fileSize uint64 = 0
	var pieceCount uint64 = 0
	var fileMg *os.File = nil
	defer fileMg.Close()

	var params interface{}
	for res.Return == rpc_api.DOWNLOAD_OK || res.Return == rpc_api.DL_OK_ASK_INFO {
		if fileMg == nil {
			exist, err := file.PathExists("./download/")
			if err != nil {
				return err
			}
			if !exist {
				if err = os.MkdirAll("./download/", 0777); err != nil {
					return re.CloseWithError(err)
				}
			}
			fileMg, err = os.OpenFile(filepath.Join("./download/", res.FileName), os.O_CREATE|os.O_RDWR, 0777)
			if err != nil {
				utils.ErrorLog("error initialize file")
				return re.CloseWithError(errors.New("can't open file"))
			}
		}

		if res.Return == rpc_api.DL_OK_ASK_INFO {
			utils.Log("- received response (return: DL_OK_ASK_INFO) after received", pieceCount, "piece(s)")
			params = downloadedFileInfoMsg(fileHash, fileSize, res.ReqId)
			err = requester.sendRequest(params, &res, "downloadedFileInfo", env)
			utils.Log("- request file information verification (method: user_downloadedFileInfo)")
		} else {
			// rpc.DOWNLOAD_OK
			utils.Log("- received response (return: DOWNLOAD_OK)")
			start := *res.OffsetStart
			end := *res.OffsetEnd
			fileSize = fileSize + (end - start)
			decoded, _ := base64.StdEncoding.DecodeString(res.FileData)
			if len(decoded) != int(end-start) {
				errMsg := "Wrong size:" + strconv.Itoa(len(decoded)) + " " + strconv.Itoa(int(end-start))
				utils.ErrorLog(errMsg)
				return re.CloseWithError(err)
			}
			pieceCount = pieceCount + 1

			_, err = fileMg.WriteAt(decoded, int64(start))
			if err != nil {
				utils.ErrorLog("error save file")
				return re.CloseWithError(errors.New("failed writing file"))
			}

			params = downloadDataMsg(fileHash, res.ReqId)
			err = requester.sendRequest(params, &res, "downloadData", env)
			utils.Log("- request downloading file date (user_downloadData)")
		}

		if err != nil {
			utils.ErrorLog(err)
			return re.CloseWithError(err)
		}
	}
	if res.Return == rpc_api.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
		return re.Emit(res)
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
		return re.CloseWithError(errors.New("failed to download"))
	}
}

func list(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
	ipfsenv, _ := env.(ipfsenv)
	requester := ipfsenv.requester
	var page uint64
	var err error

	page = 0
	if pageOption, ok := req.Options["page"]; ok {
		page, ok = pageOption.(uint64)
		if !ok {
			return re.CloseWithError(errors.New("page should be positive integer"))
		}
	}
	params, err := reqListMsg(page)
	if err != nil {
		return re.CloseWithError(err)
	}
	var res rpc_api.FileListResult
	err = requester.sendRequest(params, &res, "requestList", env)
	if err != nil {
		return re.CloseWithError(err)
	}
	return re.Emit(res)
}

func reqUploadMsg(fileName, hash string) (*rpc_api.ParamReqUploadFile, error) {
	// file size
	info := file.GetFileInfo(fileName)
	if info == nil {
		return nil, errors.New("failed to get file information")
	}

	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		return nil, errors.New("failed reading key file")
	}

	// signature
	sign, err := WalletPrivateKey.Sign([]byte(utils.GetFileUploadWalletSignMessage(hash, WalletAddress)))
	if err != nil {
		return nil, err
	}
	wpk, err := WalletPublicKey.ToBech()
	if err != nil {
		return nil, err
	}

	return &rpc_api.ParamReqUploadFile{
		FileName:     fileName,
		FileSize:     int(info.Size()),
		FileHash:     hash,
		WalletAddr:   WalletAddress,
		WalletPubkey: wpk,
		Signature:    hex.EncodeToString(sign),
	}, nil
}

func uploadDataMsg(hash, data string) rpc_api.ParamUploadData {
	return rpc_api.ParamUploadData{
		FileHash: hash,
		Data:     data,
	}
}

func reqDownloadMsg(hash, sdmPath string) (*rpc_api.ParamReqDownloadFile, error) {
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		return nil, errors.New("Failed reading key file.")
	}

	// signature
	sign, err := WalletPrivateKey.Sign([]byte(utils.GetFileDownloadWalletSignMessage(hash, WalletAddress)))
	if err != nil {
		return nil, err
	}
	wpk, err := WalletPublicKey.ToBech()
	if err != nil {
		return nil, err
	}

	return &rpc_api.ParamReqDownloadFile{
		FileHandle:   sdmPath,
		WalletAddr:   WalletAddress,
		WalletPubkey: wpk,
		Signature:    hex.EncodeToString(sign),
	}, nil
}

// downloadDataMsg
func downloadDataMsg(hash, reqid string) *rpc_api.ParamDownloadData {
	return &rpc_api.ParamDownloadData{
		FileHash: hash,
		ReqId:    reqid,
	}
}

// downloadedFileInfoMsg
func downloadedFileInfoMsg(fileHash string, fileSize uint64, reqid string) *rpc_api.ParamDownloadFileInfo {
	return &rpc_api.ParamDownloadFileInfo{
		FileHash: fileHash,
		FileSize: fileSize,
		ReqId:    reqid,
	}
}

func reqListMsg(page uint64) (*rpc_api.ParamReqFileList, error) {
	//wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		return nil, errors.New("failed reading key file")
	}

	return &rpc_api.ParamReqFileList{
		WalletAddr: WalletAddress,
		PageId:     page,
	}, nil
}

func readWalletKeys(wallet string) bool {
	if wallet == "" {
		WalletAddress = findWallet("./accounts/")
	} else {
		WalletAddress = wallet
	}
	if WalletAddress == "" {
		return false
	}

	keyjson, err := ioutil.ReadFile(filepath.Join("./accounts/", WalletAddress+".json"))
	if utils.CheckError(err) {
		utils.ErrorLog("getPublicKey ioutil.ReadFile", err)
		return false
	}

	key, err := utils.DecryptKey(keyjson, "aaa")
	if utils.CheckError(err) {
		utils.ErrorLog("getPublicKey DecryptKey", err)
		return false
	}
	WalletPrivateKey = types.BytesToAccPriveKey(key.PrivateKey)
	WalletPublicKey = WalletPrivateKey.PubKeyFromPrivKey()
	return true
}

func findWallet(folder string) string {
	var files []string
	var file string

	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return nil
		}
		file = filepath.Base(path)

		if m, _ := filepath.Match("st1*", file); !info.IsDir() && filepath.Ext(path) == ".json" && m {
			// only catch the first wallet file
			if files == nil {
				files = append(files, file[:len(file)-len(filepath.Ext(file))])
			}
		}
		return nil
	})
	if err != nil {
		return ""
	}

	if files != nil {
		return files[0]
	}
	return ""
}

func wrapJsonRpc(method string, param []byte) []byte {
	// compose json-rpc request
	request := &jsonrpcMessage{
		Version: "2.0",
		ID:      1,
		Method:  method,
		Params:  json.RawMessage(param),
	}
	r, e := json.Marshal(request)
	if e != nil {
		utils.ErrorLog("json marshal error", e)
		return nil
	}
	return r
}

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      int             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}
