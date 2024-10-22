package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/sds-msg/protos"
	msgutils "github.com/stratosnet/sds/sds-msg/utils"
)

func reqUploadMsg(filePath, hash, sn string) []byte {
	// file size
	info, err := file.GetFileInfo(filePath)
	if err != nil {
		utils.ErrorLog("Failed to get file information.", err.Error())
		return nil
	}
	fileName := info.Name()
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.GetFileUploadWalletSignMessage(hash, WalletAddress, sn, nowSec)))
	if err != nil {
		return nil
	}

	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}
	// param
	var params = []rpc.ParamReqUploadFile{}
	params = append(params, rpc.ParamReqUploadFile{
		FileName: fileName,
		FileSize: int(info.Size()),
		FileHash: hash,
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime:         nowSec,
		DesiredTier:     2,
		AllowHigherTier: true,
		SequenceNumber:  sn,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqUploadFile")
		return nil
	}

	return wrapJsonRpc("user_requestUpload", pm)
}

func uploadDataMsg(hash, data, sn string) []byte {
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.GetFileUploadWalletSignMessage(hash, WalletAddress, sn, nowSec)))
	if err != nil {
		return nil
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}

	pa := []rpc.ParamUploadData{{
		FileHash: hash,
		Data:     data,
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime:        nowSec,
		SequenceNumber: sn,
	}}
	pm, e := json.Marshal(pa)
	if e != nil {
		utils.ErrorLog("json marshal error", e)
	}

	return wrapJsonRpc("user_uploadData", pm)
}

func put(cmd *cobra.Command, args []string) error {
	sn, err := handleGetOzone()
	if err != nil {
		return err
	}

	// args[0] is the first param, instead of the subcommand "put"
	filePath := args[0]
	hash := file.GetFileHash(filePath, "")
	utils.Log("- start uploading the file:", filePath)

	// compose request file upload params
	r := reqUploadMsg(filePath, hash, sn)
	if r == nil {
		return nil
	}

	utils.Log("- request uploading file (method: user_requestUpload)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("http no response")
		return nil
	}

	// handle: unmarshal response then unmarshal result
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		utils.ErrorLog("unmarshal failed")
		return nil
	}

	var res rpc.Result
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		utils.ErrorLog("unmarshal failed")
		return nil
	}

	// Handle result:1 sending the content
	for res.Return == rpc.UPLOAD_DATA {
		utils.Log("- received response (return: UPLOAD_DATA)")
		// get the data from the file
		so := &protos.SliceOffset{
			SliceOffsetStart: *res.OffsetStart,
			SliceOffsetEnd:   *res.OffsetEnd,
		}
		rawData, err := file.GetFileData(filePath, so)
		if err != nil {
			utils.ErrorLog("failed reading file data", err.Error())
			return nil
		}
		encoded := base64.StdEncoding.EncodeToString(rawData)
		r = uploadDataMsg(hash, encoded, sn)
		utils.Log("- request upload date (method: user_uploadData)")
		body = httpRequest(r)
		if body == nil {
			utils.ErrorLog("json marshal error")
			return nil
		}

		// Handle rsp
		err = json.Unmarshal(body, &rsp)
		if err != nil {
			utils.ErrorLog("unmarshal failed")
			return nil
		}
		err = json.Unmarshal(rsp.Result, &res)
		if err != nil {
			utils.ErrorLog("unmarshal failed")
			return nil
		}
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	utils.Log("- uploading is done")
	return nil
}

func reqUploadStreamMsg(filePath, hash, sn string) []byte {
	// file size
	info, err := file.GetFileInfo(filePath)
	if err != nil {
		utils.ErrorLog("Failed to get file information.", err.Error())
		return nil
	}
	fileName := info.Name()
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.GetFileUploadWalletSignMessage(hash, WalletAddress, sn, nowSec)))
	if err != nil {
		return nil
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}
	// param
	var params = []rpc.ParamReqUploadFile{}
	params = append(params, rpc.ParamReqUploadFile{
		FileName: fileName,
		FileSize: int(info.Size()),
		FileHash: hash,
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime:         nowSec,
		DesiredTier:     2,
		AllowHigherTier: true,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqUploadFile")
		return nil
	}

	return wrapJsonRpc("user_requestUploadStream", pm)
}

