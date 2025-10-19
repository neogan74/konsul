package main

import (
	"flag"
)

// RateLimitCommands handles all rate limit related commands
type RateLimitCommands struct {
	cli *CLI
}

// NewRateLimitCommands creates a new rate limit commands handler
func NewRateLimitCommands(cli *CLI) *RateLimitCommands {
	return &RateLimitCommands{cli: cli}
}

// Handle routes rate limit subcommands
func (r *RateLimitCommands) Handle(args []string) {
	if len(args) == 0 {
		r.cli.Errorln("Rate limit subcommand required")
		r.cli.Errorln("Usage: konsulctl ratelimit <stats|config|clients|client|reset|update> [options]")
		r.cli.Exit(1)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "stats":
		r.Stats(subArgs)
	case "config":
		r.Config(subArgs)
	case "clients":
		r.Clients(subArgs)
	case "client":
		r.ClientStatus(subArgs)
	case "reset":
		r.Reset(subArgs)
	case "update":
		r.Update(subArgs)
	default:
		r.cli.Errorf("Unknown rate limit subcommand: %s\n", subcommand)
		r.cli.Errorln("Available: stats, config, clients, client, reset, update")
		r.cli.Exit(1)
	}
}

// Stats shows rate limit statistics
func (r *RateLimitCommands) Stats(args []string) {
	config, remaining, err := r.cli.ParseGlobalFlags(args, "stats")
	if err == flag.ErrHelp {
		r.cli.Println("Usage: konsulctl ratelimit stats [options]")
		return
	}
	r.cli.HandleError(err, "parsing flags")
	r.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl ratelimit stats")

	client := r.cli.CreateClient(config)
	stats, err := client.GetRateLimitStats()
	r.cli.HandleError(err, "getting rate limit stats")

	r.cli.Println("Rate Limit Statistics:")
	r.cli.Printf("  IP Limiters: %v\n", stats.Data["ip_limiters"])
	r.cli.Printf("  API Key Limiters: %v\n", stats.Data["apikey_limiters"])
}

// Config shows rate limit configuration
func (r *RateLimitCommands) Config(args []string) {
	config, remaining, err := r.cli.ParseGlobalFlags(args, "config")
	if err == flag.ErrHelp {
		r.cli.Println("Usage: konsulctl ratelimit config [options]")
		return
	}
	r.cli.HandleError(err, "parsing flags")
	r.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl ratelimit config")

	client := r.cli.CreateClient(config)
	rlConfig, err := client.GetRateLimitConfig()
	r.cli.HandleError(err, "getting rate limit config")

	r.cli.Println("Rate Limit Configuration:")
	r.cli.Printf("  Enabled: %t\n", rlConfig.Config.Enabled)
	r.cli.Printf("  Requests per second: %.1f\n", rlConfig.Config.RequestsPerSec)
	r.cli.Printf("  Burst: %d\n", rlConfig.Config.Burst)
	r.cli.Printf("  By IP: %t\n", rlConfig.Config.ByIP)
	r.cli.Printf("  By API Key: %t\n", rlConfig.Config.ByAPIKey)
	r.cli.Printf("  Cleanup interval: %s\n", rlConfig.Config.CleanupInterval)
}

// Clients lists active rate-limited clients
func (r *RateLimitCommands) Clients(args []string) {
	var clientType string
	flagSet := flag.NewFlagSet("clients", flag.ContinueOnError)
	flagSet.SetOutput(r.cli.Error)
	flagSet.StringVar(&clientType, "type", "all", "Client type: all, ip, or apikey")

	config, remaining, err := r.cli.ParseGlobalFlags(args, "clients")
	if err == flag.ErrHelp {
		r.cli.Println("Usage: konsulctl ratelimit clients [--type <type>] [options]")
		return
	}
	r.cli.HandleError(err, "parsing flags")

	err = flagSet.Parse(remaining)
	r.cli.HandleError(err, "parsing client flags")

	client := r.cli.CreateClient(config)
	clients, err := client.GetRateLimitClients(clientType)
	r.cli.HandleError(err, "getting rate limit clients")

	if clients.Count == 0 {
		r.cli.Println("No active rate-limited clients")
		return
	}

	r.cli.Printf("Active Rate-Limited Clients (%d):\n", clients.Count)
	r.cli.Println()
	for _, c := range clients.Clients {
		r.cli.Printf("  Identifier: %s\n", c.Identifier)
		r.cli.Printf("  Type: %s\n", c.Type)
		r.cli.Printf("  Tokens: %.2f / %d\n", c.Tokens, c.MaxTokens)
		r.cli.Printf("  Rate: %.1f req/s\n", c.Rate)
		r.cli.Printf("  Last update: %s\n", c.LastUpdate)
		r.cli.Println()
	}
}

