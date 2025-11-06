package main

import (
	"fmt"
	"os"
	"strings"

	"netbird-coredns/internal/api"
	"netbird-coredns/internal/config"
	"netbird-coredns/internal/logger"
	"netbird-coredns/internal/process"
	"netbird-coredns/internal/template"
)

const banner = `
███╗   ██╗███████╗████████╗██████╗ ██╗██████╗ ██████╗      ██████╗ ██████╗ ██████╗ ███████╗██████╗ ███╗   ██╗███████╗
████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██║██╔══██╗██╔══██╗    ██╔════╝██╔═══██╗██╔══██╗██╔════╝██╔══██╗████╗  ██║██╔════╝
██╔██╗ ██║█████╗     ██║   ██████╔╝██║██████╔╝██║  ██║    ██║     ██║   ██║██████╔╝█████╗  ██║  ██║██╔██╗ ██║███████╗
██║╚██╗██║██╔══╝     ██║   ██╔══██╗██║██╔══██╗██║  ██║    ██║     ██║   ██║██╔══██╗██╔══╝  ██║  ██║██║╚██╗██║╚════██║
██║ ╚████║███████╗   ██║   ██████╔╝██║██║  ██║██████╔╝    ╚██████╗╚██████╔╝██║  ██║███████╗██████╔╝██║ ╚████║███████║
╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═════╝ ╚═╝╚═╝  ╚═╝╚═════╝      ╚═════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═════╝ ╚═╝  ╚═══╝╚══════╝

A CoreDNS plugin for managing custom DNS records via API.

By Christian De Leon (https://github.com/christian-deleon/netbird-coredns)
`

var (
	globalProcessManager *process.Manager
	globalConfig         *config.Config
)

func main() {
	// Check for help flag
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help") {
		printUsage()
		os.Exit(0)
	}

	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic occurred: %v", r)
			panic(r) // Re-panic to maintain original behavior
		}
	}()

	// Load configuration first to get log level
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Set log level before any logging
	if err := logger.SetLevel(cfg.LogLevel); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set log level: %v\n", err)
		os.Exit(1)
	}

	logger.Print(banner)
	logger.Info("Starting netbird-coredns service...")

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Fatal("Invalid configuration: %v", err)
	}

	logger.Info("Configuration loaded:")
	logger.Info("  Domains: %s", strings.Join(cfg.Domains, ", "))
	logger.Info("  Forward to: %s", cfg.ForwardTo)
	logger.Info("  DNS Port: %d", cfg.DNSPort)
	logger.Info("  API Port: %d", cfg.APIPort)
	logger.Info("  Refresh interval: %d seconds", cfg.RefreshInterval)
	logger.Info("  Records file: %s", cfg.RecordsFile)
	logger.Info("  Log level: %s", cfg.LogLevel)

	// Initialize DNS records storage
	logger.Info("Initializing DNS records storage...")
	storage, err := api.NewStorage(cfg.RecordsFile)
	if err != nil {
		logger.Fatal("Failed to initialize storage: %v", err)
	}
	logger.Info("DNS records storage initialized")

	// Note: The plugin is initialized by CoreDNS when it loads the plugin
	// CoreDNS will create its own plugin instance via plugin.New() which handles
	// storage initialization from environment variables

	// Start HTTP API server
	logger.Info("Starting DNS records API server...")
	apiServer := api.NewServer(storage, cfg.APIPort)
	if err := apiServer.Start(); err != nil {
		logger.Fatal("Failed to start API server: %v", err)
	}
	logger.Info("API server started on port %d", cfg.APIPort)

	// Generate Corefile
	logger.Info("Generating Corefile...")
	generator, err := template.NewGenerator()
	if err != nil {
		logger.Fatal("Failed to create template generator: %v", err)
	}

	corefilePath := "/Corefile"
	if err := generator.WriteCorefile(cfg, corefilePath); err != nil {
		logger.Fatal("Failed to generate Corefile: %v", err)
	}

	// Print generated Corefile
	corefileContent, _ := generator.GenerateCorefile(cfg)
	logger.Debug("Generated Corefile:")
	logger.Debug("%s", corefileContent)

	// Create process manager
	processManager := process.NewManager(cfg)
	globalProcessManager = processManager
	globalConfig = cfg


	// Start CoreDNS
	logger.Info("Starting CoreDNS...")
	if err := processManager.StartCoreDNS(corefilePath); err != nil {
		logger.Fatal("Failed to start CoreDNS: %v", err)
	}

	logger.Info("All services started successfully")
	logger.Info("Service is ready and waiting for connections...")
	logger.Info("  DNS Server: port %d (UDP/TCP)", cfg.DNSPort)
	logger.Info("  API Server: http://localhost:%d", cfg.APIPort)
	logger.Info("  Health Check: http://localhost:%d/health", cfg.APIPort)

	// Run with signal handling
	if err := processManager.RunWithSignalHandling(); err != nil {
		logger.Error("Process manager error: %v", err)
	}

	logger.Info("Service shutdown completed successfully")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: %s

Environment Variables (all prefixed with NBDNS_):
  NBDNS_DOMAINS           Comma-separated domains for DNS resolution (required)
  NBDNS_FORWARD_TO        Forward server for unresolved queries (default: 8.8.8.8)
  NBDNS_DNS_PORT          DNS server port (default: 5053)
  NBDNS_API_PORT          API server port (default: 8080)
  NBDNS_REFRESH_INTERVAL  Refresh interval in seconds (default: 30)
  NBDNS_RECORDS_FILE      Path to DNS records file (default: /etc/nb-dns/records/records.json)
  NBDNS_LOG_LEVEL         Log level for the entire service (default: info)

`, os.Args[0])
}
