package plugin

import (
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"

	"netbird-coredns/internal/api"
)

type record struct {
	IPv4 net.IP
}

// NetBird represents the NetBird CoreDNS plugin
type NetBird struct {
	Next    plugin.Handler
	Domains []string
	storage *api.Storage
}

// New creates a new NetBird plugin instance
func New(domains []string) (*NetBird, error) {
	nb := &NetBird{
		Domains: domains,
	}

	// Initialize storage from environment variable
	recordsFile := os.Getenv("NBDNS_RECORDS_FILE")
	if recordsFile == "" {
		recordsFile = "/etc/nb-dns/records/records.json"
	}

	storage, err := api.NewStorage(recordsFile)
	if err != nil {
		clog.Errorf("Failed to initialize storage: %v", err)
		return nil, err
	}

	nb.storage = storage
	clog.Infof("Initialized storage with records file: %s", recordsFile)

	// Start periodic refresh for storage
	go nb.periodicRefresh()

	return nb, nil
}

// Initialize sets up the storage after configuration is loaded
func (n *NetBird) Initialize(storage *api.Storage) {
	n.storage = storage

	// Start periodic refresh
	go n.periodicRefresh()
}

// getRefreshInterval returns the refresh interval in seconds from environment variable
func getRefreshInterval() time.Duration {
	if intervalStr := os.Getenv("NBDNS_REFRESH_INTERVAL"); intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil && interval > 0 {
			return time.Duration(interval) * time.Second
		}
		clog.Warningf("invalid NBDNS_REFRESH_INTERVAL value '%s', using default 30 seconds", intervalStr)
	}
	return 30 * time.Second
}

// periodicRefresh periodically reloads the DNS records from disk
func (n *NetBird) periodicRefresh() {
	interval := getRefreshInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial refresh
	n.refresh()

	for range ticker.C {
		n.refresh()
	}
}

// refresh reloads custom DNS records from disk
func (n *NetBird) refresh() {
	// Reload custom DNS records from disk
	if n.storage != nil {
		if err := n.storage.Reload(); err != nil {
			clog.Errorf("failed to reload storage from disk: %v", err)
		} else {
			clog.Debugf("Reloaded custom DNS records from disk")
		}
	}
}

// lookupCustomRecord checks for custom DNS records in storage
func (n *NetBird) lookupCustomRecord(queryName string) (record, bool) {
	if n.storage == nil {
		return record{}, false
	}

	// Parse domain and name from query
	// queryName is in format: "name.domain."
	parts := strings.Split(strings.TrimSuffix(queryName, "."), ".")
	if len(parts) < 2 {
		return record{}, false
	}

	name := parts[0]
	domain := strings.Join(parts[1:], ".")

	clog.Debugf("Looking up custom record: domain=%s, name=%s", domain, name)
	customRecord, err := n.storage.GetRecord(domain, name)
	if err != nil {
		clog.Debugf("Custom record lookup failed: %v", err)
		return record{}, false
	}
	clog.Debugf("Found custom record: %+v", customRecord)

	var rec record

	switch customRecord.Type {
	case "A":
		rec.IPv4 = net.ParseIP(customRecord.Value)
	case "CNAME":
		// For CNAME, we need to resolve the target
		// This is handled differently in serve.go
		return record{}, false
	}

	return rec, true
}


// Name returns the plugin name
func (n *NetBird) Name() string {
	return "netbird"
}

// ResolveCNAME resolves a CNAME record from storage
func (n *NetBird) ResolveCNAME(queryName string) (string, bool) {
	if n.storage == nil {
		return "", false
	}

	// Parse domain and name from query
	parts := strings.Split(strings.TrimSuffix(queryName, "."), ".")
	if len(parts) < 2 {
		return "", false
	}

	name := parts[0]
	domain := strings.Join(parts[1:], ".")

	customRecord, err := n.storage.GetRecord(domain, name)
	if err != nil {
		return "", false
	}

	if customRecord.Type == "CNAME" {
		// Ensure CNAME value ends with dot
		target := customRecord.Value
		if !strings.HasSuffix(target, ".") {
			target += "."
		}
		return target, true
	}

	return "", false
}
