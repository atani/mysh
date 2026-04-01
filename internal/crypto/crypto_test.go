package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	password := []byte("test-master-password")
	plaintext := []byte("my-secret-db-password")

	enc, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := Decrypt(enc, password)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(got) != string(plaintext) {
		t.Errorf("got %q, want %q", got, plaintext)
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	password := []byte("correct-password")
	wrong := []byte("wrong-password")
	plaintext := []byte("secret")

	enc, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = Decrypt(enc, wrong)
	if err == nil {
		t.Error("expected error for wrong password, got nil")
	}
}

func TestMarshalUnmarshalEncrypted(t *testing.T) {
	password := []byte("test-password")
	plaintext := []byte("hello-world")

	enc, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	encoded, err := MarshalEncrypted(enc)
	if err != nil {
		t.Fatalf("MarshalEncrypted: %v", err)
	}

	decoded, err := UnmarshalEncrypted(encoded)
	if err != nil {
		t.Fatalf("UnmarshalEncrypted: %v", err)
	}

	got, err := Decrypt(decoded, password)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(got) != string(plaintext) {
		t.Errorf("got %q, want %q", got, plaintext)
	}
}

func TestEncryptDecryptEmptyPlaintext(t *testing.T) {
	password := []byte("test-password")
	plaintext := []byte("")

	enc, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := Decrypt(enc, password)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(got) != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestMasterPasswordVerification(t *testing.T) {
	// Use temp dir to avoid polluting real config
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", "")
	defer func() { _ = os.Setenv("HOME", origHome) }()

	configDir := filepath.Join(tmpDir, ".config", "mysh")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	password := []byte("my-master-pass")

	if MasterPasswordInitialized() {
		t.Error("master password should not be initialized yet")
	}

	if err := InitMasterPassword(password); err != nil {
		t.Fatalf("InitMasterPassword: %v", err)
	}

	if !MasterPasswordInitialized() {
		t.Error("master password should be initialized")
	}

	if err := VerifyMasterPassword(password); err != nil {
		t.Errorf("VerifyMasterPassword with correct password: %v", err)
	}

	if err := VerifyMasterPassword([]byte("wrong")); err == nil {
		t.Error("VerifyMasterPassword should fail with wrong password")
	}
}

func TestUnmarshalEncryptedInvalidBase64(t *testing.T) {
	_, err := UnmarshalEncrypted("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestUnmarshalEncryptedInvalidJSON(t *testing.T) {
	// Valid base64 but not valid JSON
	encoded := "aGVsbG8gd29ybGQ=" // "hello world" in base64
	_, err := UnmarshalEncrypted(encoded)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDecryptCorruptedSalt(t *testing.T) {
	data := &EncryptedData{
		Salt:       "not-valid-base64!!!",
		Nonce:      "AAAA",
		Ciphertext: "AAAA",
	}
	_, err := Decrypt(data, []byte("password"))
	if err == nil {
		t.Error("expected error for corrupted salt")
	}
}

func TestDecryptCorruptedNonce(t *testing.T) {
	password := []byte("test")
	enc, err := Encrypt([]byte("plain"), password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	enc.Nonce = "not-valid-base64!!!"
	_, err = Decrypt(enc, password)
	if err == nil {
		t.Error("expected error for corrupted nonce")
	}
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	password := []byte("test")
	enc, err := Encrypt([]byte("plain"), password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	enc.Ciphertext = "not-valid-base64!!!"
	_, err = Decrypt(enc, password)
	if err == nil {
		t.Error("expected error for corrupted ciphertext")
	}
}

func TestEncryptProducesDifferentOutput(t *testing.T) {
	password := []byte("same-password")
	plaintext := []byte("same-plaintext")

	enc1, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt 1: %v", err)
	}
	enc2, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt 2: %v", err)
	}

	// Same plaintext+password should produce different ciphertext (random salt/nonce)
	if enc1.Salt == enc2.Salt {
		t.Error("two encryptions should produce different salts")
	}
	if enc1.Ciphertext == enc2.Ciphertext {
		t.Error("two encryptions should produce different ciphertexts")
	}
}

func TestDecryptV0DataWithCurrentCode(t *testing.T) {
	// Simulate v0 encrypted data (Version=0, using argonTimeV0=1)
	password := []byte("test-password")
	plaintext := []byte("legacy-secret")

	// Manually create v0-encrypted data using the old parameters
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		t.Fatalf("generating salt: %v", err)
	}
	key := deriveKeyV0(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("creating cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("creating GCM: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		t.Fatalf("generating nonce: %v", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	v0Data := &EncryptedData{
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Version:    0, // legacy
	}

	// Decrypt with current code should work
	got, err := Decrypt(v0Data, password)
	if err != nil {
		t.Fatalf("Decrypt v0 data: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Errorf("got %q, want %q", got, plaintext)
	}
}

func TestEncryptUsesV1(t *testing.T) {
	password := []byte("test-password")
	enc, err := Encrypt([]byte("data"), password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc.Version != 1 {
		t.Errorf("Version = %d, want 1", enc.Version)
	}
}

func TestMasterPasswordPathContainsMysh(t *testing.T) {
	path := masterPasswordPath()
	if filepath.Base(path) != ".master_check" {
		t.Errorf("expected .master_check, got %q", filepath.Base(path))
	}
}

func TestVerifyMasterPasswordCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpDir, ".config", "mysh")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write corrupted check file
	checkPath := filepath.Join(configDir, ".master_check")
	if err := os.WriteFile(checkPath, []byte("corrupted-data"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := VerifyMasterPassword([]byte("any-password"))
	if err == nil {
		t.Error("expected error for corrupted check file")
	}
}

func TestMasterPasswordInitializedFalseWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", "")

	if MasterPasswordInitialized() {
		t.Error("should return false when file doesn't exist")
	}
}
