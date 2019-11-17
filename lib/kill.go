// +build !windows

package lib

import "syscall"

// Kill kills process pid
func Kill(pid int, force bool) error {
	sig := syscall.SIGTERM

	if force {
		sig = syscall.SIGKILL
	}

	return syscall.Kill(pid, sig)
}
