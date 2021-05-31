package tools

import (
	"encoding/hex"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"

	"github.com/google/uuid"
)

// FileHandleLength
const FileHandleLength = 112

// ParseFileHandle
func ParseFileHandle(handle string) (protocol, walletAddress, fileHash, fileName string, err error) {
	handleInBytes := []byte(handle)

	if handle == "" || len(handle) < FileHandleLength {
		err = errors.New("handle is null or length is not correct")
		return
	}

	protocol = string(handleInBytes[:3])
	walletAddress = string(handleInBytes[6:47])
	fileHash = string(handleInBytes[48:112])

	if len(handle) > FileHandleLength+1 {
		fileName = string(handleInBytes[113:])
	}

	if protocol != "spb" ||
		walletAddress == "" || len(walletAddress) < 41 ||
		fileHash == "" || len(fileHash) < 64 {
		err = errors.New("file handle parse fail")
	}

	return
}

// GenerateTaskID
func GenerateTaskID(mix string) string {
	return utils.CalcHash([]byte(uuid.New().String() + "#" + strconv.FormatInt(time.Now().UnixNano(), 10) + mix))
}

// LoadOrCreateAccount
func LoadOrCreateAccount(path, pass string) string {

	if path == "" && pass == "" {
		utils.ErrorLog("missing privateKeyPath or privateKeyPass")
		return ""
	}

	p, _ := os.Stat(path)
	if p.IsDir() {
		files, _ := ioutil.ReadDir(path)
		if len(files) > 0 {
			path = filepath.Join(path, files[0].Name())
		}
	}

	privKeyInStr, err := ioutil.ReadFile(path)
	if err != nil {
		keyPath := filepath.Dir(path)
		account, err := utils.CreateAccount(keyPath, "", pass, "", "", "", "", 4096, 6)
		if utils.CheckError(err) {
			utils.ErrorLog("create account failed", err)
			return ""
		}
		privKeyInStr, _ = ioutil.ReadFile(keyPath + "/" + account.String())
	}

	privKey, err := utils.DecryptKey(privKeyInStr, pass)
	if err != nil {
		utils.ErrorLog("decrypt key failed", err)
		return ""
	}

	return hex.EncodeToString(secp256k1.PrivKeyToPubKey(privKey.PrivateKey))
}

// GenerateDownloadLink
func GenerateDownloadLink(walletAddress, fileHash string) string {
	return strings.Join([]string{"spb:/", walletAddress, fileHash}, "/")
}
