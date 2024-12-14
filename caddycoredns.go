package caddycoredns

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

type CaddyDNS struct {
	Next     plugin.Handler
	proxyMap map[string]string // map[hostname]upstreamIP
	mutex    sync.RWMutex
}

// refreshProxyInfo fetches the current proxy configuration from Caddy's admin API
func (c *CaddyDNS) refreshProxyInfo() error {
	resp, err := http.Get("http://caddy:2019/config/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Clear existing mappings
	c.proxyMap = make(map[string]string)

	// Parse the Caddy config to extract reverse proxy mappings
	// This is a simplified example - you'll need to adjust the parsing
	// based on your actual Caddy configuration structure
	if apps, ok := config["apps"].(map[string]interface{}); ok {
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				log.Printf("%+v", servers)
				// Parse server blocks and extract proxy information
				// Add to c.proxyMap
			}
		}
	}

	return nil
}

func (c *CaddyDNS) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	// Handle only A record queries
	if len(r.Question) == 0 || r.Question[0].Qtype != dns.TypeA {
		return plugin.NextOrFailure(c.Name(), c.Next, ctx, w, r)
	}

	qname := r.Question[0].Name

	c.mutex.RLock()
	upstreamIP, exists := c.proxyMap[qname]
	c.mutex.RUnlock()

	if exists {
		// Construct DNS response
		resp := new(dns.Msg)
		resp.SetReply(r)

		rr := new(dns.A)
		rr.Hdr = dns.RR_Header{
			Name:   qname,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    300,
		}
		rr.A = net.ParseIP(upstreamIP)

		resp.Answer = []dns.RR{rr}
		w.WriteMsg(resp)
		return dns.RcodeSuccess, nil
	}

	return plugin.NextOrFailure(c.Name(), c.Next, ctx, w, r)
}

func (c *CaddyDNS) Name() string { return "caddydns" }
