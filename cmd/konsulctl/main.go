package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "kv":
		handleKVCommand(args)
	case "service":
		handleServiceCommand(args)
	case "backup":
		handleBackupCommand(args)
	case "dns":
		handleDNSCommand(args)
	case "ratelimit":
		handleRateLimitCommand(args)
	case "version":
		fmt.Printf("konsulctl version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("konsulctl - Konsul CLI Tool")
	fmt.Println()
	fmt.Println("Usage: konsulctl <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  kv <subcommand>    Key-value operations")
	fmt.Println("    get <key>        Get value by key")
	fmt.Println("    set <key> <value>  Set key-value pair")
	fmt.Println("    delete <key>     Delete key")
	fmt.Println("    list             List all keys")
	fmt.Println()
	fmt.Println("  service <subcommand>  Service operations")
	fmt.Println("    register <name> <address> <port>  Register service")
	fmt.Println("    list             List all services")
	fmt.Println("    deregister <name>  Deregister service")
	fmt.Println("    heartbeat <name>   Send heartbeat for service")
	fmt.Println()
	fmt.Println("  backup <subcommand>   Backup operations")
	fmt.Println("    create           Create a backup")
	fmt.Println("    restore <file>   Restore from backup file")
	fmt.Println("    list             List available backups")
	fmt.Println("    export           Export data as JSON")
	fmt.Println()
	fmt.Println("  dns <query> <service>     DNS operations")
	fmt.Println("    srv <service>    Show SRV record query for service")
	fmt.Println("    a <service>      Show A record query for service")
	fmt.Println()
	fmt.Println("  ratelimit <subcommand>    Rate limit operations")
	fmt.Println("    stats            Show rate limit statistics")
	fmt.Println("    config           Show rate limit configuration")
	fmt.Println("    clients [--type <type>]  List active rate-limited clients")
	fmt.Println("    client <id>      Show specific client status")
	fmt.Println("    reset ip <ip>    Reset rate limit for IP")
	fmt.Println("    reset apikey <key>  Reset rate limit for API key")
	fmt.Println("    reset all [--type <type>]  Reset all rate limiters")
	fmt.Println("    update <--rate <n> | --burst <n>>  Update configuration")
	fmt.Println()
	fmt.Println("  version            Show version")
	fmt.Println("  help               Show this help")
	fmt.Println()
	fmt.Println("Global Options:")
	fmt.Println("  --server <url>     Konsul server URL (default: http://localhost:8888)")
}

func handleKVCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("KV subcommand required")
		fmt.Println("Usage: konsulctl kv <get|set|delete|list> [options]")
		os.Exit(1)
	}

	var serverURL string
	var tlsSkipVerify bool
	var tlsCACert string
	var tlsClientCert string
	var tlsClientKey string
	flagSet := flag.NewFlagSet("kv", flag.ExitOnError)
	flagSet.StringVar(&serverURL, "server", "http://localhost:8888", "Konsul server URL")
	flagSet.BoolVar(&tlsSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification")
	flagSet.StringVar(&tlsCACert, "ca-cert", "", "Path to CA certificate file")
	flagSet.StringVar(&tlsClientCert, "client-cert", "", "Path to client certificate file")
	flagSet.StringVar(&tlsClientKey, "client-key", "", "Path to client key file")

	subcommand := args[0]
	subArgs := args[1:]

	flagSet.Parse(subArgs)
	remainingArgs := flagSet.Args()

	tlsConfig := &TLSConfig{
		Enabled:        strings.HasPrefix(serverURL, "https://"),
		SkipVerify:     tlsSkipVerify,
		CACertFile:     tlsCACert,
		ClientCertFile: tlsClientCert,
		ClientKeyFile:  tlsClientKey,
	}

	switch subcommand {
	case "get":
		handleKVGet(serverURL, tlsConfig, remainingArgs)
	case "set":
		handleKVSet(serverURL, tlsConfig, remainingArgs)
	case "delete":
		handleKVDelete(serverURL, tlsConfig, remainingArgs)
	case "list":
		handleKVList(serverURL, tlsConfig, remainingArgs)
	default:
		fmt.Printf("Unknown KV subcommand: %s\n", subcommand)
		fmt.Println("Available: get, set, delete, list")
		os.Exit(1)
	}
}

