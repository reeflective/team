package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/certs"
	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/server/db"
)

var namePattern = regexp.MustCompile("^[a-zA-Z0-9_-]*$") // Only allow alphanumeric chars

// NewUserConfig generates a new user client connection configuration.
func (s *Server) NewUserConfig(userName string, lhost string, lport uint16) ([]byte, error) {
	if !namePattern.MatchString(userName) {
		return nil, errors.New("invalid user name (alphanumerics only)")
	}
	if userName == "" {
		return nil, errors.New("user name required")
	}
	if lhost == "" {
		return nil, errors.New("invalid team server host (empty)")
	}
	if lport == blankPort {
		lport = s.opts.port
	}

	rawToken := s.newUserToken()
	digest := sha256.Sum256([]byte(rawToken))
	dbuser := &db.User{
		Name:  userName,
		Token: hex.EncodeToString(digest[:]),
	}
	err := s.db.Save(dbuser).Error
	if err != nil {
		return nil, err
	}

	publicKey, privateKey, err := s.certs.UserClientGenerateCertificate(userName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate %s", err)
	}

	caCertPEM, _, _ := s.certs.GetUsersCAPEM()
	config := client.Config{
		User:          userName,
		Token:         rawToken,
		Host:          lhost,
		Port:          int(lport),
		CACertificate: string(caCertPEM),
		PrivateKey:    string(privateKey),
		Certificate:   string(publicKey),
	}

	return json.Marshal(config)
}

// DeleteUser deletes a user from the teamserver database, in fact forbidding
// it to ever reconnect with the user's credentials (client configuration file)
func (s *Server) DeleteUser(name string) error {
	err := s.db.Where(&db.User{
		Name: name,
	}).Delete(&db.User{}).Error
	if err != nil {
		return err
	}

	s.userTokens = &sync.Map{}

	return s.certs.UserClientRemoveCertificate(name)
}

// GetUsersCA returns the bytes of a PEM-encoded certificate authority,
// which may contain multiple teamserver users and their master.
func (s *Server) GetUsersCA() ([]byte, []byte, error) {
	return s.certs.GetUsersCAPEM()
}

// SaveUsersCA accepts the public and private parts of a Certificate
// Authority containing one or more users to add to the teamserver.
func (s *Server) SaveUsersCA(cert, key []byte) {
	s.certs.SaveUsersCA(cert, key)
}

// newUserToken - Generate a new user authentication token.
func (s *Server) newUserToken() string {
	buf := make([]byte, 32)
	n, err := rand.Read(buf)
	if err != nil || n != len(buf) {
		panic(errors.New("failed to read from secure rand"))
	}
	return hex.EncodeToString(buf)
}

// userByToken - Select a teamserver user by token value
func (s *Server) userByToken(value string) (*db.User, error) {
	if len(value) < 1 {
		return nil, db.ErrRecordNotFound
	}
	user := &db.User{}
	err := s.db.Where(&db.User{
		Token: value,
	}).First(user).Error
	return user, err
}

// getUserTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS parameters, we choose sensible defaults instead.
func (s *Server) getUserTLSConfig(host string) *tls.Config {
	log := log.NamedLogger(s.log, "certs", "mtls")
	caCertPtr, _, err := s.certs.GetUsersCA()
	if err != nil {
		log.Error("Failed to get users certificate authority")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	_, _, err = s.certs.UserServerGetCertificate(host)
	if err == certs.ErrCertDoesNotExist {
		s.certs.UserServerGenerateCertificate(host)
	}

	certPEM, keyPEM, err := s.certs.UserServerGetCertificate(host)
	if err != nil {
		log.Errorf("Failed to generate or fetch certificate %s", err)
		return nil
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Errorf("Error loading server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	if keyLogger := s.certs.NewKeyLogger(); keyLogger != nil {
		tlsConfig.KeyLogWriter = s.certs.NewKeyLogger()
	}

	return tlsConfig
}
