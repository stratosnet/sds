package ipfs

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	cmds "github.com/ipfs/go-ipfs-cmds"

	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	rpc_api "github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/sds-msg/protos"
	msgutils "github.com/stratosnet/sds/sds-msg/utils"
)

const (
	CMD_ADD  = "add"
	CMD_GET  = "get"
	CMD_LIST = "ls"
)

// Define the root of the commands
var RootCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		CMD_ADD: {
			Arguments: []cmds.Argument{
				cmds.StringArg("fileName", true, true, "fileName"),
			},
			Run: add,
		},
		CMD_GET: {
			Arguments: []cmds.Argument{
				cmds.StringArg("sdmPath", true, true, "sdmPath"),
			},
			Run: get,
		},
		CMD_LIST: {
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

	// compose reqOzone for the SN
	paramReqGetOzone, err := reqOzone()
	if err != nil {
		return emitError(re, "failed to create request message", err)
	}

	utils.Log("- request get ozone (method: user_requestGetOzone)")

	var resOzone rpc_api.GetOzoneResult
	err = requester.sendRequest(paramReqGetOzone, &resOzone, "requestGetOzone", env)
	if err != nil {
		return emitError(re, "failed to send upload file request", err)
	}

	// compose request file upload params
	paramsFile, err := reqUploadMsg(fileName, hash, resOzone.SequenceNumber)
	if err != nil {
		return emitError(re, "failed to create request message", err)
	}

	utils.Log("- request uploading file (method: user_requestUpload)")

	var res rpc_api.Result
	err = requester.sendRequest(paramsFile, &res, "requestUpload", env)
	if err != nil {
		return emitError(re, "failed to send upload file request", err)
	}

	// Handle result:1 sending the content
	for res.Return == rpc_api.UPLOAD_DATA {
		utils.Log("- received response (return: UPLOAD_DATA)")
		// get the data from the file
		so := &protos.SliceOffset{
			SliceOffsetStart: *res.OffsetStart,
			SliceOffsetEnd:   *res.OffsetEnd,
		}
		rawData, err := file.GetFileData(fileName, so)
		if err != nil {
			return emitError(re, "failed to get data from file", err)
		}
		encoded := base64.StdEncoding.EncodeToString(rawData)
		paramsData, err := uploadDataMsg(hash, encoded, resOzone.SequenceNumber)
		if err != nil {
			return emitError(re, "failed to prepare upload data request", err)
		}
		utils.Log("- request upload date (method: user_uploadData)")

		err = requester.sendRequest(paramsData, &res, "uploadData", env)
		if err != nil {
			return emitError(re, "failed to send upload data request", err)
		}
	}
	if res.Return == rpc_api.SUCCESS {
		utils.Log("- uploading is done")
		return re.Emit("uploading is done")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
		return emitError(re, "failed to upload", nil)
	}
}

func get(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
	ipfsenv, _ := env.(ipfsenv)
	requester := ipfsenv.requester

	sdmPath := req.Arguments[0]
	utils.Log("- start downloading the file: ", sdmPath)
	_, _, fileHash, _, err := fwtypes.ParseFileHandle(sdmPath)
	if err != nil {
		return emitError(re, "sdm format error", nil)
	}

	// compose reqOzone for the SN
	paramReqGetOzone, err := reqOzone()
	if err != nil {
		return emitError(re, "failed to create request message", err)
	}

	utils.Log("- request get ozone (method: user_requestGetOzone)")

	var resOzone rpc_api.GetOzoneResult
	err = requester.sendRequest(paramReqGetOzone, &resOzone, "requestGetOzone", env)
	if err != nil {
		return emitError(re, "failed to send upload file request", err)
	}

	// compose "request file download" request
	r, err := reqDownloadMsg(fileHash, sdmPath, resOzone.SequenceNumber)
	if err != nil {
		return emitError(re, "failed to create download msg", err)
	}

	utils.Log("- request downloading the file (method: user_requestDownload)")
	// http request-respond
	var res rpc_api.Result
	err = requester.sendRequest(r, &res, "requestDownload", env)
	if err != nil {
		return emitError(re, "failed to send download file request", err)
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
				if err = os.MkdirAll("./download/", 0700); err != nil {
					return emitError(re, "failed to create download folder", err)
				}
			}
			fileMg, err = os.OpenFile(filepath.Join("./download/", res.FileName), os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				return emitError(re, "can't open file", err)
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
				return emitError(re, errMsg, nil)
			}
			pieceCount = pieceCount + 1

			_, err = fileMg.WriteAt(decoded, int64(start))
			if err != nil {
				return emitError(re, "failed writing file", nil)
			}

			params = downloadDataMsg(fileHash, res.ReqId)
			err = requester.sendRequest(params, &res, "downloadData", env)
			utils.Log("- request downloading file date (user_downloadData)")
		}

		if err != nil {
			return emitError(re, "failed to send download data request", err)
		}
	}
	if res.Return == rpc_api.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
		return re.Emit("download is done")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
		return emitError(re, "failed to download", nil)
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
			return emitError(re, "page should be positive integer", nil)
		}
	}
	params, err := reqListMsg(page)
	if err != nil {
		return emitError(re, "failed to create list request", err)
	}
	var res rpc_api.FileListResult
	err = requester.sendRequest(params, &res, "requestList", env)
	if err != nil {
		return emitError(re, "failed to send list request", err)
	}
	return re.Emit(res)
}

