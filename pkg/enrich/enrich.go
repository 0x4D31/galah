package enrich

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/bluele/gcache"
)

// LookupInfo contains the results of a performed lookup.
type LookupInfo struct {
	Host         string
	KnownScanner string
}

// Config holds configuration settings for enrichment cache.
type Config struct {
	CacheSize int
	CacheTTL  time.Duration
}

// Enricher represents the default enrichment implementation.
type Enricher struct {
	cache gcache.Cache
	ttl   time.Duration
}

// ScannerSubnets contains a list of known scanners' subnet.
// The list is taken from https://github.com/mushorg/glutton.
var ScannerSubnets = map[string][]string{
	"censys scanner": {
		"162.142.125.0/24",
		"167.94.138.0/24",
		"167.94.145.0/24",
		"167.94.146.0/24",
		"167.248.133.0/24",
	},
	"shadowserver scanner": {
		"64.62.202.96/27",
		"66.220.23.112/29",
		"74.82.47.0/26",
		"184.105.139.64/26",
		"184.105.143.128/26",
		"184.105.247.192/26",
		"216.218.206.64/26",
		"141.212.0.0/16",
	},
	"PAN Expanse scanner": {
		"144.86.173.0/24",
	},
	"rwth scanner": {
		"137.226.113.56/26",
	},
}

// New creates a new Enricher instance with the specified configuration.
func New(conf Config) *Enricher {
	return &Enricher{
		cache: gcache.New(conf.CacheSize).LFU().Build(),
		ttl:   conf.CacheTTL,
	}
}

// Process enriches the IP address and stores the result in the enrichment cache.
func (e *Enricher) Process(ip string) (*LookupInfo, error) {
	val, err := e.cache.Get(ip)
	if err == nil {
		if lookupInfo, ok := val.(LookupInfo); ok {
			return &lookupInfo, nil
		}
	}

	hosts, err := reverseIPLookup(ip)
	if err != nil {
		return nil, err
	}

	host := ""
	if len(hosts) > 0 {
		host = hosts[0]
	}

	scanner, err := isKnownScanner(ip, hosts)
	if err != nil {
		return nil, err
	}

	lookupInfo := LookupInfo{Host: host, KnownScanner: scanner}
	if err := e.cache.SetWithExpire(ip, lookupInfo, e.ttl); err != nil {
		return nil, fmt.Errorf("error updating enrichment cache for IP %q: %w", ip, err)
	}

	return &lookupInfo, nil
}

// reverseIPLookup performs a reverse IP lookup and returns the names.
func reverseIPLookup(ip string) ([]string, error) {
	names, err := net.LookupAddr(ip)
	if err != nil {
		return nil, err
	}
	return names, nil
}

// isKnownScanner checks if the given IP belongs to a known scanner.
func isKnownScanner(ip string, hosts []string) (string, error) {
	parsedIP := net.ParseIP(ip)

	for scanner, subnets := range ScannerSubnets {
		for _, subnet := range subnets {
			_, network, err := net.ParseCIDR(subnet)
			if err != nil {
				return "", err
			}
			if network.Contains(parsedIP) {
				return scanner, nil
			}
		}
	}

	for _, host := range hosts {
		switch {
		case strings.HasSuffix(host, "shodan.io."):
			return "shodan scanner", nil
		case strings.HasSuffix(host, "censys-scanner.com."):
			return "censys scanner", nil
		case strings.HasSuffix(host, "binaryedge.ninja."):
			return "binaryedge scanner", nil
		case strings.HasSuffix(host, "rwth-aachen.de."):
			return "rwth scanner", nil
		}
	}
	return "", nil
}
