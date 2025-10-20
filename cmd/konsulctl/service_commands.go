package main

import (
	"flag"
	"fmt"
)

// ServiceCommands handles all service discovery related commands
type ServiceCommands struct {
	cli *CLI
}

// NewServiceCommands creates a new service commands handler
func NewServiceCommands(cli *CLI) *ServiceCommands {
	return &ServiceCommands{cli: cli}
}

// Handle routes service subcommands
func (s *ServiceCommands) Handle(args []string) {
	if len(args) == 0 {
		s.cli.Errorln("Service subcommand required")
		s.cli.Errorln("Usage: konsulctl service <register|list|deregister|heartbeat> [options]")
		s.cli.Exit(1)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "register":
		s.Register(subArgs)
	case "list":
		s.List(subArgs)
	case "deregister":
		s.Deregister(subArgs)
	case "heartbeat":
		s.Heartbeat(subArgs)
	default:
		s.cli.Errorf("Unknown service subcommand: %s\n", subcommand)
		s.cli.Errorln("Available: register, list, deregister, heartbeat")
		s.cli.Exit(1)
	}
}

// Register registers a new service
func (s *ServiceCommands) Register(args []string) {
	config, remaining, err := s.cli.ParseGlobalFlags(args, "register")
	if err == flag.ErrHelp {
		s.cli.Println("Usage: konsulctl service register <name> <address> <port> [--check-http <url>] [--check-tcp <addr>] [--check-ttl <duration>] [options]")
		return
	}
	s.cli.HandleError(err, "parsing flags")
	s.cli.ValidateMinArgs(remaining, 3, "Usage: konsulctl service register <name> <address> <port> [--check-http <url>] [--check-tcp <addr>] [--check-ttl <duration>]")

	name := remaining[0]
	address := remaining[1]
	port := remaining[2]

	// Parse health check flags
	checks, err := s.parseHealthChecks(name, remaining[3:])
	s.cli.HandleError(err, "parsing health checks")

	client := s.cli.CreateClient(config)
	err = client.RegisterServiceWithChecks(name, address, port, checks)
	s.cli.HandleError(err, "registering service '"+name+"'")

	checkInfo := ""
	if len(checks) > 0 {
		checkInfo = fmt.Sprintf(" with %d health check(s)", len(checks))
	}

	s.cli.Printf("Successfully registered service: %s at %s:%s%s\n", name, address, port, checkInfo)
}

// parseHealthChecks parses health check flags from arguments
func (s *ServiceCommands) parseHealthChecks(serviceName string, args []string) ([]*CheckDefinition, error) {
	var checks []*CheckDefinition

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--check-http":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--check-http requires a URL")
			}
			checks = append(checks, &CheckDefinition{
				Name:     fmt.Sprintf("%s-http-check", serviceName),
				HTTP:     args[i+1],
				Interval: "30s",
				Timeout:  "10s",
			})
			i++ // Skip the URL argument

		case "--check-tcp":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--check-tcp requires an address")
			}
			checks = append(checks, &CheckDefinition{
				Name:     fmt.Sprintf("%s-tcp-check", serviceName),
				TCP:      args[i+1],
				Interval: "30s",
				Timeout:  "10s",
			})
			i++ // Skip the address argument

		case "--check-ttl":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--check-ttl requires a duration")
			}
			checks = append(checks, &CheckDefinition{
				Name: fmt.Sprintf("%s-ttl-check", serviceName),
				TTL:  args[i+1],
			})
			i++ // Skip the duration argument

		default:
			// Ignore unknown flags (might be global flags already parsed)
			if args[i][0] == '-' {
				// Skip flag and its potential value
				if i+1 < len(args) && args[i+1][0] != '-' {
					i++
				}
			}
		}
	}

	return checks, nil
}

// List lists all registered services
func (s *ServiceCommands) List(args []string) {
	config, remaining, err := s.cli.ParseGlobalFlags(args, "list")
	if err == flag.ErrHelp {
		s.cli.Println("Usage: konsulctl service list [options]")
		return
	}
	s.cli.HandleError(err, "parsing flags")
	s.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl service list")

	client := s.cli.CreateClient(config)

	services, err := client.ListServices()
	s.cli.HandleError(err, "listing services")

	if len(services) == 0 {
		s.cli.Println("No services found")
		return
	}

	s.cli.Println("Services:")
	for _, service := range services {
		s.cli.Printf("  %s - %s:%d\n", service.Name, service.Address, service.Port)
	}
}

// Deregister removes a service registration
func (s *ServiceCommands) Deregister(args []string) {
	config, remaining, err := s.cli.ParseGlobalFlags(args, "deregister")
	if err == flag.ErrHelp {
		s.cli.Println("Usage: konsulctl service deregister <name> [options]")
		return
	}
	s.cli.HandleError(err, "parsing flags")
	s.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl service deregister <name>")

	name := remaining[0]
	client := s.cli.CreateClient(config)

	err = client.DeregisterService(name)
	s.cli.HandleError(err, "deregistering service '"+name+"'")

	s.cli.Printf("Successfully deregistered service: %s\n", name)
}

// Heartbeat sends a heartbeat for a service
func (s *ServiceCommands) Heartbeat(args []string) {
	config, remaining, err := s.cli.ParseGlobalFlags(args, "heartbeat")
	if err == flag.ErrHelp {
		s.cli.Println("Usage: konsulctl service heartbeat <name> [options]")
		return
	}
	s.cli.HandleError(err, "parsing flags")
	s.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl service heartbeat <name>")

	name := remaining[0]
	client := s.cli.CreateClient(config)

	err = client.ServiceHeartbeat(name)
	s.cli.HandleError(err, "sending heartbeat for service '"+name+"'")

	s.cli.Printf("Successfully sent heartbeat for service: %s\n", name)
}
