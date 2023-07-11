package systemd

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"text/template"

	"github.com/reeflective/team/internal/version"
)

// Config is a stub to generate systemd configuration files.
type Config struct {
	User    string   // User to configure systemd for, default is current user.
	Binpath string   // Path to binary
	Args    []string // The command is the position of the daemon command in the application command tree.
}

//go:embed teamserver.service
var systemdServiceTemplate string

func NewFrom(name string, userCfg *Config) string {
	cfg := NewDefaultConfig()

	if userCfg != nil {
		cfg.User = userCfg.User
		cfg.Binpath = userCfg.Binpath
		cfg.Args = userCfg.Args
	}

	// Prepare all values before running templates
	ver := version.Semantic()
	version := fmt.Sprintf("%d.%d.%d", ver[0], ver[0], ver[0])
	desc := fmt.Sprintf("%s Teamserver daemon (v%s)", name, version)

	systemdUser := cfg.User
	if systemdUser == "" {
		systemdUser = "root"
	}

	// Command
	command := strings.Join(cfg.Args, " ")

	TemplateValues := struct {
		Application string
		Description string
		User        string
		Command     string
	}{
		Application: name,
		Description: desc,
		User:        systemdUser,
		Command:     command,
	}

	var config bytes.Buffer

	templ := template.New(name)
	parsed, err := templ.Parse(systemdServiceTemplate)
	if err != nil {
		log.Fatalf("Failed to parse: %s", err)
	}

	parsed.Execute(&config, TemplateValues)

	systemdFile := config.String()

	return systemdFile
}

// NewDefaultConfig returns a default Systemd service file configuration.
func NewDefaultConfig() *Config {
	c := &Config{}

	user, _ := user.Current()
	if user != nil {
		c.User = user.Username
	}

	currentPath, err := os.Executable()
	if err != nil {
		return c
	}

	c.Binpath = currentPath

	return c
}