func handleKVGet(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl kv get <key>")
		os.Exit(1)
	}

	key := args[0]
	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	value, err := client.GetKV(key)
	if err != nil {
		fmt.Printf("Error getting key '%s': %v\n", key, err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", value)
}

func handleKVSet(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 2 {
		fmt.Println("Usage: konsulctl kv set <key> <value>")
		os.Exit(1)
	}

	key := args[0]
	value := args[1]
	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	err := client.SetKV(key, value)
	if err != nil {
		fmt.Printf("Error setting key '%s': %v\n", key, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully set %s = %s\n", key, value)
}

func handleKVDelete(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl kv delete <key>")
		os.Exit(1)
	}

	key := args[0]
	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	err := client.DeleteKV(key)
	if err != nil {
		fmt.Printf("Error deleting key '%s': %v\n", key, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully deleted key: %s\n", key)
}

func handleKVList(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl kv list")
		os.Exit(1)
	}

	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	keys, err := client.ListKV()
	if err != nil {
		fmt.Printf("Error listing keys: %v\n", err)
		os.Exit(1)
	}

	if len(keys) == 0 {
		fmt.Println("No keys found")
		return
	}

	fmt.Println("Keys:")
	for _, key := range keys {
		fmt.Printf("  %s\n", key)
	}
}

func handleServiceCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Service subcommand required")
		fmt.Println("Usage: konsulctl service <register|list|deregister|heartbeat> [options]")
		os.Exit(1)
	}

	var serverURL string
	var tlsSkipVerify bool
	var tlsCACert string
	var tlsClientCert string
	var tlsClientKey string
	flagSet := flag.NewFlagSet("service", flag.ExitOnError)
	flagSet.StringVar(&serverURL, "server", "http://localhost:8888", "Konsul server URL")
	flagSet.BoolVar(&tlsSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification")
	flagSet.StringVar(&tlsCACert, "ca-cert", "", "Path to CA certificate file")
	flagSet.StringVar(&tlsClientCert, "client-cert", "", "Path to client certificate file")
	flagSet.StringVar(&tlsClientKey, "client-key", "", "Path to client key file")

	subcommand := args[0]
	subArgs := args[1:]

	flagSet.Parse(subArgs)
	remainingArgs := flagSet.Args()

	tlsConfig := &TLSConfig{
		Enabled:        strings.HasPrefix(serverURL, "https://"),
		SkipVerify:     tlsSkipVerify,
		CACertFile:     tlsCACert,
		ClientCertFile: tlsClientCert,
		ClientKeyFile:  tlsClientKey,
	}

	switch subcommand {
	case "register":
		handleServiceRegister(serverURL, tlsConfig, remainingArgs)
	case "list":
		handleServiceList(serverURL, tlsConfig, remainingArgs)
	case "deregister":
		handleServiceDeregister(serverURL, tlsConfig, remainingArgs)
	case "heartbeat":
		handleServiceHeartbeat(serverURL, tlsConfig, remainingArgs)
	default:
		fmt.Printf("Unknown service subcommand: %s\n", subcommand)
		fmt.Println("Available: register, list, deregister, heartbeat")
		os.Exit(1)
	}
}

func handleServiceRegister(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) < 3 {
		fmt.Println("Usage: konsulctl service register <name> <address> <port> [--check-http <url>] [--check-tcp <addr>] [--check-interval <duration>]")
		os.Exit(1)
	}

	name := args[0]
	address := args[1]
	port := args[2]

	// Parse health check flags
	var checks []*CheckDefinition

	// Simple parsing for health check flags
	for i := 3; i < len(args); i++ {
		switch args[i] {
		case "--check-http":
			if i+1 >= len(args) {
				fmt.Println("--check-http requires a URL")
				os.Exit(1)
			}
			checks = append(checks, &CheckDefinition{
				Name:     fmt.Sprintf("%s-http-check", name),
				HTTP:     args[i+1],
				Interval: "30s",
				Timeout:  "10s",
			})
			i++ // Skip the URL argument
		case "--check-tcp":
			if i+1 >= len(args) {
				fmt.Println("--check-tcp requires an address")
				os.Exit(1)
			}
			checks = append(checks, &CheckDefinition{
				Name:     fmt.Sprintf("%s-tcp-check", name),
				TCP:      args[i+1],
				Interval: "30s",
				Timeout:  "10s",
			})
			i++ // Skip the address argument
		case "--check-ttl":
			if i+1 >= len(args) {
				fmt.Println("--check-ttl requires a duration")
				os.Exit(1)
			}
			checks = append(checks, &CheckDefinition{
				Name: fmt.Sprintf("%s-ttl-check", name),
				TTL:  args[i+1],
			})
			i++ // Skip the duration argument
		}
	}

	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	err := client.RegisterServiceWithChecks(name, address, port, checks)
	if err != nil {
		fmt.Printf("Error registering service '%s': %v\n", name, err)
		os.Exit(1)
	}

	checkInfo := ""
	if len(checks) > 0 {
		checkInfo = fmt.Sprintf(" with %d health check(s)", len(checks))
	}

	fmt.Printf("Successfully registered service: %s at %s:%s%s\n", name, address, port, checkInfo)
}

func handleServiceList(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl service list")
		os.Exit(1)
	}

	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	services, err := client.ListServices()
	if err != nil {
		fmt.Printf("Error listing services: %v\n", err)
		os.Exit(1)
	}

	if len(services) == 0 {
		fmt.Println("No services found")
		return
	}

	fmt.Println("Services:")
	for _, service := range services {
		fmt.Printf("  %s - %s:%d\n", service.Name, service.Address, service.Port)
	}
}