func putstream(cmd *cobra.Command, args []string) error {
	sn, err := handleGetOzone()
	if err != nil {
		return err
	}

	// args[0] is the first param, instead of the subcommand "put"
	fileName := args[0]
	hash := file.GetFileHashForVideoStream(args[0], "")
	utils.Log("- start uploading stream video:", fileName)

	// compose request file upload params
	r := reqUploadStreamMsg(args[0], hash, sn)
	if r == nil {
		return nil
	}

	utils.Log("- request uploading video file (method: user_requestUploadStream)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("http no response")
		return nil
	}

	// handle: unmarshal response then unmarshal result
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		utils.ErrorLog("unmarshal failed")
		return nil
	}

	var res rpc.Result
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		utils.ErrorLog("unmarshal failed")
		return nil
	}

	// Handle result:1 sending the content
	for res.Return == rpc.UPLOAD_DATA {
		utils.Log("- received response (return: UPLOAD_DATA)")
		// get the data from the file
		so := &protos.SliceOffset{
			SliceOffsetStart: *res.OffsetStart,
			SliceOffsetEnd:   *res.OffsetEnd,
		}
		rawData, err := file.GetFileData(fileName, so)
		if err != nil {
			utils.ErrorLog("failed getting file data ", err.Error())
			return nil
		}
		encoded := base64.StdEncoding.EncodeToString(rawData)
		r = uploadDataMsg(hash, encoded, sn)
		utils.Log("- request upload date (method: user_uploadDataStream)")
		body = httpRequest(r)
		if body == nil {
			utils.ErrorLog("json marshal error")
			return nil
		}

		// Handle rsp
		err = json.Unmarshal(body, &rsp)
		if err != nil {
			utils.ErrorLog("unmarshal failed")
			return nil
		}
		err = json.Unmarshal(rsp.Result, &res)
		if err != nil {
			utils.ErrorLog("unmarshal failed")
			return nil
		}
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	utils.Log("- uploading is done")
	return nil
}

func getFileStatus(_ *cobra.Command, args []string) error {
	fileHash := args[0]

	timestamp := time.Now().Unix()
	signature, err := WalletPrivateKey.Sign([]byte(msgutils.GetFileStatusWalletSignMessage(fileHash, WalletAddress, timestamp)))
	if err != nil {
		return err
	}

	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return err
	}

	params := []rpc.ParamGetFileStatus{{
		FileHash: fileHash,
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(signature),
		},
		ReqTime: timestamp,
	}}
	paramsBytes, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("json marshal error", e)
	}

	req := wrapJsonRpc("user_getFileStatus", paramsBytes)

	// http request-respond
	body := httpRequest(req)
	if body == nil {
		utils.ErrorLog("http no response")
		return nil
	}

	// handle: unmarshal response then unmarshal result
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		utils.ErrorLog("unmarshal failed")
		return nil
	}

	var res rpc.FileStatusResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		utils.ErrorLog("unmarshal failed")
		return nil
	}

	utils.Logf("File status:  %v", string(rsp.Result))
	return nil
}

