package api

import (
	"fmt"
	"testing"

	"github.com/ipfs/go-cid"
	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"

	"github.com/stratosnet/sds/framework/crypto"
	fwed25519 "github.com/stratosnet/sds/framework/crypto/ed25519"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/sds-msg/header"
	"github.com/stratosnet/sds/sds-msg/protos"
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

	spP2pPrivateKey := fwed25519.GenPrivKey()
	spP2pAddr := spP2pPrivateKey.PubKey().Address()
	spP2pAddrString := fwtypes.P2PAddressBytesToBech32(spP2pAddr.Bytes())
	if spP2pAddrString == "" {
		t.Fatal(fmt.Errorf("p2p address convert failed"))
	}

	spP2pPubKey := spP2pPrivateKey.PubKey()
	spP2pPubKeyString, err := fwtypes.P2PPubKeyToBech32(spP2pPubKey)
	if err != nil {
		t.Fatal(err)
	}
	setting.SPMap.Store(spP2pAddrString, setting.SPBaseInfo{P2PPublicKey: spP2pPubKeyString})

	data := []byte("some kind of data")

	filehash, _ := mh.Sum(data, mh.KECCAK_256, 20)
	fileCid := cid.NewCidV1(uint64(crypto.VIDEO_CODEC), filehash)
	encoder, _ := mbase.NewEncoder(mbase.Base32hex)
	fh := fileCid.Encode(encoder)
	reqBody := &StreamReqBody{
		FileHash:     fh,
		P2PAddress:   "d4c3b2a1",
		SpP2pAddress: spP2pAddrString,
		Sign:         nil,
		SliceInfo:    &protos.DownloadSliceInfo{SliceNumber: 0},
	}
	sliceHash, err := crypto.CalcSliceHash(data, reqBody.FileHash, reqBody.SliceInfo.SliceNumber)
	if err != nil {
		t.Fatal(err)
	}

	toSign := []byte(reqBody.P2PAddress + reqBody.FileHash + header.ReqDownloadSlice.Name)
	signature, err := spP2pPrivateKey.Sign(toSign)
	if err != nil {
		t.Fatal(err)
	}

	reqBody.Sign = signature

	return reqBody, sliceHash, data
}
