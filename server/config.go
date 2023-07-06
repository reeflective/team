package server

import (
	"encoding/hex"
	"encoding/json"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/log"
	"github.com/sirupsen/logrus"
)

const serverConfigFileName = "server.json"

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
func (s *Server) ConfigPath() string {
	appDir := s.AppDir()

	log := log.NamedLogger(s.log, "config", "server")
	serverConfigPath := filepath.Join(appDir, "configs", serverConfigFileName)
	log.Debugf("Loading config from %s", serverConfigPath)
	return serverConfigPath
}

// GetConfig returns the team server configuration struct.
func (s *Server) GetConfig() *Config {
	cfgLog := log.NamedLogger(s.log, "config", "server")

	configPath := s.ConfigPath()
	config := s.getDefaultServerConfig()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			cfgLog.Errorf("Failed to read config file %s", err)
			return config
		}
		err = json.Unmarshal(data, config)
		if err != nil {
			cfgLog.Errorf("Failed to parse config file %s", err)
			return config
		}
	} else {
		cfgLog.Warnf("Config file does not exist, using defaults")
	}

	if config.Log.Level < 0 {
		config.Log.Level = 0
	}
	if 6 < config.Log.Level {
		config.Log.Level = 6
	}
	s.log.SetLevel(log.LevelFrom(config.Log.Level))

	// This updates the config with any missing fields
	err := s.SaveConfig(config)
	if err != nil {
		cfgLog.Errorf("Failed to save default config %s", err)
	}
	return config
}

// Save - Save config file to disk
func (s *Server) SaveConfig(c *Config) error {
	log := log.NamedLogger(s.log, "config", "server")

	configPath := s.ConfigPath()
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		log.Debugf("Creating config dir %s", configDir)
		err := os.MkdirAll(configDir, 0o700)
		if err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	log.Debugf("Saving config to %s", configPath)
	err = os.WriteFile(configPath, data, 0o600)
	if err != nil {
		log.Errorf("Failed to write config %s", err)
	}
	return nil
}

// AddListenerJob adds a teamserver listener job to the config and saves it.
func (s *Server) AddListener(host string, port uint16) error {
	listener := struct {
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	}{
		Host: host,
		Port: port,
		ID:   getRandomID(),
	}

	s.config.Listeners = append(s.config.Listeners, listener)

	return s.SaveConfig(s.config)
}

// RemoveListenerJob removes a server listener job from the configuration and saves it.
func (c *Server) RemoveListener(id string) {
	if c.config.Listeners == nil {
		return
	}

	defer c.SaveConfig(c.config)

	var listeners []struct {
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	}

	for _, listener := range c.config.Listeners {
		if listener.ID != id {
			listeners = append(listeners, listener)
		}
	}

	c.config.Listeners = listeners
}

func (c *Server) getDefaultServerConfig() *Config {
	return &Config{
		DaemonMode: struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		}{
			Port: int(c.opts.port),
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

func (s *Server) clientServerMatch(config *client.Config) bool {
	if config == nil {
		return false
	}

	if s.config.Listeners != nil {
		for _, job := range s.config.Listeners {
			if job.Host == config.Host && job.Port == uint16(config.Port) {
				return true
			}
		}
	}

	// If matching our daemon config.
	if s.config.DaemonMode.Host == config.Host && s.config.DaemonMode.Port == config.Port {
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
