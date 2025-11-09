package main

import (
	"fmt"
	"os"
)

const version = "1.0.0"

func main() {
	cli := NewCLI()

	if len(os.Args) < 2 {
		printUsage()
		cli.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "kv":
		kvCmd := NewKVCommands(cli)
		kvCmd.Handle(args)
	case "service":
		serviceCmd := NewServiceCommands(cli)
		serviceCmd.Handle(args)
	case "backup":
		backupCmd := NewBackupCommands(cli)
		backupCmd.Handle(args)
	case "dns":
		dnsCmd := NewDNSCommands(cli)
		dnsCmd.Handle(args)
	case "ratelimit":
		rateLimitCmd := NewRateLimitCommands(cli)
		rateLimitCmd.Handle(args)
	case "acl":
		aclCmd := NewACLCommands(cli)
		aclCmd.Handle(args)
	case "version":
		cli.Printf("konsulctl version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		cli.Printf("Unknown command: %s\n", command)
		printUsage()
		cli.Exit(1)
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
	fmt.Println("    watch <key|pattern>  Watch for key changes")
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
	fmt.Println("  acl <subcommand>    ACL policy operations")
	fmt.Println("    policy list      List all policies")
	fmt.Println("    policy get <name>  Get policy details")
	fmt.Println("    policy create <file>  Create policy from JSON file")
	fmt.Println("    policy update <file>  Update policy from JSON file")
	fmt.Println("    policy delete <name>  Delete a policy")
	fmt.Println("    test <policies> <resource> <path> <capability>  Test ACL permissions")
	fmt.Println()
	fmt.Println("  version            Show version")
	fmt.Println("  help               Show this help")
	fmt.Println()
	fmt.Println("Global Options:")
	fmt.Println("  --server <url>     Konsul server URL (default: http://localhost:8888)")
}
