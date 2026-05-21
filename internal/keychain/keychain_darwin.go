//go:build darwin

package keychain

import (
	"fmt"
	"os/exec"
	"strings"
)

// Name returns the human-readable name of the credential store.
func Name() string { return "macOS Keychain" }

// Get retrieves the master password from the macOS Keychain.
// Returns an empty string and an error if the password is not found.
func Get() (string, error) {
	out, err := exec.Command("security", "find-generic-password",
		"-s", service, "-a", account, "-w").Output()
	if err != nil {
		return "", fmt.Errorf("keychain lookup failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Set stores the master password in the macOS Keychain.
// Uses -U to update an existing entry if one already exists.
// Note: the password is briefly visible in the process list via the -w
// argument. The macOS `security` CLI does not support reading passwords from
// stdin for add-generic-password, so the argument approach is required.
func Set(password string) error {
	return exec.Command("security", "add-generic-password",
		"-s", service, "-a", account, "-w", password, "-U").Run()
}

// Delete removes the master password from the macOS Keychain.
func Delete() error {
	return exec.Command("security", "delete-generic-password",
		"-s", service, "-a", account).Run()
}
