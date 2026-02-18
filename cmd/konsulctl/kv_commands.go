package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// KVCommands handles all KV store related commands
type KVCommands struct {
	cli *CLI
}

// ClientConfig holds client configuration for watch connections
type ClientConfig struct {
	Address string
	TLS     *TLSConfig
	Token   string
}

// NewKVCommands creates a new KV commands handler
func NewKVCommands(cli *CLI) *KVCommands {
	return &KVCommands{cli: cli}
}

// Handle routes KV subcommands
func (k *KVCommands) Handle(args []string) {
	if len(args) == 0 {
		k.cli.Errorln("KV subcommand required")
		k.cli.Errorln("Usage: konsulctl kv <get|set|delete|list> [options]")
		k.cli.Exit(1)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "get":
		k.Get(subArgs)
	case "set":
		k.Set(subArgs)
	case "delete":
		k.Delete(subArgs)
	case "list":
		k.List(subArgs)
	case "watch":
		k.Watch(subArgs)
	default:
		k.cli.Errorf("Unknown KV subcommand: %s\n", subcommand)
		k.cli.Errorln("Available: get, set, delete, list, watch")
		k.cli.Exit(1)
	}
}

// Get retrieves a value by key
func (k *KVCommands) Get(args []string) {
	config, remaining, err := k.cli.ParseGlobalFlags(args, "get")
	if err == flag.ErrHelp {
		k.cli.Println("Usage: konsulctl kv get <key> [options]")
		return
	}
	k.cli.HandleError(err, "parsing flags")
	k.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl kv get <key>")

	key := remaining[0]
	client := k.cli.CreateClient(config)

	value, err := client.GetKV(key)
	k.cli.HandleError(err, "getting key '"+key+"'")

	k.cli.Printf("%s\n", value)
}

// Set sets a key-value pair
func (k *KVCommands) Set(args []string) {
	config, remaining, err := k.cli.ParseGlobalFlags(args, "set")
	if err == flag.ErrHelp {
		k.cli.Println("Usage: konsulctl kv set <key> <value> [options]")
		return
	}
	k.cli.HandleError(err, "parsing flags")
	k.cli.ValidateExactArgs(remaining, 2, "Usage: konsulctl kv set <key> <value>")

	key := remaining[0]
	value := remaining[1]
	client := k.cli.CreateClient(config)

	err = client.SetKV(key, value)
	k.cli.HandleError(err, "setting key '"+key+"'")

	k.cli.Printf("Successfully set %s = %s\n", key, value)
}

// Delete deletes a key
func (k *KVCommands) Delete(args []string) {
	config, remaining, err := k.cli.ParseGlobalFlags(args, "delete")
	if err == flag.ErrHelp {
		k.cli.Println("Usage: konsulctl kv delete <key> [options]")
		return
	}
	k.cli.HandleError(err, "parsing flags")
	k.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl kv delete <key>")

	key := remaining[0]
	client := k.cli.CreateClient(config)

	err = client.DeleteKV(key)
	k.cli.HandleError(err, "deleting key '"+key+"'")

	k.cli.Printf("Successfully deleted key: %s\n", key)
}

// List lists all keys
func (k *KVCommands) List(args []string) {
	config, remaining, err := k.cli.ParseGlobalFlags(args, "list")
	if err == flag.ErrHelp {
		k.cli.Println("Usage: konsulctl kv list [options]")
		return
	}
	k.cli.HandleError(err, "parsing flags")
	k.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl kv list")

	client := k.cli.CreateClient(config)

	keys, err := client.ListKV()
	k.cli.HandleError(err, "listing keys")

	if len(keys) == 0 {
		k.cli.Println("No keys found")
		return
	}

	k.cli.Println("Keys:")
	for _, key := range keys {
		k.cli.Printf("  %s\n", key)
	}
}

