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
	"runtime"
	"strings"
	"sync"

	"github.com/reeflective/team"
	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/certs"
	"github.com/reeflective/team/internal/db"
	"github.com/reeflective/team/internal/version"
)

var namePattern = regexp.MustCompile("^[a-zA-Z0-9_-]*$") // Only allow alphanumeric chars

// GetVersion returns the server binary version information.
func (ts *Server) GetVersion() team.Version {
	dirty := version.GitDirty != ""
	semVer := version.Semantic()
	compiled, _ := version.Compiled()

	return team.Version{
		Major:      int32(semVer[0]),
		Minor:      int32(semVer[1]),
		Patch:      int32(semVer[2]),
		Commit:     strings.TrimSuffix(version.GitCommit, "\n"),
		Dirty:      dirty,
		CompiledAt: compiled.Unix(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}
}

// GetUsers returns the list of users in the teamserver database, and their information.
func (ts *Server) GetUsers() ([]team.User, error) {
	var users []team.User

	usersDB := []*db.User{}
	err := ts.db.Distinct("Name").Find(&usersDB).Error

	if err != nil && len(usersDB) == 0 {
		return users, ts.errorf("%w: %w", ErrDatabase, err)
	}

	for _, user := range users {
		users = append(users, team.User{
			Name: user.Name,
			// TODO: online && num clients.
		})
	}

	return users, nil
}

// NewUserConfig generates a new user client connection configuration.
func (ts *Server) NewUserConfig(userName string, lhost string, lport uint16) ([]byte, error) {
	if !namePattern.MatchString(userName) {
		return nil, ts.errorf("%w: invalid user name (alphanumerics only)", ErrUserConfig)
	}
	if userName == "" {
		return nil, ts.errorf("%w: user name required ", ErrUserConfig)
	}
	if lhost == "" {
		return nil, ts.errorf("%w: invalid team server host (empty)", ErrUserConfig)
	}

	if lport == blankPort {
		lport = uint16(ts.opts.config.DaemonMode.Port)
	}

	rawToken, err := ts.newUserToken()
	if err != nil {
		return nil, ts.errorf("%w: %w", ErrUserConfig, err)
	}

	digest := sha256.Sum256([]byte(rawToken))
	dbuser := &db.User{
		Name:  userName,
		Token: hex.EncodeToString(digest[:]),
	}
	err = ts.db.Save(dbuser).Error
	if err != nil {
		return nil, ts.errorf("%w: %w", ErrDatabase, err)
	}

	publicKey, privateKey, err := ts.certs.UserClientGenerateCertificate(userName)
	if err != nil {
		return nil, ts.errorf("%w: failed to generate certificate %w", ErrCertificate, err)
	}

	caCertPEM, _, _ := ts.certs.GetUsersCAPEM()
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
func (ts *Server) DeleteUser(name string) error {
	err := ts.db.Where(&db.User{
		Name: name,
	}).Delete(&db.User{}).Error
	if err != nil {
		return err
	}

	ts.userTokens = &sync.Map{}

	return ts.certs.UserClientRemoveCertificate(name)
}

func (ts *Server) AuthenticateUser(rawToken string) (name string, authorized bool, err error) {
	log := ts.NamedLogger("server", "auth")
	log.Debugf("Authorization-checking user token ...")

	// Check auth cache
	digest := sha256.Sum256([]byte(rawToken))
	token := hex.EncodeToString(digest[:])

	if name, ok := ts.userTokens.Load(token); ok {
		log.Debugf("Token in cache!")
		return name.(string), true, nil
	}

	user, err := ts.userByToken(token)
	if err != nil || user == nil {
		return "", false, ts.errorf("%w: %w", ErrUnauthenticated, err)
	}

	log.Debugf("Valid user token for %s", user.Name)
	ts.userTokens.Store(token, user.Name)

	return user.Name, true, nil
}

// GetUsersCA returns the bytes of a PEM-encoded certificate authority,
// which may contain multiple teamserver users and their master.
func (ts *Server) GetUsersCA() ([]byte, []byte, error) {
	return ts.certs.GetUsersCAPEM()
}

// SaveUsersCA accepts the public and private parts of a Certificate
// Authority containing one or more users to add to the teamserver.
func (ts *Server) SaveUsersCA(cert, key []byte) {
	ts.certs.SaveUsersCA(cert, key)
}

// newUserToken - Generate a new user authentication token.
func (ts *Server) newUserToken() (string, error) {
	buf := make([]byte, 32)
	n, err := rand.Read(buf)
	if err != nil || n != len(buf) {
		return "", fmt.Errorf("failed to read from secure rand: %w", err)
	} else if n != len(buf) {
		return "", errors.New("failed to read from secure rand")
	}
	return hex.EncodeToString(buf), nil
}

// userByToken - Select a teamserver user by token value
func (ts *Server) userByToken(value string) (*db.User, error) {
	if len(value) < 1 {
		return nil, db.ErrRecordNotFound
	}
	user := &db.User{}
	err := ts.db.Where(&db.User{
		Token: value,
	}).First(user).Error
	return user, err
}

// getUserTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS parameters, we choose sensible defaults instead.
func (ts *Server) GetUserTLSConfig() (*tls.Config, error) {
	log := ts.NamedLogger("certs", "mtls")
	caCertPtr, _, err := ts.certs.GetUsersCA()
	if err != nil {
		return nil, ts.errorWith(log, "%w: failed to get users certificate authority: %w", ErrCertificate, err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	_, _, err = ts.certs.UserServerGetCertificate()
	if err == certs.ErrCertDoesNotExist {
		ts.certs.UserServerGenerateCertificate()
	}

	certPEM, keyPEM, err := ts.certs.UserServerGetCertificate()
	if err != nil {
		return nil, ts.errorWith(log, "%w: failed to generated or fetch user certificate: %w", ErrCertificate, err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, ts.errorWith(log, "%w: failed to load server certificate: %w", ErrCertificate, err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	if keyLogger := ts.certs.NewKeyLogger(); keyLogger != nil {
		tlsConfig.KeyLogWriter = ts.certs.NewKeyLogger()
	}

	return tlsConfig, nil
}
