package certs

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
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log/slog"
	"testing"

	"gorm.io/gorm"

	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/internal/db"
)

// newTestManager builds a certificate manager backed by an in-memory filesystem
// and an in-memory SQLite database. Constructing the manager also generates the
// user certificate authority, so the returned manager is ready to sign certs.
func newTestManager(t *testing.T) *Manager {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	dbConfig := &db.Config{
		Dialect:      db.Sqlite,
		Database:     db.SQLiteInMemoryHost,
		MaxIdleConns: 1,
		MaxOpenConns: 1,
		LogLevel:     "error",
	}

	database, err := db.NewClient(dbConfig, logger)
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}

	fs := assets.NewFileSystem(true)

	return NewManager(fs, database, logger, "testapp", "/app")
}

// TestNewManagerInitializesCA verifies that constructing a manager creates a
// usable user certificate authority: it is retrievable both parsed and as PEM,
// and the CA key files were written to the (in-memory) filesystem.
func TestNewManagerInitializesCA(t *testing.T) {
	certs := newTestManager(t)

	caCert, caKey, err := certs.GetUsersCA()
	if err != nil {
		t.Fatalf("GetUsersCA: %v", err)
	}
	if caCert == nil || caKey == nil {
		t.Fatal("GetUsersCA returned nil certificate or key")
	}
	if !caCert.IsCA {
		t.Fatal("user CA certificate is not marked as a CA")
	}

	certPEM, keyPEM, err := certs.GetUsersCAPEM()
	if err != nil {
		t.Fatalf("GetUsersCAPEM: %v", err)
	}
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		t.Fatal("GetUsersCAPEM returned empty certificate or key")
	}
	if block, _ := pem.Decode(certPEM); block == nil || block.Type != "CERTIFICATE" {
		t.Fatal("CA certificate PEM is not a valid CERTIFICATE block")
	}
}

// TestECCCertificateRoundTrip exercises the full lifecycle of an ECC leaf
// certificate: generate + save, fetch it back byte-for-byte, then remove it and
// confirm it is gone.
func TestECCCertificateRoundTrip(t *testing.T) {
	certs := newTestManager(t)

	cn := "roundtrip.example.com"
	cert, key := certs.GenerateECCCertificate(userCA, cn, false, false)
	if len(cert) == 0 || len(key) == 0 {
		t.Fatal("GenerateECCCertificate returned empty material")
	}

	if err := certs.saveCertificate(userCA, ECCKey, cn, cert, key); err != nil {
		t.Fatalf("saveCertificate: %v", err)
	}

	gotCert, gotKey, err := certs.GetECCCertificate(userCA, cn)
	if err != nil {
		t.Fatalf("GetECCCertificate: %v", err)
	}
	if !bytes.Equal(cert, gotCert) || !bytes.Equal(key, gotKey) {
		t.Fatal("fetched certificate/key does not match the stored material")
	}

	if err := certs.RemoveCertificate(userCA, ECCKey, cn); err != nil {
		t.Fatalf("RemoveCertificate: %v", err)
	}

	if _, _, err := certs.GetECCCertificate(userCA, cn); err != ErrCertDoesNotExist {
		t.Fatalf("expected ErrCertDoesNotExist after removal, got %v", err)
	}
}

// TestRSACertificateRoundTrip does the same lifecycle check for RSA material,
// covering the RSA key generation and PEM-encoding branches.
func TestRSACertificateRoundTrip(t *testing.T) {
	certs := newTestManager(t)

	cn := "rsa.example.com"
	cert, key := certs.GenerateRSACertificate(userCA, cn, false, false)
	if len(cert) == 0 || len(key) == 0 {
		t.Fatal("GenerateRSACertificate returned empty material")
	}

	if block, _ := pem.Decode(key); block == nil || block.Type != "RSA PRIVATE KEY" {
		t.Fatal("RSA private key PEM block is malformed")
	}

	if err := certs.saveCertificate(userCA, RSAKey, cn, cert, key); err != nil {
		t.Fatalf("saveCertificate: %v", err)
	}

	gotCert, gotKey, err := certs.GetRSACertificate(userCA, cn)
	if err != nil {
		t.Fatalf("GetRSACertificate: %v", err)
	}
	if !bytes.Equal(cert, gotCert) || !bytes.Equal(key, gotKey) {
		t.Fatal("fetched RSA certificate/key does not match stored material")
	}
}

// TestGetCertificateNotFound confirms that fetching an unknown certificate
// returns the sentinel error and no material.
func TestGetCertificateNotFound(t *testing.T) {
	certs := newTestManager(t)

	cert, key, err := certs.GetECCCertificate(userCA, "nobody.example.com")
	if err != ErrCertDoesNotExist {
		t.Fatalf("expected ErrCertDoesNotExist, got %v", err)
	}
	if cert != nil || key != nil {
		t.Fatal("expected nil material for a missing certificate")
	}
}

// TestInvalidKeyTypeRejected ensures the key-type guard rejects unknown key
// namespaces on every entry point that takes one.
func TestInvalidKeyTypeRejected(t *testing.T) {
	certs := newTestManager(t)

	if _, _, err := certs.GetCertificate(userCA, "dsa", "x"); err == nil {
		t.Fatal("GetCertificate accepted an invalid key type")
	}
	if err := certs.RemoveCertificate(userCA, "dsa", "x"); err == nil {
		t.Fatal("RemoveCertificate accepted an invalid key type")
	}
	if err := certs.saveCertificate(userCA, "dsa", "x", nil, nil); err == nil {
		t.Fatal("saveCertificate accepted an invalid key type")
	}
}

