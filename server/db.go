package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/server/db"
)

const (
	databaseConfigFileName = "database.json"
)

// GetDatabaseConfigPath - File path to config.json
func (s *Server) dbConfigPath() string {
	appDir := s.AppDir()
	databaseConfigLog := log.NamedLogger(s.log, "config", "database")
	databaseConfigPath := filepath.Join(appDir, "configs", databaseConfigFileName)
	databaseConfigLog.Debugf("Loading config from %s", databaseConfigPath)
	return databaseConfigPath
}

// Save - Save config file to disk
func (s *Server) SaveDatabaseConfig(c *db.Config) error {
	databaseConfigLog := log.NamedLogger(s.log, "config", "database")

	configPath := s.dbConfigPath()
	configDir := path.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		databaseConfigLog.Debugf("Creating config dir %s", configDir)
		err := os.MkdirAll(configDir, 0o700)
		if err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	databaseConfigLog.Infof("Saving config to %s", configPath)
	err = os.WriteFile(configPath, data, 0o600)
	if err != nil {
		databaseConfigLog.Errorf("Failed to write config %s", err)
	}
	return nil
}

// GetDatabaseConfig - Get config value
func (s *Server) GetDatabaseConfig() *db.Config {
	databaseConfigLog := log.NamedLogger(s.log, "config", "database")

	configPath := s.dbConfigPath()
	config := s.getDefaultDatabaseConfig()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			databaseConfigLog.Errorf("Failed to read config file %s", err)
			return config
		}
		err = json.Unmarshal(data, config)
		if err != nil {
			databaseConfigLog.Errorf("Failed to parse config file %s", err)
			return config
		}
	} else {
		databaseConfigLog.Warnf("Config file does not exist, using defaults")
	}

	if config.MaxIdleConns < 1 {
		config.MaxIdleConns = 1
	}
	if config.MaxOpenConns < 1 {
		config.MaxOpenConns = 1
	}

	err := s.SaveDatabaseConfig(config) // This updates the config with any missing fields
	if err != nil {
		databaseConfigLog.Errorf("Failed to save default config %s", err)
	}
	return config
}

func (s *Server) getDefaultDatabaseConfig() *db.Config {
	return &db.Config{
		Database:     filepath.Join(s.AppDir(), fmt.Sprintf("%s.db", s.name)),
		Dialect:      db.Sqlite,
		MaxIdleConns: 10,
		MaxOpenConns: 100,

		LogLevel: "warn",
	}
}

func encodeParams(rawParams map[string]string) string {
	params := url.Values{}
	for key, value := range rawParams {
		params.Add(key, value)
	}
	return params.Encode()
}
