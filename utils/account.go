package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/cosmos/go-bip39"
	"github.com/pborman/uuid"
	"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	"github.com/stratosnet/sds/utils/types"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	keyHeaderKDF = "scrypt"
	scryptN      = 4096
	scryptP      = 6
	scryptR      = 8
	scryptDKLen  = 32

	version         = 3
	mnemonicEntropy = 256
)

type KeyStorePassphrase struct {
	KeysDirPath string
	ScryptN     int
	ScryptP     int
}

type encryptedKeyJSONV3 struct {
	Address string     `json:"address"`
	Name    string     `json:"name"`
	Crypto  cryptoJSON `json:"crypto"`
	Id      string     `json:"id"`
	Version int        `json:"version"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

type hdKeyBytes struct {
	HdPath     []byte
	Mnemonic   []byte
	Passphrase []byte
	PrivKey    []byte
}

// CreateWallet creates a new stratos-chain wallet with the given nickname and password, and saves the key data into the dir folder
func CreateWallet(dir, nickname, password, hrp, mnemonic, bip39Passphrase, hdPath string) (types.Address, error) {
	privateKey, err := keys.SecpDeriveKey(mnemonic, bip39Passphrase, hdPath)
	if err != nil {
		return types.Address{}, err
	}

	return saveAccountKey(dir, nickname, password, hrp, mnemonic, bip39Passphrase, hdPath, scryptN, scryptP, privateKey, true)
}

// CreateP2PKey creates a P2P key to be used by one of the SDS nodes, and saves the key data into the dir folder
func CreateP2PKey(dir, nickname, password, hrp string) (types.Address, error) {
	privateKey := ed25519.NewKey()

	return saveAccountKey(dir, nickname, password, hrp, "", "", "", scryptN, scryptP, privateKey, false)
}

func saveAccountKey(dir, nickname, password, hrp, mnemonic, bip39Passphrase, hdPath string, scryptN, scryptP int, privateKey []byte, isWallet bool) (types.Address, error) {
	keyStore := &KeyStorePassphrase{dir, scryptN, scryptP}

	key := newKeyFromBytes(privateKey, isWallet)
	key.Name = nickname
	key.HdPath = hdPath
	key.Mnemonic = mnemonic
	key.Passphrase = bip39Passphrase

	address, err := key.Address.ToBech(hrp)
	if err != nil {
		return types.Address{}, err
	}

	filename := dir + "/" + address
	if err = keyStore.StoreKey(filename, key, password); err != nil {
		zeroKey(key.PrivateKey)
		return types.Address{}, err
	}
	return key.Address, nil
}

func NewMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(mnemonicEntropy)
	if err != nil {
		return "", err
	}

	return bip39.NewMnemonic(entropy)
}

func newKeyFromBytes(privateKey []byte, isWallet bool) *AccountKey {
	id := uuid.NewRandom()
	key := &AccountKey{
		Id:         id,
		PrivateKey: privateKey,
	}
	if isWallet {
		key.Address = crypto.PrivKeyToAddress(privateKey)
	} else {
		key.Address = ed25519.PrivKeyBytesToAddress(privateKey)
	}
	return key
}

// EncryptKey encrypts a key using the specified scrypt parameters into a json
// blob that can be decrypted later on.
func EncryptKey(key *AccountKey, auth string) ([]byte, error) {
	authArray := []byte(auth)

	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]
	hdKeyBytesObject := hdKeyBytes{
		HdPath:     []byte(key.HdPath),
		Mnemonic:   []byte(key.Mnemonic),
		Passphrase: []byte(key.Passphrase),
		PrivKey:    key.PrivateKey,
	}
	hdKeyEncoded, err := msgpack.Marshal(hdKeyBytesObject)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, aes.BlockSize) // 16
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	cipherText, err := aesCTRXOR(encryptKey, hdKeyEncoded, iv)
	if err != nil {
		return nil, err
	}
	mac := crypto.Keccak256(derivedKey[16:32], cipherText)

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = scryptN
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = scryptP
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = hex.EncodeToString(salt)

	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	cryptoStruct := cryptoJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          hex.EncodeToString(mac),
	}
	encryptedKeyJSONV3 := encryptedKeyJSONV3{
		hex.EncodeToString(key.Address[:]),
		key.Name,
		cryptoStruct,
		key.Id.String(),
		version,
	}
	return json.Marshal(encryptedKeyJSONV3)
}

// DecryptKey decrypts a key from a json blob, returning the private key itself.
func DecryptKey(keyjson []byte, auth string) (*AccountKey, error) {
	// Parse the json into a simple map to fetch the key version
	m := make(map[string]interface{})
	if err := json.Unmarshal(keyjson, &m); err != nil {
		return nil, err
	}
	// Depending on the version try to parse one way or another
	var (
		keyBytes, keyId []byte
		err             error
	)
	k := &encryptedKeyJSONV3{}
	if err = json.Unmarshal(keyjson, k); err != nil {
		return nil, err
	}
	keyBytes, keyId, err = decryptKeyV3(k, auth)
	// Handle any decryption errors and return the key
	if err != nil {
		return nil, err
	}

	hdKeyBytesObject := hdKeyBytes{}
	err = msgpack.Unmarshal(keyBytes, &hdKeyBytesObject)
	if err != nil {
		return nil, err
	}

	return &AccountKey{
		Id:         uuid.UUID(keyId),
		Name:       k.Name,
		Address:    crypto.PrivKeyToAddress(hdKeyBytesObject.PrivKey),
		HdPath:     string(hdKeyBytesObject.HdPath),
		Mnemonic:   string(hdKeyBytesObject.Mnemonic),
		Passphrase: string(hdKeyBytesObject.Passphrase),
		PrivateKey: hdKeyBytesObject.PrivKey,
	}, nil
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-128 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

func GetKeyStorePassphrase(keysDirPath string) KeyStorePassphrase {
	return KeyStorePassphrase{keysDirPath, scryptN, scryptP}
}

func (ks KeyStorePassphrase) StoreKey(filename string, key *AccountKey, auth string) error {
	keyjson, err := EncryptKey(key, auth)
	if err != nil {
		return err
	}
	if filename[len(filename)-5:] != ".json" {
		filename = filename + ".json"
	}
	return WriteKeyFile(filename, keyjson)
}

// zeroKey zeroes a private key in memory.
func zeroKey(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func WriteKeyFile(file string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

func decryptKeyV3(keyProtected *encryptedKeyJSONV3, auth string) (keyBytes []byte, keyId []byte, err error) {
	if keyProtected.Version != version {
		return nil, nil, fmt.Errorf("Version not supported: %v", keyProtected.Version)
	}

	if keyProtected.Crypto.Cipher != "aes-128-ctr" {
		return nil, nil, fmt.Errorf("Cipher not supported: %v", keyProtected.Crypto.Cipher)
	}

	keyId = uuid.Parse(keyProtected.Id)
	mac, err := hex.DecodeString(keyProtected.Crypto.MAC)
	if err != nil {
		return nil, nil, err
	}

	iv, err := hex.DecodeString(keyProtected.Crypto.CipherParams.IV)
	if err != nil {
		return nil, nil, err
	}

	cipherText, err := hex.DecodeString(keyProtected.Crypto.CipherText)
	if err != nil {
		return nil, nil, err
	}

	derivedKey, err := getKDFKey(keyProtected.Crypto, auth)
	if err != nil {
		return nil, nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, nil, errors.New("could not decrypt key with given passphrase")
	}

	plainText, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, nil, err
	}
	return plainText, keyId, err
}

func getKDFKey(cryptoJSON cryptoJSON, auth string) ([]byte, error) {
	authArray := []byte(auth)
	salt, err := hex.DecodeString(cryptoJSON.KDFParams["salt"].(string))
	if err != nil {
		return nil, err
	}
	dkLen := ensureInt(cryptoJSON.KDFParams["dklen"])

	if cryptoJSON.KDF == keyHeaderKDF {
		n := ensureInt(cryptoJSON.KDFParams["n"])
		r := ensureInt(cryptoJSON.KDFParams["r"])
		p := ensureInt(cryptoJSON.KDFParams["p"])
		return scrypt.Key(authArray, salt, n, r, p, dkLen)

	}
	if cryptoJSON.KDF == "pbkdf2" {
		c := ensureInt(cryptoJSON.KDFParams["c"])
		prf := cryptoJSON.KDFParams["prf"].(string)
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("Unsupported PBKDF2 PRF: %s", prf)
		}
		key := pbkdf2.Key(authArray, salt, c, dkLen, sha256.New)
		return key, nil
	}

	return nil, fmt.Errorf("Unsupported KDF: %s", cryptoJSON.KDF)
}

// TODO: can we do without this when unmarshalling dynamic JSON?
// why do integers in KDF params end up as float64 and not int after
// unmarshal?
func ensureInt(x interface{}) int {
	res, ok := x.(int)
	if !ok {
		res = int(x.(float64))
	}
	return res
}

type AccountKey struct {
	Id uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address types.Address
	// The HD path to use to derive this key
	HdPath string
	// The mnemonic for the underlying HD wallet
	Mnemonic string
	// a nickname
	Name string
	// The bip39 passphrase for the underlying HD wallet
	Passphrase string
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey []byte
}

// ChangePassword
func ChangePassword(walletAddress, dir, auth string, key *AccountKey) error {
	files, _ := ioutil.ReadDir(dir)
	if len(files) == 0 {
		ErrorLog("not exist")
		return nil
	}
	for _, info := range files {
		if info.Name() == walletAddress {
			continue
		}
		keyStore := &KeyStorePassphrase{dir, scryptN, scryptP}
		filename := dir + "/" + walletAddress
		err := keyStore.StoreKey(filename, key, auth)
		return err
	}

	return nil
}