func reqDownloadMsg(hash, sdmPath, sn string) []byte {
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.GetFileDownloadWalletSignMessage(hash, WalletAddress, sn, nowSec)))
	if err != nil {
		return nil
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}

	// param
	var params = []rpc.ParamReqDownloadFile{}
	params = append(params, rpc.ParamReqDownloadFile{
		FileHandle: sdmPath,
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime: nowSec,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_requestDownload", pm)
}

func downloadDataMsg(hash, reqid string) []byte {
	// param
	params := []rpc.ParamDownloadData{{
		FileHash: hash,
		ReqId:    reqid,
	}}

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_downloadData", pm)
}

func downloadedFileInfoMsg(fileHash string, fileSize uint64, reqid string) []byte {
	// param
	params := []rpc.ParamDownloadFileInfo{{
		FileHash: fileHash,
		FileSize: fileSize,
		ReqId:    reqid,
	}}

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_downloadedFileInfo", pm)
}

func get(cmd *cobra.Command, args []string) error {
	sn, err := handleGetOzone()
	if err != nil {
		return err
	}
	// args[0] is the fileHash
	sdmPath := args[0]
	utils.Log("- start downloading the file: ", sdmPath)
	_, _, fileHash, _, err := fwtypes.ParseFileHandle(sdmPath)
	if err != nil {
		utils.ErrorLog("sdm format error")
		return nil
	}

	// compose "request file download" request
	r := reqDownloadMsg(fileHash, sdmPath, sn)
	if r == nil {
		return nil
	}
	utils.Log("- request downloading the file (method: user_requestDownload)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}

	// handle rsp
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	var res rpc.Result
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		utils.ErrorLog("unmarshal failed")
		return nil
	}

	var fileSize uint64 = 0
	var pieceCount uint64 = 0
	var fileMg *os.File = nil
	// Handle result:1 sending the content
	for res.Return == rpc.DOWNLOAD_OK || res.Return == rpc.DL_OK_ASK_INFO {
		if fileMg == nil {
			exist, err := file.PathExists("./download/")
			if err != nil {
				return err
			}
			if !exist {
				if err = os.MkdirAll("./download/", 0700); err != nil {
					return err
				}
			}
			fileMg, err = os.OpenFile(filepath.Join("./download/", res.FileName), os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				utils.ErrorLog("error initialize file")
				return errors.New("can't open file")
			}
		}

		if res.Return == rpc.DL_OK_ASK_INFO {
			utils.Log("- received response (return: DL_OK_ASK_INFO) after received", pieceCount, "piece(s)")
			r = downloadedFileInfoMsg(fileHash, fileSize, res.ReqId)
			utils.Log("- request file information verification (method: user_downloadedFileInfo)")
		} else {
			// rpc.DOWNLOAD_OK
			utils.Log("- received response (return: DOWNLOAD_OK)")
			start := *res.OffsetStart
			end := *res.OffsetEnd
			fileSize = fileSize + (end - start)
			decoded, _ := base64.StdEncoding.DecodeString(res.FileData)
			if len(decoded) != int(end-start) {
				utils.ErrorLog("Wrong size:", strconv.Itoa(len(decoded)), " ", strconv.Itoa(int(end-start)))
				fileMg.Close()
				return nil
			}
			pieceCount = pieceCount + 1

			_, err = fileMg.WriteAt(decoded, int64(start))
			if err != nil {
				utils.ErrorLog("error save file")
				fileMg.Close()
				return errors.New("failed writing file")
			}

			r = downloadDataMsg(fileHash, res.ReqId)
			utils.Log("- request downloading file date (user_downloadData)")
		}

		body := httpRequest(r)
		if body == nil {
			utils.ErrorLog("json marshal error")
			fileMg.Close()
			return nil
		}

		// Handle rsp
		err := json.Unmarshal(body, &rsp)
		if err != nil {
			return nil
		}

		err = json.Unmarshal(rsp.Result, &res)
		if err != nil {
			utils.ErrorLog("unmarshal failed")
			fileMg.Close()
			return nil
		}
	}
	if fileMg != nil {
		fileMg.Close()
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	utils.Log("- downloading is done")
	return nil
}

func reqDeleteMsg(hash string) []byte {
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.DeleteShareWalletSignMessage(hash, WalletAddress, nowSec)))
	if err != nil {
		return nil
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}

	// param
	var params = []rpc.ParamReqDeleteFile{}
	params = append(params, rpc.ParamReqDeleteFile{
		FileHash: hash,
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime: nowSec,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_requestDeleteFile", pm)
}

func delete(cmd *cobra.Command, args []string) error {
	// compose "request file download" request
	fileHash := args[0]
	r := reqDeleteMsg(fileHash)
	if r == nil {
		return nil
	}
	utils.Log("- request deleting the file (method: user_requestDeleteFile)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
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
		utils.ErrorLog("unmarshal failed")
		return nil
	}

	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		utils.ErrorLog("unmarshal failed")
		return nil
	}

	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	utils.Log("- downloading is done")
	return nil
}

func reqListMsg(page uint64) []byte {
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.FindMyFileListWalletSignMessage(WalletAddress, nowSec)))
	if err != nil {
		return nil
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}
	// param
	var params = []rpc.ParamReqFileList{}
	params = append(params, rpc.ParamReqFileList{
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime: nowSec,
		PageId:  page,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqFileList")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_requestList", pm)
}