// Watch watches for changes to a key or key pattern
func (k *KVCommands) Watch(args []string) {
	// Check for help first
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		k.cli.Println("Usage: konsulctl kv watch <key|pattern> [options]")
		k.cli.Println()
		k.cli.Println("Watch for changes to a key or key pattern.")
		k.cli.Println()
		k.cli.Println("Examples:")
		k.cli.Println("  konsulctl kv watch app/config")
		k.cli.Println("  konsulctl kv watch 'app/*'")
		k.cli.Println("  konsulctl kv watch 'app/**'")
		k.cli.Println()
		k.cli.Println("Patterns:")
		k.cli.Println("  Exact match:  app/config")
		k.cli.Println("  Single level: app/* (matches app/config, app/data)")
		k.cli.Println("  Multi level:  app/** (matches all keys under app/)")
		k.cli.Println()
		k.cli.Println("Options:")
		k.cli.Println("  --transport <type>  Transport type: websocket (default) or sse")
		k.cli.Println("  --server <url>       Konsul server URL (default: http://localhost:8888)")
		k.cli.Println("  --tls-skip-verify    Skip TLS certificate verification")
		k.cli.Println("  --ca-cert <file>     Path to CA certificate file")
		k.cli.Println("  --client-cert <file> Path to client certificate file")
		k.cli.Println("  --client-key <file>  Path to client key file")
		k.cli.Println("  --token <token>      JWT token for authentication (or set KONSUL_TOKEN)")
		return
	}

	// Custom flag set with transport option
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(k.cli.Error)
	transport := fs.String("transport", "websocket", "Transport type (websocket or sse)")

	// Global flags
	serverURL := fs.String("server", "http://localhost:8888", "Konsul server URL")
	tlsSkipVerify := fs.Bool("tls-skip-verify", false, "Skip TLS certificate verification")
	tlsCACert := fs.String("ca-cert", "", "Path to CA certificate file")
	tlsClientCert := fs.String("client-cert", "", "Path to client certificate file")
	tlsClientKey := fs.String("client-key", "", "Path to client key file")
	tokenFlag := fs.String("token", "", "JWT token for authentication (or set KONSUL_TOKEN)")

	err := fs.Parse(args)
	k.cli.HandleError(err, "parsing flags")

	remaining := fs.Args()
	k.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl kv watch <key|pattern>")

	pattern := remaining[0]

	// Build client config
	config := &ClientConfig{
		Address: *serverURL,
		TLS: &TLSConfig{
			Enabled:        strings.HasPrefix(*serverURL, "https://"),
			SkipVerify:     *tlsSkipVerify,
			CACertFile:     *tlsCACert,
			ClientCertFile: *tlsClientCert,
			ClientKeyFile:  *tlsClientKey,
		},
	}
	config.Token = strings.TrimSpace(*tokenFlag)
	if config.Token == "" {
		config.Token = strings.TrimSpace(os.Getenv("KONSUL_TOKEN"))
	}

	switch *transport {
	case "websocket":
		k.watchWebSocket(config, pattern)
	case "sse":
		k.watchSSE(config, pattern)
	default:
		k.cli.Errorf("Unknown transport: %s (must be websocket or sse)\n", *transport)
		k.cli.Exit(1)
	}
}

