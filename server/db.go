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
func (ts *Server) dbConfigPath() string {
	appDir := ts.AppDir()
	log := log.NewNamed(ts.log, "config", "database")
	databaseConfigPath := filepath.Join(appDir, "configs", databaseConfigFileName)
	log.Debugf("Loading config from %s", databaseConfigPath)
	return databaseConfigPath
}

// Save - Save config file to disk
func (ts *Server) saveDatabaseConfig(c *db.Config) error {
	log := log.NewNamed(ts.log, "config", "database")

	configPath := ts.dbConfigPath()
	configDir := path.Dir(configPath)
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
	log.Infof("Saving config to %s", configPath)
	err = os.WriteFile(configPath, data, 0o600)
	if err != nil {
		log.Errorf("Failed to write config %s", err)
	}
	return nil
}

// getDatabaseConfig - Get config value
func (ts *Server) getDatabaseConfig() *db.Config {
	log := log.NewNamed(ts.log, "config", "database")

	configPath := ts.dbConfigPath()
	config := ts.getDefaultDatabaseConfig()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			log.Errorf("Failed to read config file %s", err)
			return config
		}
		err = json.Unmarshal(data, config)
		if err != nil {
			log.Errorf("Failed to parse config file %s", err)
			return config
		}
	} else {
		log.Warnf("Config file does not exist, using defaults")
	}

	if config.MaxIdleConns < 1 {
		config.MaxIdleConns = 1
	}
	if config.MaxOpenConns < 1 {
		config.MaxOpenConns = 1
	}

	err := ts.saveDatabaseConfig(config) // This updates the config with any missing fields
	if err != nil {
		log.Errorf("Failed to save default config %s", err)
	}
	return config
}

func (ts *Server) getDefaultDatabaseConfig() *db.Config {
	return &db.Config{
		Database:     filepath.Join(ts.AppDir(), fmt.Sprintf("%s.db", ts.name)),
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