func reqRpMsg() []byte {
	// param
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.RegisterNewPPWalletSignMessage(WalletAddress, nowSec)))
	if err != nil {
		return nil
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}
	// param
	params := make([]rpc.ParamReqRP, 0)
	params = append(params, rpc.ParamReqRP{
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime: nowSec,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqRP")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("owner_requestRegisterNewPP", pm)
}

func reqActivateMsg(deposit, fee string, gas uint64) []byte {
	// param
	params := []rpc.ParamReqActivate{{
		WalletAddr: WalletAddress,
		Deposit:    deposit,
		Fee:        fee,
		Gas:        gas,
	}}

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqActivate")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("owner_requestActivate", pm)
}

func reqPrepayMsg(prepayAmount, fee string, gasUint64 uint64) []byte {
	// param
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.PrepayWalletSignMessage(WalletAddress, nowSec)))
	if err != nil {
		return nil
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}

	// param
	params := make([]rpc.ParamReqPrepay, 0)
	params = append(params, rpc.ParamReqPrepay{
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime:      nowSec,
		PrepayAmount: prepayAmount,
		Fee:          fee,
		Gas:          gasUint64,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqPrepay")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("owner_requestPrepay", pm)
}

func reqStartMiningMsg() []byte {
	// param
	params := []rpc.ParamReqStartMining{{
		WalletAddr: WalletAddress,
	}}

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqStartMining")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("owner_requestStartMining", pm)
}

func reqStatusMsg() []byte {
	// param
	params := []rpc.ParamReqStatus{{
		WalletAddr: WalletAddress,
	}}

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqStatus")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("owner_requestStatus", pm)
}

func reqServiceStatusMsg() []byte {
	// param
	params := []rpc.ParamReqServiceStatus{{
		WalletAddr: WalletAddress,
	}}

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqServiceStatus")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_requestServiceStatus", pm)
}

func reqWithdrawMsg(amount, targetAddress, fee string, gasUint64 uint64) []byte {
	params := []rpc.ParamReqWithdraw{{
		Amount:        amount,
		TargetAddress: targetAddress,
		Fee:           fee,
		Gas:           gasUint64,
	}}

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqWithdraw")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("owner_requestWithdraw", pm)
}

func reqSendMsg(amount, toAddress, fee string, gasUint64 uint64) []byte {
	params := []rpc.ParamReqSend{{
		Amount: amount,
		To:     toAddress,
		Fee:    fee,
		Gas:    gasUint64,
	}}

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqSend")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("owner_requestSend", pm)
}

func printFileList(res rpc.FileListResult) {
	if res.Return == rpc.SUCCESS {
		fmt.Printf("\n%-20s %-41s %-9s %-8s\n", "File Name", "File Hash", "File Size", "Create Time")
		fmt.Printf("_____________________________________________________________________________________\n")
		for i := range res.FileInfo {
			f := res.FileInfo[i]
			fmt.Printf("%-20s %-25s %10d %8d\n", f.FileName, f.FileHash, f.FileSize, f.CreateTime)
		}
		fmt.Printf("_____________________________________________________________________________________\n")
		fmt.Printf("Total: %d\tPage: %d\n\n", res.TotalNumber, res.PageId)
	}
}

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
	utils.Log("- request listing files (method: user_requestList)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
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
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	printFileList(res)
	return nil
}

func reqGetOzoneMsg() []byte {
	// param
	params := []rpc.ParamReqGetOzone{{
		WalletAddr: WalletAddress,
	}}

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqUGetOzone")
		return nil
	}

	// wrap to the json-rpc message
	return wrapJsonRpc("user_requestGetOzone", pm)
}

