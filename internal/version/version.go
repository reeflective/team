package version

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"time"
)

//go:generate bash teamserver_version_info
var (
	// Version - The semantic version in string form
	//go:embed client_version.compiled
	Version string

	// GoVersion - Go compiler version
	//go:embed go_version.compiled
	GoVersion string

	// GitCommit - The commit id at compile time
	//go:embed commit_version.compiled
	GitCommit string

	// GitDirty - Was the commit dirty at compile time
	//go:embed dirty_version.compiled
	GitDirty string

	// CompiledAt - When was this binary compiled
	//go:embed compiled_version.compiled
	CompiledAt string
)

const (
	semVerLen = 3
)

// Semantic - Get the structured sematic version.
func Semantic() []int {
	semVer := make([]int, semVerLen)
	version := strings.TrimSuffix(Version, "\n")
	version = strings.TrimPrefix(version, "v")

	for i, part := range strings.Split(version, ".") {
		number, _ := strconv.ParseInt(part, 10, 32)
		semVer[i] = int(number)
	}

	return semVer
}

// Compiled - Get time this binary was compiled.
func Compiled() (time.Time, error) {
	compiledAt := strings.TrimSuffix(CompiledAt, "\n")

	compiled, err := strconv.ParseInt(compiledAt, 10, 64)
	if err != nil {
		return time.Unix(0, 0), err
	}

	return time.Unix(compiled, 0), nil
}

// Full - Full version string.
func Full() string {
	ver := strings.TrimSuffix(Version, "\n")
	if GitCommit != "" {
		ver += fmt.Sprintf(" - %s", GitCommit)
	}

	compiled, err := Compiled()
	if err == nil {
		ver += fmt.Sprintf(" - Compiled %s", compiled.String())
	}

	return ver
}
