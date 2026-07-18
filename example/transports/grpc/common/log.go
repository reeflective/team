package common

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
	"os"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

// This example transport demonstrates that a transport backend is free to use a
// logging library entirely of its own (here logrus), independent of the slog
// backend used by the reeflective/team core. Its gRPC middleware logs through
// the logrus logger below, while the core keeps logging through its slog
// handlers. See the sibling "grpcslog" example for a transport that instead
// plugs into the core's slog loggers via Server/Client.NamedLogger().
var exampleLogrus = newExampleLogrus()

func newExampleLogrus() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	return logger
}

// Logrus returns the shared *logrus.Logger used by this example transport's
// gRPC middleware. Consumers modelling their own logrus-based transport can use
// this as a reference for wiring a self-owned logging backend.
func Logrus() *logrus.Logger {
	return exampleLogrus
}

// LogEntry returns a *logrus.Entry tagged with a package/stream, mirroring the
// shape of the core's NamedLogger() but backed by this example's own logrus logger.
func LogEntry(pkg, stream string) *logrus.Entry {
	return exampleLogrus.WithField("logger", pkg).WithField("stream", stream)
}

// CodeToLevel maps a grpc response code to a logging level.
func CodeToLevel(code codes.Code) logrus.Level {
	switch code {
	case codes.OK:
		return logrus.DebugLevel
	case codes.Canceled:
		return logrus.DebugLevel
	case codes.Unknown:
		return logrus.ErrorLevel
	case codes.InvalidArgument:
		return logrus.WarnLevel
	case codes.DeadlineExceeded:
		return logrus.WarnLevel
	case codes.NotFound:
		return logrus.DebugLevel
	case codes.AlreadyExists:
		return logrus.DebugLevel
	case codes.PermissionDenied:
		return logrus.WarnLevel
	case codes.Unauthenticated:
		return logrus.WarnLevel
	case codes.ResourceExhausted:
		return logrus.WarnLevel
	case codes.FailedPrecondition:
		return logrus.WarnLevel
	case codes.Aborted:
		return logrus.WarnLevel
	case codes.OutOfRange:
		return logrus.WarnLevel
	case codes.Unimplemented:
		return logrus.ErrorLevel
	case codes.Internal:
		return logrus.ErrorLevel
	case codes.Unavailable:
		return logrus.WarnLevel
	case codes.DataLoss:
		return logrus.ErrorLevel
	default:
		return logrus.ErrorLevel
	}
}
