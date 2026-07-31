package security

import (
	"context"
	"net/netip"
	"testing"
)

// fakeResolver returns the specified IPs for any hostname. It is used to test
// SSRF rejection logic without making real DNS calls.
type fakeResolver struct {
	ips []netip.Addr
	err error
}

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]netip.Addr, error) {
	return f.ips, f.err
}

func mustIP(s string) netip.Addr {
	return netip.MustParseAddr(s)
}

func TestCheckURLAllowed(t *testing.T) {
	g := NewGuard([]string{"www.baiscope.lk", "cineru.lk", "piratelk.com", "zoom.lk"})
	g.SetResolver(fakeResolver{ips: []netip.Addr{mustIP("93.184.216.34")}})

	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://www.baiscope.lk/inception", false},
		{"http://cineru.lk/inception", false},
		{"https://piratelk.com/inception", false},
		{"http://zoom.lk/inception", false},
		{"https://zoom.lk/", false},
		// Wrong host.
		{"https://evil.com/malware", true},
		{"http://baiscopelk.com/inception", true}, // not in allowlist (www. missing)
		// Wrong scheme.
		{"ftp://www.baiscope.lk/file", true},
	}
	for _, tt := range tests {
		err := g.CheckURL(context.Background(), tt.url)
		gotErr := err != nil
		if gotErr != tt.wantErr {
			t.Errorf("CheckURL(%q) error = %v, wantErr = %v", tt.url, err, tt.wantErr)
		}
	}
}

func TestCheckURLPrivateIP(t *testing.T) {
	g := NewGuard([]string{"www.baiscope.lk"})

	tests := []struct {
		name string
		ip   netip.Addr
	}{
		{"loopback", mustIP("127.0.0.1")},
		{"private 10.x", mustIP("10.0.0.1")},
		{"private 192.168", mustIP("192.168.1.1")},
		{"private 172.16", mustIP("172.16.0.1")},
		{"link-local unicast", mustIP("169.254.1.1")},
		{"cloud metadata", mustIP("169.254.169.254")},
		{"unspecified", mustIP("0.0.0.0")},
		{"multicast", mustIP("224.0.0.1")},
		{"IPv6 loopback", mustIP("::1")},
		{"IPv6 private", mustIP("fc00::1")},
		{"IPv6 link-local", mustIP("fe80::1")},
	}

	for _, tt := range tests {
		g.SetResolver(fakeResolver{ips: []netip.Addr{tt.ip}})
		err := g.CheckURL(context.Background(), "https://www.baiscope.lk/inception")
		if err == nil {
			t.Errorf("CheckURL with %s resolved to %s: expected error, got nil", tt.name, tt.ip)
		} else {
			t.Logf("%s -> %s: %v", tt.name, tt.ip, err)
		}
	}
}

func TestCheckURLDNSRebinding(t *testing.T) {
	// User-provided URL passes the hostname check but DNS returns a private IP.
	g := NewGuard([]string{"www.baiscope.lk"})
	g.SetResolver(fakeResolver{ips: []netip.Addr{mustIP("10.0.0.1")}})

	err := g.CheckURL(context.Background(), "https://www.baiscope.lk/inception")
	if err == nil {
		t.Fatal("expected error for DNS-rebinding attack (hostname in allowlist but resolves to private IP), got nil")
	}
	t.Logf("DNS rebinding blocked: %v", err)
}

func TestAllowedHosts(t *testing.T) {
	sources := []BaseURLProvider{
		testBaseURLProvider{baseURL: "https://www.baiscope.lk"},
		testBaseURLProvider{baseURL: "https://cineru.lk"},
		testBaseURLProvider{baseURL: "https://piratelk.com"},
		testBaseURLProvider{baseURL: "https://zoom.lk"},
	}
	hosts := AllowedHosts(sources)
	if len(hosts) != 4 {
		t.Fatalf("expected 4 hosts, got %d: %v", len(hosts), hosts)
	}
	hostSet := make(map[string]bool)
	for _, h := range hosts {
		hostSet[h] = true
	}
	for _, want := range []string{"www.baiscope.lk", "cineru.lk", "piratelk.com", "zoom.lk"} {
		if !hostSet[want] {
			t.Errorf("missing host %q in %v", want, hosts)
		}
	}
}

type testBaseURLProvider struct {
	baseURL string
}

func (t testBaseURLProvider) BaseURL() string {
	return t.baseURL
}
