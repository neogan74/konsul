package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// ACLCommands handles ACL policy operations
type ACLCommands struct {
	cli *CLI
}

// NewACLCommands creates a new ACL commands handler
func NewACLCommands(cli *CLI) *ACLCommands {
	return &ACLCommands{cli: cli}
}

// Handle processes ACL commands
func (cmd *ACLCommands) Handle(args []string) {
	if len(args) < 1 {
		cmd.printUsage()
		cmd.cli.Exit(1)
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "policy":
		cmd.handlePolicyCommands(subargs)
	case "test":
		cmd.handleTestCommand(subargs)
	case "help", "-h", "--help":
		cmd.printUsage()
	default:
		cmd.cli.Printf("Unknown ACL subcommand: %s\n", subcommand)
		cmd.printUsage()
		cmd.cli.Exit(1)
	}
}

// handlePolicyCommands handles policy-related commands
func (cmd *ACLCommands) handlePolicyCommands(args []string) {
	if len(args) < 1 {
		cmd.printPolicyUsage()
		cmd.cli.Exit(1)
	}

	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "list":
		cmd.listPolicies(actionArgs)
	case "get":
		if len(actionArgs) < 1 {
			cmd.cli.Printf("Usage: konsulctl acl policy get <policy-name>\n")
			cmd.cli.Exit(1)
		}
		cmd.getPolicy(actionArgs)
	case "create":
		if len(actionArgs) < 1 {
			cmd.cli.Printf("Usage: konsulctl acl policy create <policy-file>\n")
			cmd.cli.Exit(1)
		}
		cmd.createPolicy(actionArgs)
	case "update":
		if len(actionArgs) < 1 {
			cmd.cli.Printf("Usage: konsulctl acl policy update <policy-file>\n")
			cmd.cli.Exit(1)
		}
		cmd.updatePolicy(actionArgs)
	case "delete":
		if len(actionArgs) < 1 {
			cmd.cli.Printf("Usage: konsulctl acl policy delete <policy-name>\n")
			cmd.cli.Exit(1)
		}
		cmd.deletePolicy(actionArgs)
	case "help", "-h", "--help":
		cmd.printPolicyUsage()
	default:
		cmd.cli.Printf("Unknown policy action: %s\n", action)
		cmd.printPolicyUsage()
		cmd.cli.Exit(1)
	}
}

// listPolicies lists all ACL policies
func (cmd *ACLCommands) listPolicies(args []string) {
	config, _, err := cmd.cli.ParseGlobalFlags(args, "list")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl acl policy list [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")

	client := cmd.cli.CreateClient(config)

	result, err := client.ListACLPolicies()
	if err != nil {
		cmd.cli.Printf("Error listing policies: %v\n", err)
		cmd.cli.Exit(1)
	}

	if result.Count == 0 {
		cmd.cli.Printf("No policies found\n")
		return
	}

	cmd.cli.Printf("ACL Policies (%d):\n", result.Count)
	for _, policy := range result.Policies {
		cmd.cli.Printf("  - %s\n", policy)
	}
}

// getPolicy retrieves and displays a specific policy
func (cmd *ACLCommands) getPolicy(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "get")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl acl policy get <name> [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl acl policy get <name>")

	name := remaining[0]
	client := cmd.cli.CreateClient(config)

	policy, err := client.GetACLPolicy(name)
	if err != nil {
		cmd.cli.Printf("Error getting policy: %v\n", err)
		cmd.cli.Exit(1)
	}

	// Pretty print JSON
	jsonData, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		cmd.cli.Printf("Error formatting policy: %v\n", err)
		cmd.cli.Exit(1)
	}

	cmd.cli.Printf("%s\n", string(jsonData))
}

// createPolicy creates a new ACL policy from a file
func (cmd *ACLCommands) createPolicy(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "create")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl acl policy create <file> [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl acl policy create <file>")

	filePath := remaining[0]
	client := cmd.cli.CreateClient(config)

	// Read policy file
	data, err := os.ReadFile(filePath)
	if err != nil {
		cmd.cli.Printf("Error reading policy file: %v\n", err)
		cmd.cli.Exit(1)
	}

	// Parse JSON to validate
	var policy map[string]interface{}
	if err := json.Unmarshal(data, &policy); err != nil {
		cmd.cli.Printf("Error parsing policy JSON: %v\n", err)
		cmd.cli.Exit(1)
	}

	// Create policy
	_, err = client.CreateACLPolicy(policy)
	if err != nil {
		cmd.cli.Printf("Error creating policy: %v\n", err)
		cmd.cli.Exit(1)
	}

	cmd.cli.Printf("Policy created successfully: %s\n", policy["name"])
}

