// +build !windows

package lib

import "syscall"

// Kill kills process pid
func Kill(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}
