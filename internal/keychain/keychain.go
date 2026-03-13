package keychain

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

const (
	service = "mysh"
	account = "master-password"
)

// Get retrieves the master password from the macOS Keychain.
// Returns an empty string and an error if the password is not found or
// the current OS is not macOS.
func Get() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("keychain is only supported on macOS")
	}

	out, err := exec.Command("security", "find-generic-password",
		"-s", service, "-a", account, "-w").Output()
	if err != nil {
		return "", fmt.Errorf("keychain lookup failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Set stores the master password in the macOS Keychain.
// Uses -U to update an existing entry if one already exists.
func Set(password string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("keychain is only supported on macOS")
	}

	return exec.Command("security", "add-generic-password",
		"-s", service, "-a", account, "-w", password, "-U").Run()
}

// Delete removes the master password from the macOS Keychain.
func Delete() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("keychain is only supported on macOS")
	}

	return exec.Command("security", "delete-generic-password",
		"-s", service, "-a", account).Run()
}
