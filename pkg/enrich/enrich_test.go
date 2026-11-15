package enrich

import (
	"net"
	"testing"
	"time"
)

func TestIsKnownScanner(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		host    []string
		scanner string
	}{
		{
			name:    "knownScannerCensysIP",
			ip:      "162.142.125.10",
			host:    []string{""},
			scanner: "censys scanner",
		},
		{
			name:    "knownScannerShodanHostname",
			ip:      "1.1.1.1",
			host:    []string{"test.shodan.io."},
			scanner: "shodan scanner",
		},
		{
			name:    "unknownIP",
			ip:      "127.0.0.1",
			host:    []string{""},
			scanner: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := isKnownScanner(tt.ip, tt.host)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if scanner != tt.scanner {
				t.Errorf("Expected %v, got %v", tt.scanner, scanner)
			}
		})
	}
}

func TestProcessNoPTR(t *testing.T) {
	orig := lookupAddr
	defer func() { lookupAddr = orig }()
	calls := 0
	lookupAddr = func(ip string) ([]string, error) {
		calls++
		return nil, &net.DNSError{Err: "no such host", Name: ip, IsNotFound: true}
	}
	e := New(Config{CacheSize: 10, CacheTTL: time.Hour})

	// A Censys address without a PTR record: no error, still tagged, and cached.
	info, err := e.Process("162.142.125.10")
	if err != nil {
		t.Fatalf("missing PTR must not be an error, got %v", err)
	}
	if info.KnownScanner != "censys scanner" || info.Host != "" {
		t.Errorf("got %+v, want censys scanner with no host", info)
	}
	if _, err := e.Process("162.142.125.10"); err != nil || calls != 1 {
		t.Errorf("second lookup should come from the cache (calls=%d, err=%v)", calls, err)
	}

	// A real DNS failure is still reported.
	lookupAddr = func(ip string) ([]string, error) {
		return nil, &net.DNSError{Err: "i/o timeout", Name: ip, IsTimeout: true}
	}
	if _, err := e.Process("198.51.100.9"); err == nil {
		t.Error("a DNS timeout should still be an error")
	}
}
