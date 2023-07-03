package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/server/db"
)

var namePattern = regexp.MustCompile("^[a-zA-Z0-9_-]*$") // Only allow alphanumeric chars

// NewUserConfig generates a new user client connection configuration.
func (s *Server) NewUserConfig(operatorName string, lhost string, lport uint16) ([]byte, error) {
	if !namePattern.MatchString(operatorName) {
		return nil, errors.New("invalid operator name (alphanumerics only)")
	}
	if operatorName == "" {
		return nil, errors.New("operator name required")
	}
	if lhost == "" {
		return nil, errors.New("invalid lhost")
	}

	rawToken := db.GenerateOperatorToken()
	digest := sha256.Sum256([]byte(rawToken))
	dbOperator := &db.User{
		Name:  operatorName,
		Token: hex.EncodeToString(digest[:]),
	}
	err := s.db.Save(dbOperator).Error
	if err != nil {
		return nil, err
	}

	publicKey, privateKey, err := s.certs.UserClientGenerateCertificate(operatorName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate %s", err)
	}

	caCertPEM, _, _ := s.certs.GetUserCertificateAutorityPEM()
	config := client.Config{
		User:          operatorName,
		Token:         rawToken,
		Host:          lhost,
		Port:          int(lport),
		CACertificate: string(caCertPEM),
		PrivateKey:    string(privateKey),
		Certificate:   string(publicKey),
	}

	return json.Marshal(config)
}

// StartPersistentJobs starts all teamserver listeners,
// aborting and returning an error if one of those raise one.
func (s *Server) StartPersistentJobs() error {
	if s.config.Jobs == nil {
		return nil
	}

	for _, j := range s.config.Jobs.Multiplayer {
		_, _, err := s.ServeAddr(j.Host, j.Port)
		if err != nil {
			return err
		}
	}

	return nil
}
