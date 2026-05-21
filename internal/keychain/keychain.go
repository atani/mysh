// Package keychain stores the mysh master password in the operating system's
// native credential store: the macOS Keychain or the Windows Credential
// Manager.
//
// On unsupported platforms (e.g. Linux) the exported functions return an error
// so the caller falls back to the MYSH_MASTER_PASSWORD environment variable or
// an interactive prompt. Each platform provides its own implementation of Get,
// Set, Delete and Name in a build-tagged file.
package keychain

const (
	service = "mysh"
	account = "master-password"
)
