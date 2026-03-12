package crypto

import (
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
	defer os.Setenv("HOME", origHome)

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
