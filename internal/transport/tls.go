package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
)

const (
	// Should be 31415, but... go to hell with your endless limits.
	DefaultPort = 31416
)

func GetTLSConfig(caCertificate string, certificate string, privateKey string) (*tls.Config, error) {
	certPEM, err := tls.X509KeyPair([]byte(certificate), []byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("Cannot parse client certificate: %v", err)
	}

	// Load CA cert
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCertificate))

	// Setup config with custom certificate validation routine
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{certPEM},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Don't worry I sorta know what I'm doing
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return RootOnlyVerifyCertificate(caCertificate, rawCerts)
		},
	}
	return tlsConfig, nil
}

// RootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func RootOnlyVerifyCertificate(caCertificate string, rawCerts [][]byte) error {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertificate))
	if !ok {
		fmt.Errorf("Failed to parse root certificate")
	}

	cert, err := x509.ParseCertificate(rawCerts[0]) // We should only get one cert
	if err != nil {
		log.Printf("Failed to parse certificate: " + err.Error())
		return err
	}

	// Basically we only care if the certificate was signed by our authority
	// Go selects sensible defaults for time and EKU, basically we're only
	// skipping the hostname check, I think?
	options := x509.VerifyOptions{
		Roots: roots,
	}
	if options.Roots == nil {
		return fmt.Errorf("Certificate root is nil")
	}
	if _, err := cert.Verify(options); err != nil {
		return fmt.Errorf("Failed to verify certificate: " + err.Error())
	}

	return nil
}
