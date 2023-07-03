package server

import (
	"encoding/hex"
	"encoding/json"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

const serverConfigFileName = "server.json"

// LogConfig - Server logging config
type LogConfig struct {
	Level              int  `json:"level"`
	GRPCUnaryPayloads  bool `json:"grpc_unary_payloads"`
	GRPCStreamPayloads bool `json:"grpc_stream_payloads"`
	TLSKeyLogger       bool `json:"tls_key_logger"`
}

// DaemonConfig - Configure daemon mode
type DaemonConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// JobConfig - Restart Jobs on Load
type JobConfig struct {
	Multiplayer []*MultiplayerJobConfig `json:"multiplayer"`
}

type MultiplayerJobConfig struct {
	Host  string `json:"host"`
	Port  uint16 `json:"port"`
	JobID string `json:"job_id"`
}

// ServerConfig - Server config
type ServerConfig struct {
	DaemonMode   bool          `json:"daemon_mode"`
	DaemonConfig *DaemonConfig `json:"daemon"`
	Logs         *LogConfig    `json:"logs"`
	Jobs         *JobConfig    `json:"jobs,omitempty"`
	GoProxy      string        `json:"go_proxy"`
}

// GetServerConfigPath - File path to the server config.json file.
func (s *Server) ConfigPath() string {
	appDir := s.AppDir()

	serverConfigLog := s.NamedLogger("config", "server")
	serverConfigPath := filepath.Join(appDir, "configs", serverConfigFileName)
	serverConfigLog.Debugf("Loading config from %s", serverConfigPath)
	return serverConfigPath
}

// GetConfig returns the team server configuration struct.
func (s *Server) GetConfig() *ServerConfig {
	serverConfigLog := s.NamedLogger("config", "server")

	configPath := s.ConfigPath()
	config := getDefaultServerConfig()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			serverConfigLog.Errorf("Failed to read config file %s", err)
			return config
		}
		err = json.Unmarshal(data, config)
		if err != nil {
			serverConfigLog.Errorf("Failed to parse config file %s", err)
			return config
		}
	} else {
		serverConfigLog.Warnf("Config file does not exist, using defaults")
	}

	if config.Logs.Level < 0 {
		config.Logs.Level = 0
	}
	if 6 < config.Logs.Level {
		config.Logs.Level = 6
	}
	s.log.SetLevel(levelFrom(config.Logs.Level))

	err := s.SaveConfig(config) // This updates the config with any missing fields
	if err != nil {
		serverConfigLog.Errorf("Failed to save default config %s", err)
	}
	return config
}

// Save - Save config file to disk
func (s *Server) SaveConfig(c *ServerConfig) error {
	serverConfigLog := s.NamedLogger("config", "server")

	configPath := s.ConfigPath()
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		serverConfigLog.Debugf("Creating config dir %s", configDir)
		err := os.MkdirAll(configDir, 0o700)
		if err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	serverConfigLog.Infof("Saving config to %s", configPath)
	err = os.WriteFile(configPath, data, 0o600)
	if err != nil {
		serverConfigLog.Errorf("Failed to write config %s", err)
	}
	return nil
}

// AddMultiplayerJob adds a teamserver listener job to the config and saves it.
func (s *Server) AddMultiplayerJob(config *MultiplayerJobConfig) error {
	if s.config.Jobs == nil {
		s.config.Jobs = &JobConfig{}
	}
	config.JobID = getRandomID()
	s.config.Jobs.Multiplayer = append(s.config.Jobs.Multiplayer, config)

	return s.SaveConfig(s.config)
}

// RemoveJob removes a server listener job from the configuration and saves it.
func (c *Server) RemoveJob(jobID string) {
	if c.config.Jobs == nil {
		return
	}
	defer c.SaveConfig(c.config)
}

func getDefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		DaemonMode: false,
		DaemonConfig: &DaemonConfig{
			Host: "",
			Port: 31336,
		},
		Logs: &LogConfig{
			Level:              int(logrus.InfoLevel),
			GRPCUnaryPayloads:  false,
			GRPCStreamPayloads: false,
		},
		Jobs: &JobConfig{},
	}
}

func getRandomID() string {
	seededRand := insecureRand.New(insecureRand.NewSource(time.Now().UnixNano()))
	buf := make([]byte, 32)
	seededRand.Read(buf)
	return hex.EncodeToString(buf)
}
