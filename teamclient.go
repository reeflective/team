package team

import "time"

type Client interface {
	Users() ([]User, error)
	Version() (Version, error)
}

type User struct {
	Name     string
	Online   bool
	LastSeen time.Time
	Clients  int
}

type Version struct {
	Major      int32
	Minor      int32
	Patch      int32
	Commit     string
	Dirty      bool
	CompiledAt int64
	OS         string
	Arch       string
}