func handleServiceDeregister(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl service deregister <name>")
		os.Exit(1)
	}

	name := args[0]
	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	err := client.DeregisterService(name)
	if err != nil {
		fmt.Printf("Error deregistering service '%s': %v\n", name, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully deregistered service: %s\n", name)
}

func handleServiceHeartbeat(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl service heartbeat <name>")
		os.Exit(1)
	}

	name := args[0]
	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	err := client.ServiceHeartbeat(name)
	if err != nil {
		fmt.Printf("Error sending heartbeat for service '%s': %v\n", name, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully sent heartbeat for service: %s\n", name)
}

func handleDNSCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("DNS subcommand required")
		fmt.Println("Usage: konsulctl dns <query> <service-name> [options]")
		fmt.Println("  query: 'srv' or 'a' (record type)")
		os.Exit(1)
	}

	var dnsServer string
	var dnsPort int
	flagSet := flag.NewFlagSet("dns", flag.ExitOnError)
	flagSet.StringVar(&dnsServer, "server", "localhost", "DNS server address")
	flagSet.IntVar(&dnsPort, "port", 8600, "DNS server port")

	subcommand := args[0]
	if len(args) < 2 {
		fmt.Println("Service name required")
		os.Exit(1)
	}
	serviceName := args[1]

	subArgs := args[2:]
	flagSet.Parse(subArgs)

	fmt.Printf("DNS %s query for service '%s' (server: %s:%d)\n", subcommand, serviceName, dnsServer, dnsPort)

	switch subcommand {
	case "srv":
		fmt.Printf("SRV Record: _%s._tcp.service.consul\n", serviceName)
		fmt.Printf("Run: dig @%s -p %d _%s._tcp.service.consul SRV\n", dnsServer, dnsPort, serviceName)
	case "a":
		fmt.Printf("A Record: %s.service.consul\n", serviceName)
		fmt.Printf("Run: dig @%s -p %d %s.service.consul A\n", dnsServer, dnsPort, serviceName)
	default:
		fmt.Printf("Unknown DNS query type: %s\n", subcommand)
		fmt.Println("Supported types: srv, a")
		os.Exit(1)
	}
}

func handleBackupCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Backup subcommand required")
		fmt.Println("Usage: konsulctl backup <create|restore|list|export> [options]")
		os.Exit(1)
	}

	var serverURL string
	var tlsSkipVerify bool
	var tlsCACert string
	var tlsClientCert string
	var tlsClientKey string
	flagSet := flag.NewFlagSet("backup", flag.ExitOnError)
	flagSet.StringVar(&serverURL, "server", "http://localhost:8888", "Konsul server URL")
	flagSet.BoolVar(&tlsSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification")
	flagSet.StringVar(&tlsCACert, "ca-cert", "", "Path to CA certificate file")
	flagSet.StringVar(&tlsClientCert, "client-cert", "", "Path to client certificate file")
	flagSet.StringVar(&tlsClientKey, "client-key", "", "Path to client key file")

	subcommand := args[0]
	subArgs := args[1:]

	flagSet.Parse(subArgs)
	remainingArgs := flagSet.Args()

	tlsConfig := &TLSConfig{
		Enabled:        strings.HasPrefix(serverURL, "https://"),
		SkipVerify:     tlsSkipVerify,
		CACertFile:     tlsCACert,
		ClientCertFile: tlsClientCert,
		ClientKeyFile:  tlsClientKey,
	}

	switch subcommand {
	case "create":
		handleBackupCreate(serverURL, tlsConfig, remainingArgs)
	case "restore":
		handleBackupRestore(serverURL, tlsConfig, remainingArgs)
	case "list":
		handleBackupList(serverURL, tlsConfig, remainingArgs)
	case "export":
		handleBackupExport(serverURL, tlsConfig, remainingArgs)
	default:
		fmt.Printf("Unknown backup subcommand: %s\n", subcommand)
		fmt.Println("Available: create, restore, list, export")
		os.Exit(1)
	}
}

func handleBackupCreate(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl backup create")
		os.Exit(1)
	}

	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	filename, err := client.CreateBackup()
	if err != nil {
		fmt.Printf("Error creating backup: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created backup: %s\n", filename)
}

func handleBackupRestore(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl backup restore <backup-file>")
		os.Exit(1)
	}

	backupFile := args[0]
	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	err := client.RestoreBackup(backupFile)
	if err != nil {
		fmt.Printf("Error restoring backup: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully restored from backup: %s\n", backupFile)
}

func handleBackupList(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl backup list")
		os.Exit(1)
	}

	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	backups, err := client.ListBackups()
	if err != nil {
		fmt.Printf("Error listing backups: %v\n", err)
		os.Exit(1)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found")
		return
	}

	fmt.Println("Available backups:")
	for _, backup := range backups {
		fmt.Printf("  %s\n", backup)
	}
}

func handleBackupExport(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl backup export")
		os.Exit(1)
	}

	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	data, err := client.ExportData()
	if err != nil {
		fmt.Printf("Error exporting data: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", data)
}

func handleRateLimitCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Rate limit subcommand required")
		fmt.Println("Usage: konsulctl ratelimit <stats|config|clients|client|reset|update> [options]")
		os.Exit(1)
	}

	var serverURL string
	var tlsSkipVerify bool
	var tlsCACert string
	var tlsClientCert string
	var tlsClientKey string
	flagSet := flag.NewFlagSet("ratelimit", flag.ExitOnError)
	flagSet.StringVar(&serverURL, "server", "http://localhost:8888", "Konsul server URL")
	flagSet.BoolVar(&tlsSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification")
	flagSet.StringVar(&tlsCACert, "ca-cert", "", "Path to CA certificate file")
	flagSet.StringVar(&tlsClientCert, "client-cert", "", "Path to client certificate file")
	flagSet.StringVar(&tlsClientKey, "client-key", "", "Path to client key file")

	subcommand := args[0]
	subArgs := args[1:]

	flagSet.Parse(subArgs)
	remainingArgs := flagSet.Args()

	tlsConfig := &TLSConfig{
		Enabled:        strings.HasPrefix(serverURL, "https://"),
		SkipVerify:     tlsSkipVerify,
		CACertFile:     tlsCACert,
		ClientCertFile: tlsClientCert,
		ClientKeyFile:  tlsClientKey,
	}

	switch subcommand {
	case "stats":
		handleRateLimitStats(serverURL, tlsConfig, remainingArgs)
	case "config":
		handleRateLimitConfig(serverURL, tlsConfig, remainingArgs)
	case "clients":
		handleRateLimitClients(serverURL, tlsConfig, remainingArgs)
	case "client":
		handleRateLimitClientStatus(serverURL, tlsConfig, remainingArgs)
	case "reset":
		handleRateLimitReset(serverURL, tlsConfig, remainingArgs)
	case "update":
		handleRateLimitUpdate(serverURL, tlsConfig, remainingArgs)
	default:
		fmt.Printf("Unknown rate limit subcommand: %s\n", subcommand)
		fmt.Println("Available: stats, config, clients, client, reset, update")
		os.Exit(1)
	}
}

func handleRateLimitStats(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl ratelimit stats")
		os.Exit(1)
	}

	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	stats, err := client.GetRateLimitStats()
	if err != nil {
		fmt.Printf("Error getting rate limit stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Rate Limit Statistics:")
	fmt.Printf("  IP Limiters: %v\n", stats.Data["ip_limiters"])
	fmt.Printf("  API Key Limiters: %v\n", stats.Data["apikey_limiters"])
}

func handleRateLimitConfig(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl ratelimit config")
		os.Exit(1)
	}

	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	config, err := client.GetRateLimitConfig()
	if err != nil {
		fmt.Printf("Error getting rate limit config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Rate Limit Configuration:")
	fmt.Printf("  Enabled: %t\n", config.Config.Enabled)
	fmt.Printf("  Requests per second: %.1f\n", config.Config.RequestsPerSec)
	fmt.Printf("  Burst: %d\n", config.Config.Burst)
	fmt.Printf("  By IP: %t\n", config.Config.ByIP)
	fmt.Printf("  By API Key: %t\n", config.Config.ByAPIKey)
	fmt.Printf("  Cleanup interval: %s\n", config.Config.CleanupInterval)
}

func handleRateLimitClients(serverURL string, tlsConfig *TLSConfig, args []string) {
	// Parse optional --type flag
	var clientType string
	flagSet := flag.NewFlagSet("clients", flag.ExitOnError)
	flagSet.StringVar(&clientType, "type", "all", "Client type: all, ip, or apikey")
	flagSet.Parse(args)

	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	clients, err := client.GetRateLimitClients(clientType)
	if err != nil {
		fmt.Printf("Error getting rate limit clients: %v\n", err)
		os.Exit(1)
	}

	if clients.Count == 0 {
		fmt.Println("No active rate-limited clients")
		return
	}

	fmt.Printf("Active Rate-Limited Clients (%d):\n", clients.Count)
	fmt.Println()
	for _, c := range clients.Clients {
		fmt.Printf("  Identifier: %s\n", c.Identifier)
		fmt.Printf("  Type: %s\n", c.Type)
		fmt.Printf("  Tokens: %.2f / %d\n", c.Tokens, c.MaxTokens)
		fmt.Printf("  Rate: %.1f req/s\n", c.Rate)
		fmt.Printf("  Last update: %s\n", c.LastUpdate)
		fmt.Println()
	}
}