func handleGetOzone() (string, error) {
	// compose "request get ozone" request
	r := reqGetOzoneMsg()
	if r == nil {
		return "", errors.New("failed composing message")
	}

	utils.Log("- request ozone balance (method: user_requestGetOzone)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return "", errors.New("failed request")
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
		return "", errors.New("failed unmarshal response")
	}

	var res rpc.GetOzoneResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		utils.ErrorLog("unmarshal failed")
		return "", errors.New("failed unmarshal result")
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
		ozone, _ := strconv.ParseFloat(res.Ozone, 64)
		utils.Log("OZONE balance: ", ozone/1000000000.0)
		utils.Log("SN:            ", res.SequenceNumber)
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}

	return res.SequenceNumber, nil
}

func getozone(cmd *cobra.Command, args []string) error {
	_, err := handleGetOzone()
	return err
}

func printSharedFileList(res rpc.FileShareResult) {
	if res.Return == rpc.SUCCESS {
		fmt.Printf("\n%-20s %-41s %-9s %-8s  %-8s   %-15s  %-15s\n", "File Name", "File Hash", "File Size", "Link Time", "Link Exp", "Share ID", "Share Link")
		fmt.Printf("________________________________________________________________________________________________________________________________________\n")
		for i := range res.FileInfo {
			f := res.FileInfo[i]
			fmt.Printf("%-20s %-25s %10d %8d %8d %-15s %-15s\n", f.FileName, f.FileHash, f.FileSize, f.LinkTime, f.LinkTimeExp, f.ShareId, f.ShareLink)
		}
		fmt.Printf("________________________________________________________________________________________________________________________________________\n")
		fmt.Printf("Total: %d\tPage: %d\n\n", res.TotalNumber, res.PageId)
	}
}

func reqListShareMsg(page uint64) []byte {
	// param
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.FindMyFileListWalletSignMessage(WalletAddress, nowSec)))
	if err != nil {
		return nil
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}
	// param
	var params = []rpc.ParamReqListShared{}
	params = append(params, rpc.ParamReqListShared{
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime: nowSec,
		PageId:  page,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap into request message
	return wrapJsonRpc("user_requestListShare", pm)
}

func listshare(cmd *cobra.Command, args []string) error {
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
	utils.Log("- request listing files (method: user_requestListShare)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
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
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	printSharedFileList(res)
	return nil
}

func reqShareMsg(hash string) []byte {
	// param
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.GetShareFileWalletSignMessage(hash, WalletAddress, nowSec)))
	if err != nil {
		return nil
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}
	// param
	var params = []rpc.ParamReqShareFile{}
	params = append(params, rpc.ParamReqShareFile{
		FileHash: hash,
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime: nowSec,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqUploadFile")
		return nil
	}

	// wrap into request message
	return wrapJsonRpc("user_requestShare", pm)
}

func share(cmd *cobra.Command, args []string) error {
	// check input
	if len(args) != 1 {
		utils.ErrorLog("file hash is not provided")
		return nil
	}
	// compose request
	r := reqShareMsg(args[0])
	if r == nil {
		return nil
	}
	utils.Log("- request sharing file (method: user_requestShare)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
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
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
		fmt.Println("ShareId: ", res.ShareId)
		fmt.Println("ShareLink: ", res.ShareLink)
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	return nil
}

func reqStopShareMsg(shareId string) []byte {
	// param
	nowSec := time.Now().Unix()
	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.DeleteShareWalletSignMessage(shareId, WalletAddress, nowSec)))
	if err != nil {
		return nil
	}
	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}
	// param
	var params = []rpc.ParamReqStopShare{}
	params = append(params, rpc.ParamReqStopShare{
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime: nowSec,
		ShareId: shareId,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqStopShare")
		return nil
	}

	// wrap into request message
	return wrapJsonRpc("user_requestStopShare", pm)
}

func stopshare(cmd *cobra.Command, args []string) error {

	// compose request
	r := reqStopShareMsg(args[0])
	if r == nil {
		return nil
	}
	utils.Log("- request stop sharing (method: user_requestStopShare)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
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
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}

	return nil
}

