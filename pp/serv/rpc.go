package serv

import (
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/hex"
	"sync"
	"encoding/json"
	"time"
	"github.com/stratosnet/sds/msg/header"
	rpc_api "github.com/stratosnet/sds/pp/api/rpc"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/rpc"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	"github.com/stratosnet/stratos-chain/types"
	"github.com/tendermint/tendermint/libs/bech32"
)

const (
	// the length of request shall be shorter than 5242880 bytes
	// this equals 3932160 bytes after
	FILE_DATA_SAFE_SIZE = 3500000

	// timeout for waiting result from external source, in seconds
	WAIT_TIMEOUT time.Duration = 3
)


var (
	// key: fileHash value: file
	FileOffset = make(map[string]*FileFetchOffset)
	FileOffsetMutex sync.Mutex
)

type FileFetchOffset struct {
	RemoteRequested    uint64
	ResourceNodeAsked  uint64
}

type rpcApi struct {

}

func RpcApi() *rpcApi {
	return &rpcApi{}
}

// apis returns the collection of built-in RPC APIs.
func apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "user",
			Version:   "1.0",
			Service:   RpcApi(),
			Public:    true,
		},
	}
}

func ResultHook(r rpc_api.Result, fileHash string) rpc_api.Result {
	if r.Return == rpc_api.UPLOAD_DATA {
		start := *r.OffsetStart
		end := *r.OffsetEnd
		// have to cut the requested data block into smaller pieces when the size is greater than the limit
		if end - start > FILE_DATA_SAFE_SIZE {
			f := &FileFetchOffset{RemoteRequested: start + FILE_DATA_SAFE_SIZE, ResourceNodeAsked: end}

			FileOffsetMutex.Lock()
			FileOffset[fileHash] = f
			FileOffsetMutex.Unlock()

			e := start + FILE_DATA_SAFE_SIZE
			nr := rpc_api.Result {
				Return: r.Return,
				OffsetStart: &start,
				OffsetEnd: &e,
			}
			return nr
		}
	}
	return r
}

func (api *rpcApi) RequestUpload(param rpc_api.ParamReqUploadFile) rpc_api.Result {
	fileName := param.FileName
	fileSize := param.FileSize
	fileHash := param.FileHash
	walletAddr := param.WalletAddr
	pubkey := param.WalletPubkey
	signature := param.Signature

	size:= fileSize

	// the input for signature is hashed by SHA256
	hs := sha256.Sum256([]byte(fileHash + walletAddr))
	ds, _ := hex.DecodeString(signature)

	// decode public key
	pubPref, pubkey64, err := bech32.DecodeAndConvert(pubkey)
	if pubPref != types.AccountPubKeyPrefix || err != nil {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	pk, e := b64.StdEncoding.DecodeString(string(pubkey64))
	if e != nil {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}
	if !secp256k1.VerifySignature(pk, hs[:], ds) {
		return rpc_api.Result{Return: rpc_api.SIGNATURE_FAILURE}
	}

	// start to upload file
	p := requests.RequestUploadFile(fileName, fileHash, uint64(size),  "rpc", walletAddr, false)
	peers.SendMessageToSPServer(p, header.ReqUploadFile)

	var result rpc_api.Result
	var found bool
	var done = make(chan bool)

	go func() {
		for {
			result, found = file.GetRemoteFileEvent(fileHash)
			if found {
				result = ResultHook(result, fileHash)
				done <- true
				return
			}
		}
	}()

	select {
	case <-time.After(WAIT_TIMEOUT * time.Second):
		utils.DebugLog("TO QUIT TIMEOUT")
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case <-done:
		mj, _ := json.Marshal(&result)
		utils.DebugLog("Marshal result:", string(mj))

		return result
	}
}

func (api *rpcApi) UploadData(param rpc_api.ParamUploadData) rpc_api.Result {

	content := param.Data
	fileHash := param.FileHash
	// content in base64
	dec, _ := b64.StdEncoding.DecodeString(content)

	file.SendFileDataBack(fileHash, dec)

	// first part: if the amount of bytes server requested haven't been finished,
	// go on asking from the client
	FileOffsetMutex.Lock()
	fo, found := FileOffset[fileHash]
	FileOffsetMutex.Unlock()
	if found {
		if fo.ResourceNodeAsked - fo.RemoteRequested > FILE_DATA_SAFE_SIZE {
			start := fo.RemoteRequested
			end := fo.RemoteRequested + FILE_DATA_SAFE_SIZE
			nr := rpc_api.Result{
				Return: rpc_api.UPLOAD_DATA,
				OffsetStart: &start,
				OffsetEnd: &end,
			}

			FileOffsetMutex.Lock()
			FileOffset[fileHash].RemoteRequested = fo.RemoteRequested + FILE_DATA_SAFE_SIZE
			FileOffsetMutex.Unlock()
			return nr
		} else {
			nr := rpc_api.Result{
				Return: rpc_api.UPLOAD_DATA,
				OffsetStart: &fo.RemoteRequested,
				OffsetEnd: &fo.ResourceNodeAsked,
			}

			FileOffsetMutex.Lock()
			delete(FileOffset, fileHash)
			FileOffsetMutex.Unlock()
			return nr
		}
	}

	// second part: let the server decide what will be the next step
	var result rpc_api.Result
	var done = make(chan bool)

	go func() {
		for {
			result, found = file.GetRemoteFileEvent(fileHash)
			if found {
				result = ResultHook(result, fileHash)
				done <- true
				return
			}
		}
	}()

	select {
	case <-time.After(WAIT_TIMEOUT * time.Second):
		return rpc_api.Result{Return: rpc_api.TIME_OUT}
	case <-done:
		return result
	}
}

type test struct {
	FileName     string    `json:"filename"`
	FileSize     int       `json:"filesize"`
	FileHash     string    `json:"filehash"`
	WalletAddr   string    `json:"walletaddr"`
	WalletPubkey string    `json:"walletpubkey"`
	Signature    string    `json:"signatur"`
}

func (api *rpcApi) Test(t test) []string {

	utils.DebugLog("hello, the world", t.FileName, t.FileHash, t.WalletAddr)
	return []string{"0"}
}
