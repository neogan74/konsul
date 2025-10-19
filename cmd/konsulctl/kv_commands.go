package main

import (
	"flag"
)

// KVCommands handles all KV store related commands
type KVCommands struct {
	cli *CLI
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
	default:
		k.cli.Errorf("Unknown KV subcommand: %s\n", subcommand)
		k.cli.Errorln("Available: get, set, delete, list")
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