func handleRateLimitClientStatus(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl ratelimit client <identifier>")
		os.Exit(1)
	}

	identifier := args[0]
	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	status, err := client.GetRateLimitClientStatus(identifier)
	if err != nil {
		fmt.Printf("Error getting client status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Client Status: %s\n", status.Identifier)
	fmt.Printf("  Type: %s\n", status.Type)
	fmt.Printf("  Tokens: %.2f / %d\n", status.Tokens, status.MaxTokens)
	fmt.Printf("  Rate: %.1f req/s\n", status.Rate)
	fmt.Printf("  Last update: %s\n", status.LastUpdate)

	// Show percentage
	percentage := (status.Tokens / float64(status.MaxTokens)) * 100
	fmt.Printf("  Capacity: %.1f%%\n", percentage)
}

func handleRateLimitReset(serverURL string, tlsConfig *TLSConfig, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: konsulctl ratelimit reset <ip|apikey|all> <value>")
		fmt.Println("  ip <ip-address>        Reset rate limit for specific IP")
		fmt.Println("  apikey <key-id>        Reset rate limit for specific API key")
		fmt.Println("  all [--type <type>]    Reset all rate limiters (type: all, ip, or apikey)")
		os.Exit(1)
	}

	resetType := args[0]
	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	switch resetType {
	case "ip":
		if len(args) != 2 {
			fmt.Println("Usage: konsulctl ratelimit reset ip <ip-address>")
			os.Exit(1)
		}
		ip := args[1]
		err := client.ResetRateLimitIP(ip)
		if err != nil {
			fmt.Printf("Error resetting rate limit for IP: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully reset rate limit for IP: %s\n", ip)

	case "apikey":
		if len(args) != 2 {
			fmt.Println("Usage: konsulctl ratelimit reset apikey <key-id>")
			os.Exit(1)
		}
		keyID := args[1]
		err := client.ResetRateLimitAPIKey(keyID)
		if err != nil {
			fmt.Printf("Error resetting rate limit for API key: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully reset rate limit for API key: %s\n", keyID)

	case "all":
		// Parse optional --type flag
		var limiterType string
		flagSet := flag.NewFlagSet("reset-all", flag.ExitOnError)
		flagSet.StringVar(&limiterType, "type", "all", "Limiter type: all, ip, or apikey")
		flagSet.Parse(args[1:])

		err := client.ResetRateLimitAll(limiterType)
		if err != nil {
			fmt.Printf("Error resetting rate limiters: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully reset all %s rate limiters\n", limiterType)

	default:
		fmt.Printf("Unknown reset type: %s\n", resetType)
		fmt.Println("Available: ip, apikey, all")
		os.Exit(1)
	}
}

func handleRateLimitUpdate(serverURL string, tlsConfig *TLSConfig, args []string) {
	var rate float64
	var burst int
	flagSet := flag.NewFlagSet("update", flag.ExitOnError)
	flagSet.Float64Var(&rate, "rate", 0, "Requests per second")
	flagSet.IntVar(&burst, "burst", 0, "Burst size")
	flagSet.Parse(args)

	if rate == 0 && burst == 0 {
		fmt.Println("Usage: konsulctl ratelimit update --rate <n> --burst <n>")
		fmt.Println("  At least one of --rate or --burst must be specified")
		os.Exit(1)
	}

	client := NewKonsulClientWithTLS(serverURL, tlsConfig)

	var ratePtr *float64
	var burstPtr *int
	if rate > 0 {
		ratePtr = &rate
	}
	if burst > 0 {
		burstPtr = &burst
	}

	resp, err := client.UpdateRateLimitConfig(ratePtr, burstPtr)
	if err != nil {
		fmt.Printf("Error updating rate limit config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", resp.Message)
	if resp.Config.RequestsPerSec > 0 || resp.Config.Burst > 0 {
		fmt.Println("Updated configuration:")
		if resp.Config.RequestsPerSec > 0 {
			fmt.Printf("  Requests per second: %.1f\n", resp.Config.RequestsPerSec)
		}
		if resp.Config.Burst > 0 {
			fmt.Printf("  Burst: %d\n", resp.Config.Burst)
		}
		fmt.Println()
		fmt.Println("Note: Changes apply to new limiters only.")
		fmt.Println("To apply to existing clients, run: konsulctl ratelimit reset all")
	}
}
