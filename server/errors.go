package server

import "errors"

var (
	//
	// Filesystem errors.
	//

	// ErrDirectory is an error related to directories used by the teamserver.
	ErrDirectory = errors.New("teamserver directory")

	// ErrDirectoryUnwritable is an error returned when the teamserver checked for write permissions
	// on a directory path it needs, and that the Go code of the teamserver has determined that the
	// file is really non-user writable. This error is NEVER returned "because the path does not exist".
	ErrDirectoryUnwritable = errors.New("The directory seems to be unwritable (to the app runtime)")

	// ErrLogging is an error related with the logging backend.
	// Some errors can be about writable files/directories.
	ErrLogging = errors.New("logging")

	// ErrSecureRandFailed indicates that the teamserver could not read from the system secure random source.
	ErrSecureRandFailed = errors.New("failed to read from secure rand")

	//
	// Teamserver core errors.
	//

	// ErrConfig is an error related to the teamserver configuration.
	ErrConfig = errors.New("teamserver config")

	// ErrDatabaseConfig is an error related to the database configuration.
	ErrDatabaseConfig = errors.New("teamserver database configuration")

	// ErrDatabase is an error raised by the database backend.
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

	//
	// Listener errors.
	//

	// ErrListenerNotFound indicates that for a given ID, no running or persistent listener could be found.
	ErrListenerNotFound = errors.New("no listener exists with ID")

	// ErrListener indicates an error raised by a listener stack/implementation.
	ErrListener = errors.New("teamserver listener")
)
