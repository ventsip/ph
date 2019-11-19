// +build windows

package lib

import "syscall"

// Kill kills process pid
func Kill(pid int) error {
	h, err := syscall.OpenProcess(1, false, uint32(pid))
	if err != nil {
		return err
	}

	return syscall.TerminateProcess(h, 0)
}
