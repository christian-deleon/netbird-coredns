package plugin

import (
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var log = clog.NewWithPlugin("netbird")

// init registers the netbird plugin with CoreDNS
func init() {
	plugin.Register("netbird", setup)
}

// setup configures the NetBird plugin with the given domains
func setup(c *caddy.Controller) error {
	var domains []string

	c.Next() // 'netbird'

	// Parse all domains on the same line
	for c.NextArg() {
		domain := c.Val()
		// Split by comma if multiple domains are provided together
		if strings.Contains(domain, ",") {
			parts := strings.Split(domain, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					domains = append(domains, trimmed)
				}
			}
		} else {
			domains = append(domains, domain)
		}
	}

	if len(domains) == 0 {
		return c.ArgErr()
	}

	nb, err := New(domains)
	if err != nil {
		return plugin.Error("netbird", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		nb.Next = next
		return nb
	})

	return nil
}