// ClientStatus shows status for a specific client
func (r *RateLimitCommands) ClientStatus(args []string) {
	config, remaining, err := r.cli.ParseGlobalFlags(args, "client")
	if err == flag.ErrHelp {
		r.cli.Println("Usage: konsulctl ratelimit client <identifier> [options]")
		return
	}
	r.cli.HandleError(err, "parsing flags")
	r.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl ratelimit client <identifier>")

	identifier := remaining[0]
	client := r.cli.CreateClient(config)
	status, err := client.GetRateLimitClientStatus(identifier)
	r.cli.HandleError(err, "getting client status")

	r.cli.Printf("Client Status: %s\n", status.Identifier)
	r.cli.Printf("  Type: %s\n", status.Type)
	r.cli.Printf("  Tokens: %.2f / %d\n", status.Tokens, status.MaxTokens)
	r.cli.Printf("  Rate: %.1f req/s\n", status.Rate)
	r.cli.Printf("  Last update: %s\n", status.LastUpdate)

	// Show percentage
	percentage := (status.Tokens / float64(status.MaxTokens)) * 100
	r.cli.Printf("  Capacity: %.1f%%\n", percentage)
}

// Reset handles rate limit reset operations
func (r *RateLimitCommands) Reset(args []string) {
	config, remaining, err := r.cli.ParseGlobalFlags(args, "reset")
	if err == flag.ErrHelp {
		r.cli.Println("Usage: konsulctl ratelimit reset <ip|apikey|all> <value> [options]")
		return
	}
	r.cli.HandleError(err, "parsing flags")
	r.cli.ValidateMinArgs(remaining, 1, "Usage: konsulctl ratelimit reset <ip|apikey|all> <value>")

	resetType := remaining[0]
	client := r.cli.CreateClient(config)

	switch resetType {
	case "ip":
		r.cli.ValidateExactArgs(remaining, 2, "Usage: konsulctl ratelimit reset ip <ip-address>")
		ip := remaining[1]
		err := client.ResetRateLimitIP(ip)
		r.cli.HandleError(err, "resetting rate limit for IP")
		r.cli.Printf("Successfully reset rate limit for IP: %s\n", ip)

	case "apikey":
		r.cli.ValidateExactArgs(remaining, 2, "Usage: konsulctl ratelimit reset apikey <key-id>")
		keyID := remaining[1]
		err := client.ResetRateLimitAPIKey(keyID)
		r.cli.HandleError(err, "resetting rate limit for API key")
		r.cli.Printf("Successfully reset rate limit for API key: %s\n", keyID)

	case "all":
		var limiterType string
		flagSet := flag.NewFlagSet("reset-all", flag.ContinueOnError)
		flagSet.SetOutput(r.cli.Error)
		flagSet.StringVar(&limiterType, "type", "all", "Limiter type: all, ip, or apikey")
		err := flagSet.Parse(remaining[1:])
		r.cli.HandleError(err, "parsing reset flags")

		err = client.ResetRateLimitAll(limiterType)
		r.cli.HandleError(err, "resetting rate limiters")
		r.cli.Printf("Successfully reset all %s rate limiters\n", limiterType)

	default:
		r.cli.Errorf("Unknown reset type: %s\n", resetType)
		r.cli.Errorln("Available: ip, apikey, all")
		r.cli.Exit(1)
	}
}

// Update updates rate limit configuration
func (r *RateLimitCommands) Update(args []string) {
	var rate float64
	var burst int

	flagSet := flag.NewFlagSet("update", flag.ContinueOnError)
	flagSet.SetOutput(r.cli.Error)
	flagSet.Float64Var(&rate, "rate", 0, "Requests per second")
	flagSet.IntVar(&burst, "burst", 0, "Burst size")

	config, remaining, err := r.cli.ParseGlobalFlags(args, "update")
	if err == flag.ErrHelp {
		r.cli.Println("Usage: konsulctl ratelimit update --rate <n> --burst <n> [options]")
		return
	}
	r.cli.HandleError(err, "parsing flags")

	err = flagSet.Parse(remaining)
	r.cli.HandleError(err, "parsing update flags")

	if rate == 0 && burst == 0 {
		r.cli.Errorln("Usage: konsulctl ratelimit update --rate <n> --burst <n>")
		r.cli.Errorln("  At least one of --rate or --burst must be specified")
		r.cli.Exit(1)
		return
	}

	client := r.cli.CreateClient(config)

	var ratePtr *float64
	var burstPtr *int
	if rate > 0 {
		ratePtr = &rate
	}
	if burst > 0 {
		burstPtr = &burst
	}

	resp, err := client.UpdateRateLimitConfig(ratePtr, burstPtr)
	r.cli.HandleError(err, "updating rate limit config")

	r.cli.Printf("%s\n", resp.Message)
	if resp.Config.RequestsPerSec > 0 || resp.Config.Burst > 0 {
		r.cli.Println("Updated configuration:")
		if resp.Config.RequestsPerSec > 0 {
			r.cli.Printf("  Requests per second: %.1f\n", resp.Config.RequestsPerSec)
		}
		if resp.Config.Burst > 0 {
			r.cli.Printf("  Burst: %d\n", resp.Config.Burst)
		}
		r.cli.Println()
		r.cli.Println("Note: Changes apply to new limiters only.")
		r.cli.Println("To apply to existing clients, run: konsulctl ratelimit reset all")
	}
}
