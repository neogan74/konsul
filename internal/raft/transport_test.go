package raft

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to generate certs. passing parentCert/Key allows signing.
func generateCert(t *testing.T, dir string, name string, isCA bool, parentCert *x509.Certificate, parentKey *rsa.PrivateKey) (string, string, *x509.Certificate, *rsa.PrivateKey) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: name,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  isCA,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	if isCA {
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	var parent *x509.Certificate
	var signerKey *rsa.PrivateKey

	if parentCert != nil {
		parent = parentCert
		signerKey = parentKey
	} else {
		parent = &template
		signerKey = priv
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, parent, &priv.PublicKey, signerKey)
	require.NoError(t, err)

	certPath := filepath.Join(dir, name+".crt")
	certOut, err := os.Create(certPath)
	require.NoError(t, err)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyPath := filepath.Join(dir, name+".key")
	keyOut, err := os.Create(keyPath)
	require.NoError(t, err)
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	keyOut.Close()

	// Parse the generated cert to return it
	cert, err := x509.ParseCertificate(derBytes)
	require.NoError(t, err)

	return certPath, keyPath, cert, priv
}

func TestTransport_TCP(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BindAddr = "127.0.0.1:0"
	cfg.TLS.Enabled = false

	logger := hclog.NewNullLogger()

	trans, err := NewTransport(cfg, logger)
	require.NoError(t, err)
	defer trans.Close()

	assert.NotNil(t, trans)
}

func TestTransport_TLS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _, _ := generateCert(t, dir, "server", false, nil, nil)

	cfg := DefaultConfig()
	cfg.BindAddr = "127.0.0.1:0"
	cfg.TLS.Enabled = true
	cfg.TLS.CertFile = certFile
	cfg.TLS.KeyFile = keyFile

	logger := hclog.NewNullLogger()

	trans, err := NewTransport(cfg, logger)
	require.NoError(t, err)
	defer trans.Close()

	assert.NotNil(t, trans)
}

func TestTransport_mTLS(t *testing.T) {
	dir := t.TempDir()
	// Generate CA
	caCert, _, caObj, caKey := generateCert(t, dir, "ca", true, nil, nil)
	// Generate Server Cert (signed by CA - simplified here using self-signed for test speed,
	// ideally should sign with CA but for basic config loading test self-signed is fine if we don't verify trust chain in creation)
	// Wait, NewTransport loads CA file but doesn't verify the server cert against it during creation.
	// It uses CA to verify PEERS.

	// Generate Server Cert (signed by CA)
	certFile, keyFile, _, _ := generateCert(t, dir, "server", false, caObj, caKey)

	cfg := DefaultConfig()
	cfg.BindAddr = "127.0.0.1:0"
	cfg.TLS.Enabled = true
	cfg.TLS.CertFile = certFile
	cfg.TLS.KeyFile = keyFile
	cfg.TLS.VerifyPeer = true
	cfg.TLS.CAFile = caCert

	logger := hclog.NewNullLogger()

	trans, err := NewTransport(cfg, logger)
	require.NoError(t, err)
	defer trans.Close()

	assert.NotNil(t, trans)
}
