//go:build !darwin && !windows

package keychain

import (
	"fmt"
	"runtime"
)

// Name returns the human-readable name of the credential store. Platforms
// without a supported store return an empty string.
func Name() string { return "" }

// Get always fails on platforms without a supported credential store so the
// caller falls back to the environment variable or an interactive prompt.
func Get() (string, error) {
	return "", fmt.Errorf("no OS credential store available on %s", runtime.GOOS)
}

// Set always fails on platforms without a supported credential store.
func Set(string) error {
	return fmt.Errorf("no OS credential store available on %s", runtime.GOOS)
}

// Delete always fails on platforms without a supported credential store.
func Delete() error {
	return fmt.Errorf("no OS credential store available on %s", runtime.GOOS)
}
