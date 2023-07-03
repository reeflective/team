package client

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/AlecAivazis/survey.v1"
)

// Config is a JSON client connection configuration.
// It contains the addresses of a team server, the name of the user
// allowed to connect to it, and cryptographic material to secure and
// authenticate the client-server connection (using Mutual TLS).
type Config struct {
	User          string `json:"user"` // This value is actually ignored for the most part (cert CN is used instead)
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Token         string `json:"token"`
	CACertificate string `json:"ca_certificate"`
	PrivateKey    string `json:"private_key"`
	Certificate   string `json:"certificate"`
}

// GetConfigDir - Returns the path to the config dir
func (c *Client) ConfigsDir() string {
	rootDir, _ := filepath.Abs(c.AppDir())
	dir := filepath.Join(rootDir, configsDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetConfigs returns a list of available configs in
// the application config directory (~/.app/configs)
func (c *Client) GetConfigs() map[string]*Config {
	configDir := c.ConfigsDir()
	configFiles, err := os.ReadDir(configDir)
	if err != nil {
		c.log.Error(fmt.Sprintf("No configs found %v", err))
		return map[string]*Config{}
	}

	confs := map[string]*Config{}
	for _, confFile := range configFiles {
		confFilePath := filepath.Join(configDir, confFile.Name())

		conf, err := c.ReadConfig(confFilePath)
		if err != nil {
			continue
		}
		digest := sha256.Sum256([]byte(conf.Certificate))
		confs[fmt.Sprintf("%s@%s (%x)", conf.User, conf.Host, digest[:8])] = conf
	}
	return confs
}

// ReadConfig loads a client config into a struct.
func (c *Client) ReadConfig(confFilePath string) (*Config, error) {
	confFile, err := os.Open(confFilePath)
	if err != nil {
		c.log.Error(fmt.Sprintf("Open failed %v", err))
		return nil, err
	}
	defer confFile.Close()
	data, err := io.ReadAll(confFile)
	if err != nil {
		c.log.Error(fmt.Sprintf("Read failed %v", err))
		return nil, err
	}
	conf := &Config{}
	err = json.Unmarshal(data, conf)
	if err != nil {
		c.log.Error(fmt.Sprintf("Parse failed %v", err))
		return nil, err
	}
	return conf, nil
}

// SaveConfig saves a client config to disk.
func (c *Client) SaveConfig(config *Config) error {
	if config.Host == "" || config.User == "" {
		return errors.New("empty config")
	}
	configDir := c.ConfigsDir()
	filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(config.User), filepath.Base(config.Host))
	saveTo, _ := filepath.Abs(filepath.Join(configDir, filename))
	configJSON, _ := json.Marshal(config)
	err := os.WriteFile(saveTo, configJSON, 0o600)
	if err != nil {
		c.log.Error(fmt.Sprintf("Failed to write config to: %s (%v)", saveTo, err))
		return err
	}
	c.log.Error(fmt.Sprintf("Saved new client config to: %s", saveTo))
	return nil
}

// SelectConfig either returns the only configuration found in the
// application client configs directory, or prompts the user to select one.
// This call might thus be blocking, and expect user input.
func (c *Client) SelectConfig() *Config {
	configs := c.GetConfigs()

	if len(configs) == 0 {
		return nil
	}

	if len(configs) == 1 {
		for _, config := range configs {
			return config
		}
	}

	answer := struct{ Config string }{}
	qs := getPromptForConfigs(configs)
	err := survey.Ask(qs, &answer)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	return configs[answer.Config]
}

func getPromptForConfigs(configs map[string]*Config) []*survey.Question {
	keys := []string{}
	for k := range configs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return []*survey.Question{
		{
			Name: "config",
			Prompt: &survey.Select{
				Message: "Select a server:",
				Options: keys,
				Default: keys[0],
			},
		},
	}
}
