package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/0x4d31/galah/galah"
	"github.com/0x4d31/galah/internal/app"
	"github.com/0x4d31/galah/internal/config"
	"github.com/0x4d31/galah/internal/server"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func TestStartServers(t *testing.T) {
	tests := []struct {
		name       string
		portConfig config.PortConfig
		wantErr    bool
	}{
		{
			name: "startServerUnknownProtocol",
			portConfig: config.PortConfig{
				Port:     8081,
				Protocol: "UNKNOWN_PROTOCOL",
			},
			wantErr: true,
		},
	}

	a := &app.App{
		Config: &config.Config{},
		Logger: logrus.New(),
	}

	for _, tt := range tests {
		a.Config.Ports = []config.PortConfig{tt.portConfig}
		t.Run(tt.name, func(t *testing.T) {

			srv := server.Server{
				Cache:       a.Cache,
				Config:      a.Config,
				EventLogger: a.EventLogger,
				LLMConfig:   a.LLMConfig,
				Logger:      a.Logger,
				Model:       a.Model,
				Service:     &galah.Service{},
			}
			err := srv.StartServers()

			if (err != nil) != tt.wantErr {
				t.Errorf("StartServers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStartHTTPServer(t *testing.T) {
	tests := []struct {
		name       string
		portConfig config.PortConfig
		wantErr    bool
	}{
		{
			name: "startServerOnOccupiedPort",
			portConfig: config.PortConfig{
				Port:     8080, // Use a known port
				Protocol: "HTTP",
			},
			wantErr: true,
		},
	}

	a := &app.App{
		Config: &config.Config{},
		Logger: logrus.New(),
	}

	// Start a dummy server on port 8080 to occupy it
	dummyServer := &http.Server{Addr: ":8080"}
	go dummyServer.ListenAndServe()
	time.Sleep(100 * time.Millisecond)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := server.Server{
				Cache:       a.Cache,
				Config:      a.Config,
				EventLogger: a.EventLogger,
				LLMConfig:   a.LLMConfig,
				Logger:      a.Logger,
				Model:       a.Model,
				Service:     &galah.Service{},
			}
			HTTPServer := srv.SetupServer(tt.portConfig)

			var g errgroup.Group
			g.Go(func() error {
				return srv.StartHTTPServer(HTTPServer, tt.portConfig)
			})

			err := g.Wait()
			if (err != nil) != tt.wantErr {
				t.Errorf("startHTTPServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	dummyServer.Shutdown(context.Background())
}

func TestStartTLSServer(t *testing.T) {
	tests := []struct {
		name       string
		portConfig config.PortConfig
		wantErr    bool
	}{
		{
			name: "startServerWithInvalidTLSProfile",
			portConfig: config.PortConfig{
				Port:       8444,
				Protocol:   "TLS",
				TLSProfile: "invalidTLSProfile",
			},
			wantErr: true,
		},
		{
			name: "startServerWithMissingTLSProfile",
			portConfig: config.PortConfig{
				Port:     8445,
				Protocol: "TLS",
			},
			wantErr: true,
		},
	}

	a := &app.App{
		Config: &config.Config{
			Profiles: map[string]config.TLSConfig{
				"invalidTLSProfile": {
					Certificate: "non-existent_cert.pem",
					Key:         "non-existent_key.pem",
				},
			},
		},
		Logger: logrus.New(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := server.Server{
				Cache:       a.Cache,
				Config:      a.Config,
				EventLogger: a.EventLogger,
				LLMConfig:   a.LLMConfig,
				Logger:      a.Logger,
				Model:       a.Model,
				Service:     &galah.Service{},
			}
			HTTPServer := srv.SetupServer(tt.portConfig)
			err := srv.StartTLSServer(HTTPServer, tt.portConfig)

			if (err != nil) != tt.wantErr {
				t.Errorf("startTLSServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
