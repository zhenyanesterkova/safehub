package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"golang.org/x/crypto/pbkdf2"
)

const (
	KeyLength   = 32 // AES-256
	SaltLength  = 16
	NonceLength = 12 // GCM nonce
)

type CryptoService struct{}

func NewCryptoService() *CryptoService {
	return &CryptoService{}
}

// DeriveKey выводит ключ из пароля и соли
func (cs *CryptoService) DeriveKey(password, salt []byte) []byte {
	return pbkdf2.Key(password, salt, 100000, KeyLength, sha256.New)
}

// GenerateSalt генерирует случайную соль
func (cs *CryptoService) GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLength)
	_, err := rand.Read(salt)
	return salt, err
}

// Encrypt шифрует данные с помощью AES-GCM
func (cs *CryptoService) Encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, NonceLength)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt расшифровывает данные
func (cs *CryptoService) Decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < NonceLength {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:NonceLength], ciphertext[NonceLength:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
