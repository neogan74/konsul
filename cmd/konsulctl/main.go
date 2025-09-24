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
	fmt.Println("  version            Show version")
	fmt.Println("  help               Show this help")
	fmt.Println()
	fmt.Println("Global Options:")
	fmt.Println("  --server <url>     Konsul server URL (default: http://localhost:8888)")
}

func handleKVCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("KV subcommand required")
		fmt.Println("Usage: konsulctl kv <get|set> [options]")
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
	default:
		fmt.Printf("Unknown KV subcommand: %s\n", subcommand)
		fmt.Println("Available: get, set")
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