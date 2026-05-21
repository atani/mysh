//go:build darwin || windows

package keychain

// service and account identify the mysh master password entry within the OS
// credential store. They are shared by the macOS and Windows implementations.
const (
	service = "mysh"
	account = "master-password"
)
