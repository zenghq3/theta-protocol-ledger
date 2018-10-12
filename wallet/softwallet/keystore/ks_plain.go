package keystore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/pborman/uuid"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
)

var _ Keystore = (*KeystorePlain)(nil)

type KeystorePlain struct {
	keysDirPath string
}

func NewKeystorePlain(keysDirPath string) (KeystorePlain, error) {
	err := os.MkdirAll(keysDirPath, 0700)
	if err != nil {
		return KeystorePlain{}, err
	}

	fi, err := os.Lstat(keysDirPath)
	if err != nil {
		return KeystorePlain{}, err
	}
	if fi.Mode().Perm() != 0700 {
		return KeystorePlain{}, fmt.Errorf("%s must have permission set to 0700", keysDirPath)
	}

	ks := KeystorePlain{
		keysDirPath: keysDirPath,
	}

	return ks, nil
}

func (ks KeystorePlain) GetKey(address common.Address, auth string) (*Key, error) {
	filePath := ks.getFilePath(address)
	fd, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	plainKeyJs := new(plainKeyJSON)
	if err := json.NewDecoder(fd).Decode(plainKeyJs); err != nil {
		return nil, err
	}
	if plainKeyJs.Address != hex.EncodeToString(address[:]) {
		return nil, fmt.Errorf("key content mismatch: have address %x, want %x", plainKeyJs.Address, address.Hex())
	}

	privKeyBytes, err := hex.DecodeString(plainKeyJs.PrivateKey)
	if err != nil {
		return nil, err
	}

	privKey, err := crypto.PrivateKeyFromBytes(privKeyBytes)
	if err != nil {
		return nil, err
	}

	keyId := uuid.Parse(plainKeyJs.Id)
	key := &Key{
		Id:         keyId,
		Address:    common.HexToAddress(plainKeyJs.Address),
		PrivateKey: privKey,
	}
	return key, nil
}

func (ks KeystorePlain) StoreKey(key *Key, auth string) error {
	address := key.Address
	filePath := ks.getFilePath(address)
	plainKeyJs := &plainKeyJSON{
		Address:    hex.EncodeToString(key.Address[:]),
		PrivateKey: hex.EncodeToString(key.PrivateKey.ToBytes()),
		Id:         key.Id.String(),
		Version:    version,
	}
	content, err := json.Marshal(plainKeyJs)
	if err != nil {
		return err
	}
	return writeKeyFile(filePath, content)
}

func (ks KeystorePlain) getFilePath(address common.Address) string {
	filePath := path.Join(ks.keysDirPath, address.Hex()[2:])
	return filePath
}

type plainKeyJSON struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privatekey"`
	Id         string `json:"id"`
	Version    int    `json:"version"`
}
