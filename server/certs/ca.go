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
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// -----------------------
//  CERTIFICATE AUTHORITY
// -----------------------

func (c *Manager) getCertDir() string {
	rootDir := c.appDir
	certDir := filepath.Join(rootDir, "certs")
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		err := os.MkdirAll(certDir, 0o700)
		if err != nil {
			c.log.Fatalf("Failed to create cert dir %s", err)
		}
	}
	return certDir
}

// GetUserCertificateAutority returns the certificate authority for teamserver users.
func (c *Manager) GetUserCertificateAutority() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	return c.GetCertificateAuthority(userCA)
}

// GetUserCertificateAutorityPEM returns the certificate authority for teamserver users, PEM-encoded.
func (c *Manager) GetUserCertificateAutorityPEM() ([]byte, []byte, error) {
	return c.GetCertificateAuthorityPEM(userCA)
}

// GenerateCertificateAuthority - Creates a new CA cert for a given type
func (c *Manager) GenerateCertificateAuthority(caType string, commonName string) (*x509.Certificate, *ecdsa.PrivateKey) {
	storageDir := c.getCertDir()
	certFilePath := filepath.Join(storageDir, fmt.Sprintf("%s-ca-cert.pem", caType))
	if _, err := os.Stat(certFilePath); os.IsNotExist(err) {
		c.log.Infof("Generating certificate authority for '%s'", caType)
		cert, key := c.GenerateECCCertificate(caType, commonName, true, false)
		c.SaveCertificateAuthority(caType, cert, key)
	}
	cert, key, err := c.GetCertificateAuthority(caType)
	if err != nil {
		c.log.Fatalf("Failed to load CA %s", err)
	}
	return cert, key
}

// GetCertificateAuthority - Get the current CA certificate
func (c *Manager) GetCertificateAuthority(caType string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certPEM, keyPEM, err := c.GetCertificateAuthorityPEM(caType)
	if err != nil {
		return nil, nil, err
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		c.log.Error("Failed to parse certificate PEM")
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		c.log.Error("Failed to parse certificate: " + err.Error())
		return nil, nil, err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		c.log.Error("Failed to parse certificate PEM")
		return nil, nil, err
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		c.log.Error(err)
		return nil, nil, err
	}

	return cert, key, nil
}

// GetCertificateAuthorityPEM - Get PEM encoded CA cert/key
func (c *Manager) GetCertificateAuthorityPEM(caType string) ([]byte, []byte, error) {
	caType = filepath.Base(caType)
	caCertPath := filepath.Join(c.getCertDir(), fmt.Sprintf("%s-ca-cert.pem", caType))
	caKeyPath := filepath.Join(c.getCertDir(), fmt.Sprintf("%s-ca-key.pem", caType))

	certPEM, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		c.log.Error(err)
		return nil, nil, err
	}

	keyPEM, err := ioutil.ReadFile(caKeyPath)
	if err != nil {
		c.log.Error(err)
		return nil, nil, err
	}
	return certPEM, keyPEM, nil
}

// SaveCertificateAuthority - Save the certificate and the key to the filesystem
// doesn't return an error because errors are fatal. If we can't generate CAs,
// then we can't secure communication and we should die a horrible death.
func (c *Manager) SaveCertificateAuthority(caType string, cert []byte, key []byte) {
	storageDir := c.getCertDir()
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		os.MkdirAll(storageDir, 0o700)
	}

	// CAs get written to the filesystem since we control the names and makes them
	// easier to move around/backup
	certFilePath := filepath.Join(storageDir, fmt.Sprintf("%s-ca-cert.pem", caType))
	keyFilePath := filepath.Join(storageDir, fmt.Sprintf("%s-ca-key.pem", caType))

	err := ioutil.WriteFile(certFilePath, cert, 0o600)
	if err != nil {
		c.log.Fatalf("Failed write certificate data to: %s", certFilePath)
	}

	err = ioutil.WriteFile(keyFilePath, key, 0o600)
	if err != nil {
		c.log.Fatalf("Failed write certificate data to: %s", keyFilePath)
	}
}
