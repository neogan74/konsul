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

func handleBackupRestore(serverURL string, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl backup restore <backup-file>")
		os.Exit(1)
	}

	backupFile := args[0]
	client := NewKonsulClient(serverURL)

	err := client.RestoreBackup(backupFile)
	if err != nil {
		fmt.Printf("Error restoring backup: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully restored from backup: %s\n", backupFile)
}

func handleBackupList(serverURL string, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl backup list")
		os.Exit(1)
	}

	client := NewKonsulClient(serverURL)

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

func handleBackupExport(serverURL string, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl backup export")
		os.Exit(1)
	}

	client := NewKonsulClient(serverURL)

	data, err := client.ExportData()
	if err != nil {
		fmt.Printf("Error exporting data: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", data)
}
