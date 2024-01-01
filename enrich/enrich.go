package enrich

import (
	"errors"
	"log"
	"net"
	"strings"
	"time"

	"github.com/bluele/gcache"
)

type LookupInfo struct {
	Host         string
	KnownScanner string
}

type Config struct {
	CacheSize int
	CacheTTL  time.Duration
}

type Default struct {
	Cache    gcache.Cache
	CacheTTL time.Duration
}

// The list is taken from https://github.com/mushorg/glutton.
var scannerSubnets = map[string][]string{
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

func New(conf *Config) *Default {
	return &Default{
		Cache:    gcache.New(conf.CacheSize).LFU().Build(),
		CacheTTL: conf.CacheTTL,
	}
}

func (e *Default) Process(ip string) (*LookupInfo, error) {
	// Check the cache
	val, err := e.Cache.Get(ip)
	if err != nil {
		if errors.Is(err, gcache.KeyNotFoundError) {
			// log.Printf("Enrichment cache miss for IP %q: %s", ip, err)
		} else {
			log.Printf("Failed to retrieve %q from cache: %s", ip, err)
		}
	}
	if l, ok := val.(LookupInfo); ok && l != (LookupInfo{}) {
		return &l, nil
	}

	hosts, err := reverseIPLookup(ip)
	if err != nil {
		return (&LookupInfo{}), err
	}
	host := hosts[0]

	scanner, err := isKnownScanner(ip, hosts)
	if err != nil {
		log.Println(err)
	}

	// Update the cache
	if err := e.Cache.SetWithExpire(ip, LookupInfo{Host: host, KnownScanner: scanner}, e.CacheTTL); err != nil {
		log.Printf("Failed to update cache for IP %q: %s", ip, err)
	}

	return &LookupInfo{
		Host:         host,
		KnownScanner: scanner,
	}, nil
}

func reverseIPLookup(ip string) ([]string, error) {
	names, err := net.LookupAddr(ip)
	if err != nil {
		return nil, err
	}
	return names, nil
}

// IsKnownScanner checks if the given IP belongs to a known scanner.
func isKnownScanner(ip string, hosts []string) (string, error) {
	parsedIP := net.ParseIP(ip)

	for scanner, subnets := range scannerSubnets {
		for _, subnet := range subnets {
			_, net, err := net.ParseCIDR(subnet)
			if err != nil {
				return "", err
			}
			if net.Contains(parsedIP) {
				return scanner, nil
			}
		}
	}

	for _, host := range hosts {
		if strings.HasSuffix(host, "shodan.io.") {
			return "shodan scanner", nil
		}
		if strings.HasSuffix(host, "censys-scanner.com.") {
			return "shodan scanner", nil
		}
		if strings.HasSuffix(host, "binaryedge.ninja.") {
			return "binaryedge scanner", nil
		}
		if strings.HasSuffix(host, "rwth-aachen.de.") {
			return "rwth scanner", nil
		}
	}
	return "", nil
}
