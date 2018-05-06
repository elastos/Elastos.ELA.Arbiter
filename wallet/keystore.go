package wallet

import (
	"sync"
	"bytes"
	"errors"
	"crypto/rand"
	"crypto/sha256"

	. "github.com/elastos/Elastos.ELA/core"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
)

const (
	KeystoreVersion = "1.0"
)

type Keystore interface {
	ChangePassword(oldPassword, newPassword []byte) error

	GetPublicKey() *crypto.PublicKey
	GetRedeemScript() []byte
	GetProgramHash() *Uint168
	Address() string

	Sign(password []byte, txn *Transaction) ([]byte, error)
}

type KeystoreImpl struct {
	sync.Mutex

	*KeystoreFile

	publicKey    *crypto.PublicKey
	redeemScript []byte
	programHash  *Uint168
	address      string
}

func CreateKeystore(name string, password []byte) (Keystore, error) {

	keystoreFile, err := CreateKeystoreFile(name)
	if err != nil {
		return nil, err
	}

	keystore := &KeystoreImpl{
		KeystoreFile: keystoreFile,
	}

	iv := make([]byte, 16)
	_, err = rand.Read(iv)
	if err != nil {
		return nil, err
	}
	// Set IV
	keystoreFile.SetIV(iv)

	masterKey := make([]byte, 32)
	_, err = rand.Read(masterKey)
	if err != nil {
		return nil, err
	}

	passwordKey := crypto.ToAesKey(password)
	defer ClearBytes(passwordKey)
	passwordHash := sha256.Sum256(passwordKey)
	defer ClearBytes(passwordHash[:])
	// Set password hash
	keystoreFile.SetPasswordHash(passwordHash[:])

	masterKeyEncrypted, err := keystore.encryptMasterKey(passwordKey, masterKey)
	if err != nil {
		return nil, err
	}
	// Set master key encrypted
	keystoreFile.SetMasterKeyEncrypted(masterKeyEncrypted)

	// Generate new key pair
	privateKey, publicKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	privateKeyEncrypted, err := keystore.encryptPrivateKey(masterKey, passwordKey, privateKey, publicKey)
	defer ClearBytes(privateKeyEncrypted)
	// Set private key encrypted
	keystoreFile.SetPrivateKeyEncrypted(privateKeyEncrypted)

	// Init keystore parameters
	keystore.init(privateKey, publicKey)

	err = keystoreFile.SaveToFile()
	if err != nil {
		return nil, err
	}
	// Handle system interrupt signals
	keystore.catchSystemSignals()

	return keystore, nil
}

func OpenKeystore(name string, password []byte) (Keystore, error) {

	keystoreFile, err := OpenKeystoreFile(name)
	if err != nil {
		return nil, err
	}

	keystore := &KeystoreImpl{
		KeystoreFile: keystoreFile,
	}

	err = keystore.verifyPassword(password)
	if err != nil {
		return nil, err
	}

	privateKey, publicKey, err := keystore.decryptPrivateKey(crypto.ToAesKey(password))
	if err != nil {
		return nil, err
	}

	keystore.init(privateKey, publicKey)

	// Handle system interrupt signals
	keystore.catchSystemSignals()

	return keystore, nil
}

func (store *KeystoreImpl) init(privateKey []byte, publicKey *crypto.PublicKey) error {
	defer ClearBytes(privateKey)

	// Set public key
	store.publicKey = publicKey

	var err error
	// Set redeem script
	store.redeemScript, err = crypto.CreateStandardRedeemScript(publicKey)
	if err != nil {
		return err
	}

	// Set program hash
	store.programHash, err = crypto.ToProgramHash(store.redeemScript)
	if err != nil {
		return err
	}

	// Set address
	store.address, err = store.programHash.ToAddress()
	if err != nil {
		return err
	}

	return nil
}

func (store *KeystoreImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.Lock()
	})
}

func (store *KeystoreImpl) verifyPassword(password []byte) error {
	passwordKey := crypto.ToAesKey(password)
	defer ClearBytes(passwordKey)
	passwordHash := sha256.Sum256(passwordKey)
	defer ClearBytes(passwordHash[:])

	origin, err := store.GetPasswordHash()
	if err != nil {
		return err
	}
	if IsEqualBytes(origin, passwordHash[:]) {
		return nil
	}
	return errors.New("password wrong")
}

