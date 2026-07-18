package db

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"net/url"
)

const (
	// Sqlite - SQLite protocol.
	Sqlite = "sqlite3"
	// Postgres - Postgresql protocol.
	Postgres = "postgresql"
	// MySQL - MySQL protocol.
	MySQL = "mysql"
)

// Config - Server database configuration.
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

	// EncryptionKey, when set, enables transparent encryption-at-rest for
	// on-disk SQLite databases through the pure-Go adiantum VFS (available on
	// the default and wasm_sqlite builds). It is deliberately NOT serialized:
	// persisting the key next to the database it protects would defeat the
	// purpose. Applications supply it out-of-band (option, env, KMS, prompt).
	EncryptionKey string `json:"-"`
}

// DSN - Get the db connections string
// https://github.com/go-sql-driver/mysql#examples
func (c *Config) DSN() (string, error) {
	switch c.Dialect {
	case Sqlite:
		filePath := c.Database

		params := url.Values{}
		for key, value := range c.Params {
			params.Add(key, value)
		}

		// Enable transparent encryption-at-rest when a key is provided and the
		// database actually lives on disk. The adiantum VFS is pure-Go, so this
		// works on the default and wasm_sqlite builds; in-memory databases have
		// nothing on disk to encrypt, and the cgo_sqlite build has no such VFS.
		if c.EncryptionKey != "" && filePath != SQLiteInMemoryHost {
			params.Set("vfs", "adiantum")
			params.Set("textkey", c.EncryptionKey)
		}

		return fmt.Sprintf("file:%s?%s", filePath, params.Encode()), nil

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
		return "", ErrUnsupportedDialect
	}
}

func encodeParams(rawParams map[string]string) string {
	params := url.Values{}
	for key, value := range rawParams {
		params.Add(key, value)
	}

	return params.Encode()
}
