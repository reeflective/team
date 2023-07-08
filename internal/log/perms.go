//go:build !windows
// +build !windows

package log

import (
	"fmt"
	"os"
	"syscall"
)

// IsWritable checks that the given path can be created.
func IsWritable(path string) (isWritable bool, err error) {
	isWritable = false
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	err = nil
	if !info.IsDir() {
		return false, fmt.Errorf("Path isn't a directory")
	}

	// Check if the user bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		return false, fmt.Errorf("Write permission bit is not set on this file for user")
	}

	var stat syscall.Stat_t
	if err = syscall.Stat(path, &stat); err != nil {
		return false, fmt.Errorf("Unable to get stat")
	}

	err = nil
	if uint32(os.Geteuid()) != stat.Uid {
		return isWritable, fmt.Errorf("User doesn't have permission to write to this directory")
	}

	isWritable = true
	return
}
