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
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/reeflective/team/server/db"
)

const (
	// userCA - Directory containing user certificates
	userCA = "user"

	clientNamespace = "client" // User clients
	serverNamespace = "server" // User servers
)

// UserClientGenerateCertificate - Generate a certificate signed with a given CA
func (c *Manager) UserClientGenerateCertificate(operator string) ([]byte, []byte, error) {
	cert, key := c.GenerateECCCertificate(userCA, operator, false, true)
	err := c.saveCertificate(userCA, ECCKey, fmt.Sprintf("%s.%s", clientNamespace, operator), cert, key)
	return cert, key, err
}

// UserClientGetCertificate - Helper function to fetch a client cert
func (c *Manager) UserClientGetCertificate(operator string) ([]byte, []byte, error) {
	return c.GetECCCertificate(userCA, fmt.Sprintf("%s.%s", clientNamespace, operator))
}

// UserClientRemoveCertificate - Helper function to remove a client cert
func (c *Manager) UserClientRemoveCertificate(operator string) error {
	return c.RemoveCertificate(userCA, ECCKey, fmt.Sprintf("%s.%s", clientNamespace, operator))
}

// UserServerGetCertificate - Helper function to fetch a server cert
func (c *Manager) UserServerGetCertificate(hostname string) ([]byte, []byte, error) {
	return c.GetECCCertificate(userCA, fmt.Sprintf("%s.%s", serverNamespace, hostname))
}

// UserServerGenerateCertificate - Generate a certificate signed with a given CA
func (c *Manager) UserServerGenerateCertificate(hostname string) ([]byte, []byte, error) {
	cert, key := c.GenerateECCCertificate(userCA, hostname, false, false)
	err := c.saveCertificate(userCA, ECCKey, fmt.Sprintf("%s.%s", serverNamespace, hostname), cert, key)
	return cert, key, err
}

// UserClientListCertificates - Get all client certificates
func (c *Manager) UserClientListCertificates() []*x509.Certificate {
	operatorCerts := []*db.Certificate{}
	result := c.db.Where(&db.Certificate{CAType: userCA}).Find(&operatorCerts)
	if result.Error != nil {
		c.log.Error(result.Error)
		return []*x509.Certificate{}
	}

	c.log.Infof("Found %d operator certs ...", len(operatorCerts))

	certs := []*x509.Certificate{}
	for _, operator := range operatorCerts {
		block, _ := pem.Decode([]byte(operator.CertificatePEM))
		if block == nil {
			c.log.Warn("failed to parse certificate PEM")
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			c.log.Warnf("failed to parse x.509 certificate %v", err)
			continue
		}
		certs = append(certs, cert)
	}
	return certs
}