func reqGetSharedMsg(shareLink, sn string) []byte {

	wpk, err := fwtypes.WalletPubKeyToBech32(WalletPublicKey)
	if err != nil {
		return nil
	}

	// param
	nowSec := time.Now().Unix()

	parsedLink, err := fwtypes.ParseShareLink(shareLink)
	if err != nil {
		utils.ErrorLog("failed to parse share link")
		return nil
	}

	// signature
	sign, err := WalletPrivateKey.Sign([]byte(msgutils.GetDownloadShareFileWalletSignMessage(parsedLink.Link, WalletAddress, sn, nowSec)))
	if err != nil {
		return nil
	}
	// param
	var params = []rpc.ParamReqGetShared{}
	params = append(params, rpc.ParamReqGetShared{
		Signature: rpc.Signature{
			Address:   WalletAddress,
			Pubkey:    wpk,
			Signature: hex.EncodeToString(sign),
		},
		ReqTime:   nowSec,
		ShareLink: shareLink,
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.ErrorLog("failed marshal param for ReqStopShare")
		return nil
	}

	// wrap into request message
	return wrapJsonRpc("user_requestGetShared", pm)
}

func getshared(cmd *cobra.Command, args []string) error {
	sn, err := handleGetOzone()
	if err != nil {
		return err
	}

	utils.Log("- start downloading the file:", args[0])
	// compose request: get shared file
	r := reqGetSharedMsg(args[0], sn)
	if r == nil {
		return nil
	}

	utils.Log("- request shared file information (method: user_requestGetShared)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}
	// handle response
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return err
	}
	var res rpc.Result
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return nil
	}

	fileHash := res.FileHash

	var fileSize uint64 = 0
	var pieceCount uint64 = 0
	var fileMg *os.File = nil
	// Handle result:1 sending the content
	for res.Return == rpc.DOWNLOAD_OK || res.Return == rpc.DL_OK_ASK_INFO {
		exist, err := file.PathExists("./download/")
		if err != nil {
			return err
		}
		if !exist {
			if err = os.MkdirAll("./download/", 0700); err != nil {
				return err
			}
		}
		if fileMg == nil {
			fileMg, err = os.OpenFile(filepath.Join("./download/", res.FileName), os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				utils.ErrorLog("error initialize file")
				return errors.New("can't open file")
			}
		}
		if res.Return == rpc.DL_OK_ASK_INFO {
			utils.Log("- received response (return: DL_OK_ASK_INFO)")
			r = downloadedFileInfoMsg(fileHash, fileSize, res.ReqId)
			utils.Log("- request file information verification (method: user_downloadedFileInfo)")
		} else {
			utils.Log("- received response (return: DOWNLOAD_OK)")
			start := *res.OffsetStart
			end := *res.OffsetEnd
			fileSize = fileSize + (end - start)
			decoded, _ := base64.StdEncoding.DecodeString(res.FileData)
			if len(decoded) != int(end-start) {
				utils.ErrorLog("Wrong size:", strconv.Itoa(len(decoded)), " ", strconv.Itoa(int(end-start)))
				fileMg.Close()
				return nil
			}
			pieceCount = pieceCount + 1
			_, err = fileMg.WriteAt(decoded, int64(start))
			if err != nil {
				utils.ErrorLog("error save file")
				fileMg.Close()
				return errors.New("failed writing file")
			}
			r = downloadDataMsg(fileHash, res.ReqId)
			utils.Log("- request downloading file data (method: user_downloadData)")
		}

		body := httpRequest(r)
		if body == nil {
			utils.ErrorLog("json marshal error")
			fileMg.Close()
			return nil
		}

		// Handle rsp
		err = json.Unmarshal(body, &rsp)
		if err != nil {
			fileMg.Close()
			return nil
		}

		err = json.Unmarshal(rsp.Result, &res)
		if err != nil {
			utils.ErrorLog("unmarshal failed")
			fileMg.Close()
			return nil
		}
	}
	if fileMg != nil {
		fileMg.Close()
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}

	utils.Log("- downloading is done")
	return nil
}

