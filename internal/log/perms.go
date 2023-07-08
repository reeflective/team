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
		fmt.Println("Path doesn't exist")
		return
	}

	err = nil
	if !info.IsDir() {
		fmt.Println("Path isn't a directory")
		return
	}

	// Check if the user bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		fmt.Println("Write permission bit is not set on this file for user")
		return
	}

	var stat syscall.Stat_t
	if err = syscall.Stat(path, &stat); err != nil {
		fmt.Println("Unable to get stat")
		return
	}

	err = nil
	if uint32(os.Geteuid()) != stat.Uid {
		isWritable = false
		fmt.Println("User doesn't have permission to write to this directory")
		return
	}

	isWritable = true
	return
}
