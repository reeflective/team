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
	"log/slog"

	"google.golang.org/grpc/codes"
)

// CodeToLevel maps a grpc response code to an slog logging level.
func CodeToLevel(code codes.Code) slog.Level {
	switch code {
	case codes.OK:
		return slog.LevelDebug
	case codes.Canceled:
		return slog.LevelDebug
	case codes.Unknown:
		return slog.LevelError
	case codes.InvalidArgument:
		return slog.LevelWarn
	case codes.DeadlineExceeded:
		return slog.LevelWarn
	case codes.NotFound:
		return slog.LevelDebug
	case codes.AlreadyExists:
		return slog.LevelDebug
	case codes.PermissionDenied:
		return slog.LevelWarn
	case codes.Unauthenticated:
		return slog.LevelWarn
	case codes.ResourceExhausted:
		return slog.LevelWarn
	case codes.FailedPrecondition:
		return slog.LevelWarn
	case codes.Aborted:
		return slog.LevelWarn
	case codes.OutOfRange:
		return slog.LevelWarn
	case codes.Unimplemented:
		return slog.LevelError
	case codes.Internal:
		return slog.LevelError
	case codes.Unavailable:
		return slog.LevelWarn
	case codes.DataLoss:
		return slog.LevelError
	default:
		return slog.LevelError
	}
}
