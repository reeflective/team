package server

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"text/template"

	"github.com/reeflective/team/client"
)

type SystemdConfig struct {
	User    string   // User to configure systemd for, default is current user.
	Binpath string   // Path to binary
	Args    []string // The command is the position of the daemon command in the application command tree.
}

//go:embed assets/teamserver.service
var systemdServiceTemplate string

// GenerateServiceFile generates a systemd service file for the application.
func (s *Server) GenerateServiceFile(userCfg *SystemdConfig) string {
	cfg := DefaultSystemdConfig()

	if userCfg != nil {
		cfg.User = userCfg.User
		cfg.Binpath = userCfg.Binpath
		cfg.Args = userCfg.Args
	}

	// Prepare all values before running templates
	ver := client.SemanticVersion()
	version := fmt.Sprintf("%d.%d.%d", ver[0], ver[0], ver[0])
	desc := fmt.Sprintf("%s Teamserver daemon (v%s)", s.name, version)

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
		Application: s.Name(),
		Description: desc,
		User:        systemdUser,
		Command:     command,
	}

	var config bytes.Buffer

	templ := template.New(s.Name())
	parsed, err := templ.Parse(systemdServiceTemplate)
	if err != nil {
		log.Fatalf("Failed to parse: %s", err)
	}

	parsed.Execute(&config, TemplateValues)

	systemdFile := config.String()

	return systemdFile
}

// DefaultSystemdConfig returns a default Systemd service file configuration.
func DefaultSystemdConfig() *SystemdConfig {
	c := &SystemdConfig{}

	user, _ := user.Current()
	if user != nil {
		c.User = user.Name
	}

	currentPath, err := os.Executable()
	if err != nil {
		return c
	}

	c.Binpath = currentPath

	return c
}
