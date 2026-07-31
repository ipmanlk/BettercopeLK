// Package security validates that user-supplied URLs are safe to fetch. The
// guard uses a host allowlist (derived from enabled source configs at startup)
// and rejects requests whose resolved IPs are private or loopback addresses.
package security

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

// Resolver looks up IP addresses for a hostname. This abstraction exists so
// tests can inject a fake resolver that returns arbitrary IPs without touching
// the real DNS.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]netip.Addr, error)
}

// netResolver uses the system DNS resolver.
type netResolver struct{}

func (netResolver) LookupIPAddr(ctx context.Context, host string) ([]netip.Addr, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]netip.Addr, 0, len(addrs))
	for _, a := range addrs {
		if ip, ok := netip.AddrFromSlice(a.IP); ok {
			ips = append(ips, ip.Unmap())
		}
	}
	return ips, nil
}

// Guard holds an allowlist of permitted hosts and a resolver for IP lookups.
type Guard struct {
	allowedHosts map[string]struct{}
	resolver     Resolver
}

// NewGuard creates a Guard from the given host allowlist. Each entry should be
// a lowercased hostname with no port (e.g. "www.baiscope.lk").
func NewGuard(allowedHosts []string) *Guard {
	g := &Guard{
		allowedHosts: make(map[string]struct{}, len(allowedHosts)),
		resolver:     netResolver{},
	}
	for _, h := range allowedHosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h != "" {
			g.allowedHosts[h] = struct{}{}
		}
	}
	return g
}

// BaseURLProvider is the interface AllowedHosts needs to extract base URLs
// from any source-like type.
type BaseURLProvider interface {
	BaseURL() string
}

// AllowedHosts extracts the host portion of each provider's BaseURL and
// returns them as a string slice suitable for NewGuard.
func AllowedHosts(providers []BaseURLProvider) []string {
	seen := make(map[string]bool)
	var hosts []string
	for _, p := range providers {
		u, err := url.Parse(p.BaseURL())
		if err != nil {
			continue
		}
		h := strings.ToLower(u.Host)
		if h != "" && !seen[h] {
			seen[h] = true
			hosts = append(hosts, h)
		}
	}
	return hosts
}

// CheckURL validates rawURL against the guard's allowlist and resolves its
// host to ensure it does not point at a private, loopback, or otherwise
// internal IP address.
func (g *Guard) CheckURL(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme %q not allowed (must be http or https)", u.Scheme)
	}

	host := strings.ToLower(u.Hostname())
	if _, ok := g.allowedHosts[host]; !ok {
		return fmt.Errorf("host %q is not in the allowlist", host)
	}

	ips, err := g.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolving host %q: %w", host, err)
	}

	for _, ip := range ips {
		if err := g.checkIP(ip); err != nil {
			return fmt.Errorf("host %q resolved to blocked IP %s: %w", host, ip, err)
		}
	}

	return nil
}

// checkIP returns an error if ip is a private, loopback, link-local,
// unspecified, multicast, or well-known metadata endpoint address.
func (g *Guard) checkIP(ip netip.Addr) error {
	switch {
	case ip.IsLoopback():
		return fmt.Errorf("loopback address")
	case ip.IsPrivate():
		return fmt.Errorf("private address")
	case ip.IsLinkLocalUnicast():
		return fmt.Errorf("link-local unicast address")
	case ip.IsLinkLocalMulticast():
		return fmt.Errorf("link-local multicast address")
	case ip.IsUnspecified():
		return fmt.Errorf("unspecified address")
	case ip.IsMulticast():
		return fmt.Errorf("multicast address")
	case ip == netip.MustParseAddr("169.254.169.254"):
		return fmt.Errorf("cloud metadata endpoint")
	}
	return nil
}

// SetResolver replaces the default system resolver. Used by tests to inject a
// fake resolver.
func (g *Guard) SetResolver(r Resolver) {
	g.resolver = r
}
