package server

import "errors"

var (
	// ErrDirectory is an error related to directories used by the teamserver.
	ErrDirectory = errors.New("teamserver directory")

	// ErrLogging is an error related with the logging backend.
	// Some errors can be about writable files/directories.
	ErrLogging = errors.New("logging")

	// ErrConfig is an error related to the teamserver configuration.
	ErrConfig = errors.New("teamserver config")

	// ErrDatabaseConfig is an error related to the database configuration.
	ErrDatabaseConfig = errors.New("teamserver database configuration")

	// ErrDatabase is an error raised by the database backend
	ErrDatabase = errors.New("database")

	// ErrTeamServer is an error raised by the teamserver code.
	ErrTeamServer = errors.New("teamserver")

	// ErrCertificate is an error related to the certificate infrastructure.
	ErrCertificate = errors.New("certificates")

	// ErrUserConfig is an error related to users (teamclients) configuration files.
	ErrUserConfig = errors.New("user configuration")

	// ErrUnauthenticated indicates that a client user could not authenticate itself,
	// whether at connection time, or when requesting server-side features/info.
	ErrUnauthenticated = errors.New("User authentication failure")
)