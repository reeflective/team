package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/internal/transport"
	"github.com/sirupsen/logrus"
)

const (
	configFileExt = "teamserver.json"
	blankHost           = "-"
	blankPort           = uint16(0)
)

type Config struct {
	DaemonMode struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"daemon_mode"`

	Log struct {
		Level              int  `json:"level"`
		GRPCUnaryPayloads  bool `json:"grpc_unary_payloads"`
		GRPCStreamPayloads bool `json:"grpc_stream_payloads"`
		TLSKeyLogger       bool `json:"tls_key_logger"`
	} `json:"log"`

	Listeners []struct {
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	} `json:"listeners"`
}

// GetServerConfigPath - File path to the server config.json file.
func (ts *Server) ConfigPath() string {
	appDir := ts.AppDir()

	serverConfigPath := filepath.Join(appDir, "configs", fmt.Sprintf("%s.%s", ts.Name(), configFileExt))
	return serverConfigPath
}

// GetConfig returns the team server configuration struct.
func (ts *Server) GetConfig() *Config {
	cfgLog := ts.NamedLogger("config", "server")

	configPath := ts.ConfigPath()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		cfgLog.Debugf("Loading config from %s", configPath)

		data, err := os.ReadFile(configPath)
		if err != nil {
			cfgLog.Errorf("Failed to read config file %s", err)
			return ts.opts.config
		}
		err = json.Unmarshal(data, ts.opts.config)
		if err != nil {
			cfgLog.Errorf("Failed to parse config file %s", err)
			return ts.opts.config
		}
	} else {
		cfgLog.Warnf("Config file does not exist, using defaults")
	}

	if ts.opts.config.Log.Level < 0 {
		ts.opts.config.Log.Level = 0
	}
	if 6 < ts.opts.config.Log.Level {
		ts.opts.config.Log.Level = 6
	}
	ts.log.SetLevel(log.LevelFrom(ts.opts.config.Log.Level))

	// This updates the config with any missing fields
	err := ts.SaveConfig(ts.opts.config)
	if err != nil {
		cfgLog.Errorf("Failed to save default config %s", err)
	}

	return ts.opts.config
}

// Save - Save config file to disk
func (ts *Server) SaveConfig(c *Config) error {
	log := ts.NamedLogger("config", "server")

	configPath := ts.ConfigPath()
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		log.Debugf("Creating config dir %s", configDir)
		err := os.MkdirAll(configDir, 0o700)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrConfig, err)
		}
	}

	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}

	log.Debugf("Saving config to %s", configPath)
	err = os.WriteFile(configPath, data, 0o600)
	if err != nil {
		return fmt.Errorf("%w: failed to write config: %s", ErrConfig, err)
	}
	return nil
}

// AddListenerJob adds a teamserver listener job to the config and saves it.
func (ts *Server) AddListener(host string, port uint16) error {
	listener := struct {
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	}{
		Host: host,
		Port: port,
		ID:   getRandomID(),
	}

	ts.opts.config.Listeners = append(ts.opts.config.Listeners, listener)

	return ts.SaveConfig(ts.opts.config)
}

// RemoveListenerJob removes a server listener job from the configuration and saves it.
func (ts *Server) RemoveListener(id string) {
	if ts.opts.config.Listeners == nil {
		return
	}

	defer ts.SaveConfig(ts.opts.config)

	var listeners []struct {
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	}

	for _, listener := range ts.opts.config.Listeners {
		if listener.ID != id {
			listeners = append(listeners, listener)
		}
	}

	ts.opts.config.Listeners = listeners
}

// startPersistentListeners starts all teamserver listeners,
// aborting and returning an error if one of those raise one.
func (ts *Server) startPersistentListeners() error {
	if ts.opts.config.Listeners == nil {
		return nil
	}

	for _, j := range ts.opts.config.Listeners {
		_, err := ts.ServeAddr(j.Host, j.Port)
		if err != nil {
			return err
		}
	}

	return nil
}

func getDefaultServerConfig() *Config {
	return &Config{
		DaemonMode: struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		}{
			Port: transport.DefaultPort, // 31416
		},
		Log: struct {
			Level              int  `json:"level"`
			GRPCUnaryPayloads  bool `json:"grpc_unary_payloads"`
			GRPCStreamPayloads bool `json:"grpc_stream_payloads"`
			TLSKeyLogger       bool `json:"tls_key_logger"`
		}{
			Level: int(logrus.InfoLevel),
		},
		Listeners: []struct {
			Host string `json:"host"`
			Port uint16 `json:"port"`
			ID   string `json:"id"`
		}{},
	}
}

func (ts *Server) clientServerMatch(config *client.Config) bool {
	if config == nil {
		return false
	}

	if ts.opts.config.Listeners != nil {
		for _, job := range ts.opts.config.Listeners {
			if job.Host == config.Host && job.Port == uint16(config.Port) {
				return true
			}
		}
	}

	// If matching our daemon config.
	if ts.opts.config.DaemonMode.Host == config.Host && ts.opts.config.DaemonMode.Port == config.Port {
		return true
	}

	return false
}

func getRandomID() string {
	seededRand := insecureRand.New(insecureRand.NewSource(time.Now().UnixNano()))
	buf := make([]byte, 32)
	seededRand.Read(buf)
	return hex.EncodeToString(buf)
}
