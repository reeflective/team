package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/reeflective/team/server/db"
)

const (
	databaseConfigFileName = "database.json"
)

// GetDatabaseConfigPath - File path to config.json
func (s *Server) DatabaseConfigPath() string {
	appDir := s.AppDir()
	databaseConfigLog := s.NamedLogger("config", "database")
	databaseConfigPath := filepath.Join(appDir, "configs", databaseConfigFileName)
	databaseConfigLog.Debugf("Loading config from %s", databaseConfigPath)
	return databaseConfigPath
}

func encodeParams(rawParams map[string]string) string {
	params := url.Values{}
	for key, value := range rawParams {
		params.Add(key, value)
	}
	return params.Encode()
}

// Save - Save config file to disk
func (s *Server) SaveDatabaseConfig(c *db.Config) error {
	databaseConfigLog := s.NamedLogger("config", "database")

	configPath := s.DatabaseConfigPath()
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
	databaseConfigLog := s.NamedLogger("config", "database")

	configPath := s.DatabaseConfigPath()
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

// UserByToken - Select a teamserver user by token value
func (s *Server) UserByToken(value string) (*db.User, error) {
	if len(value) < 1 {
		return nil, db.ErrRecordNotFound
	}
	operator := &db.User{}
	err := s.db.Where(&db.User{
		Token: value,
	}).First(operator).Error
	return operator, err
}

// UserAll - Select all teamserver users from the database
func (s *Server) UserAll() ([]*db.User, error) {
	operators := []*db.User{}
	err := s.db.Distinct("Name").Find(&operators).Error
	return operators, err
}

// GetKeyValue - Get a value from a key
func (s *Server) GetKeyValue(key string) (string, error) {
	keyValue := &db.KeyValue{}
	err := s.db.Where(&db.KeyValue{
		Key: key,
	}).First(keyValue).Error
	return keyValue.Value, err
}

// SetKeyValue - Set the value for a key/value pair
func (s *Server) SetKeyValue(key string, value string) error {
	err := s.db.Where(&db.KeyValue{
		Key: key,
	}).First(&db.KeyValue{}).Error
	if err == db.ErrRecordNotFound {
		err = s.db.Create(&db.KeyValue{
			Key:   key,
			Value: value,
		}).Error
	} else {
		err = s.db.Where(&db.KeyValue{
			Key: key,
		}).Updates(db.KeyValue{
			Key:   key,
			Value: value,
		}).Error
	}
	return err
}

// DeleteKeyValue - Delete a key/value pair
func (s *Server) DeleteKeyValue(key string, value string) error {
	return s.db.Delete(&db.KeyValue{
		Key: key,
	}).Error
}
