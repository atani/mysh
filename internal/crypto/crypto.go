package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

const (
	saltSize    = 16
	keySize     = 32
	argonTime   = 1
	argonMemory = 64 * 1024
	argonThread = 4
)

type EncryptedData struct {
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func deriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, argonTime, argonMemory, argonThread, keySize)
}

func Encrypt(plaintext, password []byte) (*EncryptedData, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedData{
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func Decrypt(data *EncryptedData, password []byte) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(data.Salt)
	if err != nil {
		return nil, fmt.Errorf("decoding salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(data.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decoding nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(data.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decoding ciphertext: %w", err)
	}

	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: wrong master password or corrupted data")
	}

	return plaintext, nil
}

func MarshalEncrypted(data *EncryptedData) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func UnmarshalEncrypted(s string) (*EncryptedData, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var data EncryptedData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ReadPassword reads a password from the terminal with echo disabled.
// The caller is responsible for printing the prompt.
func ReadPassword() (string, error) {
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	return string(password), nil
}

func masterPasswordPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "mysh", ".master_check")
}

func InitMasterPassword(password []byte) error {
	checkData, err := Encrypt([]byte("mysh-check"), password)
	if err != nil {
		return err
	}
	encoded, err := MarshalEncrypted(checkData)
	if err != nil {
		return err
	}
	return os.WriteFile(masterPasswordPath(), []byte(encoded), 0600)
}

func VerifyMasterPassword(password []byte) error {
	path := masterPasswordPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // first time, no check file yet
		}
		return err
	}

	enc, err := UnmarshalEncrypted(string(data))
	if err != nil {
		return err
	}

	plain, err := Decrypt(enc, password)
	if err != nil {
		return errors.New("wrong master password")
	}

	if string(plain) != "mysh-check" {
		return errors.New("wrong master password")
	}
	return nil
}

func MasterPasswordInitialized() bool {
	_, err := os.Stat(masterPasswordPath())
	return err == nil
}
