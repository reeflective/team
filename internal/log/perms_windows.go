package log

import (
	"fmt"
	"os"
)

// IsWritable checks that the given path can be created, on Windows.
func IsWritable(path string) (isWritable bool, err error) {
	isWritable = false
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	err = nil
	if !info.IsDir() {
		return false, fmt.Errorf("Path isn't a directory")
	}

	// Check if the user bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		return false, fmt.Errorf("Write permission bit is not set on this file for user")
	}

	isWritable = true
	return
}
