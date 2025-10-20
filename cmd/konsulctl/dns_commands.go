package main

import (
	"flag"
)

// DNSCommands handles all DNS query related commands
type DNSCommands struct {
	cli *CLI
}

// NewDNSCommands creates a new DNS commands handler
func NewDNSCommands(cli *CLI) *DNSCommands {
	return &DNSCommands{cli: cli}
}

// Handle routes DNS subcommands
func (d *DNSCommands) Handle(args []string) {
	if len(args) == 0 {
		d.cli.Errorln("DNS subcommand required")
		d.cli.Errorln("Usage: konsulctl dns <query> <service-name> [options]")
		d.cli.Errorln("  query: 'srv' or 'a' (record type)")
		d.cli.Exit(1)
		return
	}

	if len(args) < 2 {
		d.cli.Errorln("Service name required")
		d.cli.Exit(1)
		return
	}

	queryType := args[0]
	serviceName := args[1]
	queryArgs := args[2:]

	switch queryType {
	case "srv":
		d.SRV(serviceName, queryArgs)
	case "a":
		d.A(serviceName, queryArgs)
	default:
		d.cli.Errorf("Unknown DNS query type: %s\n", queryType)
		d.cli.Errorln("Supported types: srv, a")
		d.cli.Exit(1)
	}
}

// SRV shows SRV record query for a service
func (d *DNSCommands) SRV(serviceName string, args []string) {
	var dnsServer string
	var dnsPort int

	flagSet := flag.NewFlagSet("srv", flag.ContinueOnError)
	flagSet.SetOutput(d.cli.Error)
	flagSet.StringVar(&dnsServer, "server", "localhost", "DNS server address")
	flagSet.IntVar(&dnsPort, "port", 8600, "DNS server port")

	err := flagSet.Parse(args)
	if err == flag.ErrHelp {
		d.cli.Println("Usage: konsulctl dns srv <service-name> [--server <addr>] [--port <port>]")
		return
	}
	d.cli.HandleError(err, "parsing flags")

	d.cli.Printf("DNS SRV query for service '%s' (server: %s:%d)\n", serviceName, dnsServer, dnsPort)
	d.cli.Printf("SRV Record: _%s._tcp.service.consul\n", serviceName)
	d.cli.Printf("Run: dig @%s -p %d _%s._tcp.service.consul SRV\n", dnsServer, dnsPort, serviceName)
}

// A shows A record query for a service
func (d *DNSCommands) A(serviceName string, args []string) {
	var dnsServer string
	var dnsPort int

	flagSet := flag.NewFlagSet("a", flag.ContinueOnError)
	flagSet.SetOutput(d.cli.Error)
	flagSet.StringVar(&dnsServer, "server", "localhost", "DNS server address")
	flagSet.IntVar(&dnsPort, "port", 8600, "DNS server port")

	err := flagSet.Parse(args)
	if err == flag.ErrHelp {
		d.cli.Println("Usage: konsulctl dns a <service-name> [--server <addr>] [--port <port>]")
		return
	}
	d.cli.HandleError(err, "parsing flags")

	d.cli.Printf("DNS A query for service '%s' (server: %s:%d)\n", serviceName, dnsServer, dnsPort)
	d.cli.Printf("A Record: %s.service.consul\n", serviceName)
	d.cli.Printf("Run: dig @%s -p %d %s.service.consul A\n", dnsServer, dnsPort, serviceName)
}
