package dns

import (
	"fmt"
	"net"
	"strings"
)

// RecordType represents the type of DNS record
type RecordType string

const (
	RecordTypeA     RecordType = "A"
	RecordTypeCNAME RecordType = "CNAME"
)

// Record represents a DNS record
type Record struct {
	Name   string     `json:"name"`
	Domain string     `json:"domain"`
	Type   RecordType `json:"type"`
	Value  string     `json:"value"`
	TTL    uint32     `json:"ttl,omitempty"`
}

// Validate checks if a record is valid
func (r *Record) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("record name cannot be empty")
	}
	if r.Domain == "" {
		return fmt.Errorf("record domain cannot be empty")
	}
	if r.Type == "" {
		return fmt.Errorf("record type cannot be empty")
	}
	if r.Value == "" {
		return fmt.Errorf("record value cannot be empty")
	}

	// Validate based on type
	switch r.Type {
	case RecordTypeA:
		if ip := net.ParseIP(r.Value); ip == nil || ip.To4() == nil {
			return fmt.Errorf("invalid IPv4 address: %s", r.Value)
		}
	case RecordTypeCNAME:
		// CNAME value should be a valid domain name
		if !isValidDomain(r.Value) {
			return fmt.Errorf("invalid CNAME target: %s", r.Value)
		}
	default:
		return fmt.Errorf("unsupported record type: %s", r.Type)
	}

	return nil
}

// FQDN returns the fully qualified domain name for this record
func (r *Record) FQDN() string {
	return fmt.Sprintf("%s.%s.", r.Name, r.Domain)
}

// isValidDomain checks if a string is a valid domain name
func isValidDomain(domain string) bool {
	domain = strings.TrimSuffix(domain, ".")
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		// Check if label contains only valid characters
		for _, c := range label {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
		// Label cannot start or end with hyphen
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
	}

	return true
}
