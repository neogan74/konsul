package main

import (
	"flag"
)

// BackupCommands handles all backup and restore related commands
type BackupCommands struct {
	cli *CLI
}

// NewBackupCommands creates a new backup commands handler
func NewBackupCommands(cli *CLI) *BackupCommands {
	return &BackupCommands{cli: cli}
}

// Handle routes backup subcommands
func (b *BackupCommands) Handle(args []string) {
	if len(args) == 0 {
		b.cli.Errorln("Backup subcommand required")
		b.cli.Errorln("Usage: konsulctl backup <create|restore|list|export> [options]")
		b.cli.Exit(1)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "create":
		b.Create(subArgs)
	case "restore":
		b.Restore(subArgs)
	case "list":
		b.List(subArgs)
	case "export":
		b.Export(subArgs)
	default:
		b.cli.Errorf("Unknown backup subcommand: %s\n", subcommand)
		b.cli.Errorln("Available: create, restore, list, export")
		b.cli.Exit(1)
	}
}

// Create creates a new backup
func (b *BackupCommands) Create(args []string) {
	config, remaining, err := b.cli.ParseGlobalFlags(args, "create")
	if err == flag.ErrHelp {
		b.cli.Println("Usage: konsulctl backup create [options]")
		return
	}
	b.cli.HandleError(err, "parsing flags")
	b.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl backup create")

	client := b.cli.CreateClient(config)

	filename, err := client.CreateBackup()
	b.cli.HandleError(err, "creating backup")

	b.cli.Printf("Successfully created backup: %s\n", filename)
}

// Restore restores from a backup file
func (b *BackupCommands) Restore(args []string) {
	config, remaining, err := b.cli.ParseGlobalFlags(args, "restore")
	if err == flag.ErrHelp {
		b.cli.Println("Usage: konsulctl backup restore <backup-file> [options]")
		return
	}
	b.cli.HandleError(err, "parsing flags")
	b.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl backup restore <backup-file>")

	backupFile := remaining[0]
	client := b.cli.CreateClient(config)

	err = client.RestoreBackup(backupFile)
	b.cli.HandleError(err, "restoring backup")

	b.cli.Printf("Successfully restored from backup: %s\n", backupFile)
}

// List lists all available backups
func (b *BackupCommands) List(args []string) {
	config, remaining, err := b.cli.ParseGlobalFlags(args, "list")
	if err == flag.ErrHelp {
		b.cli.Println("Usage: konsulctl backup list [options]")
		return
	}
	b.cli.HandleError(err, "parsing flags")
	b.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl backup list")

	client := b.cli.CreateClient(config)

	backups, err := client.ListBackups()
	b.cli.HandleError(err, "listing backups")

	if len(backups) == 0 {
		b.cli.Println("No backups found")
		return
	}

	b.cli.Println("Available backups:")
	for _, backup := range backups {
		b.cli.Printf("  %s\n", backup)
	}
}

// Export exports all data as JSON
func (b *BackupCommands) Export(args []string) {
	config, remaining, err := b.cli.ParseGlobalFlags(args, "export")
	if err == flag.ErrHelp {
		b.cli.Println("Usage: konsulctl backup export [options]")
		return
	}
	b.cli.HandleError(err, "parsing flags")
	b.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl backup export")

	client := b.cli.CreateClient(config)

	data, err := client.ExportData()
	b.cli.HandleError(err, "exporting data")

	b.cli.Printf("%s\n", data)
}
