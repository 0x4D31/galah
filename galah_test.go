package main

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

func TestStartServers(t *testing.T) {
	tests := []struct {
		name       string
		portConfig PortConfig
		wantErr    bool
	}{
		{
			name: "Start server with unknown protocol",
			portConfig: PortConfig{
				Port:     8081,
				Protocol: []string{"UNKNOWN_PROTOCOL"},
			},
			wantErr: true,
		},
	}

	app := &App{Config: &Config{}}

	for _, tt := range tests {
		app.Config.Ports = []PortConfig{tt.portConfig}
		t.Run(tt.name, func(t *testing.T) {
			err := app.startServers()
			log.Println(err)

			if (err != nil) != tt.wantErr {
				t.Errorf("StartServers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStartHTTPServer(t *testing.T) {
	tests := []struct {
		name       string
		portConfig PortConfig
		wantErr    bool
	}{
		{
			name: "Start server on an already occupied port",
			portConfig: PortConfig{
				Port:     8080, // Use a known port
				Protocol: []string{"HTTP"},
			},
			wantErr: true,
		},
	}

	app := &App{Config: &Config{}}

	// Start a dummy server on port 8080 to occupy it
	dummyServer := &http.Server{Addr: ":8080"}
	go dummyServer.ListenAndServe()
	time.Sleep(100 * time.Millisecond) // Give it some time to start

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := app.setupServer(tt.portConfig)

			var g errgroup.Group
			g.Go(func() error {
				return app.startHTTPServer(server, tt.portConfig)
			})

			// Wait for the goroutine to complete and get the error
			err := g.Wait()
			log.Println(err)

			if (err != nil) != tt.wantErr {
				t.Errorf("startHTTPServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Cleanup: Shutdown the dummy server
	dummyServer.Shutdown(context.Background())
}

func TestStartTLSServer(t *testing.T) {
	tests := []struct {
		name       string
		portConfig PortConfig
		wantErr    bool
	}{
		{
			name: "Start server with invalid TLS profile",
			portConfig: PortConfig{
				Port:       8444,
				Protocol:   []string{"TLS"},
				TLSProfile: "invalidTLSProfile",
			},
			wantErr: true,
		},
		{
			name: "Start server with missing TLS profile",
			portConfig: PortConfig{
				Port:     8445,
				Protocol: []string{"TLS"},
			},
			wantErr: true,
		},
	}

	app := &App{
		Config: &Config{
			TLS: map[string]TLSConfig{
				"invalidTLSProfile": {
					Certificate: "non-existent_cert.pem",
					Key:         "non-existent_key.pem",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := app.setupServer(tt.portConfig)
			err := app.startTLSServer(server, tt.portConfig)

			if (err != nil) != tt.wantErr {
				t.Errorf("startTLSServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
