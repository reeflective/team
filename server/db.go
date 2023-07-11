package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/reeflective/team/internal/db"
)

const (
	maxIdleConns = 10
	maxOpenConns = 100
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
func (ts *Server) saveDatabaseConfig(cfg *db.Config) error {
	log := ts.NamedLogger("config", "database")

	configPath := ts.dbConfigPath()
	configDir := path.Dir(configPath)

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		log.Debugf("Creating config dir %s", configDir)

		err := os.MkdirAll(configDir, dirWriteModePerm)
		if err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}

	log.Infof("Saving config to %s", configPath)

	return os.WriteFile(configPath, data, FileWriteModePerm)
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
			return nil, fmt.Errorf("Failed to read config file %w", err)
		}

		err = json.Unmarshal(data, config)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse config file %w", err)
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
		MaxIdleConns: maxIdleConns,
		MaxOpenConns: maxOpenConns,

		LogLevel: "warn",
	}
}
