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
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/stratos-chain/types"
	"github.com/tendermint/tendermint/libs/bech32"
)

var (
	WalletPrivateKey []byte
	WalletPublicKey  string
	WalletAddress    string
)

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      int             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func readWalletKeys() bool {
	WalletAddress = "st1macvxhdy33kphmwv7kvvk28hpg0xn7nums5klu"
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

func main() {
	args := os.Args
	if len(args[1:]) != 1 {
		fmt.Println("usage: ", args[0], " file")
		return
	}

	// initialize log
	utils.NewDefaultLogger("./logs/stdout.log", true, true)

	ret := readWalletKeys()
	if !ret {
		utils.DebugLog("Failed reading key file.")
		return
	}
	// size
	info := file.GetFileInfo(args[1])
	if info == nil {
		utils.DebugLog("Failed to get file information.")
		return
	}

	// hash
	hash := file.GetFileHash(args[1], "")

	// signature
	hs := sha256.Sum256([]byte(hash + WalletAddress))
	sign, err := secp256k1.Sign(hs[:], WalletPrivateKey)
	if err != nil {
		utils.DebugLog("failed to sign, error:", err)
		return
	}
	signature := sign[:len(sign)-1]

	var params = []rpc.ParamReqUploadFile{}
	params = append(params, rpc.ParamReqUploadFile {
		FileName      : args[1],
		FileSize      : int(info.Size()),
		FileHash      : hash,
		WalletAddr    : WalletAddress,
		WalletPubkey  : WalletPublicKey,
		Signature     : hex.EncodeToString(signature),
	})

	pm, e := json.Marshal(params)
	if e != nil {
		utils.DebugLog("failed marshal param for ReqUploadFile")
		return
	}

	request := &jsonrpcMessage {
		Version: "2.0",
			ID:  1,
		 Method: "user_requestUpload",
		 Params: json.RawMessage(pm),
	 }

	r, e := json.Marshal(&request)
	if e != nil {
		utils.DebugLog("failed marshal ReqUploadFile", e)
		return
	}

	// http post
	fmt.Println("--> ", string(r))

	url := "http://127.0.0.1:8235"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(r))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("<-- ", string(body))
	resp.Body.Close()

	// Handle rsp
	var rsp jsonrpcMessage
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return
	}

	var res rpc.Result
	err = json.Unmarshal(rsp.Result, &res)
	if err != nil {
		return
	}

	// Handle result:1 sending the content
	for res.Return == rpc.UPLOAD_DATA {
		request.Method = "user_uploadData"

		// get the data from the file
		so := &protos.SliceOffset {
			SliceOffsetStart: *res.OffsetStart,
			SliceOffsetEnd: *res.OffsetEnd,
		}

		rawData := file.GetFileData(args[1], so)
		encoded := base64.StdEncoding.EncodeToString(rawData)

		var pa = []rpc.ParamUploadData{}
		pa = append(pa, rpc.ParamUploadData {
			FileHash      : hash,
			Data          : encoded,
		})
		pm, e := json.Marshal(pa)
		if e != nil {
			utils.DebugLog("json marshal error", e)
		}

		request.Params = pm
		r, e := json.Marshal(request)
		if e != nil {
			utils.DebugLog("json marshal error", e)
		}

		// http post again
		if len(r) > 500 {
			fmt.Println("--> ", string(r[:130]), "... \"}]}")
		}

		req, err = http.NewRequest("POST", url, bytes.NewBuffer(r))
		req.Header.Set("X-Custom-Header", "myvalue")
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		if err != nil {
			panic(err)
		}

		body, _ = ioutil.ReadAll(resp.Body)
		fmt.Println("<-- ", string(body))
		resp.Body.Close()

		// Handle rsp
		err = json.Unmarshal(body, &rsp)
		if err != nil {
			return
		}

		err = json.Unmarshal(rsp.Result, &res)
		if err != nil {
			return
		}
	}

	return
}