// WatchEvent represents a watch event from the server
type WatchEvent struct {
	Type      string `json:"type"`
	Key       string `json:"key"`
	Value     string `json:"value,omitempty"`
	OldValue  string `json:"old_value,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// watchWebSocket watches for key changes using WebSocket
func (k *KVCommands) watchWebSocket(config *ClientConfig, pattern string) {
	// Build WebSocket URL
	wsURL, err := k.buildWebSocketURL(config, pattern)
	k.cli.HandleError(err, "building WebSocket URL")

	k.cli.Printf("Watching %s (WebSocket)...\n", pattern)
	k.cli.Println("Press Ctrl+C to stop")
	k.cli.Println()

	// Setup signal handler for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		k.cli.Println("\nStopping watch...")
		cancel()
	}()

	// Connect to WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Add TLS configuration if needed
	if config.TLS != nil && config.TLS.Enabled {
		tlsClientConfig, err := k.buildTLSClientConfig(config.TLS)
		k.cli.HandleError(err, "configuring TLS")
		dialer.TLSClientConfig = tlsClientConfig
	}

	// Add authorization header
	headers := http.Header{}
	if config.Token != "" {
		headers.Add("Authorization", "Bearer "+config.Token)
	}

	conn, _, err := dialer.Dial(wsURL, headers)
	k.cli.HandleError(err, "connecting to WebSocket")
	defer func() {
		if err := conn.Close(); err != nil {
			k.cli.Errorf("Error closing WebSocket: %v\n", err)
		}
	}()

	// Read events
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var event WatchEvent
			err := conn.ReadJSON(&event)
			if err != nil {
				if ctx.Err() != nil {
					return // Context cancelled, exit gracefully
				}
				k.cli.Errorf("Error reading event: %v\n", err)
				return
			}

			k.printWatchEvent(&event)
		}
	}
}

// watchSSE watches for key changes using Server-Sent Events
func (k *KVCommands) watchSSE(config *ClientConfig, pattern string) {
	httpURL, err := k.buildHTTPWatchURL(config, pattern)
	k.cli.HandleError(err, "building SSE URL")

	k.cli.Printf("Watching %s (SSE)...\n", pattern)
	k.cli.Println("Press Ctrl+C to stop")
	k.cli.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		k.cli.Println("\nStopping watch...")
		cancel()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpURL, nil)
	k.cli.HandleError(err, "creating SSE request")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+config.Token)
	}

	client := k.newWatchHTTPClient(config)
	resp, err := client.Do(req)
	k.cli.HandleError(err, "connecting to SSE endpoint")
	defer func() {
		if err := resp.Body.Close(); err != nil {
			k.cli.Errorf("Error closing SSE response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			k.cli.ExitError("SSE request failed (%s): %s\n", resp.Status, msg)
		}
		k.cli.ExitError("SSE request failed (%s)\n", resp.Status)
		return
	}

	reader := bufio.NewReader(resp.Body)
	var (
		eventName string
		dataBuf   strings.Builder
	)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return
			}
			if errors.Is(err, io.EOF) {
				k.cli.Println("\nConnection closed by server")
				return
			}
			k.cli.Errorf("Error reading SSE stream: %v\n", err)
			return
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			payload := strings.TrimSuffix(dataBuf.String(), "\n")
			k.handleSSEEvent(eventName, payload)
			eventName = ""
			dataBuf.Reset()
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(line[len("event:"):])
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimSpace(line[len("data:"):])
			dataBuf.WriteString(data)
			dataBuf.WriteByte('\n')
		default:
			// Ignore other SSE fields (id, retry, etc)
		}
	}
}

// buildWebSocketURL builds the WebSocket URL for watching
func (k *KVCommands) buildWebSocketURL(config *ClientConfig, pattern string) (string, error) {
	u, err := url.Parse(config.Address)
	if err != nil {
		return "", fmt.Errorf("invalid server URL: %w", err)
	}

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	u.Scheme = scheme
	u.Path = buildWatchPath(u.Path, pattern)
	u.RawQuery = ""
	u.Fragment = ""

	return u.String(), nil
}

func (k *KVCommands) buildHTTPWatchURL(config *ClientConfig, pattern string) (string, error) {
	u, err := url.Parse(config.Address)
	if err != nil {
		return "", fmt.Errorf("invalid server URL: %w", err)
	}

	u.Path = buildWatchPath(u.Path, pattern)
	u.RawQuery = ""
	u.Fragment = ""

	return u.String(), nil
}

func buildWatchPath(basePath, pattern string) string {
	base := strings.TrimSuffix(basePath, "/")
	if base == "" {
		return fmt.Sprintf("/kv/watch/%s", url.PathEscape(pattern))
	}

	return fmt.Sprintf("%s/kv/watch/%s", base, url.PathEscape(pattern))
}

func (k *KVCommands) newWatchHTTPClient(config *ClientConfig) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if config.TLS != nil && config.TLS.Enabled {
		tlsClientConfig, err := k.buildTLSClientConfig(config.TLS)
		k.cli.HandleError(err, "configuring TLS")
		if tlsClientConfig != nil {
			transport.TLSClientConfig = tlsClientConfig
		}
	}

	return &http.Client{
		Timeout:   0,
		Transport: transport,
	}
}

func (k *KVCommands) buildTLSClientConfig(tlsConfig *TLSConfig) (*tls.Config, error) {
	if tlsConfig == nil || !tlsConfig.Enabled {
		return nil, nil
	}

	clientConfig := &tls.Config{
		InsecureSkipVerify: tlsConfig.SkipVerify,
	}

	if tlsConfig.CACertFile != "" {
		caCert, err := os.ReadFile(tlsConfig.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("reading CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("parsing CA certificate: invalid PEM data")
		}
		clientConfig.RootCAs = caCertPool
	}

	if tlsConfig.ClientCertFile != "" || tlsConfig.ClientKeyFile != "" {
		if tlsConfig.ClientCertFile == "" || tlsConfig.ClientKeyFile == "" {
			return nil, fmt.Errorf("both client-cert and client-key must be provided")
		}

		cert, err := tls.LoadX509KeyPair(tlsConfig.ClientCertFile, tlsConfig.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("loading client certificate: %w", err)
		}
		clientConfig.Certificates = []tls.Certificate{cert}
	}

	return clientConfig, nil
}

func (k *KVCommands) handleSSEEvent(eventName, payload string) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return
	}

	if eventName != "" && eventName != "kv-change" {
		k.cli.Printf("Received %s event: %s\n", eventName, payload)
		return
	}

	var event WatchEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		k.cli.Errorf("Failed to parse watch event: %v\n", err)
		return
	}

	k.printWatchEvent(&event)
}

// printWatchEvent prints a watch event in a readable format
func (k *KVCommands) printWatchEvent(event *WatchEvent) {
	timestamp := time.Unix(event.Timestamp, 0).Format("2006-01-02 15:04:05")

	switch event.Type {
	case "set":
		if event.OldValue != "" {
			k.cli.Printf("[%s] UPDATE %s: %s -> %s\n", timestamp, event.Key, event.OldValue, event.Value)
		} else {
			k.cli.Printf("[%s] CREATE %s: %s\n", timestamp, event.Key, event.Value)
		}
	case "delete":
		k.cli.Printf("[%s] DELETE %s (was: %s)\n", timestamp, event.Key, event.OldValue)
	default:
		k.cli.Printf("[%s] UNKNOWN event type: %s for key %s\n", timestamp, event.Type, event.Key)
	}
}
