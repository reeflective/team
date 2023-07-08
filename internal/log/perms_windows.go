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

	isWritable = true
	return
}
