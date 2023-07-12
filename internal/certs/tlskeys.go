package certs

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
	"os"

	"github.com/reeflective/team/internal/log"
)

const (
	mtlsCA = "mtls-server"
)

func (c *Manager) NewKeyLogger() *os.File {
	keyFilePath, present := os.LookupEnv("SSLKEYLOGFILE")
	if present {
		keyFile, err := os.OpenFile(keyFilePath, log.FileWriteOpenMode, log.FileReadPerm)
		if err != nil {
			c.log.Errorf("Failed to open TLS key file %v", err)
			return nil
		}

		c.log.Warnf("NOTICE: TLS Keys logged to '%s'\n", keyFilePath)

		return keyFile
	}

	return nil
}