func (store *KeystoreImpl) ChangePassword(oldPassword, newPassword []byte) error {
	// Get old passwordKey
	oldPasswordKey := crypto.ToAesKey(oldPassword)
	defer ClearBytes(oldPasswordKey)

	masterKeyEncrypted, err := store.GetMasterKeyEncrypted()
	if err != nil {
		return err
	}
	defer ClearBytes(masterKeyEncrypted)

	masterKey, err := store.decryptMasterKey(oldPasswordKey)
	if err != nil {
		return err
	}
	defer ClearBytes(masterKey)

	// Decrypt private key
	privateKey, publicKey, err := store.decryptPrivateKey(oldPasswordKey)
	if err != nil {
		return err
	}
	defer ClearBytes(privateKey)

	// Encrypt private key with new password
	newPasswordKey := crypto.ToAesKey(newPassword)
	defer ClearBytes(newPasswordKey)
	newPasswordHash := sha256.Sum256(newPasswordKey)
	defer ClearBytes(newPasswordHash[:])

	masterKeyEncrypted, err = store.encryptMasterKey(newPasswordKey, masterKey)
	if err != nil {
		return err
	}

	privateKeyEncrypted, err := store.encryptPrivateKey(masterKey, newPasswordKey, privateKey, publicKey)
	if err != nil {
		return err
	}
	defer ClearBytes(privateKeyEncrypted)

	store.SetPasswordHash(newPasswordHash[:])
	store.SetMasterKeyEncrypted(masterKeyEncrypted)
	store.SetPrivateKeyEncrypted(privateKeyEncrypted)

	err = store.SaveToFile()
	if err != nil {
		return err
	}

	return nil
}

func (store *KeystoreImpl) GetPublicKey() *crypto.PublicKey {
	return store.publicKey
}

func (store *KeystoreImpl) GetRedeemScript() []byte {
	return store.redeemScript
}

func (store *KeystoreImpl) GetProgramHash() *Uint168 {
	return store.programHash
}

func (store *KeystoreImpl) Address() string {
	return store.address
}

func (store *KeystoreImpl) Sign(password []byte, txn *Transaction) ([]byte, error) {
	privateKey, _, err := store.decryptPrivateKey(crypto.ToAesKey(password))
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	txn.SerializeUnsigned(buf)
	signedData, err := crypto.Sign(privateKey, buf.Bytes())
	if err != nil {
		return nil, err
	}

	return signedData, nil
}

func (store *KeystoreImpl) encryptMasterKey(passwordKey, masterKey []byte) ([]byte, error) {
	iv, err := store.GetIV()
	if err != nil {
		return nil, err
	}

	masterKeyEncrypted, err := crypto.AesEncrypt(masterKey, passwordKey, iv)
	if err != nil {
		return nil, err
	}

	return masterKeyEncrypted, nil
}

func (store *KeystoreImpl) decryptMasterKey(passwordKey []byte) (masterKey []byte, err error) {
	iv, err := store.GetIV()
	if err != nil {
		return nil, err
	}

	masterKeyEncrypted, err := store.GetMasterKeyEncrypted()
	if err != nil {
		return nil, err
	}

	masterKey, err = crypto.AesDecrypt(masterKeyEncrypted, passwordKey, iv)
	if err != nil {
		return nil, err
	}

	return masterKey, nil
}

func (store *KeystoreImpl) encryptPrivateKey(masterKey, passwordKey, privateKey []byte, publicKey *crypto.PublicKey) ([]byte, error) {
	decryptedPrivateKey := make([]byte, 96)
	defer ClearBytes(decryptedPrivateKey)

	publicKeyBytes, err := publicKey.EncodePoint(false)
	if err != nil {
		return nil, err
	}
	for i := 1; i <= 64; i++ {
		decryptedPrivateKey[i-1] = publicKeyBytes[i]
	}
	for i := len(privateKey) - 1; i >= 0; i-- {
		decryptedPrivateKey[96+i-len(privateKey)] = privateKey[i]
	}

	iv, err := store.GetIV()
	if err != nil {
		return nil, err
	}

	encryptedPrivateKey, err := crypto.AesEncrypt(decryptedPrivateKey, masterKey, iv)
	if err != nil {
		return nil, err
	}
	return encryptedPrivateKey, nil
}

func (store *KeystoreImpl) decryptPrivateKey(passwordKey []byte) ([]byte, *crypto.PublicKey, error) {
	privateKeyEncrypted, err := store.GetPrivetKeyEncrypted()
	if err != nil {
		return nil, nil, err
	}
	if len(privateKeyEncrypted) != 96 {
		return nil, nil, errors.New("invalid encrypted private key")
	}

	iv, err := store.GetIV()
	if err != nil {
		return nil, nil, err
	}

	masterKeyEncrypted, err := store.GetMasterKeyEncrypted()
	if err != nil {
		return nil, nil, err
	}
	defer ClearBytes(masterKeyEncrypted)

	masterKey, err := store.decryptMasterKey(passwordKey)
	if err != nil {
		return nil, nil, err
	}
	defer ClearBytes(masterKey)

	keyPair, err := crypto.AesDecrypt(privateKeyEncrypted, masterKey, iv)
	if err != nil {
		return nil, nil, err
	}
	privateKey := keyPair[64:96]

	return privateKey, crypto.NewPubKey(privateKey), nil
}
