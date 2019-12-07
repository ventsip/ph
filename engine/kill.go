// +build !windows

package engine

import "syscall"

// Kill kills process pid
func Kill(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}
