package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

func init() {
	// Override DNS resolution only when /etc/resolv.conf is missing
	// (for example on Android-like environments).
	if _, err := os.Stat("/etc/resolv.conf"); err == nil {
		return
	}

	// Read DNS servers from the environment, separated by ";".
	// Example: JAMECLAW_DNS_SERVER="8.8.8.8:53;1.1.1.1:53;223.5.5.5:53"
	dnsEnv := os.Getenv("JAMECLAW_DNS_SERVER")
	if dnsEnv == "" {
		dnsEnv = "8.8.8.8:53;1.1.1.1:53"
	}

	var dnsServers []string
	for _, s := range strings.Split(dnsEnv, ";") {
		s = strings.TrimSpace(s)
		if s != "" {
			// Add the default DNS port when one is not provided.
			if _, _, err := net.SplitHostPort(s); err != nil {
				s = s + ":53"
			}
			dnsServers = append(dnsServers, s)
		}
	}

	// Round-robin index across configured DNS servers.
	var idx uint64

	customResolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			// Try the configured DNS servers in round-robin order.
			server := dnsServers[atomic.AddUint64(&idx, 1)%uint64(len(dnsServers))]
			return d.DialContext(ctx, "udp", server)
		},
	}

	// Replace the global resolver.
	net.DefaultResolver = customResolver

	// Replace the default HTTP transport dialer so it uses the custom resolver.
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  customResolver,
	}

	if tr, ok := http.DefaultTransport.(*http.Transport); ok {
		tr.DialContext = dialer.DialContext
	}
}
