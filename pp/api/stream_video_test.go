package api

import (
	ed25519crypto "crypto/ed25519"
	"crypto/md5"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	//"github.com/tendermint/tendermint/libs/bech32"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

func init() {
	utils.NewDefaultLogger("", false, false)
}

func TestVerifySignature(t *testing.T) {
	reqBody, sliceHash, data := setup(t)

	success := verifySignature(reqBody, sliceHash, data)
	if !success {
		t.Fatal("Invalid signature")
	}
}

func TestVerifySignatureMissingSPInfo(t *testing.T) {
	reqBody, sliceHash, data := setup(t)
	setting.SPMap.Delete(reqBody.SpP2pAddress)

	success := verifySignature(reqBody, sliceHash, data)
	if success {
		t.Fatal("Verify should have been false because SP info is missing in setting.SPMap")
	}
}

func setup(t *testing.T) (*StreamReqBody, string, []byte) {
	bechPrefix := "stsdsp2p"

	spP2pPrivateKey := ed25519.NewKey()
	spP2pAddr := ed25519.PrivKeyBytesToAddress(spP2pPrivateKey)
	spP2pAddrString, err := spP2pAddr.ToBech(bechPrefix)
	if err != nil {
		t.Fatal(err)
	}
	spP2pPubKey := ed25519.PrivKeyBytesToPubKey(spP2pPrivateKey)
	spP2pPubKeyString, err := bech32.ConvertAndEncode(bechPrefix, spP2pPubKey.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	setting.SPMap.Store(spP2pAddrString, setting.SPBaseInfo{P2PPublicKey: spP2pPubKeyString})

	data := []byte("some kind of data")
	reqBody := &StreamReqBody{
		FileHash:     utils.CalcFileHashFromData(md5.New().Sum(data), cid.Raw),
		P2PAddress:   "d4c3b2a1",
		SpP2pAddress: spP2pAddrString,
		Sign:         nil,
		SliceInfo:    &protos.DownloadSliceInfo{SliceNumber: 0},
	}
	sliceHash := utils.CalcSliceHash(data, reqBody.FileHash, reqBody.SliceInfo.SliceNumber)

	toSign := []byte(reqBody.P2PAddress + reqBody.FileHash + header.ReqDownloadSlice)
	signature := ed25519crypto.Sign(spP2pPrivateKey, toSign)
	reqBody.Sign = signature

	return reqBody, sliceHash, data
}
