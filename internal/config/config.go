package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the netbird-coredns service
type Config struct {
	// General configuration
	LogLevel string

	// DNS configuration
	Domains     []string
	ForwardTo   string
	RecordsFile string
	DNSPort     int

	// API configuration
	APIPort int

	// Refresh settings
	RefreshInterval int
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	config := &Config{}

	// Required: Domains
	domainsStr := os.Getenv("NBDNS_DOMAINS")
	if domainsStr == "" {
		return nil, fmt.Errorf("NBDNS_DOMAINS is required")
	}
	config.Domains = parseDomains(domainsStr)
	if len(config.Domains) == 0 {
		return nil, fmt.Errorf("NBDNS_DOMAINS must contain at least one valid domain")
	}

	// Optional: Forward server
	config.ForwardTo = os.Getenv("NBDNS_FORWARD_TO")
	if config.ForwardTo == "" {
		config.ForwardTo = "8.8.8.8"
	}

	// Optional: DNS port
	dnsPortStr := os.Getenv("NBDNS_DNS_PORT")
	if dnsPortStr != "" {
		port, err := strconv.Atoi(dnsPortStr)
		if err != nil || port <= 0 || port > 65535 {
			return nil, fmt.Errorf("invalid NBDNS_DNS_PORT value: %s", dnsPortStr)
		}
		config.DNSPort = port
	} else {
		config.DNSPort = 5053 // Default to 5053 to avoid conflicts with system DNS (53) and mDNS (5353)
	}

	// Optional: API port
	apiPortStr := os.Getenv("NBDNS_API_PORT")
	if apiPortStr != "" {
		port, err := strconv.Atoi(apiPortStr)
		if err != nil || port <= 0 || port > 65535 {
			return nil, fmt.Errorf("invalid NBDNS_API_PORT value: %s", apiPortStr)
		}
		config.APIPort = port
	} else {
		config.APIPort = 8080
	}

	// Optional: Refresh interval
	intervalStr := os.Getenv("NBDNS_REFRESH_INTERVAL")
	if intervalStr != "" {
		interval, err := strconv.Atoi(intervalStr)
		if err != nil || interval <= 0 {
			return nil, fmt.Errorf("invalid NBDNS_REFRESH_INTERVAL value: %s", intervalStr)
		}
		config.RefreshInterval = interval
	} else {
		config.RefreshInterval = 15
	}

	// Optional: Records file
	config.RecordsFile = os.Getenv("NBDNS_RECORDS_FILE")
	if config.RecordsFile == "" {
		config.RecordsFile = "/etc/nb-dns/records/records.json"
	}

	// Optional: Log level
	logLevel := strings.ToLower(os.Getenv("NBDNS_LOG_LEVEL"))
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if logLevel == "" {
		config.LogLevel = "info"
	} else if validLogLevels[logLevel] {
		config.LogLevel = logLevel
	} else {
		return nil, fmt.Errorf("invalid NBDNS_LOG_LEVEL value: %s. Must be one of: debug, info, warn, error", logLevel)
	}

	return config, nil
}

// Validate ensures all required configuration is present and valid
func (c *Config) Validate() error {
	if len(c.Domains) == 0 {
		return fmt.Errorf("at least one domain is required")
	}

	if c.RefreshInterval <= 0 {
		return fmt.Errorf("refresh interval must be positive")
	}

	if c.APIPort <= 0 || c.APIPort > 65535 {
		return fmt.Errorf("API port must be between 1 and 65535")
	}

	if c.DNSPort <= 0 || c.DNSPort > 65535 {
		return fmt.Errorf("DNS port must be between 1 and 65535")
	}

	return nil
}

// GetPrimaryDomain returns the first domain in the list
func (c *Config) GetPrimaryDomain() string {
	if len(c.Domains) > 0 {
		return c.Domains[0]
	}
	return ""
}

// parseDomains parses a comma-separated list of domains
func parseDomains(domainsStr string) []string {
	return parseList(domainsStr)
}

// parseList parses a comma-separated list of strings
func parseList(listStr string) []string {
	parts := strings.Split(listStr, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
