package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/reeflective/team/internal/db"
)

// GetDatabaseConfigPath - File path to config.json.
func (ts *Server) dbConfigPath() string {
	appDir := ts.AppDir()
	log := ts.NamedLogger("config", "database")
	databaseConfigPath := filepath.Join(appDir, "configs", fmt.Sprintf("%s.%s", ts.Name()+"_database", configFileExt))
	log.Debugf("Loading config from %s", databaseConfigPath)
	return databaseConfigPath
}

// Save - Save config file to disk.
func (ts *Server) saveDatabaseConfig(c *db.Config) error {
	log := ts.NamedLogger("config", "database")

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
	return os.WriteFile(configPath, data, 0o600)
}

// getDatabaseConfig returns a working database configuration,
// either fetched from the file system, adjusted with in-code
// options, or a default one.
// If an error happens, it is returned with a nil configuration.
func (ts *Server) getDatabaseConfig() (*db.Config, error) {
	log := ts.NamedLogger("config", "database")

	config := ts.getDefaultDatabaseConfig()

	configPath := ts.dbConfigPath()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to read config file %s", err)
		}
		err = json.Unmarshal(data, config)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse config file %s", err)
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

	// This updates the config with any missing fields,
	// failing to save is not critical for operation.
	err := ts.saveDatabaseConfig(config)
	if err != nil {
		log.Errorf("Failed to save default config %s", err)
	}

	return config, nil
}

func (ts *Server) getDefaultDatabaseConfig() *db.Config {
	return &db.Config{
		Database:     filepath.Join(ts.AppDir(), fmt.Sprintf("%s.teamserver.db", ts.name)),
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
