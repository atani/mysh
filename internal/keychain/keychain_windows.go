//go:build windows

package keychain

import (
	"fmt"

	"github.com/danieljoos/wincred"
)

// target is the Windows Credential Manager entry name. It combines the service
// and account so the entry is namespaced and easy to identify in the
// Credential Manager UI.
const target = service + ":" + account

// Name returns the human-readable name of the credential store.
func Name() string { return "Windows Credential Manager" }

// Get retrieves the master password from the Windows Credential Manager.
// Returns an empty string and an error if the entry is not found.
func Get() (string, error) {
	cred, err := wincred.GetGenericCredential(target)
	if err != nil {
		return "", fmt.Errorf("credential manager lookup failed: %w", err)
	}
	return string(cred.CredentialBlob), nil
}

// Set stores the master password in the Windows Credential Manager, creating
// the entry or overwriting an existing one. The credential is persisted for
// the local machine so it survives logoff, mirroring the macOS login keychain.
func Set(password string) error {
	cred := wincred.NewGenericCredential(target)
	cred.CredentialBlob = []byte(password)
	cred.Persist = wincred.PersistLocalMachine
	return cred.Write()
}

// Delete removes the master password from the Windows Credential Manager.
func Delete() error {
	cred, err := wincred.GetGenericCredential(target)
	if err != nil {
		return fmt.Errorf("credential manager lookup failed: %w", err)
	}
	return cred.Delete()
}
