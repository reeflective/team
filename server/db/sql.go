package db

// Wiregost - Post-Exploitation & Implant Framework
// Copyright © 2020 Para
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ErrRecordNotFound - Record not found error
var ErrRecordNotFound = gorm.ErrRecordNotFound

// NewDatabaseClient - Initialize the db client
func NewDatabaseClient(dbConfig *Config, log *logrus.Logger) *gorm.DB {
	var dbClient *gorm.DB

	dsn, err := dbConfig.DSN()
	if err != nil {
		panic(err)
	}

	dbLog := getGormLogger(log, dbConfig.LogLevel)

	switch dbConfig.Dialect {
	case Sqlite:
		dbClient = sqliteClient(dsn, dbLog)
		log.Infof("Connecting to SQLite database %s", dsn)
	case Postgres:
		dbClient = postgresClient(dsn, dbLog)
		log.Infof("Connecting to PostgreSQL database %s", dsn)
	case MySQL:
		dbClient = mySQLClient(dsn, dbLog)
		log.Infof("Connecting to MySQL database %s", dsn)
	default:
		panic(fmt.Sprintf("Unknown DB Dialect: '%s'", dbConfig.Dialect))
	}

	err = dbClient.AutoMigrate(
		&Certificate{},
		&User{},
		&KeyValue{},
	)
	if err != nil {
		log.Error(err)
	}

	// Get generic database object sql.DB to use its functions
	sqlDB, err := dbClient.DB()
	if err != nil {
		log.Error(err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	return dbClient.Session(&gorm.Session{
		FullSaveAssociations: true,
	})
}

func postgresClient(dsn string, log logger.Interface) *gorm.DB {
	dbClient, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      log,
	})
	if err != nil {
		panic(err)
	}
	return dbClient
}

func mySQLClient(dsn string, log logger.Interface) *gorm.DB {
	dbClient, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      log,
	})
	if err != nil {
		panic(err)
	}
	return dbClient
}
