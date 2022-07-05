package main

import (
	"fmt"
	"os"
	"bytes"
	"io/ioutil"
	"encoding/base64"
	"encoding/hex"
	"path/filepath"
	"encoding/json"
	"crypto/sha256"
	"net/http"
	"strconv"
	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/stratos-chain/types"
	"github.com/tendermint/tendermint/libs/bech32"
)

const (
	DEFAULT_URL = "http://127.0.0.1:8235"
)

var (
	WalletPrivateKey []byte
	WalletPublicKey  string
	WalletAddress    string

	Url              string
)

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      int             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
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
				files = append(files, file[:len(file) - len(filepath.Ext(file))])
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

func readWalletKeys(wallet string) bool {
	if wallet == "" {
		WalletAddress = findWallet("./account/")
	} else {
		WalletAddress = wallet
	}
	if WalletAddress == "" {
		return false
	}

	keyjson, err := ioutil.ReadFile(filepath.Join("./account/", WalletAddress+".json"))
	if utils.CheckError(err) {
		utils.ErrorLog("getPublicKey ioutil.ReadFile", err)
		return false
	}
	key, err := utils.DecryptKey(keyjson, "aaa")

	if utils.CheckError(err) {
		utils.ErrorLog("getPublicKey DecryptKey", err)
		return false
	}
	WalletPrivateKey = key.PrivateKey

	rawPubkey := secp256k1.PrivKeyToPubKeyCompressed(key.PrivateKey)
	pubkey64 := base64.StdEncoding.EncodeToString(rawPubkey)
	WalletPublicKey, err = bech32.ConvertAndEncode(types.AccountPubKeyPrefix, []byte(pubkey64))
	if err != nil {
		utils.DebugLog(err)
		return false
	}

	return true
}

func wrapJsonRpc(method string, param []byte) []byte {
    // compose json-rpc request
	request := &jsonrpcMessage {
		Version: "2.0",
			ID:  1,
		 Method: method,
		 Params: json.RawMessage(param),
	 }
	r, e := json.Marshal(request)
	if e != nil {
		utils.DebugLog("json marshal error", e)
		return nil
	}
	return r
}

