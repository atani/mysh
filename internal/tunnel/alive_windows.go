//go:build windows

package tunnel

import "syscall"

const processQueryLimitedInformation = 0x1000

func isAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	_ = syscall.CloseHandle(h)
	return true
}
