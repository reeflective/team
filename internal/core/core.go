package core

import (
	"runtime"
	"strings"

	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/internal/version"
	"github.com/reeflective/team/server/db"
	"gorm.io/gorm"
)

// Version returns the teamserver version.
func Version() (*proto.Version, error) {
	dirty := version.GitDirty != ""
	semVer := version.Semantic()
	compiled, _ := version.Compiled()
	return &proto.Version{
		Major:      int32(semVer[0]),
		Minor:      int32(semVer[1]),
		Patch:      int32(semVer[2]),
		Commit:     strings.TrimSuffix(version.GitCommit, "\n"),
		Dirty:      dirty,
		CompiledAt: compiled.Unix(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}, nil
}

// Users returns the list of teamserver users and their status.
func Users(database *gorm.DB) (*proto.Users, error) {
	users := []*db.User{}
	err := database.Distinct("Name").Find(&users).Error

	var userspb *proto.Users
	for _, user := range users {
		userspb.Users = append(userspb.Users, &proto.User{
			Name: user.Name,
		})
	}

	return userspb, err
}
