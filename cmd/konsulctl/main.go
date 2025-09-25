package main

import (
	"flag"
	"fmt"
	"os"
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
	flagSet := flag.NewFlagSet("kv", flag.ExitOnError)
	flagSet.StringVar(&serverURL, "server", "http://localhost:8888", "Konsul server URL")

	subcommand := args[0]
	subArgs := args[1:]

	flagSet.Parse(subArgs)
	remainingArgs := flagSet.Args()

	switch subcommand {
	case "get":
		handleKVGet(serverURL, remainingArgs)
	case "set":
		handleKVSet(serverURL, remainingArgs)
	case "delete":
		handleKVDelete(serverURL, remainingArgs)
	case "list":
		handleKVList(serverURL, remainingArgs)
	default:
		fmt.Printf("Unknown KV subcommand: %s\n", subcommand)
		fmt.Println("Available: get, set, delete, list")
		os.Exit(1)
	}
}

func handleKVGet(serverURL string, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl kv get <key>")
		os.Exit(1)
	}

	key := args[0]
	client := NewKonsulClient(serverURL)

	value, err := client.GetKV(key)
	if err != nil {
		fmt.Printf("Error getting key '%s': %v\n", key, err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", value)
}

func handleKVSet(serverURL string, args []string) {
	if len(args) != 2 {
		fmt.Println("Usage: konsulctl kv set <key> <value>")
		os.Exit(1)
	}

	key := args[0]
	value := args[1]
	client := NewKonsulClient(serverURL)

	err := client.SetKV(key, value)
	if err != nil {
		fmt.Printf("Error setting key '%s': %v\n", key, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully set %s = %s\n", key, value)
}

func handleKVDelete(serverURL string, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl kv delete <key>")
		os.Exit(1)
	}

	key := args[0]
	client := NewKonsulClient(serverURL)

	err := client.DeleteKV(key)
	if err != nil {
		fmt.Printf("Error deleting key '%s': %v\n", key, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully deleted key: %s\n", key)
}

func handleKVList(serverURL string, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl kv list")
		os.Exit(1)
	}

	client := NewKonsulClient(serverURL)

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
	flagSet := flag.NewFlagSet("service", flag.ExitOnError)
	flagSet.StringVar(&serverURL, "server", "http://localhost:8888", "Konsul server URL")

	subcommand := args[0]
	subArgs := args[1:]

	flagSet.Parse(subArgs)
	remainingArgs := flagSet.Args()

	switch subcommand {
	case "register":
		handleServiceRegister(serverURL, remainingArgs)
	case "list":
		handleServiceList(serverURL, remainingArgs)
	case "deregister":
		handleServiceDeregister(serverURL, remainingArgs)
	case "heartbeat":
		handleServiceHeartbeat(serverURL, remainingArgs)
	default:
		fmt.Printf("Unknown service subcommand: %s\n", subcommand)
		fmt.Println("Available: register, list, deregister, heartbeat")
		os.Exit(1)
	}
}

func handleServiceRegister(serverURL string, args []string) {
	if len(args) != 3 {
		fmt.Println("Usage: konsulctl service register <name> <address> <port>")
		os.Exit(1)
	}

	name := args[0]
	address := args[1]
	port := args[2]

	client := NewKonsulClient(serverURL)

	err := client.RegisterService(name, address, port)
	if err != nil {
		fmt.Printf("Error registering service '%s': %v\n", name, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully registered service: %s at %s:%s\n", name, address, port)
}

func handleServiceList(serverURL string, args []string) {
	if len(args) != 0 {
		fmt.Println("Usage: konsulctl service list")
		os.Exit(1)
	}

	client := NewKonsulClient(serverURL)

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

func handleServiceDeregister(serverURL string, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl service deregister <name>")
		os.Exit(1)
	}

	name := args[0]
	client := NewKonsulClient(serverURL)

	err := client.DeregisterService(name)
	if err != nil {
		fmt.Printf("Error deregistering service '%s': %v\n", name, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully deregistered service: %s\n", name)
}

func handleServiceHeartbeat(serverURL string, args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: konsulctl service heartbeat <name>")
		os.Exit(1)
	}

	name := args[0]
	client := NewKonsulClient(serverURL)

	err := client.ServiceHeartbeat(name)
	if err != nil {
		fmt.Printf("Error sending heartbeat for service '%s': %v\n", name, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully sent heartbeat for service: %s\n", name)
}