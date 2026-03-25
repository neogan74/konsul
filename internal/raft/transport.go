package raft

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

// NewTransport creates a new Raft network transport.
// If TLS is enabled in config, it uses secure communication.
func NewTransport(cfg *Config, logger hclog.Logger) (*raft.NetworkTransport, error) {
	// Resolve advertise address
	advertiseAddrStr := cfg.GetAdvertiseAddr()
	addr, err := net.ResolveTCPAddr("tcp", advertiseAddrStr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve advertise address: %w", err)
	}

	if !cfg.TLS.Enabled {
		return raft.NewTCPTransport(cfg.BindAddr, addr, 3, 10*time.Second, os.Stderr)
	}

	// TLS Enabled
	return newTLSTransport(cfg, addr, logger)
}

func newTLSTransport(cfg *Config, advertiseAddr net.Addr, logger hclog.Logger) (*raft.NetworkTransport, error) {
	// Load certificates
	cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		ServerName:   cfg.TLS.ServerName,
	}

	// Configure mTLS if verifying peers
	if cfg.TLS.VerifyPeer {
		caCert, err := os.ReadFile(cfg.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.ClientCAs = caPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.RootCAs = caPool
	}

	logger.Info("enabling Raft TLS transport",
		"verify_peer", cfg.TLS.VerifyPeer,
		"server_name", cfg.TLS.ServerName,
	)

	// Create a TCP listener
	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", cfg.BindAddr, err)
	}

	// Wrap listener with TLS
	tlsListener := tls.NewListener(listener, tlsConfig)

	// Create a StreamLayer that uses TLS
	streamLayer := &tlsStreamLayer{
		Listener:  tlsListener,
		tlsConfig: tlsConfig,
		advertise: advertiseAddr,
	}

	// Create NetworkTransport
	transConfig := &raft.NetworkTransportConfig{
		Stream:  streamLayer,
		MaxPool: 3,
		Timeout: 10 * time.Second,
		Logger:  logger,
	}

	return raft.NewNetworkTransportWithConfig(transConfig), nil
}

// tlsStreamLayer implements raft.StreamLayer interface
type tlsStreamLayer struct {
	net.Listener
	tlsConfig *tls.Config
	advertise net.Addr
}

// Dial implements raft.StreamLayer.Dial
func (t *tlsStreamLayer) Dial(addr raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	return tls.DialWithDialer(dialer, "tcp", string(addr), t.tlsConfig)
}

// Addr implements raft.StreamLayer.Addr
// Note: This must return the Advertise address, not the Bind address
func (t *tlsStreamLayer) Addr() net.Addr {
	return t.advertise
}
