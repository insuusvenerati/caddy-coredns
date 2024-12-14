package caddycoredns

import (
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

// init registers this plugin.
func init() { plugin.Register("caddydns", setup) }

// setup is the function that gets called when the config parser see the token "caddydns". Setup is responsible
// for parsing any extra options the caddydns plugin may have. The first token this function sees is "caddydns".
func setup(c *caddy.Controller) error {
	c.Next() // Skip plugin name
	if c.NextArg() {
		// If there was another token, return an error, because we don't have any configuration.
		// Any errors returned from this setup function should be wrapped with plugin.Error, so we
		// can present a slightly nicer error message to the user.
		return plugin.Error("caddydns", c.ArgErr())
	}

	caddyDNS := &CaddyDNS{
		proxyMap: make(map[string]string),
	}

	// Do initial fetch of proxy information
	if err := caddyDNS.refreshProxyInfo(); err != nil {
		return plugin.Error("caddydns", err)
	}

	// Set up periodic refresh (every 30 seconds)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			caddyDNS.refreshProxyInfo()
		}
	}()

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		caddyDNS.Next = next
		return caddyDNS
	})

	// All OK, return a nil error.
	return nil
}
