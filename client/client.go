package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/reeflective/team/internal/proto"
)

const (
	// configsDirName - Directory name containing config files
	configsDirName  = "configs"
	versionFileName = "version"
	logFileName     = "client.log"
)

// Client is a team client wrapper.
// It offers the core functionality of any team client.
type Client struct {
	name    string
	opts    *opts
	conn    *grpc.ClientConn
	rpc     proto.TeamClient
	log     *slog.Logger
	logFile *os.File
}

// New returns an application client ready to work.
// The application client log file is opened and served to the client builtin logger.
// The client will panic if it can't open or create this log file as ~/.app/client.log
func New(application string, options ...Options) *Client {
	c := &Client{
		opts: &opts{},
		name: application,
	}

	c.logFile = c.initLogging(c.AppDir())

	c.apply(options...)

	return c
}

// Setup - Extract or create local assets
func (c *Client) Setup(force bool, echo bool) {
	appDir := c.AppDir()
	localVer := c.assetVersion()
	if force || localVer == "" || localVer != GitCommit {
		c.saveAssetVersion(appDir)
	}
	// if _, err := os.Stat(filepath.Join(appDir, settingsFileName)); os.IsNotExist(err) {
	// 	SaveSettings(nil)
	// }
}

// Connect establishes the required RPC connection and returns once the latter is established,
// potentially returning a failure which may or may not be critical, and therefore checked.
func (c *Client) Connect() (err error) {
	defer func() {
		c.rpc = proto.NewTeamClient(c.conn)
	}()

	// Our connection is already existing and configured.
	if c.conn != nil {
		return
	}

	// Else connect with any available configuration.
	c.conn, err = c.ConnectDefault()

	return
}

// ConnectDefault uses the default client configurations to connect to the team server.
// Note that this call might be blocking and expect user input, in the case where more
// than one server configuration is found in the application directory: the application
// will prompt the user to choose one of them.
func (c *Client) ConnectDefault() (*grpc.ClientConn, error) {
	configs := c.GetConfigs()
	if len(configs) == 0 {
		return nil, fmt.Errorf("no config files found at %s", c.ConfigsDir())
	}
	cfg := c.SelectConfig()
	if cfg == nil {
		return nil, errors.New("no application was selected or parsed")
	}

	return c.ConnectWith(cfg)
}

// ConnectWith establishes a working gRPC client connection to the server specified in the configuration.
func (c *Client) ConnectWith(config *Config) (*grpc.ClientConn, error) {
	tlsConfig, err := getTLSConfig(config.CACertificate, config.Certificate, config.PrivateKey)
	if err != nil {
		return nil, err
	}
	transportCreds := credentials.NewTLS(tlsConfig)
	callCreds := credentials.PerRPCCredentials(tokenAuth{token: config.Token})
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithPerRPCCredentials(callCreds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	connection, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", config.Host, config.Port), options...)
	if err != nil {
		return nil, err
	}

	// Register the core RPC methods
	c.conn = connection
	c.rpc = proto.NewTeamClient(c.conn)

	return connection, nil
}

// Conn returns the gRPC client connection it uses.
func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

// ServerVersion returns the version information of the server to which
// the client is connected, or nil and an error if it could not retrieve it.
func (c *Client) ServerVersion() (ver *proto.Version, err error) {
	if c.rpc == nil {
		return nil, errors.New("no working client RPC is attached to this client")
	}

	res, err := c.rpc.GetVersion(context.Background(), &proto.Empty{})
	if err != nil {
		return nil, errors.New(status.Convert(err).Message())
	}

	return res, nil
}

// Users returns a list of all users registered to the application server.
func (c *Client) Users() (users []proto.User) {
	if c.rpc == nil {
		return nil
	}

	res, err := c.rpc.GetUsers(context.Background(), &proto.Empty{})
	if err != nil {
		return nil
	}

	for _, user := range res.GetUsers() {
		users = append(users, *user)
	}

	return
}

// Disconnect disconnects the client from the server,
// closing the connection and the client log file.
// Any errors are logged to the log file, not returned.
func (c *Client) Disconnect() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.log.Error(fmt.Sprintf("error closing connection: %v", err))
		}
	}

	if c.logFile != nil {
		c.logFile.Close()
	}
}

// Name returns the name of the client application.
func (c *Client) Name() string {
	return c.name
}

func (c *Client) assetVersion() string {
	appDir := c.AppDir()
	data, err := os.ReadFile(filepath.Join(appDir, versionFileName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (c *Client) saveAssetVersion(appDir string) {
	versionFilePath := filepath.Join(appDir, versionFileName)
	fVer, _ := os.Create(versionFilePath)
	defer fVer.Close()
	fVer.Write([]byte(GitCommit))
}