// TestUserClientCertificateLifecycle covers the user-facing helpers used when a
// teamserver mints, lists and revokes a client's credentials.
func TestUserClientCertificateLifecycle(t *testing.T) {
	certs := newTestManager(t)

	if _, _, err := certs.UserClientGenerateCertificate("alice"); err != nil {
		t.Fatalf("UserClientGenerateCertificate: %v", err)
	}

	cert, key, err := certs.UserClientGetCertificate("alice")
	if err != nil {
		t.Fatalf("UserClientGetCertificate: %v", err)
	}
	if len(cert) == 0 || len(key) == 0 {
		t.Fatal("client certificate material is empty")
	}

	listed := certs.UserClientListCertificates()
	if len(listed) != 1 {
		t.Fatalf("expected exactly 1 listed client certificate, got %d", len(listed))
	}
	if listed[0].Subject.CommonName != "alice" {
		t.Fatalf("listed certificate CN = %q, want alice", listed[0].Subject.CommonName)
	}

	if err := certs.UserClientRemoveCertificate("alice"); err != nil {
		t.Fatalf("UserClientRemoveCertificate: %v", err)
	}
	if _, _, err := certs.UserClientGetCertificate("alice"); err != ErrCertDoesNotExist {
		t.Fatalf("expected ErrCertDoesNotExist after revocation, got %v", err)
	}
	if got := certs.UserClientListCertificates(); len(got) != 0 {
		t.Fatalf("expected no client certificates after revocation, got %d", len(got))
	}
}

// TestUserServerCertificate covers the lazy generate-then-fetch pattern used by
// UsersTLSConfig for the server-side leaf certificate.
func TestUserServerCertificate(t *testing.T) {
	certs := newTestManager(t)

	// Not generated yet.
	if _, _, err := certs.UserServerGetCertificate(); err != ErrCertDoesNotExist {
		t.Fatalf("expected ErrCertDoesNotExist before generation, got %v", err)
	}

	if _, _, err := certs.UserServerGenerateCertificate(); err != nil {
		t.Fatalf("UserServerGenerateCertificate: %v", err)
	}

	cert, key, err := certs.UserServerGetCertificate()
	if err != nil {
		t.Fatalf("UserServerGetCertificate: %v", err)
	}
	if len(cert) == 0 || len(key) == 0 {
		t.Fatal("server certificate material is empty")
	}
}

// TestClientCertificateChainsToCA is the core PKI assertion: a generated client
// certificate must actually verify against the user CA. This proves the signing
// chain (not just that bytes round-trip through the database).
func TestClientCertificateChainsToCA(t *testing.T) {
	certs := newTestManager(t)

	caCert, _, err := certs.GetUsersCA()
	if err != nil {
		t.Fatalf("GetUsersCA: %v", err)
	}

	leafPEM, _, err := certs.UserClientGenerateCertificate("bob")
	if err != nil {
		t.Fatalf("UserClientGenerateCertificate: %v", err)
	}

	block, _ := pem.Decode(leafPEM)
	if block == nil {
		t.Fatal("failed to decode leaf certificate PEM")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate(leaf): %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	if _, err := leaf.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); err != nil {
		t.Fatalf("client certificate does not chain to the user CA: %v", err)
	}
}

// TestRootOnlyVerifyCertificate revives the (previously commented-out) contract
// for the hostname-skipping verifier: a certificate signed by the CA passes,
// one signed by a different CA is rejected.
func TestRootOnlyVerifyCertificate(t *testing.T) {
	certs := newTestManager(t)

	caPEM, _, err := certs.GetUsersCAPEM()
	if err != nil {
		t.Fatalf("GetUsersCAPEM: %v", err)
	}

	// RootOnlyVerifyCertificate skips only the hostname check; it still enforces
	// Go's default EKU (server-auth), so we verify with server certificates.
	//
	// A leaf signed by our CA must verify.
	leafPEM, _ := certs.GenerateECCCertificate(userCA, "localhost", false, false)
	leafBlock, _ := pem.Decode(leafPEM)
	if leafBlock == nil {
		t.Fatal("failed to decode leaf certificate PEM")
	}
	if err := RootOnlyVerifyCertificate(string(caPEM), [][]byte{leafBlock.Bytes}); err != nil {
		t.Fatalf("RootOnlyVerifyCertificate rejected a validly-signed cert: %v", err)
	}

	// A leaf signed by a DIFFERENT CA must be rejected.
	other := newTestManagerWithApp(t, "otherapp", "/other")
	foreignPEM, _ := other.GenerateECCCertificate(userCA, "localhost", false, false)
	foreignBlock, _ := pem.Decode(foreignPEM)
	if foreignBlock == nil {
		t.Fatal("failed to decode foreign certificate PEM")
	}
	if err := RootOnlyVerifyCertificate(string(caPEM), [][]byte{foreignBlock.Bytes}); err == nil {
		t.Fatal("RootOnlyVerifyCertificate accepted a certificate signed by a foreign CA")
	}

	// A malformed CA PEM must be rejected outright (regression guard: the error
	// used to be constructed but never returned).
	if err := RootOnlyVerifyCertificate("not a pem", [][]byte{leafBlock.Bytes}); err == nil {
		t.Fatal("RootOnlyVerifyCertificate accepted a malformed root certificate")
	}
}

// newTestManagerWithApp is like newTestManager but lets a test create a second,
// independent CA (distinct app name + filesystem root + database).
func newTestManagerWithApp(t *testing.T, appName, appDir string) *Manager {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	dbConfig := &db.Config{
		Dialect:      db.Sqlite,
		Database:     db.SQLiteInMemoryHost,
		MaxIdleConns: 1,
		MaxOpenConns: 1,
		LogLevel:     "error",
	}

	var database *gorm.DB
	database, err := db.NewClient(dbConfig, logger)
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}

	return NewManager(assets.NewFileSystem(true), database, logger, appName, appDir)
}
