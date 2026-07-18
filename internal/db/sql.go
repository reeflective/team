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
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	// SQLiteInMemoryHost is the default string used by SQLite
	// as a host when ran in memory (string value is ":memory:").
	SQLiteInMemoryHost = ":memory:"
)

var (
	// ErrRecordNotFound - Record not found error.
	ErrRecordNotFound = gorm.ErrRecordNotFound

	// ErrUnsupportedDialect - An invalid dialect was specified.
	ErrUnsupportedDialect = errors.New("Unknown/unsupported DB Dialect")
)

// NewClient initializes a database client connection to a backend specified in config.
func NewClient(dbConfig *Config, dbLogger *slog.Logger) (*gorm.DB, error) {
	var dbClient *gorm.DB

	dsn, err := dbConfig.DSN()
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal database DSN: %w", err)
	}

	// Logging middleware (queries)
	dbLog := newGormLogger(dbLogger, dbConfig.LogLevel)
	logDbDsn := fmt.Sprintf("%s (%s:%d)", dbConfig.Database, dbConfig.Host, dbConfig.Port)

	switch dbConfig.Dialect {
	case Sqlite:
		dbLogger.Info(fmt.Sprintf("Connecting to SQLite database %s", logDbDsn))

		dbClient, err = sqliteClient(dsn, dbLog)
		if err != nil {
			return nil, fmt.Errorf("Database connection failed: %w", err)
		}

	case Postgres:
		dbLogger.Info(fmt.Sprintf("Connecting to PostgreSQL database %s", logDbDsn))

		dbClient, err = postgresClient(dsn, dbLog)
		if err != nil {
			return nil, fmt.Errorf("Database connection failed: %w", err)
		}

	case MySQL:
		dbLogger.Info(fmt.Sprintf("Connecting to MySQL database %s", logDbDsn))

		dbClient, err = mySQLClient(dsn, dbLog)
		if err != nil {
			return nil, fmt.Errorf("Database connection failed: %w", err)
		}
	default:
		return nil, fmt.Errorf("%w: '%s'", ErrUnsupportedDialect, dbConfig.Dialect)
	}

	// For SQLite, force an actual page read now so that a wrong encryption key
	// (or an otherwise corrupt/unreadable file) surfaces here as a clean error,
	// instead of panicking later inside AutoMigrate's schema introspection.
	if dbConfig.Dialect == Sqlite {
		var count int
		if err := dbClient.Raw("SELECT count(*) FROM sqlite_master").Scan(&count).Error; err != nil {
			return nil, fmt.Errorf("Database open failed (wrong encryption key or corrupt database?): %w", err)
		}
	}

	err = dbClient.AutoMigrate(Schema()...)
	if err != nil {
		dbLogger.Error(err.Error())
	}

	// Get generic database object sql.DB to use its functions
	sqlDB, err := dbClient.DB()
	if err != nil {
		dbLogger.Error(err.Error())
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	return dbClient, nil
}

// Schema returns all objects which should be registered
// to the teamserver database backend.
func Schema() []any {
	return []any{
		&Certificate{},
		&User{},
	}
}

func postgresClient(dsn string, log logger.Interface) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      log,
	})
}

func mySQLClient(dsn string, log logger.Interface) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      log,
	})
}
