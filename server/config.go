package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/internal/transport"
	"github.com/sirupsen/logrus"
)

const (
	blankHost = "-"
	blankPort = uint16(0)
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
		Name string `json:"name"`
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	} `json:"listeners"`
}

// GetServerConfigPath - File path to the server config.json file.
func (ts *Server) ConfigPath() string {
	appDir := ts.TeamDir()
	configDir := filepath.Join(appDir, "configs")

	err := ts.fs.MkdirAll(configDir, log.DirPerm)
	if err != nil {
		ts.log().Errorf("cannot write to %s config dir: %s", configDir, err)
	}

	serverConfigPath := filepath.Join(configDir, fmt.Sprintf("%s.%s", ts.Name(), command.ServerConfigExt))

	return serverConfigPath
}

// GetConfig returns the team server configuration struct.
func (ts *Server) GetConfig() *Config {
	cfgLog := ts.NamedLogger("config", "server")

	if ts.opts.inMemory {
		return ts.opts.config
	}

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
		cfgLog.Warnf("Teamserver: no config file found, using and saving defaults")
	}

	if ts.opts.config.Log.Level < 0 {
		ts.opts.config.Log.Level = 0
	}

	if int(logrus.TraceLevel) < ts.opts.config.Log.Level {
		ts.opts.config.Log.Level = int(logrus.TraceLevel)
	}

	// This updates the config with any missing fields
	err := ts.SaveConfig(ts.opts.config)
	if err != nil {
		cfgLog.Errorf("Failed to save default config %s", err)
	}

	return ts.opts.config
}

// Save - Save config file to disk.
func (ts *Server) SaveConfig(cfg *Config) error {
	cfgLog := ts.NamedLogger("config", "server")

	if ts.opts.inMemory {
		return nil
	}

	configPath := ts.ConfigPath()
	configDir := filepath.Dir(configPath)

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		cfgLog.Debugf("Creating config dir %s", configDir)

		err := os.MkdirAll(configDir, log.DirPerm)
		if err != nil {
			return ts.errorf("%w: %w", ErrConfig, err)
		}
	}

	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}

	cfgLog.Debugf("Saving config to %s", configPath)

	err = os.WriteFile(configPath, data, log.FileReadPerm)
	if err != nil {
		return ts.errorf("%w: failed to write config: %s", ErrConfig, err)
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
			Name string `json:"name"`
			Host string `json:"host"`
			Port uint16 `json:"port"`
			ID   string `json:"id"`
		}{},
	}
}

func getRandomID() string {
	seededRand := insecureRand.New(insecureRand.NewSource(time.Now().UnixNano()))
	buf := make([]byte, transport.TokenLength)
	seededRand.Read(buf)

	return hex.EncodeToString(buf)
}
