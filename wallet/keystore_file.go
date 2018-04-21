package wallet

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"sync"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

const (
	OldWalletFile       = "wallet.dat"
	DefaultKeystoreFile = "keystore.dat"
)

type KeystoreFile struct {
	sync.Mutex

	fileName string
	Version  string

	IV                  string
	PasswordHash        string
	MasterKeyEncrypted  string
	PrivateKeyEncrypted string
}

func CreateKeystoreFile(name string) (*KeystoreFile, error) {

	if FileExisted(name) {
		return nil, errors.New("key store file already exist")
	}

	file := &KeystoreFile{
		fileName: name,
		Version:  KeystoreVersion,
	}

	return file, nil
}

func OpenKeystoreFile(name string) (*KeystoreFile, error) {

	file := &KeystoreFile{
		fileName: name,
	}

	err := file.LoadFromFile()
	if err != nil {
		// Try to open keystore file from old version
		file, err = OpenFromOldVersion()
		if err == nil {
			return file, nil
		}
		return nil, err
	}

	return file, nil
}

func OpenFromOldVersion() (*KeystoreFile, error) {
	if _, err := os.Stat(OldWalletFile); err != nil {
		return nil, errors.New("wallet file not exist")
	}

	file, err := os.OpenFile(OldWalletFile, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	content := make(map[string]interface{})
	err = json.Unmarshal(data, &content)
	if err != nil {
		return nil, err
	}

	accounts := content["Account"].([]interface{})

	var privateKeyEncrypted string

	for _, account := range accounts {
		accountType := account.(map[string]interface{})["Type"].(string)
		if accountType == "main-account" {
			privateKeyEncrypted = account.(map[string]interface{})["PrivateKeyEncrypted"].(string)
			break
		}
	}

	keystoreFile := &KeystoreFile{
		fileName:            DefaultKeystoreFile,
		Version:             KeystoreVersion,
		IV:                  content["IV"].(string),
		PasswordHash:        content["PasswordHash"].(string),
		MasterKeyEncrypted:  content["MasterKey"].(string),
		PrivateKeyEncrypted: privateKeyEncrypted,
	}

	err = keystoreFile.SaveToFile()
	if err != nil {
		return nil, err
	}

	return keystoreFile, nil
}

func (store *KeystoreFile) SetIV(iv []byte) {
	store.IV = BytesToHexString(iv)
}

func (store *KeystoreFile) SetPasswordHash(passwordHash []byte) {
	store.PasswordHash = BytesToHexString(passwordHash)
}

func (store *KeystoreFile) SetMasterKeyEncrypted(masterKeyEncrypted []byte) {
	store.MasterKeyEncrypted = BytesToHexString(masterKeyEncrypted)
}

func (store *KeystoreFile) SetPrivateKeyEncrypted(privateKeyEncrypted []byte) {
	store.PrivateKeyEncrypted = BytesToHexString(privateKeyEncrypted)
}

func (store *KeystoreFile) GetIV() ([]byte, error) {

	iv, err := HexStringToBytes(store.IV)
	if err != nil {
		return nil, err
	}

	return iv, nil
}

func (store *KeystoreFile) GetPasswordHash() ([]byte, error) {

	passwordHash, err := HexStringToBytes(store.PasswordHash)
	if err != nil {
		return nil, err
	}

	return passwordHash, nil
}

func (store *KeystoreFile) GetMasterKeyEncrypted() ([]byte, error) {

	masterKeyEncrypted, err := HexStringToBytes(store.MasterKeyEncrypted)
	if err != nil {
		return nil, err
	}

	return masterKeyEncrypted, nil
}

func (store *KeystoreFile) GetPrivetKeyEncrypted() ([]byte, error) {

	privateKeyEncrypted, err := HexStringToBytes(store.PrivateKeyEncrypted)
	if err != nil {
		return nil, err
	}

	return privateKeyEncrypted, nil
}

func (store *KeystoreFile) LoadFromFile() error {
	store.Lock()
	defer store.Unlock()

	if _, err := os.Stat(store.fileName); err != nil {
		return errors.New("keystore file not exist")
	}

	file, err := os.OpenFile(store.fileName, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, store)
	if err != nil {
		return err
	}

	return nil
}

func (store *KeystoreFile) SaveToFile() error {
	store.Lock()
	defer store.Unlock()

	file, err := os.OpenFile(store.fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	data, err := json.Marshal(*store)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}