// updatePolicy updates an existing ACL policy
func (cmd *ACLCommands) updatePolicy(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "update")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl acl policy update <file> [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl acl policy update <file>")

	filePath := remaining[0]
	client := cmd.cli.CreateClient(config)

	// Read policy file
	data, err := os.ReadFile(filePath)
	if err != nil {
		cmd.cli.Printf("Error reading policy file: %v\n", err)
		cmd.cli.Exit(1)
	}

	// Parse JSON to get policy name
	var policy map[string]interface{}
	if err := json.Unmarshal(data, &policy); err != nil {
		cmd.cli.Printf("Error parsing policy JSON: %v\n", err)
		cmd.cli.Exit(1)
	}

	name, ok := policy["name"].(string)
	if !ok || name == "" {
		cmd.cli.Printf("Policy must have a 'name' field\n")
		cmd.cli.Exit(1)
	}

	// Update policy
	_, err = client.UpdateACLPolicy(name, policy)
	if err != nil {
		cmd.cli.Printf("Error updating policy: %v\n", err)
		cmd.cli.Exit(1)
	}

	cmd.cli.Printf("Policy updated successfully: %s\n", name)
}

// deletePolicy deletes an ACL policy
func (cmd *ACLCommands) deletePolicy(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "delete")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl acl policy delete <name> [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl acl policy delete <name>")

	name := remaining[0]
	client := cmd.cli.CreateClient(config)

	// Confirm deletion
	cmd.cli.Printf("Are you sure you want to delete policy '%s'? (yes/no): ", name)
	var confirm string
	fmt.Scanln(&confirm)

	if strings.ToLower(confirm) != "yes" && strings.ToLower(confirm) != "y" {
		cmd.cli.Printf("Deletion cancelled\n")
		return
	}

	_, err = client.DeleteACLPolicy(name)
	if err != nil {
		cmd.cli.Printf("Error deleting policy: %v\n", err)
		cmd.cli.Exit(1)
	}

	cmd.cli.Printf("Policy deleted successfully: %s\n", name)
}

// handleTestCommand tests ACL permissions
func (cmd *ACLCommands) handleTestCommand(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "test")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl acl test <policies> <resource> <path> <capability> [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")

	if len(remaining) < 4 {
		cmd.cli.Printf("Usage: konsulctl acl test <policies> <resource> <path> <capability>\n")
		cmd.cli.Printf("Example: konsulctl acl test developer,readonly kv app/config read\n")
		cmd.cli.Exit(1)
	}

	policies := strings.Split(remaining[0], ",")
	resource := remaining[1]
	path := remaining[2]
	capability := remaining[3]

	client := cmd.cli.CreateClient(config)

	result, err := client.TestACLPolicy(policies, resource, path, capability)
	if err != nil {
		cmd.cli.Printf("Error testing policy: %v\n", err)
		cmd.cli.Exit(1)
	}

	allowed, _ := result["allowed"].(bool)

	cmd.cli.Printf("\nACL Test Result:\n")
	cmd.cli.Printf("  Policies:   %s\n", strings.Join(policies, ", "))
	cmd.cli.Printf("  Resource:   %s\n", resource)
	cmd.cli.Printf("  Path:       %s\n", path)
	cmd.cli.Printf("  Capability: %s\n", capability)
	cmd.cli.Printf("  Result:     ")

	if allowed {
		cmd.cli.Printf("ALLOWED ✓\n")
	} else {
		cmd.cli.Printf("DENIED ✗\n")
	}
}

// printUsage prints ACL command usage
func (cmd *ACLCommands) printUsage() {
	fmt.Println("ACL Commands - Access Control List management")
	fmt.Println()
	fmt.Println("Usage: konsulctl acl <subcommand> [options]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  policy <action>    Policy management")
	fmt.Println("    list             List all policies")
	fmt.Println("    get <name>       Get policy details")
	fmt.Println("    create <file>    Create policy from JSON file")
	fmt.Println("    update <file>    Update policy from JSON file")
	fmt.Println("    delete <name>    Delete a policy")
	fmt.Println()
	fmt.Println("  test <policies> <resource> <path> <capability>")
	fmt.Println("                     Test if policies allow an operation")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # List all policies")
	fmt.Println("  konsulctl acl policy list")
	fmt.Println()
	fmt.Println("  # Get policy details")
	fmt.Println("  konsulctl acl policy get developer")
	fmt.Println()
	fmt.Println("  # Create policy from file")
	fmt.Println("  konsulctl acl policy create policies/developer.json")
	fmt.Println()
	fmt.Println("  # Test ACL permissions")
	fmt.Println("  konsulctl acl test developer,readonly kv app/config read")
	fmt.Println()
}

// printPolicyUsage prints policy command usage
func (cmd *ACLCommands) printPolicyUsage() {
	fmt.Println("Policy Commands - ACL Policy management")
	fmt.Println()
	fmt.Println("Usage: konsulctl acl policy <action> [options]")
	fmt.Println()
	fmt.Println("Actions:")
	fmt.Println("  list             List all policies")
	fmt.Println("  get <name>       Get policy details")
	fmt.Println("  create <file>    Create policy from JSON file")
	fmt.Println("  update <file>    Update policy from JSON file")
	fmt.Println("  delete <name>    Delete a policy")
	fmt.Println()
}
