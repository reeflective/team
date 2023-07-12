package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/reeflective/team/internal/db"
	"github.com/reeflective/team/internal/log"
)

const (
	maxIdleConns = 10
	maxOpenConns = 100
)

// initDatabase should be called once when a teamserver is created.
func (ts *Server) initDatabase() (err error) {
	ts.dbInitOnce.Do(func() {
		dbLogger := ts.NamedLogger("database", "database")

		if ts.db != nil {
			return
		}

		ts.opts.dbConfig, err = ts.getDatabaseConfig()
		if err != nil {
			return
		}

		ts.db, err = db.NewClient(ts.opts.dbConfig, dbLogger)
		if err != nil {
			return
		}
	})

	return nil
}

// GetDatabaseConfigPath - File path to config.json.
func (ts *Server) dbConfigPath() string {
	appDir := ts.AppDir()
	log := ts.NamedLogger("config", "database")
	databaseConfigPath := filepath.Join(appDir, "configs", fmt.Sprintf("%s.%s", ts.Name()+"_database", configFileExt))
	log.Debugf("Loading config from %s", databaseConfigPath)

	return databaseConfigPath
}

// Save - Save config file to disk. If the server is configured
// to run in-memory only, the config is not saved.
func (ts *Server) saveDatabaseConfig(cfg *db.Config) error {
	if ts.opts.inMemory {
		return nil
	}

	dblog := ts.NamedLogger("config", "database")

	configPath := ts.dbConfigPath()
	configDir := path.Dir(configPath)

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		dblog.Debugf("Creating config dir %s", configDir)

		err := os.MkdirAll(configDir, log.DirPerm)
		if err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}

	dblog.Infof("Saving config to %s", configPath)

	return os.WriteFile(configPath, data, log.FilePerm)
}

// getDatabaseConfig returns a working database configuration,
// either fetched from the file system, adjusted with in-code
// options, or a default one.
// If an error happens, it is returned with a nil configuration.
func (ts *Server) getDatabaseConfig() (*db.Config, error) {
	log := ts.NamedLogger("config", "database")

	config := ts.getDefaultDatabaseConfig()
	if config.Database == ":memory:" {
		return config, nil
	}

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
	cfg := &db.Config{
		Dialect:      db.Sqlite,
		MaxIdleConns: maxIdleConns,
		MaxOpenConns: maxOpenConns,

		LogLevel: "warn",
	}

	if ts.opts.inMemory {
		cfg.Database = ":memory:"
	} else {
		cfg.Database = filepath.Join(ts.AppDir(), fmt.Sprintf("%s.teamserver.db", ts.name))
	}

	return cfg
}
