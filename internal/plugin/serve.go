package plugin

import (
	"context"
	"strings"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// ServeDNS handles DNS requests for the NetBird domains
func (n *NetBird) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	queryName := state.Name()

	// Check if query is for any of our NetBird domains
	matchesDomain := false
	for _, domain := range n.Domains {
		if strings.HasSuffix(queryName, domain+".") {
			matchesDomain = true
			clog.Debugf("Query %s matches configured domain %s", queryName, domain)
			break
		}
	}

	if !matchesDomain {
		clog.Debugf("Query %s does not match any configured domains: %v", queryName, n.Domains)
		return plugin.NextOrFailure(n.Name(), n.Next, ctx, w, r)
	}

	// Check custom records (CNAME)
	if state.QType() == dns.TypeCNAME || state.QType() == dns.TypeA {
		if target, ok := n.ResolveCNAME(queryName); ok {
			m := new(dns.Msg)
			m.SetReply(r)
			m.Authoritative = true

			header := dns.RR_Header{
				Name:   queryName,
				Rrtype: dns.TypeCNAME,
				Class:  state.QClass(),
				Ttl:    60,
			}

			m.Answer = append(m.Answer, &dns.CNAME{
				Hdr:    header,
				Target: target,
			})

			if err := w.WriteMsg(m); err != nil {
				return dns.RcodeServerFailure, err
			}
			return dns.RcodeSuccess, nil
		}
	}

	// Check custom A records
	customRec, ok := n.lookupCustomRecord(queryName)
	if ok {
		clog.Debugf("Found custom record for %s: %v", queryName, customRec)
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true

		header := dns.RR_Header{Name: queryName, Rrtype: state.QType(), Class: state.QClass(), Ttl: 60}

		switch state.QType() {
		case dns.TypeA:
			if customRec.IPv4 != nil {
				m.Answer = append(m.Answer, &dns.A{Hdr: header, A: customRec.IPv4})
				if err := w.WriteMsg(m); err != nil {
					return dns.RcodeServerFailure, err
				}
				return dns.RcodeSuccess, nil
			}
		}
	}

	// No custom records found, pass to next plugin
	return plugin.NextOrFailure(n.Name(), n.Next, ctx, w, r)
}