func rp(cmd *cobra.Command, args []string) error {
	r := reqRpMsg()
	if r == nil {
		return nil
	}
	utils.Log("- request register new pp (method: owner_requestRP)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	var res rpc.RPResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return nil
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	return nil
}

func activate(cmd *cobra.Command, args []string) error {
	if len(args) != 3 {
		utils.ErrorLog("wrong number of arguments")
		return nil
	}
	deposit := args[0]
	fee := args[1]
	gas := args[2]
	gasUint64, err := strconv.ParseUint(gas, 10, 64)
	if err != nil {
		utils.ErrorLog("wrong number of gas")
		return nil
	}

	r := reqActivateMsg(deposit, fee, gasUint64)
	if r == nil {
		return nil
	}
	utils.Log("- request register new pp (method: owner_requestActivate)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	var res rpc.ActivateResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return nil
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	return nil
}

func prepay(cmd *cobra.Command, args []string) error {
	if len(args) != 3 {
		utils.ErrorLog("wrong number of arguments")
		return nil
	}
	prepayAmount := args[0]
	fee := args[1]
	gas := args[2]
	gasUint64, err := strconv.ParseUint(gas, 10, 64)
	if err != nil {
		utils.ErrorLog("wrong number of gas")
		return nil
	}

	r := reqPrepayMsg(prepayAmount, fee, gasUint64)
	if r == nil {
		return nil
	}
	utils.Log("- request prepay (method: owner_requestPrepay)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	var res rpc.PrepayResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return nil
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	return nil
}

func startmining(cmd *cobra.Command, args []string) error {
	r := reqStartMiningMsg()
	if r == nil {
		return nil
	}
	utils.Log("- request register pp (method: owner_requestStartMining)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	var res rpc.StartMiningResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return nil
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	return nil
}

func status(cmd *cobra.Command, args []string) error {
	r := reqStatusMsg()
	if r == nil {
		return nil
	}
	utils.Log("- request register pp (method: owner_requestStatus)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	var res rpc.StatusResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return nil
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
		utils.Log(res.Message)
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	return nil
}

func servicestatus(cmd *cobra.Command, args []string) error {
	r := reqServiceStatusMsg()
	if r == nil {
		return nil
	}
	utils.Log("- request register pp (method: owner_requestStatus)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err := json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	var res rpc.StatusResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return nil
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
		utils.Log(res.Message)
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	return nil
}

func withdraw(cmd *cobra.Command, args []string) error {
	if len(args) != 4 {
		utils.ErrorLog("wrong number of arguments")
		return nil
	}
	amount := args[0]
	targetAddress := args[1]
	fee := args[2]
	gas := args[3]
	gasUint64, err := strconv.ParseUint(gas, 10, 64)
	if err != nil {
		utils.ErrorLog("wrong number of gas")
		return nil
	}

	r := reqWithdrawMsg(amount, targetAddress, fee, gasUint64)
	if r == nil {
		return nil
	}
	utils.Log("- request withdraw (method: owner_requestWithdraw)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	var res rpc.WithdrawResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return nil
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	return nil
}

func send(cmd *cobra.Command, args []string) error {
	if len(args) != 4 {
		utils.ErrorLog("wrong number of arguments")
		return nil
	}
	to := args[0]
	amount := args[1]
	fee := args[2]
	gas := args[3]
	gasUint64, err := strconv.ParseUint(gas, 10, 64)
	if err != nil {
		utils.ErrorLog("wrong number of gas")
		return nil
	}

	r := reqSendMsg(amount, to, fee, gasUint64)
	if r == nil {
		return nil
	}
	utils.Log("- request send (method: owner_requestSend)")
	// http request-respond
	body := httpRequest(r)
	if body == nil {
		utils.ErrorLog("json marshal error")
		return nil
	}

	// Handle rsp
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil
	}

	var res rpc.SendResult
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return nil
	}
	if res.Return == rpc.SUCCESS {
		utils.Log("- received response (return: SUCCESS)")
	} else {
		utils.Log("- received response (return: ", res.Return, ")")
	}
	return nil
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

func httpRequest(request []byte) []byte {
	if len(request) < 300 {
		utils.DebugLog("--> ", string(request))
	} else {
		utils.DebugLog("--> ", string(request[:230]), "... \"}]}")
	}

	// http post
	req, err := http.NewRequest("POST", Url, bytes.NewBuffer(request))
	if err != nil {
		panic(err)
	}
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) < 300 {
		utils.DebugLog("<-- ", string(body))
	} else {
		utils.DebugLog("<-- ", string(body[:230]), "... \"}]}")
	}

	resp.Body.Close()
	return body
}
