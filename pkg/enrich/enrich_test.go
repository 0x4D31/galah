package enrich

import (
	"testing"
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
			scanner, err := IsKnownScanner(tt.ip, tt.host)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if scanner != tt.scanner {
				t.Errorf("Expected %v, got %v", tt.scanner, scanner)
			}
		})
	}
}
