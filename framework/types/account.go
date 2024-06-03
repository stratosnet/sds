package types

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
	"io"
	"os"
	"path/filepath"

	"github.com/cosmos/go-bip39"
	"github.com/pborman/uuid"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"

	fwcrypto "github.com/stratosnet/sds/framework/crypto"
	fwed25519 "github.com/stratosnet/sds/framework/crypto/ed25519"
	fwsecp256k1 "github.com/stratosnet/sds/framework/crypto/secp256k1"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
	"github.com/stratosnet/sds/framework/utils"
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
func CreateWallet(dir, nickname, password, mnemonic, bip39Passphrase, hdPath string) (walletAddress fwcryptotypes.Address, newlyCreated bool, err error) {
	privateKeyBytes, err := fwsecp256k1.Derive(mnemonic, bip39Passphrase, hdPath)
	if err != nil {
		return nil, false, err
	}

	privateKey := fwsecp256k1.Generate(privateKeyBytes)

	walletAddress = privateKey.PubKey().Address()
	exists, err := KeyExists(dir, WalletAddressBytesToBech32(walletAddress.Bytes()))
	if exists || err != nil {
		return walletAddress, false, err
	}

	_, err = saveAccountKey(dir, nickname, password, mnemonic, bip39Passphrase, hdPath, scryptN, scryptP, privateKey, true)
	return walletAddress, true, err
}

// KeyExists verifies whether the given key (wallet or p2p) exists in the dir folder
func KeyExists(dir, address string) (bool, error) {
	_, err := os.Stat(filepath.Join(dir, address+".json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

// CreateP2PKey creates a P2P key to be used by one of the SDS nodes, and saves the key data into the dir folder
func CreateP2PKey(dir, nickname, password, privateKeyHex string) (fwcryptotypes.Address, error) {
	privateKey := fwed25519.GenPrivKey()
	if privateKeyHex != "" {
		privateKeyBytes, err := hex.DecodeString(privateKeyHex)
		if err != nil {
			return nil, err
		}
		privateKey = fwed25519.Generate(privateKeyBytes)
	}

	return saveAccountKey(dir, nickname, password, "", "", "", scryptN, scryptP, privateKey, false)
}

// CreateP2PKeyFromHdPath uses a wallet mnemonic to create a P2P key to be used by one of the SDS nodes, and saves the key data into the dir folder
func CreateP2PKeyFromHdPath(dir, nickname, password, mnemonic, bip39Passphrase, hdPath string) (p2pAddress fwcryptotypes.Address, newlyCreated bool, err error) {
	privateKey, err := GenerateP2pKeyFromHdPath(mnemonic, bip39Passphrase, hdPath)
	if err != nil {
		return nil, false, err
	}

	p2pAddress = privateKey.PubKey().Address()
	exists, err := KeyExists(dir, P2PAddressBytesToBech32(p2pAddress.Bytes()))
	if exists || err != nil {
		return p2pAddress, false, err
	}

	_, err = saveAccountKey(dir, nickname, password, mnemonic, bip39Passphrase, hdPath, scryptN, scryptP, privateKey, false)
	return p2pAddress, true, err
}

// GenerateP2pKeyFromHdPath uses a wallet mnemonic to create a P2P key, but doesn't store it anywhere
func GenerateP2pKeyFromHdPath(mnemonic, bip39Passphrase, hdPath string) (*fwed25519.PrivKey, error) {
	secpPrivateKeyBytes, err := fwsecp256k1.Derive(mnemonic, bip39Passphrase, hdPath)
	if err != nil {
		return nil, err
	}

	return fwed25519.GenPrivKeyFromSecret(secpPrivateKeyBytes), nil
}

func saveAccountKey(dir, nickname, password, mnemonic, bip39Passphrase, hdPath string, scryptN, scryptP int,
	privateKey fwcryptotypes.PrivKey, isWallet bool) (fwcryptotypes.Address, error) {

	keyStore := &KeyStorePassphrase{dir, scryptN, scryptP}

	id := uuid.NewRandom()
	key := &AccountKey{
		Id:         id,
		PrivateKey: privateKey,
		Address:    privateKey.PubKey().Address(),
		Name:       nickname,
		HdPath:     hdPath,
		Mnemonic:   mnemonic,
		Passphrase: bip39Passphrase,
	}

	var address string
	if isWallet {
		address = WalletAddressBytesToBech32(key.Address.Bytes())
	} else {
		address = P2PAddressBytesToBech32(key.Address.Bytes())
	}
	if address == "" {
		return nil, fmt.Errorf("Failed to parse address. ")
	}

	filename := dir + "/" + address
	if err := keyStore.StoreKey(filename, key, password); err != nil {
		zeroKey(key.PrivateKey.Bytes())
		return nil, err
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
		PrivKey:    key.PrivateKey.Bytes(),
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
	mac := fwcrypto.Keccak256(derivedKey[16:32], cipherText)

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
func DecryptKey(keyjson []byte, auth string, isWallet bool) (*AccountKey, error) {
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

	var privKey fwcryptotypes.PrivKey
	if isWallet {
		privKey = fwsecp256k1.Generate(hdKeyBytesObject.PrivKey)
	} else {
		privKey = fwed25519.Generate(hdKeyBytesObject.PrivKey)
	}

	return &AccountKey{
		Id:         uuid.UUID(keyId),
		Name:       k.Name,
		Address:    privKey.PubKey().Address(),
		HdPath:     string(hdKeyBytesObject.HdPath),
		Mnemonic:   string(hdKeyBytesObject.Mnemonic),
		Passphrase: string(hdKeyBytesObject.Passphrase),
		PrivateKey: privKey,
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
	f, err := os.CreateTemp(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
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

	calculatedMAC := fwcrypto.Keccak256(derivedKey[16:32], cipherText)
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
	Address fwcryptotypes.Address
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
	PrivateKey fwcryptotypes.PrivKey
}

// ChangePassword
func ChangePassword(walletAddress, dir, auth string, key *AccountKey) error {
	files, _ := os.ReadDir(dir)
	if len(files) == 0 {
		utils.ErrorLog("not exist")
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
