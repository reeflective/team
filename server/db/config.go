package db

import (
	"errors"
	"fmt"
	"net/url"
)

const (
	// Sqlite - SQLite protocol
	Sqlite = "sqlite3"
	// Postgres - Postgresql protocol
	Postgres = "postgresql"
	// MySQL - MySQL protocol
	MySQL = "mysql"

	databaseConfigFileName = "database.json"
)

// ErrInvalidDialect - An invalid dialect was specified
var ErrInvalidDialect = errors.New("invalid SQL Dialect")

// Config - Server config
type Config struct {
	Dialect  string `json:"dialect"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`

	Params map[string]string `json:"params"`

	MaxIdleConns int `json:"max_idle_conns"`
	MaxOpenConns int `json:"max_open_conns"`

	LogLevel string `json:"log_level"`
}

// DSN - Get the db connections string
// https://github.com/go-sql-driver/mysql#examples
func (c *Config) DSN() (string, error) {
	switch c.Dialect {
	case Sqlite:
		filePath := c.Database
		params := encodeParams(c.Params)
		return fmt.Sprintf("file:%s?%s", filePath, params), nil
	case MySQL:
		user := url.QueryEscape(c.Username)
		password := url.QueryEscape(c.Password)
		db := url.QueryEscape(c.Database)
		host := fmt.Sprintf("%s:%d", url.QueryEscape(c.Host), c.Port)
		params := encodeParams(c.Params)
		return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", user, password, host, db, params), nil
	case Postgres:
		user := url.QueryEscape(c.Username)
		password := url.QueryEscape(c.Password)
		db := url.QueryEscape(c.Database)
		host := url.QueryEscape(c.Host)
		port := c.Port
		params := encodeParams(c.Params)
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s %s", host, port, user, password, db, params), nil
	default:
		return "", ErrInvalidDialect
	}
}

func encodeParams(rawParams map[string]string) string {
	params := url.Values{}
	for key, value := range rawParams {
		params.Add(key, value)
	}
	return params.Encode()
}
