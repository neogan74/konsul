package main

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

// RBACCommands handles RBAC role and assignment operations.
type RBACCommands struct {
	cli *CLI
}

// NewRBACCommands creates a new RBAC commands handler.
func NewRBACCommands(cli *CLI) *RBACCommands {
	return &RBACCommands{cli: cli}
}

// Handle processes RBAC commands.
func (cmd *RBACCommands) Handle(args []string) {
	if len(args) < 1 {
		cmd.printUsage()
		cmd.cli.Exit(1)
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "role":
		cmd.handleRoleCommands(subargs)
	case "assign":
		cmd.assignRole(subargs)
	case "revoke":
		cmd.revokeRole(subargs)
	case "list-assignments":
		cmd.listAssignments(subargs)
	case "get-assignment":
		cmd.getAssignment(subargs)
	case "effective-policies":
		cmd.effectivePolicies(subargs)
	case "help", "-h", "--help":
		cmd.printUsage()
	default:
		cmd.cli.Printf("Unknown RBAC subcommand: %s\n", subcommand)
		cmd.printUsage()
		cmd.cli.Exit(1)
	}
}

// handleRoleCommands handles role-related commands.
func (cmd *RBACCommands) handleRoleCommands(args []string) {
	if len(args) < 1 {
		cmd.printRoleUsage()
		cmd.cli.Exit(1)
	}

	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "create":
		cmd.createRole(actionArgs)
	case "list":
		cmd.listRoles(actionArgs)
	case "get":
		cmd.getRole(actionArgs)
	case "update":
		cmd.updateRole(actionArgs)
	case "delete":
		cmd.deleteRole(actionArgs)
	case "help", "-h", "--help":
		cmd.printRoleUsage()
	default:
		cmd.cli.Printf("Unknown role action: %s\n", action)
		cmd.printRoleUsage()
		cmd.cli.Exit(1)
	}
}

// roleFlags holds the parsed --name/--description/--policies/--inherits-from flags
// shared by `role create` and `role update`.
type roleFlags struct {
	name         string
	description  string
	policies     string
	inheritsFrom string
}

