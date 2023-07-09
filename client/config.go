package client

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
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

func (tc *Client) initConfig() (*Config, error) {
	cfg := tc.opts.config

	if !tc.opts.local {
		configs := tc.GetConfigs()
		if len(configs) == 0 {
			err := fmt.Errorf("no config files found at %s", tc.ConfigsDir())
			return nil, err
		}
		cfg = tc.SelectConfig()
	}

	if cfg == nil {
		return nil, errors.New("no application was selected or parsed")
	}

	tc.opts.config = cfg

	return nil, nil
}

// GetConfigs returns a list of available configs in
// the application config directory (~/.app/configs)
func (tc *Client) GetConfigs() map[string]*Config {
	configDir := tc.ConfigsDir()
	configFiles, err := os.ReadDir(configDir)
	if err != nil {
		tc.log.Errorf("No configs found: %w", err)
		return map[string]*Config{}
	}

	confs := map[string]*Config{}
	for _, confFile := range configFiles {
		confFilePath := filepath.Join(configDir, confFile.Name())

		conf, err := tc.ReadConfig(confFilePath)
		if err != nil {
			continue
		}
		digest := sha256.Sum256([]byte(conf.Certificate))
		confs[fmt.Sprintf("%s@%s (%x)", conf.User, conf.Host, digest[:8])] = conf
	}
	return confs
}

// ReadConfig loads a client config into a struct.
func (tc *Client) ReadConfig(confFilePath string) (*Config, error) {
	confFile, err := os.Open(confFilePath)
	if err != nil {
		return nil, fmt.Errorf("open failed: %w", err)
	}
	defer confFile.Close()
	data, err := io.ReadAll(confFile)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	conf := &Config{}
	err = json.Unmarshal(data, conf)
	if err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	return conf, nil
}

// SaveConfig saves a client config to disk.
func (tc *Client) SaveConfig(config *Config) error {
	if config.User == "" {
		return ErrConfigNoUser
	}

	configDir := tc.ConfigsDir()

	filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(config.User), filepath.Base(config.Host))
	saveTo, _ := filepath.Abs(filepath.Join(configDir, filename))
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConfig, err)
	}

	err = os.WriteFile(saveTo, configJSON, 0o600)
	if err != nil {
		err = fmt.Errorf("Failed to write config to: %s (%w)", saveTo, err)

		tc.log.Error(err)
		return err
	}
	tc.log.Infof("Saved new client config to: %w", saveTo)

	return nil
}

// SelectConfig either returns the only configuration found in the
// application client configs directory, or prompts the user to select one.
// This call might thus be blocking, and expect user input.
func (tc *Client) SelectConfig() *Config {
	configs := tc.GetConfigs()

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
		// TODO: Println here should not
		fmt.Println(err.Error())
		return nil
	}

	return configs[answer.Config]
}

// Config returns the current teamclient server configuration.
func (tc *Client) Config() *Config {
	return tc.opts.config
}

// defaultUserConfig returns the default user configuration for this application.
// the file is of the following form: ~/.app/configs/app_USERNAME_default.cfg.
// If the latter is found, it returned, otherwise no config is returned.
func (tc *Client) defaultUserConfig() (cfg *Config) {
	user, err := user.Current()
	if err != nil {
		return nil
	}

	filename := fmt.Sprintf("%s_%s_default", tc.Name(), user.Username)
	saveTo := tc.ConfigsDir()

	configPath := filepath.Join(saveTo, filename+".cfg")
	if _, err := os.Stat(configPath); err == nil {
		cfg, _ = tc.ReadConfig(configPath)
	}

	return cfg
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