func reqUploadMsg(fileName, hash string) []byte {
	// file size
	info := file.GetFileInfo(fileName)
	if info == nil {
		utils.DebugLog("Failed to get file information.")
		return nil
	}

	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		utils.DebugLog("Failed reading key file.")
		return nil
	}

	// signature
	hs := sha256.Sum256([]byte(hash + WalletAddress))
	sign, err := secp256k1.Sign(hs[:], WalletPrivateKey)
	if err != nil {
		utils.DebugLog("failed to sign, error:", err)
		return nil
	}
	signature := sign[:len(sign)-1]

	// param
	var params = []rpc.ParamReqUploadFile{}
	params = append(params, rpc.ParamReqUploadFile {
		FileName      : fileName,
		FileSize      : int(info.Size()),
		FileHash      : hash,
		WalletAddr    : WalletAddress,
		WalletPubkey  : WalletPublicKey,
		Signature     : hex.EncodeToString(signature),
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.DebugLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// 
	return wrapJsonRpc("user_requestUpload", pm)
}

func uploadDataMsg(hash, data string) []byte {
	var pa = []rpc.ParamUploadData{}
	pa = append(pa, rpc.ParamUploadData {
		FileHash      : hash,
		Data          : data,
	})
	pm, e := json.Marshal(pa)
	if e != nil {
		utils.DebugLog("json marshal error", e)
	}

	return wrapJsonRpc("user_uploadData", pm)
}


// put
func put(cmd *cobra.Command, args []string) error {
	// args[0] is the first param, instead of the subcommand "put"
	fileName := args[0]
	hash := file.GetFileHash(args[0], "")

    // compose request file upload params
	r := reqUploadMsg(args[0], hash)
	if r == nil {
		return nil
	}

    // http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.DebugLog("http no response")
		return nil
	}

	// handle: unmarshal response then unmarshal result
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
		utils.DebugLog("unmarshal failed")
		return nil
	}

	var res rpc.Result
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		utils.DebugLog("unmarshal failed")
		return nil
	}

	// Handle result:1 sending the content
	for res.Return == rpc.UPLOAD_DATA {
		// get the data from the file
		so := &protos.SliceOffset {
			SliceOffsetStart: *res.OffsetStart,
			SliceOffsetEnd: *res.OffsetEnd,
		}
		rawData := file.GetFileData(fileName, so)
		encoded := base64.StdEncoding.EncodeToString(rawData)
		r = uploadDataMsg(hash, encoded)

		body = httpRequest(r)
		if body == nil {
			utils.DebugLog("json marshal error")
			return nil
		}
		
		// Handle rsp
		err = json.Unmarshal(body, &rsp)
		if err != nil {
			utils.DebugLog("unmarshal failed")
			return nil
		}
		err = json.Unmarshal(rsp.Result, &res)
		if err != nil {
			utils.DebugLog("unmarshal failed")
			return nil
		}
				fmt.Println("D")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// getParams
func reqDownloadMsg(hash string) []byte {
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		utils.DebugLog("Failed reading key file.")
		return nil
	}

	// param
	var params = []rpc.ParamReqDownloadFile{}
	params = append(params, rpc.ParamReqDownloadFile {
		FileHash: hash,
		WalletAddr: WalletAddress,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.DebugLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_requestDownload", pm)
}

// downloadDataMsg
func downloadDataMsg(hash, reqid string) []byte {
	// param
	var params = []rpc.ParamDownloadData{}
	params = append(params, rpc.ParamDownloadData {
		FileHash: hash,
		ReqId: reqid,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.DebugLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_downloadData", pm)
}

// downloadedFileInfoMsg
func downloadedFileInfoMsg(fileHash string, fileSize uint64, reqid string) []byte  {
	// param
	var params = []rpc.ParamDownloadFileInfo{}
	params = append(params, rpc.ParamDownloadFileInfo {
		FileHash: fileHash,
		FileSize: fileSize,
		ReqId: reqid,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.DebugLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_downloadedFileInfo", pm)
}

// get
func get(cmd *cobra.Command, args []string) error {

	// args[0] is the fileHash
	fileHash := args[0]

	// compose "request file download" request
	r := reqDownloadMsg(fileHash)
	if r == nil {
		return nil
	}

	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.DebugLog("json marshal error")
		return nil
	}

	// handle rsp
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
	  return nil
	}

	var res rpc.Result
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		utils.DebugLog("unmarshal failed")
		return nil
	}

	var fileSize uint64 = 0
	var pieceCount uint64 = 0
	// Handle result:1 sending the content
	for res.Return == rpc.DOWNLOAD_OK || res.Return == rpc.DL_OK_ASK_INFO {
		// TODO: save the piece to the file
		if res.Return == rpc.DL_OK_ASK_INFO {
			r = downloadedFileInfoMsg(fileHash, fileSize, res.ReqId)
			fmt.Println("There are", pieceCount, "pieces received.")
		}else {
			start := *res.OffsetStart
			end := *res.OffsetEnd
			fileSize = fileSize + (end - start)
			decoded, _ := base64.StdEncoding.DecodeString(res.FileData)
			if len(decoded) != int(end - start) {
				utils.DebugLog("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
				utils.DebugLog("Wrong size:", strconv.Itoa(len(decoded)), " ", strconv.Itoa(int(end-start)))
				return nil
			}
			pieceCount = pieceCount + 1
			r = downloadDataMsg(fileHash, res.ReqId)
		}

		body := httpRequest(r)
		if body == nil {
			utils.DebugLog("json marshal error")
			return nil
		}

		// Handle rsp
		err := json.Unmarshal(body, &rsp)
		if err != nil {
			return nil
		}

		err = json.Unmarshal(rsp.Result, &res)
		if err != nil {
			utils.DebugLog("unmarshal failed")
			return nil
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// reqListMsg
func reqListMsg(page uint64) []byte {
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		utils.DebugLog("Failed reading key file.")
		return nil
	}

	// param
	var params = []rpc.ParamReqFileList{}
	params = append(params, rpc.ParamReqFileList {
		WalletAddr: WalletAddress,
		PageId: page,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.DebugLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_requestList", pm)
}

// printFileList
func printFileList(res rpc.FileListResult){
	if res.Return == rpc.SUCCESS {
		fmt.Printf("\n%-20s %-41s %-9s %-8s\n", "File Name", "File Hash", "File Size", "Create Time")
		fmt.Printf("_____________________________________________________________________________________\n")
		for i:= range res.FileInfo {
			f := res.FileInfo[i]
			fmt.Printf("%-20s %-25s %10d %8d\n", f.FileName, f.FileHash, f.FileSize, f.CreateTime)
		}
		fmt.Printf("_____________________________________________________________________________________\n")
		fmt.Printf("Total: %d\tPage: %d\n\n", res.TotalNumber, res.PageId)
	}

}

// list
func list(cmd *cobra.Command, args []string) error {
	var page uint64
	var e error
	page = 0
	if len(args) == 1 {
		page, e = strconv.ParseUint(args[0], 10, 64)
		if e != nil {
			return e
		}
	}
	r := reqListMsg(page)
	if r == nil {
		return nil
	}
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.DebugLog("json marshal error")
		return nil
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
	  return nil
	}

	var res rpc.FileListResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
	  return nil
	}

	printFileList(res)
	return nil
}

// printSharedFileList
func printSharedFileList(res rpc.FileShareResult){
	if res.Return == rpc.SUCCESS {
		fmt.Printf("\n%-20s %-41s %-9s %-8s  %-8s   %-15s  %-15s\n", "File Name", "File Hash", "File Size", "Link Time", "Link Exp", "Share ID", "Share Link")
		fmt.Printf("________________________________________________________________________________________________________________________________________\n")
		for i:= range res.FileInfo {
			f := res.FileInfo[i]
			fmt.Printf("%-20s %-25s %10d %8d %8d %-15s %-15s\n", f.FileName, f.FileHash, f.FileSize, f.LinkTime, f.LinkTimeExp, f.ShareId, f.ShareLink)
		}
		fmt.Printf("________________________________________________________________________________________________________________________________________\n")
		fmt.Printf("Total: %d\tPage: %d\n\n", res.TotalNumber, res.PageId)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// reqListShareMsg
func reqListShareMsg(page uint64) []byte {
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		utils.DebugLog("Failed reading key file.")
		return nil
	}

	// param
	var params = []rpc.ParamReqListShared{}
	params = append(params, rpc.ParamReqListShared {
		WalletAddr    : WalletAddress,
		PageId        : page,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.DebugLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap into request message
	return wrapJsonRpc("user_requestListShare", pm)
}

// listshared
func listshare(cmd *cobra.Command, args[]string) error {
	var page uint64
	var e error
	page = 0
	if len(args) == 1 {
		page, e = strconv.ParseUint(args[0], 10, 64)
		if e != nil {
			return e
		}
	}
	// compose request
	r := reqListShareMsg(page)
	if r == nil {
		return nil
	}
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.DebugLog("json marshal error")
		return nil
	}
	// handle response
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
	  return err
	}
	var res rpc.FileShareResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
	  return nil
	}
	printSharedFileList(res)
	return nil
}

// reqShareMsg
func reqShareMsg(hash string) []byte {
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		utils.DebugLog("Failed reading key file.")
		return nil
	}

	// param
	var params = []rpc.ParamReqShareFile{}
	params = append(params, rpc.ParamReqShareFile {
		FileHash      : hash,
		WalletAddr    : WalletAddress,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.DebugLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap into request message
	return wrapJsonRpc("user_requestShare", pm)
}

// share
func share(cmd *cobra.Command, args[]string) error {
	// check input
	if len(args) != 1 {
		utils.DebugLog("file hash is not provided")
		return nil
	}
	// compose request
	r := reqShareMsg(args[0])
	if r == nil {
		return nil
	}

	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.DebugLog("json marshal error")
		return nil
	}
	// handle response
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
	  return err
	}
	var res rpc.FileShareResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
	  return nil
	}

	return nil
}

// reqStopShareMsg
func reqStopShareMsg(shareId string) []byte {
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		utils.DebugLog("Failed reading key file.")
		return nil
	}

	// param
	var params = []rpc.ParamReqStopShare{}
	params = append(params, rpc.ParamReqStopShare {
		WalletAddr    : WalletAddress,
		ShareId       : shareId,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.DebugLog("failed marshal param for ReqStopShare")
		return nil
	}

	// wrap into request message
	return wrapJsonRpc("user_requestStopShare", pm)
}

// stopshare
func stopshare(cmd *cobra.Command, args[]string) error {

	// compose request
	r := reqStopShareMsg(args[0])
	if r == nil {
		return nil
	}
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.DebugLog("json marshal error")
		return nil
	}
	// handle response
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
	  return err
	}
	var res rpc.FileShareResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
	  return nil
	}

	return nil
}

// reqGetSharedMsg
func reqGetSharedMsg(shareLink string) []byte {
	// wallet address
	ret := readWalletKeys(WalletAddress)
	if !ret {
		utils.DebugLog("Failed reading key file.")
		return nil
	}

	// param
	var params = []rpc.ParamReqGetShared{}
	params = append(params, rpc.ParamReqGetShared {
		WalletAddr    : WalletAddress,
		ShareLink     : shareLink,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.DebugLog("failed marshal param for ReqStopShare")
		return nil
	}

	// wrap into request message
	return wrapJsonRpc("user_requestGetShared", pm)
}

// getshared
func getshared(cmd *cobra.Command, args[]string) error {

	// compose request
	r := reqGetSharedMsg(args[0])
	if r == nil {
		return nil
	}
	fileHash := args[1]

	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.DebugLog("json marshal error")
		return nil
	}
	// handle response
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
	  return err
	}
	var res rpc.Result
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
	  return nil
	}

	var fileSize uint64 = 0
	var pieceCount uint64 = 0
	// Handle result:1 sending the content
	for res.Return == rpc.DOWNLOAD_OK || res.Return == rpc.DL_OK_ASK_INFO {
		// TODO: save the piece to the file
		if res.Return == rpc.DL_OK_ASK_INFO {
			r = downloadedFileInfoMsg(fileHash, fileSize, res.ReqId)
			fmt.Println("There are", pieceCount, "pieces received.")
		}else {
			start := *res.OffsetStart
			end := *res.OffsetEnd
			fileSize = fileSize + (end - start)
			decoded, _ := base64.StdEncoding.DecodeString(res.FileData)
			if len(decoded) != int(end - start) {
				utils.DebugLog("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
				utils.DebugLog("Wrong size:", strconv.Itoa(len(decoded)), " ", strconv.Itoa(int(end-start)))
				return nil
			}
			pieceCount = pieceCount + 1
			r = downloadDataMsg(fileHash, res.ReqId)
		}

		body := httpRequest(r)
		if body == nil {
			utils.DebugLog("json marshal error")
			return nil
		}

		// Handle rsp
		err := json.Unmarshal(body, &rsp)
		if err != nil {
			return nil
		}

		err = json.Unmarshal(rsp.Result, &res)
		if err != nil {
			utils.DebugLog("unmarshal failed")
			return nil
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// httpRequest
func httpRequest(request []byte) []byte {
	if len(request) < 300 {
		fmt.Println("--> ", string(request))
	} else {
		fmt.Println("--> ", string(request[:230]), "... \"}]}")
	}

	// http post
	req, err := http.NewRequest("POST", Url, bytes.NewBuffer(request))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
		return nil
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if len(body) < 300 {
		fmt.Println("<-- ", string(body))
	} else {
		fmt.Println("<-- ", string(body[:230]), "... \"}]}")
	}

	resp.Body.Close()
	return body
}

// rootPreRunE
func rootPreRunE(cmd *cobra.Command, args []string) error {
	var err error
	Url, err = cmd.Flags().GetString("url")
	if err != nil {
		utils.ErrorLog("failed to get 'home' path for the node")
		return err
	}
	WalletAddress, err = cmd.Flags().GetString("wallet")
	if err != nil {
		utils.ErrorLog("failed to get 'home' path for the node")
		return err
	}

	return nil
}

// main
func main() {
	rootCmd := &cobra.Command{
		Use:     "rpc_client",
		Short:   "rpc client for test purpose",
		PersistentPreRunE: rootPreRunE,
	}
	rootCmd.PersistentFlags().StringP("url", "u", DEFAULT_URL, "url to the RPC server, e.g. http://3.24.59.6:8235")
	rootCmd.PersistentFlags().StringP("wallet", "w", "", "wallet address to be used (default: the first wallet in folder ./account/)")

	putCmd := &cobra.Command{
		Use:     "put",
		Short:   "upload a file",
		RunE:    put,
	}

	getCmd := &cobra.Command{
		Use:     "get",
		Short:   "download a file",
		RunE:    get,
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "list files",
		RunE:    list,
	}

	shareCmd := &cobra.Command{
		Use:     "share",
		Short:   "share a file from uploaded files",
		RunE:    share,
	}

	listsharedCmd := &cobra.Command{
		Use:     "listshared",
		Short:   "list shared files",
		RunE:    listshare,
	}

	stopsharedCmd := &cobra.Command{
		Use:     "stopshare",
		Short:   "stop sharing a file",
		RunE:    stopshare,
	}

	getsharedCmd := &cobra.Command{
		Use:     "getshared",
		Short:   "download a shared file",
		RunE:    getshared,
	}

	rootCmd.AddCommand(putCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(shareCmd)
	rootCmd.AddCommand(listsharedCmd)
	rootCmd.AddCommand(stopsharedCmd)
	rootCmd.AddCommand(getsharedCmd)

	utils.NewDefaultLogger("./logs/stdout.log", true, true)

	err := rootCmd.Execute()
	if err != nil {
		utils.ErrorLog(err)
	}
}