func reqOzone() (*rpc_api.ParamReqGetOzone, error) {
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		return nil, errors.New("failed reading key file")
	}
	return &rpc_api.ParamReqGetOzone{
		WalletAddr: WalletAddress,
	}, nil
}

func reqUploadMsg(filePath, hash, sn string) (*rpc_api.ParamReqUploadFile, error) {
	// file size
	info, err := file.GetFileInfo(filePath)
	if info == nil || err != nil {
		return nil, errors.New("failed to get file information")
	}
	fileName := info.Name()
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		return nil, errors.New("failed reading key file")
	}
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.GetFileUploadWalletSignMessage(hash, WalletAddress, sn, nowSec)))
	if err != nil {
		return nil, err
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil, err
	}

	return &rpc_api.ParamReqUploadFile{
		FileName: fileName,
		FileSize: int(info.Size()),
		FileHash: hash,
		Signature: rpc_api.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime:         nowSec,
		DesiredTier:     2,
		AllowHigherTier: true,
	}, nil
}

func uploadDataMsg(hash, data, sn string) (rpc_api.ParamUploadData, error) {
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		return rpc_api.ParamUploadData{}, errors.New("failed reading key file")
	}
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.GetFileUploadWalletSignMessage(hash, WalletAddress, sn, nowSec)))
	if err != nil {
		return rpc_api.ParamUploadData{}, err
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return rpc_api.ParamUploadData{}, err
	}

	return rpc_api.ParamUploadData{
		FileHash: hash,
		Data:     data,
		Signature: rpc_api.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime: nowSec,
	}, nil
}

func reqDownloadMsg(hash, sdmPath, sn string) (*rpc_api.ParamReqDownloadFile, error) {
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		return nil, errors.New("Failed reading key file.")
	}
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.GetFileDownloadWalletSignMessage(hash, WalletAddress, sn, nowSec)))
	if err != nil {
		return nil, err
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil, err
	}

	return &rpc_api.ParamReqDownloadFile{
		FileHandle: sdmPath,
		Signature: rpc_api.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime: nowSec,
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
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.FindMyFileListWalletSignMessage(WalletAddress, nowSec)))
	if err != nil {
		return nil, err
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil, err
	}
	return &rpc_api.ParamReqFileList{
		Signature: rpc_api.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		PageId:  page,
		ReqTime: nowSec,
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

	keyjson, err := os.ReadFile(filepath.Join("./accounts/", WalletAddress+".json"))
	if utils.CheckError(err) {
		utils.ErrorLog("getPublicKey ioutil.ReadFile", err)
		return false
	}

	key, err := fwtypes.DecryptKey(keyjson, WalletPassword, true)
	if utils.CheckError(err) {
		utils.ErrorLog("getPublicKey DecryptKey", err)
		return false
	}
	WalletPrivateKey = key.PrivateKey
	WalletPublicKey = WalletPrivateKey.PubKey()
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

func emitError(re cmds.ResponseEmitter, msg string, err error) error {
	if err != nil {
		utils.ErrorLog(msg, err)
	} else {
		utils.ErrorLog(msg)
	}
	return re.CloseWithError(errors.New(msg))
}