func (cmd *RBACCommands) parseRoleFlags(args []string, fsName string) (*GlobalConfig, *roleFlags, []string, error) {
	rf := &roleFlags{}

	flagSet := flag.NewFlagSet(fsName, flag.ContinueOnError)
	flagSet.SetOutput(cmd.cli.Error)
	flagSet.StringVar(&rf.name, "name", "", "Role name")
	flagSet.StringVar(&rf.description, "description", "", "Role description")
	flagSet.StringVar(&rf.policies, "policies", "", "Comma-separated list of policy names")
	flagSet.StringVar(&rf.inheritsFrom, "inherits-from", "", "Comma-separated list of parent role names")

	config := &GlobalConfig{}
	flagSet.StringVar(&config.ServerURL, "server", "http://localhost:8888", "Konsul server URL")
	flagSet.BoolVar(&config.TLSSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification")
	flagSet.StringVar(&config.TLSCACert, "ca-cert", "", "Path to CA certificate file")
	flagSet.StringVar(&config.TLSClientCert, "client-cert", "", "Path to client certificate file")
	flagSet.StringVar(&config.TLSClientKey, "client-key", "", "Path to client key file")

	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		return nil, nil, nil, flag.ErrHelp
	}

	if err := flagSet.Parse(args); err != nil {
		return nil, nil, nil, err
	}

	return config, rf, flagSet.Args(), nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// createRole creates a new RBAC role.
func (cmd *RBACCommands) createRole(args []string) {
	config, rf, remaining, err := cmd.parseRoleFlags(args, "role create")
	if err == flag.ErrHelp {
		cmd.printRoleCreateUsage()
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateMaxArgs(remaining, 0, "Usage: konsulctl rbac role create --name <name> [options]")

	if rf.name == "" {
		cmd.cli.Printf("--name is required\n")
		cmd.printRoleCreateUsage()
		cmd.cli.Exit(1)
	}

	client := cmd.cli.CreateClient(config)

	role := &RBACRole{
		Name:        rf.name,
		Description: rf.description,
		Policies:    splitCSV(rf.policies),
		ParentRoles: splitCSV(rf.inheritsFrom),
	}

	created, err := client.CreateRBACRole(role)
	if err != nil {
		cmd.cli.Printf("Error creating role: %v\n", err)
		cmd.cli.Exit(1)
	}

	cmd.cli.Printf("Role created successfully: %s\n", created.Name)
}

// listRoles lists all RBAC roles.
func (cmd *RBACCommands) listRoles(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "role list")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl rbac role list [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateMaxArgs(remaining, 0, "Usage: konsulctl rbac role list [options]")

	client := cmd.cli.CreateClient(config)

	result, err := client.ListRBACRoles()
	if err != nil {
		cmd.cli.Printf("Error listing roles: %v\n", err)
		cmd.cli.Exit(1)
	}

	if result.Count == 0 {
		cmd.cli.Printf("No roles found\n")
		return
	}

	cmd.cli.Printf("RBAC Roles (%d):\n", result.Count)
	for _, role := range result.Roles {
		parents := "-"
		if len(role.ParentRoles) > 0 {
			parents = strings.Join(role.ParentRoles, ", ")
		}
		cmd.cli.Printf("  - %-20s policies=[%s] inherits-from=[%s]\n",
			role.Name, strings.Join(role.Policies, ", "), parents)
	}
}

// getRole retrieves and displays a specific role.
func (cmd *RBACCommands) getRole(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "role get")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl rbac role get <name> [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl rbac role get <name>")

	name := remaining[0]
	client := cmd.cli.CreateClient(config)

	role, err := client.GetRBACRole(name)
	if err != nil {
		cmd.cli.Printf("Error getting role: %v\n", err)
		cmd.cli.Exit(1)
	}

	cmd.printRole(role)
}

// updateRole replaces an existing role's definition. Since the API performs a full
// replace (PUT), any flag not supplied falls back to the role's current value.
func (cmd *RBACCommands) updateRole(args []string) {
	config, rf, remaining, err := cmd.parseRoleFlags(args, "role update")
	if err == flag.ErrHelp {
		cmd.printRoleUpdateUsage()
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl rbac role update <name> [options]")

	name := remaining[0]
	client := cmd.cli.CreateClient(config)

	existing, err := client.GetRBACRole(name)
	if err != nil {
		cmd.cli.Printf("Error fetching existing role: %v\n", err)
		cmd.cli.Exit(1)
	}

	updated := &RBACRole{
		Name:        name,
		Description: existing.Description,
		Policies:    existing.Policies,
		ParentRoles: existing.ParentRoles,
	}
	if rf.description != "" {
		updated.Description = rf.description
	}
	if rf.policies != "" {
		updated.Policies = splitCSV(rf.policies)
	}
	if rf.inheritsFrom != "" {
		updated.ParentRoles = splitCSV(rf.inheritsFrom)
	}

	result, err := client.UpdateRBACRole(name, updated)
	if err != nil {
		cmd.cli.Printf("Error updating role: %v\n", err)
		cmd.cli.Exit(1)
	}

	cmd.cli.Printf("Role updated successfully: %s\n", result.Name)
}

// deleteRole deletes an RBAC role.
func (cmd *RBACCommands) deleteRole(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "role delete")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl rbac role delete <name> [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl rbac role delete <name>")

	name := remaining[0]
	client := cmd.cli.CreateClient(config)

	cmd.cli.Printf("Are you sure you want to delete role '%s'? (yes/no): ", name)
	var confirm string
	_, _ = fmt.Scanln(&confirm)

	if !strings.EqualFold(confirm, "yes") && !strings.EqualFold(confirm, "y") {
		cmd.cli.Printf("Deletion canceled\n")
		return
	}

	if err := client.DeleteRBACRole(name); err != nil {
		cmd.cli.Printf("Error deleting role: %v\n", err)
		cmd.cli.Exit(1)
	}

	cmd.cli.Printf("Role deleted successfully: %s\n", name)
}

// assignRole assigns one or more roles to a subject, with an optional TTL.
func (cmd *RBACCommands) assignRole(args []string) {
	var subjectFlag, rolesFlag, ttlFlag string

	flagSet := flag.NewFlagSet("assign", flag.ContinueOnError)
	flagSet.SetOutput(cmd.cli.Error)
	flagSet.StringVar(&subjectFlag, "subject", "", "Subject ID (user or token) to assign roles to")
	flagSet.StringVar(&rolesFlag, "roles", "", "Comma-separated list of role names to assign")
	flagSet.StringVar(&ttlFlag, "ttl", "", "Optional duration after which the assignment expires (e.g. 24h)")

	config := &GlobalConfig{}
	flagSet.StringVar(&config.ServerURL, "server", "http://localhost:8888", "Konsul server URL")
	flagSet.BoolVar(&config.TLSSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification")
	flagSet.StringVar(&config.TLSCACert, "ca-cert", "", "Path to CA certificate file")
	flagSet.StringVar(&config.TLSClientCert, "client-cert", "", "Path to client certificate file")
	flagSet.StringVar(&config.TLSClientKey, "client-key", "", "Path to client key file")

	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		cmd.printAssignUsage()
		return
	}
	err := flagSet.Parse(args)
	cmd.cli.HandleError(err, "parsing flags")

	if subjectFlag == "" || rolesFlag == "" {
		cmd.cli.Printf("--subject and --roles are required\n")
		cmd.printAssignUsage()
		cmd.cli.Exit(1)
	}

	roleNames := splitCSV(rolesFlag)

	var expiresAt *time.Time
	if ttlFlag != "" {
		d, err := time.ParseDuration(ttlFlag)
		if err != nil {
			cmd.cli.Printf("Invalid --ttl value %q: %v\n", ttlFlag, err)
			cmd.cli.Exit(1)
		}
		t := time.Now().Add(d)
		expiresAt = &t
	}

	client := cmd.cli.CreateClient(config)

	if err := client.AssignRBACRole(subjectFlag, roleNames, expiresAt); err != nil {
		cmd.cli.Printf("Error assigning roles: %v\n", err)
		cmd.cli.Exit(1)
	}

	cmd.cli.Printf("Roles assigned to %s: %s\n", subjectFlag, strings.Join(roleNames, ", "))
	if expiresAt != nil {
		cmd.cli.Printf("Expires at: %s\n", expiresAt.Format(time.RFC3339))
	}
}

// revokeRole removes all role assignments for a subject.
func (cmd *RBACCommands) revokeRole(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "revoke")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl rbac revoke <subject-id> [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl rbac revoke <subject-id>")

	subjectID := remaining[0]
	client := cmd.cli.CreateClient(config)

	if err := client.UnassignRBACRole(subjectID); err != nil {
		cmd.cli.Printf("Error revoking roles: %v\n", err)
		cmd.cli.Exit(1)
	}

	cmd.cli.Printf("Roles revoked for: %s\n", subjectID)
}

// listAssignments lists all RBAC role assignments.
func (cmd *RBACCommands) listAssignments(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "list-assignments")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl rbac list-assignments [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateMaxArgs(remaining, 0, "Usage: konsulctl rbac list-assignments [options]")

	client := cmd.cli.CreateClient(config)

	result, err := client.ListRBACAssignments()
	if err != nil {
		cmd.cli.Printf("Error listing assignments: %v\n", err)
		cmd.cli.Exit(1)
	}

	if result.Count == 0 {
		cmd.cli.Printf("No assignments found\n")
		return
	}

	cmd.cli.Printf("RBAC Assignments (%d):\n", result.Count)
	for _, a := range result.Assignments {
		expiry := "never"
		if a.ExpiresAt != nil {
			expiry = a.ExpiresAt.Format(time.RFC3339)
		}
		cmd.cli.Printf("  - %-20s roles=[%s] expires=%s\n",
			a.SubjectID, strings.Join(a.RoleNames, ", "), expiry)
	}
}

// getAssignment retrieves the role assignment for a single subject.
func (cmd *RBACCommands) getAssignment(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "get-assignment")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl rbac get-assignment <subject-id> [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl rbac get-assignment <subject-id>")

	subjectID := remaining[0]
	client := cmd.cli.CreateClient(config)

	assignment, err := client.GetRBACAssignment(subjectID)
	if err != nil {
		cmd.cli.Printf("Error getting assignment: %v\n", err)
		cmd.cli.Exit(1)
	}

	expiry := "never"
	if assignment.ExpiresAt != nil {
		expiry = assignment.ExpiresAt.Format(time.RFC3339)
	}
	cmd.cli.Printf("Subject:     %s\n", assignment.SubjectID)
	cmd.cli.Printf("Roles:       %s\n", strings.Join(assignment.RoleNames, ", "))
	cmd.cli.Printf("Expires:     %s\n", expiry)
}

// effectivePolicies retrieves the resolved (inherited) policies for a subject.
func (cmd *RBACCommands) effectivePolicies(args []string) {
	config, remaining, err := cmd.cli.ParseGlobalFlags(args, "effective-policies")
	if err == flag.ErrHelp {
		cmd.cli.Println("Usage: konsulctl rbac effective-policies <subject-id> [options]")
		return
	}
	cmd.cli.HandleError(err, "parsing flags")
	cmd.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl rbac effective-policies <subject-id>")

	subjectID := remaining[0]
	client := cmd.cli.CreateClient(config)

	result, err := client.GetRBACEffectivePolicies(subjectID)
	if err != nil {
		cmd.cli.Printf("Error getting effective policies: %v\n", err)
		cmd.cli.Exit(1)
	}

	if result.Count == 0 {
		cmd.cli.Printf("No effective policies for: %s\n", subjectID)
		return
	}

	cmd.cli.Printf("Effective policies for %s (%d):\n", subjectID, result.Count)
	for _, p := range result.Policies {
		cmd.cli.Printf("  - %s\n", p)
	}
}

// printRole pretty-prints a single role.
func (cmd *RBACCommands) printRole(role *RBACRole) {
	cmd.cli.Printf("Name:          %s\n", role.Name)
	cmd.cli.Printf("Description:   %s\n", role.Description)
	cmd.cli.Printf("Policies:      %s\n", strings.Join(role.Policies, ", "))
	cmd.cli.Printf("Inherits From: %s\n", strings.Join(role.ParentRoles, ", "))
	if !role.CreatedAt.IsZero() {
		cmd.cli.Printf("Created At:    %s\n", role.CreatedAt.Format(time.RFC3339))
	}
	if !role.UpdatedAt.IsZero() {
		cmd.cli.Printf("Updated At:    %s\n", role.UpdatedAt.Format(time.RFC3339))
	}
}

// printUsage prints RBAC command usage.
func (cmd *RBACCommands) printUsage() {
	fmt.Println("RBAC Commands - Role-Based Access Control management")
	fmt.Println()
	fmt.Println("Usage: konsulctl rbac <subcommand> [options]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  role <action>      Role management (create, list, get, update, delete)")
	fmt.Println("  assign             Assign one or more roles to a subject")
	fmt.Println("  revoke <subject>   Revoke all role assignments for a subject")
	fmt.Println("  list-assignments   List all role assignments")
	fmt.Println("  get-assignment <subject>      Get the role assignment for a subject")
	fmt.Println("  effective-policies <subject> Show a subject's resolved (inherited) policies")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Create a role")
	fmt.Println("  konsulctl rbac role create --name developer --policies kv-rw,service-rw")
	fmt.Println()
	fmt.Println("  # Create a role that inherits from another")
	fmt.Println("  konsulctl rbac role create --name senior-dev --policies backup-create --inherits-from developer")
	fmt.Println()
	fmt.Println("  # Assign a role")
	fmt.Println("  konsulctl rbac assign --subject alice --roles developer")
	fmt.Println()
	fmt.Println("  # Assign a temporary role")
	fmt.Println("  konsulctl rbac assign --subject bob --roles oncall-admin --ttl 24h")
	fmt.Println()
	fmt.Println("  # Revoke a subject's roles")
	fmt.Println("  konsulctl rbac revoke alice")
	fmt.Println()
	fmt.Println("  # Show a subject's effective policies")
	fmt.Println("  konsulctl rbac effective-policies alice")
	fmt.Println()
}

// printRoleUsage prints role command usage.
func (cmd *RBACCommands) printRoleUsage() {
	fmt.Println("Role Commands - RBAC role management")
	fmt.Println()
	fmt.Println("Usage: konsulctl rbac role <action> [options]")
	fmt.Println()
	fmt.Println("Actions:")
	fmt.Println("  create   --name <name> [--description <desc>] [--policies <p1,p2>] [--inherits-from <r1,r2>]")
	fmt.Println("  list")
	fmt.Println("  get      <name>")
	fmt.Println("  update   <name> [--description <desc>] [--policies <p1,p2>] [--inherits-from <r1,r2>]")
	fmt.Println("  delete   <name>")
	fmt.Println()
}

func (cmd *RBACCommands) printRoleCreateUsage() {
	fmt.Println("Usage: konsulctl rbac role create --name <name> [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --name <name>              Role name (required)")
	fmt.Println("  --description <text>       Role description")
	fmt.Println("  --policies <p1,p2,...>     Comma-separated policy names")
	fmt.Println("  --inherits-from <r1,r2,...> Comma-separated parent role names")
}

func (cmd *RBACCommands) printRoleUpdateUsage() {
	fmt.Println("Usage: konsulctl rbac role update <name> [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --description <text>       New role description")
	fmt.Println("  --policies <p1,p2,...>     New comma-separated policy names")
	fmt.Println("  --inherits-from <r1,r2,...> New comma-separated parent role names")
	fmt.Println()
	fmt.Println("Unspecified fields keep their current value.")
}

func (cmd *RBACCommands) printAssignUsage() {
	fmt.Println("Usage: konsulctl rbac assign --subject <id> --roles <r1,r2> [--ttl <duration>] [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --subject <id>      Subject (user or token) ID (required)")
	fmt.Println("  --roles <r1,r2,...> Comma-separated role names to assign (required)")
	fmt.Println("  --ttl <duration>    Optional expiry, e.g. 24h, 30m")
}
